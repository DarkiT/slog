package outputnet

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/darkit/slog/modules"
)

type captureWriter struct {
	mu   sync.Mutex
	buf  bytes.Buffer
	done chan struct{}
}

func newCaptureWriter() *captureWriter {
	return &captureWriter{done: make(chan struct{}, 1)}
}

func (w *captureWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	n, err := w.buf.Write(p)
	select {
	case w.done <- struct{}{}:
	default:
	}
	return n, err
}

func (w *captureWriter) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.String()
}

type failedWriter struct{}

func (w *failedWriter) Write(_ []byte) (int, error) {
	return 0, context.Canceled
}

type panicWriter struct{}

func (w *panicWriter) Write(_ []byte) (int, error) {
	panic("writer panic")
}

func TestRawHandler_Handle(t *testing.T) {
	w := newCaptureWriter()
	h := NewRawHandler(w, slog.LevelInfo)
	r := slog.Record{Time: time.Now(), Level: slog.LevelInfo, Message: "hello"}
	r.AddAttrs(slog.String("user", "alice"))

	if err := h.Handle(context.Background(), r); err != nil {
		t.Fatalf("handle: %v", err)
	}
	select {
	case <-w.done:
	case <-time.After(300 * time.Millisecond):
		t.Fatal("timeout waiting write")
	}

	got := w.String()
	if !strings.Contains(got, "level=INFO") || !strings.Contains(got, "msg=hello") || !strings.Contains(got, "user=alice") {
		t.Fatalf("unexpected payload: %q", got)
	}
}

func TestRawHandler_JSONCodec(t *testing.T) {
	w := newCaptureWriter()
	codec, ok := GetCodec("json")
	if !ok {
		t.Fatal("json codec missing")
	}
	h := NewRawHandlerWithOption(RawOption{
		Level:  slog.LevelInfo,
		Writer: w,
		Codec:  codec,
	})
	r := slog.Record{Time: time.Now(), Level: slog.LevelInfo, Message: "hello-json"}

	if err := h.Handle(context.Background(), r); err != nil {
		t.Fatalf("handle: %v", err)
	}
	select {
	case <-w.done:
	case <-time.After(300 * time.Millisecond):
		t.Fatal("timeout waiting write")
	}
	if got := w.String(); !strings.Contains(got, "\"message\":\"hello-json\"") {
		t.Fatalf("unexpected json payload: %q", got)
	}
}

func TestRawHandler_NilWriterReturnsError(t *testing.T) {
	h := NewRawHandlerWithOption(RawOption{Writer: nil})
	r := slog.Record{Time: time.Now(), Level: slog.LevelInfo, Message: "nil writer"}

	if err := h.Handle(context.Background(), r); !errors.Is(err, errNilWriter) {
		t.Fatalf("Handle() error = %v, want %v", err, errNilWriter)
	}
}

func TestRawHandler_ReportsAsyncError(t *testing.T) {
	errWriter := newCaptureWriter()
	modules.SetAsyncErrorWriter(errWriter)
	modules.EnableAsyncErrorLogging(true)
	t.Cleanup(func() { modules.SetAsyncErrorWriter(nil) })

	h := NewRawHandler(&failedWriter{}, slog.LevelInfo)
	r := slog.Record{Time: time.Now(), Level: slog.LevelInfo, Message: "err"}
	if err := h.Handle(context.Background(), r); err != nil {
		t.Fatalf("handle: %v", err)
	}
	select {
	case <-errWriter.done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected async error output")
	}
	if !strings.Contains(errWriter.String(), "output.net") {
		t.Fatalf("expected output.net async error message, got %q", errWriter.String())
	}
}

func TestRawHandler_ReportsAsyncPanic(t *testing.T) {
	errWriter := newCaptureWriter()
	modules.SetAsyncErrorWriter(errWriter)
	modules.EnableAsyncErrorLogging(true)
	t.Cleanup(func() { modules.SetAsyncErrorWriter(nil) })

	h := NewRawHandler(&panicWriter{}, slog.LevelInfo)
	r := slog.Record{Time: time.Now(), Level: slog.LevelInfo, Message: "panic"}
	if err := h.Handle(context.Background(), r); err != nil {
		t.Fatalf("handle: %v", err)
	}
	select {
	case <-errWriter.done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected async panic output")
	}
	if out := errWriter.String(); !strings.Contains(out, "panic: writer panic") {
		t.Fatalf("expected panic output, got %q", out)
	}
}

func TestNewNetAdapter_Configure(t *testing.T) {
	adapter := NewNetAdapter()
	err := adapter.Configure(modules.Config{
		"network": "udp",
		"addr":    "127.0.0.1:9999",
		"level":   "warn",
		"codec":   "raw",
	})
	if err != nil {
		t.Fatalf("configure: %v", err)
	}
	if adapter.Handler() == nil {
		t.Fatal("expected handler to be set")
	}
	if adapter.Type() != modules.TypeSink {
		t.Fatalf("unexpected type: %v", adapter.Type())
	}
}

func TestNewNetAdapter_InvalidCodec(t *testing.T) {
	adapter := NewNetAdapter()
	err := adapter.Configure(modules.Config{
		"network": "udp",
		"addr":    "127.0.0.1:9999",
		"codec":   "not-exist",
	})
	if err == nil {
		t.Fatal("expected invalid codec error")
	}
}
