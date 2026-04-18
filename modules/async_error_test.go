package modules

import (
	"bytes"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestReportAsyncError_Disabled(t *testing.T) {
	var buf bytes.Buffer
	SetAsyncErrorWriter(&buf)
	EnableAsyncErrorLogging(false)
	t.Cleanup(func() {
		EnableAsyncErrorLogging(true)
		SetAsyncErrorWriter(nil)
	})

	ReportAsyncError("webhook", nil)
	ReportAsyncError("webhook", assertErr("boom"))
	if buf.Len() != 0 {
		t.Fatalf("expected no output when disabled, got %q", buf.String())
	}
}

func TestReportAsyncError_Enabled(t *testing.T) {
	var buf bytes.Buffer
	SetAsyncErrorWriter(&buf)
	EnableAsyncErrorLogging(true)
	t.Cleanup(func() {
		SetAsyncErrorWriter(nil)
	})

	ReportAsyncError("syslog", assertErr("write failed"))
	out := buf.String()
	if !strings.Contains(out, "syslog") || !strings.Contains(out, "write failed") {
		t.Fatalf("unexpected async error output: %q", out)
	}
}

type assertErr string

func (e assertErr) Error() string { return string(e) }

type safeBuffer struct {
	mu   sync.Mutex
	buf  bytes.Buffer
	done chan struct{}
}

func newSafeBuffer() *safeBuffer {
	return &safeBuffer{done: make(chan struct{}, 1)}
}

func (b *safeBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	n, err := b.buf.Write(p)
	select {
	case b.done <- struct{}{}:
	default:
	}
	return n, err
}

func (b *safeBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func TestRunAsync_ReportsError(t *testing.T) {
	buf := newSafeBuffer()
	SetAsyncErrorWriter(io.Writer(buf))
	EnableAsyncErrorLogging(true)
	t.Cleanup(func() {
		SetAsyncErrorWriter(nil)
	})

	RunAsync("dispatcher", func() error {
		return assertErr("task failed")
	})

	select {
	case <-buf.done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("async task timeout")
	}
	if out := buf.String(); !strings.Contains(out, "dispatcher") || !strings.Contains(out, "task failed") {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestRunAsync_ReportsPanic(t *testing.T) {
	buf := newSafeBuffer()
	SetAsyncErrorWriter(io.Writer(buf))
	EnableAsyncErrorLogging(true)
	t.Cleanup(func() {
		SetAsyncErrorWriter(nil)
	})

	RunAsync("panic-case", func() error {
		panic("boom")
	})

	select {
	case <-buf.done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("async panic task timeout")
	}
	if out := buf.String(); !strings.Contains(out, "panic-case") || !strings.Contains(out, "panic: boom") {
		t.Fatalf("unexpected panic output: %q", out)
	}
}

func TestRunAsync_DropsWhenQueueFull(t *testing.T) {
	originalDrop := asyncDropCount.Load()
	originalLogging := asyncErrorLogEnable.Load()
	originalOptions := GetAsyncExecutorOptions()
	defer func() {
		asyncDropCount.Store(originalDrop)
		EnableAsyncErrorLogging(originalLogging)
		SetAsyncExecutorOptions(originalOptions)
	}()
	asyncDropCount.Store(0)
	EnableAsyncErrorLogging(false)
	SetAsyncExecutorOptions(AsyncExecutorOptions{
		WorkerCount: 1,
		QueueSize:   1,
	})

	// Fill workers and queue with blocking tasks so subsequent submissions must drop.
	block := make(chan struct{})
	defer close(block)
	blockingTask := func() error {
		<-block
		return nil
	}
	opts := GetAsyncExecutorOptions()
	for i := 0; i < opts.WorkerCount+opts.QueueSize+32; i++ {
		RunAsync("overflow-fill", blockingTask)
	}

	deadline := time.Now().Add(500 * time.Millisecond)
	for asyncDropCount.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	var calls atomic.Int32
	RunAsync("overflow", func() error {
		calls.Add(1)
		return nil
	})

	time.Sleep(20 * time.Millisecond)

	if got := calls.Load(); got != 0 {
		t.Fatalf("task should not execute when queue is full, got calls=%d", got)
	}
	if dropped := asyncDropCount.Load(); dropped == 0 {
		t.Fatalf("expected dropped tasks when queue is full, got %d", dropped)
	}
}

func TestAsyncExecutorOptions_DefaultsAndOverride(t *testing.T) {
	original := GetAsyncExecutorOptions()
	defer SetAsyncExecutorOptions(original)

	SetAsyncExecutorOptions(AsyncExecutorOptions{})
	current := GetAsyncExecutorOptions()
	if current.WorkerCount != defaultAsyncWorkerCount || current.QueueSize != defaultAsyncQueueSize {
		t.Fatalf("unexpected defaults: %+v", current)
	}

	SetAsyncExecutorOptions(AsyncExecutorOptions{
		WorkerCount: 2,
		QueueSize:   8,
	})
	current = GetAsyncExecutorOptions()
	if current.WorkerCount != 2 || current.QueueSize != 8 {
		t.Fatalf("unexpected override: %+v", current)
	}
}
