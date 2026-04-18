package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/darkit/slog/modules"
)

func TestWebhookHandler_Enabled(t *testing.T) {
	tests := []struct {
		name     string
		level    slog.Leveler
		check    slog.Level
		expected bool
	}{
		{"debug_allows_debug", slog.LevelDebug, slog.LevelDebug, true},
		{"debug_allows_info", slog.LevelDebug, slog.LevelInfo, true},
		{"info_blocks_debug", slog.LevelInfo, slog.LevelDebug, false},
		{"info_allows_info", slog.LevelInfo, slog.LevelInfo, true},
		{"error_blocks_warn", slog.LevelError, slog.LevelWarn, false},
		{"error_allows_error", slog.LevelError, slog.LevelError, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := Option{Level: tt.level, Endpoint: "http://localhost"}.NewWebhookHandler()
			if got := h.Enabled(context.Background(), tt.check); got != tt.expected {
				t.Errorf("Enabled() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestWebhookHandler_Handle(t *testing.T) {
	var (
		mu       sync.Mutex
		received []map[string]any
	)

	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected application/json, got %s", r.Header.Get("Content-Type"))
		}

		body, _ := io.ReadAll(r.Body)
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Errorf("Failed to unmarshal payload: %v", err)
		}

		mu.Lock()
		received = append(received, payload)
		mu.Unlock()

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	h := Option{
		Level:    slog.LevelInfo,
		Endpoint: server.URL,
		Timeout:  5 * time.Second,
	}.NewWebhookHandler()

	record := slog.Record{
		Time:    time.Now(),
		Level:   slog.LevelInfo,
		Message: "test message",
	}
	record.AddAttrs(slog.String("key", "value"))

	err := h.Handle(context.Background(), record)
	if err != nil {
		t.Errorf("Handle() error = %v", err)
	}

	// 等待异步发送完成
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if len(received) != 1 {
		t.Fatalf("Expected 1 request, got %d", len(received))
	}

	payload := received[0]
	if payload["message"] != "test message" {
		t.Errorf("Expected message 'test message', got %v", payload["message"])
	}
	if payload["level"] != "INFO" {
		t.Errorf("Expected level 'INFO', got %v", payload["level"])
	}
}

func TestWebhookHandler_WithAttrs(t *testing.T) {
	h := Option{
		Level:    slog.LevelInfo,
		Endpoint: "http://localhost",
	}.NewWebhookHandler()

	h2 := h.WithAttrs([]slog.Attr{
		slog.String("key1", "value1"),
		slog.Int("key2", 42),
	})

	// 验证返回新的 handler
	if h == h2 {
		t.Error("WithAttrs should return a new handler")
	}

	// 验证类型正确
	wh, ok := h2.(*WebhookHandler)
	if !ok {
		t.Fatal("WithAttrs should return *WebhookHandler")
	}

	if len(wh.attrs) != 2 {
		t.Errorf("Expected 2 attrs, got %d", len(wh.attrs))
	}
}

func TestWebhookHandler_WithGroup(t *testing.T) {
	h := Option{
		Level:    slog.LevelInfo,
		Endpoint: "http://localhost",
	}.NewWebhookHandler()

	// 空组名应返回原 handler
	h2 := h.WithGroup("")
	if h != h2 {
		t.Error("WithGroup('') should return same handler")
	}

	// 非空组名应返回新 handler
	h3 := h.WithGroup("mygroup")
	if h == h3 {
		t.Error("WithGroup should return a new handler")
	}

	wh, ok := h3.(*WebhookHandler)
	if !ok {
		t.Fatal("WithGroup should return *WebhookHandler")
	}

	if len(wh.groups) != 1 || wh.groups[0] != "mygroup" {
		t.Errorf("Expected groups ['mygroup'], got %v", wh.groups)
	}
}

func TestWebhookHandler_Timeout(t *testing.T) {
	// 创建一个慢速服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	h := Option{
		Level:    slog.LevelInfo,
		Endpoint: server.URL,
		Timeout:  50 * time.Millisecond, // 超时时间短于服务器响应时间
	}.NewWebhookHandler()

	record := slog.Record{
		Time:    time.Now(),
		Level:   slog.LevelInfo,
		Message: "timeout test",
	}

	// Handle 应该立即返回（非阻塞）
	start := time.Now()
	err := h.Handle(context.Background(), record)
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("Handle() error = %v", err)
	}

	// 验证 Handle 是非阻塞的
	if elapsed > 50*time.Millisecond {
		t.Errorf("Handle() took too long: %v", elapsed)
	}
}

func TestOption_Defaults(t *testing.T) {
	h := Option{
		Endpoint: "http://localhost",
	}.NewWebhookHandler()

	wh, ok := h.(*WebhookHandler)
	if !ok {
		t.Fatal("Expected *WebhookHandler")
	}

	// 验证默认值
	if wh.option.Level.Level() != slog.LevelInfo {
		t.Errorf("Expected default level Info, got %v", wh.option.Level)
	}

	if wh.option.Timeout != 10*time.Second {
		t.Errorf("Expected default timeout 10s, got %v", wh.option.Timeout)
	}
	if wh.option.Codec == nil {
		t.Error("Expected default codec")
	}
	if wh.option.Transport == nil {
		t.Error("Expected default transport")
	}
}

