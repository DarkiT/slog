package slog

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestLoggerBuilder_BuildsLogger(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLoggerBuilder().
		WithWriter(buf).
		WithModule("order").
		WithGroup("api").
		WithAttrs(String("req_id", "r1")).
		EnableJSON(false).
		EnableText(true).
		Build()

	logger.Info("ok")
	out := buf.String()
	if !strings.Contains(out, "module=order") {
		t.Fatalf("expected module field, got %s", out)
	}
	if !strings.Contains(out, "api.req_id=r1") {
		t.Fatalf("expected grouped attr, got %s", out)
	}
}

func TestLoggerBuilder_ContextHelper(t *testing.T) {
	SetContextPropagator(func(ctx context.Context) []Attr {
		if v, ok := ctx.Value("trace_id").(string); ok {
			return []Attr{String("trace_id", v)}
		}
		return nil
	})

	buf := &bytes.Buffer{}
	logger := NewLoggerBuilder().WithWriter(buf).Build()
	ctx := context.WithValue(context.Background(), "trace_id", "abc-123")
	logger.InfoContext(ctx, "ctx message")

	out := buf.String()
	if !strings.Contains(out, "trace_id=abc-123") {
		t.Fatalf("expected propagated trace_id, got %s", out)
	}
}

func TestLoggerBuilder_LogfmtMode(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLoggerBuilder().WithWriter(buf).UseLogfmt().Build()
	logger.Info("lfmt", String("k", "v"))
	out := buf.String()
	if !strings.Contains(out, "k=v") || strings.Contains(out, "{") {
		t.Fatalf("logfmt output malformed: %s", out)
	}
}
