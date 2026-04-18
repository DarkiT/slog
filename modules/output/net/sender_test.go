package outputnet

import (
	"errors"
	"io"
	stdnet "net"
	"sync/atomic"
	"testing"
	"time"
)

func TestSender_TCPWriteWithDelimiter(t *testing.T) {
	ln, err := stdnet.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	readCh := make(chan []byte, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		buf := make([]byte, 128)
		n, _ := conn.Read(buf)
		cp := make([]byte, n)
		copy(cp, buf[:n])
		readCh <- cp
	}()

	s := NewSender(SenderOption{
		Network:      "tcp",
		Addr:         ln.Addr().String(),
		WriteTimeout: time.Second,
		Delimiter:    []byte("\n"),
	})
	t.Cleanup(func() { _ = s.Close() })

	n, err := s.Write([]byte("hello"))
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	if n != 5 {
		t.Fatalf("written len=%d want=5", n)
	}

	select {
	case got := <-readCh:
		if string(got) != "hello\n" {
			t.Fatalf("payload=%q want=%q", string(got), "hello\n")
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for server read")
	}
}

func TestSender_BackoffOnDialFailure(t *testing.T) {
	var dialCount int32
	s := NewSender(SenderOption{
		Network:      "tcp",
		Addr:         "127.0.0.1:12345",
		DialTimeout:  20 * time.Millisecond,
		RetryBackoff: 200 * time.Millisecond,
		Dial: func(network, addr string, timeout time.Duration) (stdnet.Conn, error) {
			_ = network
			_ = addr
			_ = timeout
			atomic.AddInt32(&dialCount, 1)
			return nil, errors.New("dial failed")
		},
	})
	t.Cleanup(func() { _ = s.Close() })

	start := time.Now()
	_, err1 := s.Write([]byte("x"))
	if err1 == nil {
		t.Fatal("expected first write error")
	}
	_, err2 := s.Write([]byte("x"))
	if err2 == nil {
		t.Fatal("expected second write error")
	}
	if time.Since(start) > 120*time.Millisecond {
		t.Fatal("second write should return quickly due to backoff")
	}
	if atomic.LoadInt32(&dialCount) != 1 {
		t.Fatalf("expected one dial attempt due to backoff, got %d", atomic.LoadInt32(&dialCount))
	}
}

type flakyConn struct {
	writeCount int32
}

func (c *flakyConn) Read(_ []byte) (int, error)        { return 0, io.EOF }
func (c *flakyConn) Close() error                      { return nil }
func (c *flakyConn) LocalAddr() stdnet.Addr            { return nil }
func (c *flakyConn) RemoteAddr() stdnet.Addr           { return nil }
func (c *flakyConn) SetDeadline(_ time.Time) error     { return nil }
func (c *flakyConn) SetReadDeadline(_ time.Time) error { return nil }
func (c *flakyConn) SetWriteDeadline(_ time.Time) error {
	return nil
}

func (c *flakyConn) Write(p []byte) (int, error) {
	if atomic.AddInt32(&c.writeCount, 1) == 1 {
		return 0, errors.New("boom")
	}
	return len(p), nil
}

func TestSender_DisableReconnectReturnsWriteErr(t *testing.T) {
	s := NewSender(SenderOption{
		Network:          "tcp",
		Addr:             "unused",
		DisableReconnect: true,
	})
	s.conn = &flakyConn{}
	t.Cleanup(func() { _ = s.Close() })

	_, err := s.Write([]byte("x"))
	if err == nil || err.Error() != "boom" {
		t.Fatalf("expected write error boom, got %v", err)
	}
}

func TestSender_Close(t *testing.T) {
	s := NewSender(SenderOption{})
	if err := s.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	_, err := s.Write([]byte("x"))
	if !errors.Is(err, errSenderClosed) {
		t.Fatalf("expected errSenderClosed, got %v", err)
	}
}
