package webhook

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"time"

	svr "github.com/darkit/slog"
)

var defaultHTTPClient = &http.Client{
	Timeout: 10 * time.Second,
}

// Transport sends already-encoded payload bytes.
type Transport interface {
	Send(ctx context.Context, payload []byte) error
}

// HTTPTransport is the default webhook transport.
type HTTPTransport struct {
	Endpoint string
	Timeout  time.Duration
	Client   *http.Client
}

func (t *HTTPTransport) Send(ctx context.Context, payload []byte) error {
	client := t.Client
	if client == nil {
		client = defaultHTTPClient
	}

	if t.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, t.Timeout)
		defer cancel()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.Endpoint, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	req.Header.Add("content-type", "application/json")
	req.Header.Add("user-agent", svr.Name)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	return nil
}
