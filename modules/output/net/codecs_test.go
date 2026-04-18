package outputnet

import (
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"
)

type customCodec struct{}

func (c customCodec) Name() string { return "custom" }

func (c customCodec) Encode(record *slog.Record, attrs []slog.Attr, groups []string) ([]byte, error) {
	_ = attrs
	_ = groups
	return []byte("x:" + record.Message), nil
}

func TestCodecRegistry_Defaults(t *testing.T) {
	if _, ok := GetCodec("raw"); !ok {
		t.Fatal("raw codec missing")
	}
	if _, ok := GetCodec("json"); !ok {
		t.Fatal("json codec missing")
	}
}

func TestCodecRegistry_RegisterCustom(t *testing.T) {
	if err := RegisterCodec(customCodec{}); err != nil {
		t.Fatalf("register codec: %v", err)
	}
	codec, ok := GetCodec("custom")
	if !ok {
		t.Fatal("custom codec not found")
	}
	rec := slog.Record{Time: time.Now(), Level: slog.LevelInfo, Message: "hello"}
	out, err := codec.Encode(&rec, nil, nil)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if string(out) != "x:hello" {
		t.Fatalf("unexpected codec output: %q", string(out))
	}
}

func TestJSONCodec_Encode(t *testing.T) {
	codec, _ := GetCodec("json")
	rec := slog.Record{Time: time.Now(), Level: slog.LevelWarn, Message: "warn"}
	rec.AddAttrs(slog.String("trace_id", "t-1"))
	out, err := codec.Encode(&rec, []slog.Attr{slog.String("user", "alice")}, []string{"req"})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if !strings.Contains(string(out), "\"message\":\"warn\"") {
		t.Fatalf("unexpected json output: %s", string(out))
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if m["user"] != "alice" {
		t.Fatalf("expected user attr, got %v", m["user"])
	}
	req, ok := m["req"].(map[string]any)
	if !ok {
		t.Fatalf("expected grouped attrs in req, got %T", m["req"])
	}
	if req["trace_id"] != "t-1" {
		t.Fatalf("expected req.trace_id attr, got %v", req["trace_id"])
	}
}
