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

func TestSubscribeReceivesNormalizedPublishView(t *testing.T) {
	resetForTest()
	EnableDLPLogger()
	defer DisableDLPLogger()

	SetContextPropagator(func(context.Context) []slog.Attr {
		return []slog.Attr{slog.String("request_id", "req-1")}
	})
	defer SetContextPropagator(nil)

	formatterID := RegisterFormatter("upper-user", func(_ []string, attr slog.Attr) (slog.Value, bool) {
		if attr.Key == "user" {
			return slog.StringValue(strings.ToUpper(attr.Value.String())), true
		}
		return attr.Value, false
	})
	defer RemoveFormatter(formatterID)

	records, cancel := Subscribe(10)
	defer cancel()

	var buf bytes.Buffer
	ResetGlobalLogger(&buf, true, false)
	SetLevelInfo()
	EnableTextLogger()
	DisableJSONLogger()

	logger := Default("billing").WithGroup("request").With("trace_id", "t-1")
	logger.InfoContext(context.Background(), "phone 13812345678", "user", "alice", "phone", "13812345678")

	select {
	case event := <-records:
		record := event.Record
		if !strings.HasPrefix(record.Message, "[") || !strings.Contains(record.Message, "] ") {
			t.Fatalf("expected message to retain module prefix shape, got %q", record.Message)
		}
		if strings.Contains(record.Message, "13812345678") {
			t.Fatalf("expected message to be desensitized, got %q", record.Message)
		}

		attrs := flattenRecordAttrs(record)
		if attrs["request.trace_id"] != "t-1" {
			t.Fatalf("expected With attr in publish view, got %v", attrs)
		}
		if attrs["request.request_id"] != "req-1" {
			t.Fatalf("expected propagated context attr in publish view, got %v", attrs)
		}
		if attrs["request.user"] != "ALICE" {
			t.Fatalf("expected formatter-applied attr in publish view, got %v", attrs)
		}
		if strings.Contains(attrs["request.phone"], "13812345678") {
			t.Fatalf("expected attr to be desensitized, got %v", attrs)
		}
		if event.Format != "text" {
			t.Fatalf("expected text render preference, got %q", event.Format)
		}
		if strings.TrimSpace(buf.String()) != event.Rendered {
			t.Fatalf("expected subscription text render to match sink output, got sink=%q event=%q", strings.TrimSpace(buf.String()), event.Rendered)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for normalized publish view")
	}
}

func TestSubscribeJSONRenderedMatchesSinkOutput(t *testing.T) {
	resetForTest()
	DisableTextLogger()
	EnableJSONLogger()
	defer func() {
		EnableTextLogger()
		DisableJSONLogger()
	}()

	var buf bytes.Buffer
	config := DefaultConfig()
	config.SetEnableText(false)
	config.SetEnableJSON(true)
	logger := NewLoggerWithConfig(&buf, config)

	events, cancel := Subscribe(10)
	defer cancel()

	logger.Info("json only", "key", "value")

	select {
	case event := <-events:
		if event.Format != "json" {
			t.Fatalf("expected json render preference, got %q", event.Format)
		}
		if strings.TrimSpace(buf.String()) != event.Rendered {
			t.Fatalf("expected JSON render to match sink output, got sink=%q event=%q", strings.TrimSpace(buf.String()), event.Rendered)
		}
		if !json.Valid([]byte(event.Rendered)) {
			t.Fatalf("expected JSON render to be valid, got %q", event.Rendered)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for JSON subscription event")
	}
}

func TestSubscribeDisablesSemanticRenderWhenNoOutputEnabled(t *testing.T) {
	resetForTest()

	config := DefaultConfig()
	config.SetEnableText(false)
	config.SetEnableJSON(false)
	logger := NewLoggerWithConfig(&bytes.Buffer{}, config)

	events, cancel := Subscribe(10)
	defer cancel()

	logger.Info("structured only", "key", "value")

	select {
	case event := <-events:
		if event.Format != "" {
			t.Fatalf("expected no semantic format when all outputs disabled, got %q", event.Format)
		}
		if event.Rendered != "" {
			t.Fatalf("expected no semantic render when all outputs disabled, got %q", event.Rendered)
		}
		attrs := flattenRecordAttrs(event.Record)
		if attrs["key"] != "value" {
			t.Fatalf("expected structured record to remain available, got %v", attrs)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for subscription event")
	}
}

func flattenRecordAttrs(record slog.Record) map[string]string {
	flattened := make(map[string]string)
	record.Attrs(func(attr slog.Attr) bool {
		flattenAttr(flattened, "", attr)
		return true
	})
	return flattened
}

func flattenAttr(dst map[string]string, prefix string, attr slog.Attr) {
	attr.Value = attr.Value.Resolve()
	key := attr.Key
	if prefix != "" {
		key = prefix + "." + key
	}
	if attr.Value.Kind() == slog.KindGroup {
		for _, child := range attr.Value.Group() {
			flattenAttr(dst, key, child)
		}
		return
	}
	dst[key] = attr.Value.String()
}
