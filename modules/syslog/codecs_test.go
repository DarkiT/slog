package syslog

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/darkit/slog/modules"
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
	out, err := codec.Encode(context.Background(), &rec, nil, nil)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if !strings.Contains(string(out), "\"hello\"") {
		t.Fatalf("unexpected output: %q", string(out))
	}
}

func TestDefaultCodec_Encode(t *testing.T) {
	codec, _ := GetCodec("default")
	rec := slog.Record{Time: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC), Level: slog.LevelInfo, Message: "test message"}
	rec.AddAttrs(slog.String("custom", "value"))
	out, err := codec.Encode(context.Background(), &rec, nil, nil)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if m["message"] != "test message" || m["level"] != "INFO" {
		t.Fatalf("unexpected payload: %#v", m)
	}
	extra, ok := m["extra"].(map[string]any)
	if !ok || extra["custom"] != "value" {
		t.Fatalf("unexpected extra: %#v", m["extra"])
	}
}

func TestJSONCodec_Encode(t *testing.T) {
	codec, _ := GetCodec("json")
	rec := slog.Record{Time: time.Now(), Level: slog.LevelWarn, Message: "warn"}
	rec.AddAttrs(slog.String("user", "alice"))
	out, err := codec.Encode(context.Background(), &rec, nil, nil)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if m["message"] != "warn" || m["user"] != "alice" {
		t.Fatalf("unexpected payload: %#v", m)
	}
}

func TestSyslogAdapter_ConfigureWithCodec(t *testing.T) {
	adapter := NewSyslogAdapter()
	err := adapter.Configure(modules.Config{
		"network": "udp",
		"addr":    "127.0.0.1:9999",
		"codec":   "json",
	})
	if err != nil {
		t.Fatalf("configure: %v", err)
	}
	if adapter.Handler() == nil {
		t.Fatal("expected handler")
	}
}

func TestSyslogAdapter_InvalidCodec(t *testing.T) {
	adapter := NewSyslogAdapter()
	err := adapter.Configure(modules.Config{
		"network": "udp",
		"addr":    "127.0.0.1:9999",
		"codec":   "not-exist",
	})
	if err == nil {
		t.Fatal("expected invalid codec error")
	}
}
