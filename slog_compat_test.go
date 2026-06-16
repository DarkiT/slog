package slog_test

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/darkit/slog"
)

func TestRootPackageProvidesStdSlogSurface(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	stdLogger := slog.New(handler)

	slog.SetDefault(stdLogger)
	slog.SetLogLoggerLevel(slog.LevelDebug)

	logger := slog.Default().With(
		slog.String("service", "billing"),
		slog.GroupAttrs("request", slog.String("id", "req-1")),
	)

	if !logger.Enabled(context.Background(), slog.LevelInfo) {
		t.Fatal("expected logger to be enabled for info level")
	}

	logger.LogAttrs(context.Background(), slog.LevelInfo, "hello",
		slog.Int("attempt", 1),
		slog.Duration("elapsed", time.Millisecond),
	)

	record := slog.NewRecord(time.Now(), slog.LevelWarn, "manual", 0)
	record.AddAttrs(slog.Bool("ok", true), slog.Any("payload", map[string]int{"n": 1}))
	if err := logger.Handler().Handle(context.Background(), record); err != nil {
		t.Fatalf("handle record: %v", err)
	}

	value := slog.AnyValue("value")
	if value.Kind() != slog.KindString || value.String() != "value" {
		t.Fatalf("unexpected value mapping: kind=%s value=%s", value.Kind(), value.String())
	}

	var levelVar slog.LevelVar
	levelVar.Set(slog.LevelWarn)
	if levelVar.Level() != slog.LevelWarn {
		t.Fatalf("unexpected level var value: %s", levelVar.Level())
	}

	output := buf.String()
	for _, want := range []string{"hello", "service=billing", "request.id=req-1", "attempt=1", "manual", "ok=true"} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected output to contain %q, got %q", want, output)
		}
	}
}

func TestRootPackageMultiAndDiscardHandlers(t *testing.T) {
	var left bytes.Buffer
	var right bytes.Buffer

	multi := slog.NewMultiHandler(
		slog.NewTextHandler(&left, nil),
		slog.NewTextHandler(&right, nil),
		slog.DiscardHandler,
	)
	logger := slog.New(multi).With(slog.String("scope", "compat"))
	logger.Info("fanout")

	for name, output := range map[string]string{"left": left.String(), "right": right.String()} {
		if !strings.Contains(output, "fanout") || !strings.Contains(output, "scope=compat") {
			t.Fatalf("%s handler did not receive expected output: %q", name, output)
		}
	}
}
