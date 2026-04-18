package slog

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestProgressUsesPipelineForDLPAndSubscriptions(t *testing.T) {
	resetForTest()
	EnableDLPLogger()
	defer DisableDLPLogger()

	var buf bytes.Buffer
	logger := NewLogger(&buf, true, false)

	ch, cancel := Subscribe(10)
	defer cancel()

	logger.Progress("phone 13812345678", 0)

	output := buf.String()
	if strings.Contains(output, "13812345678") {
		t.Fatalf("expected progress output to be desensitized, got %q", output)
	}
	if !strings.Contains(output, "100.0%") {
		t.Fatalf("expected progress output to contain final percentage, got %q", output)
	}
	if strings.Contains(output, "\r") {
		t.Fatalf("expected non-interactive progress output to avoid carriage returns, got %q", output)
	}

	select {
	case event := <-ch:
		if !strings.Contains(event.Record.Message, "100.0%") {
			t.Fatalf("expected subscription to receive progress record, got %q", event.Record.Message)
		}
		if !strings.Contains(event.Rendered, "100.0%") {
			t.Fatalf("expected semantic text render for progress, got %q", event.Rendered)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected progress record to reach subscription")
	}
}

func TestProgressJSONOutputIsValid(t *testing.T) {
	resetForTest()
	SetLevelDebug()

	var buf bytes.Buffer
	config := DefaultConfig()
	config.SetEnableText(false)
	config.SetEnableJSON(true)
	logger := NewLoggerWithConfig(&buf, config)

	logger.Progress("a\"b", 0)

	raw := strings.TrimSpace(buf.String())
	if raw == "" {
		t.Fatal("expected JSON output from progress")
	}

	var entry map[string]any
	if err := json.Unmarshal([]byte(raw), &entry); err != nil {
		t.Fatalf("expected valid JSON output, err=%v raw=%q", err, raw)
	}
	if entry["msg"] != "a\"b 100.0%" {
		t.Fatalf("unexpected msg field: %v", entry["msg"])
	}
	if entry["level"] != "Info" {
		t.Fatalf("unexpected level field: %v", entry["level"])
	}
}

func TestConsoleHandlerDynamicRenderKeepsSingleLine(t *testing.T) {
	var buf bytes.Buffer
	h := &handler{
		w:          &buf,
		state:      &handlerState{},
		level:      LevelInfo,
		timeFormat: TimeFormat,
		noColor:    true,
		dynamicTTY: true,
	}

	if err := h.Handle(withDynamicRender(context.Background(), dynamicRenderState{final: false}), slog.NewRecord(time.Time{}, LevelInfo, "working", 0)); err != nil {
		t.Fatalf("unexpected dynamic handle error: %v", err)
	}
	if err := h.Handle(context.Background(), slog.NewRecord(time.Time{}, LevelInfo, "done", 0)); err != nil {
		t.Fatalf("unexpected normal handle error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "\r\x1b[K[I] working") {
		t.Fatalf("expected inline render output, got %q", output)
	}
	if !strings.Contains(output, "\n[I] done\n") {
		t.Fatalf("expected subsequent log to continue on next line, got %q", output)
	}
}

func TestLoadingFallsBackToStructuredMilestones(t *testing.T) {
	resetForTest()

	var buf bytes.Buffer
	logger := NewLogger(&buf, true, false)

	logger.Loading("sync task", 0)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected start and done milestones, got %d lines: %q", len(lines), buf.String())
	}
	if !strings.Contains(lines[0], "sync task started") {
		t.Fatalf("expected start milestone, got %q", lines[0])
	}
	if !strings.Contains(lines[1], "sync task done") {
		t.Fatalf("expected done milestone, got %q", lines[1])
	}
}
