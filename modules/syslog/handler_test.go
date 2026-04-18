package syslog

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/darkit/slog/modules"
)

// mockWriter 用于测试的 mock writer
type mockWriter struct {
	mu   sync.Mutex
	data [][]byte
}

func (w *mockWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	// 复制数据以避免竞态
	cp := make([]byte, len(p))
	copy(cp, p)
	w.data = append(w.data, cp)
	return len(p), nil
}

func (w *mockWriter) getData() [][]byte {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.data
}

func TestSyslogHandler_Enabled(t *testing.T) {
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
			w := &mockWriter{}
			h := NewSyslogHandler(w, &Option{Level: tt.level})
			if got := h.Enabled(context.Background(), tt.check); got != tt.expected {
				t.Errorf("Enabled() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestSyslogHandler_Handle(t *testing.T) {
	w := &mockWriter{}
	h := NewSyslogHandler(w, &Option{
		Level: slog.LevelInfo,
	})

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

	// 等待异步写入完成
	time.Sleep(100 * time.Millisecond)

	data := w.getData()
	if len(data) != 1 {
		t.Fatalf("Expected 1 write, got %d", len(data))
	}

	// 验证 CEE 前缀
	if !bytes.HasPrefix(data[0], []byte(ceePrefix)) {
		t.Errorf("Expected CEE prefix, got %s", string(data[0][:20]))
	}

	// 解析 JSON 部分
	jsonPart := data[0][len(ceePrefix):]
	var payload map[string]any
	if err := json.Unmarshal(jsonPart, &payload); err != nil {
		t.Fatalf("Failed to unmarshal payload: %v", err)
	}

	if payload["message"] != "test message" {
		t.Errorf("Expected message 'test message', got %v", payload["message"])
	}

	if payload["level"] != "INFO" {
		t.Errorf("Expected level 'INFO', got %v", payload["level"])
	}
}

func TestSyslogHandler_WithAttrs(t *testing.T) {
	w := &mockWriter{}
	h := NewSyslogHandler(w, &Option{
		Level: slog.LevelInfo,
	})

	h2 := h.WithAttrs([]slog.Attr{
		slog.String("key1", "value1"),
		slog.Int("key2", 42),
	})

	// 验证返回新的 handler
	if h == h2 {
		t.Error("WithAttrs should return a new handler")
	}

	// 验证类型正确
	sh, ok := h2.(*SyslogHandler)
	if !ok {
		t.Fatal("WithAttrs should return *SyslogHandler")
	}

	if len(sh.attrs) != 2 {
		t.Errorf("Expected 2 attrs, got %d", len(sh.attrs))
	}
}

func TestSyslogHandler_WithGroup(t *testing.T) {
	w := &mockWriter{}
	h := NewSyslogHandler(w, &Option{
		Level: slog.LevelInfo,
	})

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

	sh, ok := h3.(*SyslogHandler)
	if !ok {
		t.Fatal("WithGroup should return *SyslogHandler")
	}

	if len(sh.groups) != 1 || sh.groups[0] != "mygroup" {
		t.Errorf("Expected groups ['mygroup'], got %v", sh.groups)
	}
}

func TestNewSyslogHandler_Defaults(t *testing.T) {
	w := &mockWriter{}
	h := NewSyslogHandler(w, &Option{})

	sh, ok := h.(*SyslogHandler)
	if !ok {
		t.Fatal("Expected *SyslogHandler")
	}

	// 验证默认值
	if sh.option.Level.Level() != slog.LevelInfo {
		t.Errorf("Expected default level Info, got %v", sh.option.Level)
	}

	if sh.option.Writer != w {
		t.Error("Expected writer to be set")
	}
	if sh.option.Codec == nil {
		t.Error("Expected default codec")
	}
}

func TestSyslogHandler_CEEPrefix(t *testing.T) {
	w := &mockWriter{}
	h := NewSyslogHandler(w, &Option{
		Level: slog.LevelInfo,
	})

	record := slog.Record{
		Time:    time.Now(),
		Level:   slog.LevelInfo,
		Message: "cee test",
	}

	_ = h.Handle(context.Background(), record)

	time.Sleep(100 * time.Millisecond)

	data := w.getData()
	if len(data) == 0 {
		t.Fatal("No data written")
	}

	// 验证 CEE 前缀格式
	if !strings.HasPrefix(string(data[0]), "@cee: ") {
		t.Errorf("Expected '@cee: ' prefix, got %s", string(data[0][:10]))
	}
}

type customSyslogCodec struct{}

func (c customSyslogCodec) Name() string { return "custom-handler" }

func (c customSyslogCodec) Encode(_ context.Context, record *slog.Record, attrs []slog.Attr, groups []string) ([]byte, error) {
	_ = attrs
	_ = groups
	return json.Marshal(map[string]any{"custom": record.Message})
}

func TestSyslogHandler_CustomCodec(t *testing.T) {
	w := &mockWriter{}
	h := NewSyslogHandler(w, &Option{
		Level: slog.LevelInfo,
		Codec: customSyslogCodec{},
	})

	record := slog.Record{
		Time:    time.Now(),
		Level:   slog.LevelInfo,
		Message: "custom payload",
	}
	_ = h.Handle(context.Background(), record)
	time.Sleep(100 * time.Millisecond)

	data := w.getData()
	if len(data) == 0 {
		t.Fatal("No data written")
	}
	jsonPart := data[0][len(ceePrefix):]
	var payload map[string]any
	if err := json.Unmarshal(jsonPart, &payload); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}
	if payload["custom"] != "custom payload" {
		t.Fatalf("unexpected custom payload: %#v", payload)
	}
}

func TestSyslogHandler_ChainedOperations(t *testing.T) {
	w := &mockWriter{}
	h := NewSyslogHandler(w, &Option{
		Level: slog.LevelInfo,
	})

	// 链式调用
	h2 := h.WithGroup("group1").WithAttrs([]slog.Attr{
		slog.String("attr1", "value1"),
	}).WithGroup("group2").WithAttrs([]slog.Attr{
		slog.String("attr2", "value2"),
	})

	sh, ok := h2.(*SyslogHandler)
	if !ok {
		t.Fatal("Expected *SyslogHandler")
	}

	if len(sh.groups) != 2 {
		t.Errorf("Expected 2 groups, got %d", len(sh.groups))
	}
}

func TestSyslogHandler_NonBlocking(t *testing.T) {
	// 创建一个慢速 writer
	slowWriter := &slowMockWriter{delay: 200 * time.Millisecond}

	h := NewSyslogHandler(slowWriter, &Option{
		Level: slog.LevelInfo,
	})

	record := slog.Record{
		Time:    time.Now(),
		Level:   slog.LevelInfo,
		Message: "non-blocking test",
	}

	// Handle 应该立即返回
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

type errorWriter struct{}

func (w *errorWriter) Write(_ []byte) (n int, err error) {
	return 0, errors.New("write failed")
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

func TestSyslogHandler_ReportsAsyncErrors(t *testing.T) {
	errWriter := newAsyncCaptureWriter()
	modules.SetAsyncErrorWriter(errWriter)
	modules.EnableAsyncErrorLogging(true)
	t.Cleanup(func() {
		modules.SetAsyncErrorWriter(nil)
	})

	h := NewSyslogHandler(&errorWriter{}, &Option{
		Level: slog.LevelInfo,
	})
	record := slog.Record{
		Time:    time.Now(),
		Level:   slog.LevelInfo,
		Message: "async error test",
	}
	if err := h.Handle(context.Background(), record); err != nil {
		t.Fatalf("Handle() should remain non-blocking, err=%v", err)
	}

	select {
	case <-errWriter.done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected async error output")
	}
	if out := errWriter.String(); !strings.Contains(out, "write failed") {
		t.Fatalf("expected async error output, got %q", out)
	}
}

type slowMockWriter struct {
	delay time.Duration
}

func (w *slowMockWriter) Write(p []byte) (n int, err error) {
	time.Sleep(w.delay)
	return len(p), nil
}

// 基准测试
func BenchmarkSyslogHandler_Handle(b *testing.B) {
	w := &mockWriter{}
	h := NewSyslogHandler(w, &Option{
		Level: slog.LevelInfo,
	})

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
