package multi

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"
)

// TestRouterPropagatesAttrs 确认 Router 在匹配后保留附加属性
func TestRouterPropagatesAttrs(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, nil)

	r := Router().Add(handler, func(ctx context.Context, record slog.Record) bool {
		return true
	})

	routed := r.Handler().WithAttrs([]slog.Attr{slog.String("route", "matched")}).WithGroup("meta")

	record := slog.NewRecord(time.Now(), slog.LevelInfo, "router-test", 0)
	record.AddAttrs(slog.String("payload", "ok"))

	if err := routed.Handle(context.Background(), record); err != nil {
		t.Fatalf("router handler returned error: %v", err)
	}

	t.Logf("router json output: %s", buf.Bytes())

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to decode router output: %v", err)
	}

	if entry["route"] != "matched" {
		t.Fatalf("expected route attribute to propagate, got %v", entry["route"])
	}

	meta, ok := entry["meta"].(map[string]any)
	if !ok {
		t.Fatalf("expected meta group in JSON output, got %T", entry["meta"])
	}

	nested, ok := meta["meta"].(map[string]any)
	if !ok {
		t.Fatalf("expected nested meta payload container, got %T", meta["meta"])
	}
	if nested["payload"] != "ok" {
		t.Fatalf("expected payload in nested meta group, got %v", nested["payload"])
	}
}