type asyncCaptureWriter struct {
	mu   sync.Mutex
	buf  bytes.Buffer
	done chan struct{}
}

func newAsyncCaptureWriter() *asyncCaptureWriter {
	return &asyncCaptureWriter{done: make(chan struct{}, 1)}
}

func (w *asyncCaptureWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	n, err := w.buf.Write(p)
	select {
	case w.done <- struct{}{}:
	default:
	}
	return n, err
}

func (w *asyncCaptureWriter) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.String()
}

func TestWebhookHandler_ReportsAsyncErrors(t *testing.T) {
	errWriter := newAsyncCaptureWriter()
	modules.SetAsyncErrorWriter(errWriter)
	modules.EnableAsyncErrorLogging(true)
	t.Cleanup(func() {
		modules.SetAsyncErrorWriter(nil)
	})

	h := Option{
		Level:    slog.LevelInfo,
		Endpoint: "://bad-url",
		Timeout:  10 * time.Millisecond,
	}.NewWebhookHandler()

	record := slog.Record{
		Time:    time.Now(),
		Level:   slog.LevelInfo,
		Message: "test async error",
	}

	if err := h.Handle(context.Background(), record); err != nil {
		t.Fatalf("Handle() should remain non-blocking, err=%v", err)
	}

	select {
	case <-errWriter.done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected async error output for invalid webhook endpoint")
	}
	if out := errWriter.String(); !strings.Contains(out, "webhook") {
		t.Fatalf("expected webhook async error output, got %q", out)
	}
}

type testTransport struct {
	mu       sync.Mutex
	payloads [][]byte
	done     chan struct{}
}

func (t *testTransport) Send(_ context.Context, payload []byte) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	cp := make([]byte, len(payload))
	copy(cp, payload)
	t.payloads = append(t.payloads, cp)
	if t.done != nil {
		select {
		case t.done <- struct{}{}:
		default:
		}
	}
	return nil
}

func TestWebhookHandler_CustomCodec(t *testing.T) {
	tr := &testTransport{done: make(chan struct{}, 1)}
	codec := customCodec{}

	h := Option{
		Level:     slog.LevelInfo,
		Endpoint:  "http://localhost",
		Codec:     codec,
		Transport: tr,
	}.NewWebhookHandler()

	record := slog.Record{
		Time:    time.Now(),
		Level:   slog.LevelInfo,
		Message: "codec-test",
	}
	_ = h.Handle(context.Background(), record)
	select {
	case <-tr.done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting transport send")
	}
	tr.mu.Lock()
	defer tr.mu.Unlock()
	if len(tr.payloads) != 1 {
		t.Fatalf("expected 1 payload, got %d", len(tr.payloads))
	}
	if !strings.Contains(string(tr.payloads[0]), "codec-test") {
		t.Fatalf("unexpected payload: %s", string(tr.payloads[0]))
	}
}

func TestWebhookHandler_ChainedOperations(t *testing.T) {
	h := Option{
		Level:    slog.LevelInfo,
		Endpoint: "http://localhost",
	}.NewWebhookHandler()

	// 链式调用
	h2 := h.WithGroup("group1").WithAttrs([]slog.Attr{
		slog.String("attr1", "value1"),
	}).WithGroup("group2").WithAttrs([]slog.Attr{
		slog.String("attr2", "value2"),
	})

	wh, ok := h2.(*WebhookHandler)
	if !ok {
		t.Fatal("Expected *WebhookHandler")
	}

	if len(wh.groups) != 2 {
		t.Errorf("Expected 2 groups, got %d", len(wh.groups))
	}
}

// 基准测试
func BenchmarkWebhookHandler_Handle(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	h := Option{
		Level:    slog.LevelInfo,
		Endpoint: server.URL,
	}.NewWebhookHandler()

	record := slog.Record{
		Time:    time.Now(),
		Level:   slog.LevelInfo,
		Message: "benchmark message",
	}
	record.AddAttrs(slog.String("key", "value"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Handle(context.Background(), record)
	}
}

func BenchmarkDefaultCodec_Encode(b *testing.B) {
	codec, ok := GetCodec("default")
	if !ok {
		b.Fatal("default codec missing")
	}
	record := slog.Record{
		Time:    time.Now(),
		Level:   slog.LevelInfo,
		Message: "benchmark message",
	}
	record.AddAttrs(
		slog.String("key1", "value1"),
		slog.Int("key2", 42),
		slog.Bool("key3", true),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := codec.Encode(context.Background(), &record, nil, nil); err != nil {
			b.Fatalf("encode failed: %v", err)
		}
	}
}
