package gelf

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"
)

func TestGELFHandler(t *testing.T) {
	buf := &bytes.Buffer{}
	h := New(Options{Writer: buf, Host: "test-host", Facility: "app"})

	r := slog.Record{Time: time.Unix(1_700_000_000, 0), Level: slog.LevelWarn, Message: "warn msg"}
	r.AddAttrs(slog.String("user", "bob"))

	if err := h.Handle(context.Background(), r); err != nil {
		t.Fatalf("handle error: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &payload); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if payload["version"] != "1.1" || payload["host"] != "test-host" || payload["facility"] != "app" {
		t.Fatalf("header mismatch: %v", payload)
	}
	if payload["short_message"] != "warn msg" {
		t.Fatalf("message mismatch: %v", payload["short_message"])
	}
	if payload["_user"] != "bob" {
		t.Fatalf("user mismatch: %v", payload["_user"])
	}
}
