package outputnet

import (
	"errors"
	"io"
	stdnet "net"
	"sync"
	"time"
)

var errSenderClosed = errors.New("outputnet: sender is closed")

// DialFunc customizes dialing behavior.
type DialFunc func(network, addr string, timeout time.Duration) (stdnet.Conn, error)

// SenderOption defines transport-level options for network output.
type SenderOption struct {
	Network          string
	Addr             string
	DialTimeout      time.Duration
	WriteTimeout     time.Duration
	RetryBackoff     time.Duration
	DisableReconnect bool
	Delimiter        []byte
	Dial             DialFunc
}

// Sender is a concurrency-safe network writer with lazy dialing and reconnect.
type Sender struct {
	opt SenderOption

	mu              sync.Mutex
	conn            stdnet.Conn
	closed          bool
	lastDialAttempt time.Time
	lastDialErr     error
}

// NewSender creates a reusable network sender.
func NewSender(opt SenderOption) *Sender {
	if opt.DialTimeout <= 0 {
		opt.DialTimeout = 3 * time.Second
	}
	if opt.WriteTimeout <= 0 {
		opt.WriteTimeout = 3 * time.Second
	}
	if opt.RetryBackoff <= 0 {
		opt.RetryBackoff = time.Second
	}
	if len(opt.Delimiter) > 0 {
		cp := make([]byte, len(opt.Delimiter))
		copy(cp, opt.Delimiter)
		opt.Delimiter = cp
	}
	return &Sender{opt: opt}
}

// Write sends bytes to the configured network endpoint.
func (s *Sender) Write(p []byte) (int, error) {
	payload := p
	if len(s.opt.Delimiter) > 0 {
		payload = make([]byte, 0, len(p)+len(s.opt.Delimiter))
		payload = append(payload, p...)
		payload = append(payload, s.opt.Delimiter...)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return 0, errSenderClosed
	}

	if err := s.ensureConnLocked(); err != nil {
		return 0, err
	}

	if err := s.writeLocked(payload); err == nil {
		return len(p), nil
	}

	if s.opt.DisableReconnect {
		return 0, s.lastDialErr
	}

	s.closeConnLocked()
	if err := s.ensureConnLocked(); err != nil {
		return 0, err
	}
	if err := s.writeLocked(payload); err != nil {
		s.closeConnLocked()
		return 0, err
	}
	return len(p), nil
}

// Close closes the underlying connection.
func (s *Sender) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	return s.closeConnLocked()
}

func (s *Sender) writeLocked(payload []byte) error {
	if s.conn == nil {
		return errors.New("outputnet: connection not initialized")
	}
	if s.opt.WriteTimeout > 0 {
		_ = s.conn.SetWriteDeadline(time.Now().Add(s.opt.WriteTimeout))
	}
	n, err := s.conn.Write(payload)
	if err != nil {
		s.lastDialErr = err
		return err
	}
	if n < len(payload) {
		s.lastDialErr = io.ErrShortWrite
		return io.ErrShortWrite
	}
	s.lastDialErr = nil
	return nil
}

func (s *Sender) ensureConnLocked() error {
	if s.conn != nil {
		return nil
	}

	if s.opt.Network == "" || s.opt.Addr == "" {
		s.lastDialErr = errors.New("outputnet: network and addr are required")
		return s.lastDialErr
	}

	now := time.Now()
	if s.lastDialErr != nil && now.Sub(s.lastDialAttempt) < s.opt.RetryBackoff {
		return s.lastDialErr
	}
	s.lastDialAttempt = now

	dial := s.opt.Dial
	if dial == nil {
		dial = defaultDial
	}
	conn, err := dial(s.opt.Network, s.opt.Addr, s.opt.DialTimeout)
	if err != nil {
		s.lastDialErr = err
		return err
	}
	s.conn = conn
	s.lastDialErr = nil
	return nil
}

func (s *Sender) closeConnLocked() error {
	if s.conn == nil {
		return nil
	}
	err := s.conn.Close()
	s.conn = nil
	return err
}

func defaultDial(network, addr string, timeout time.Duration) (stdnet.Conn, error) {
	d := stdnet.Dialer{Timeout: timeout}
	return d.Dial(network, addr)
}
