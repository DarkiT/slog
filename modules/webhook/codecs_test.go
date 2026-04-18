package webhook

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"
)

type customCodec struct{}

func (c customCodec) Name() string { return "custom" }

func (c customCodec) Encode(_ context.Context, record *slog.Record, attrs []slog.Attr, groups []string) ([]byte, error) {
	_ = attrs
	_ = groups
	return []byte(`{"message":"` + record.Message + `"}`), nil
}

func TestCodecRegistry_Defaults(t *testing.T) {
	if _, ok := GetCodec("default"); !ok {
		t.Fatal("default codec missing")
	}
	if _, ok := GetCodec("json"); !ok {
		t.Fatal("json codec missing")
	}
}

func TestCodecRegistry_RegisterCustom(t *testing.T) {
	if err := RegisterCodec(customCodec{}); err != nil {
		t.Fatalf("register: %v", err)
	}
	codec, ok := GetCodec("custom")
	if !ok {
		t.Fatal("custom codec missing")
	}
	rec := slog.Record{Time: time.Now(), Level: slog.LevelInfo, Message: "hello"}
	b, err := codec.Encode(context.Background(), &rec, nil, nil)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if !strings.Contains(string(b), "\"hello\"") {
		t.Fatalf("unexpected output: %s", string(b))
	}
}

func TestJSONCodec_Encode(t *testing.T) {
	codec, _ := GetCodec("json")
	rec := slog.Record{Time: time.Now(), Level: slog.LevelWarn, Message: "warn"}
	rec.AddAttrs(slog.String("user", "alice"))
	b, err := codec.Encode(context.Background(), &rec, nil, nil)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if m["message"] != "warn" || m["user"] != "alice" {
		t.Fatalf("unexpected payload: %#v", m)
	}
}

func TestDefaultCodec_Encode(t *testing.T) {
	codec, ok := GetCodec("default")
	if !ok {
		t.Fatal("default codec missing")
	}
	rec := slog.Record{Time: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC), Level: slog.LevelInfo, Message: "test message"}
	rec.AddAttrs(slog.String("custom", "value"))
	b, err := codec.Encode(context.Background(), &rec, nil, nil)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if m["message"] != "test message" || m["level"] != "INFO" {
		t.Fatalf("unexpected core fields: %#v", m)
	}
	extra, ok := m["extra"].(map[string]any)
	if !ok {
		t.Fatalf("missing extra: %#v", m)
	}
	if extra["custom"] != "value" {
		t.Fatalf("unexpected extra: %#v", extra)
	}
}
