package logfmt

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestLogfmtHandler(t *testing.T) {
	buf := &bytes.Buffer{}
	h := New(Option{Writer: buf})

	r := slog.Record{Time: time.Unix(0, 0), Level: slog.LevelInfo, Message: "hello world"}
	r.AddAttrs(slog.String("user", "alice"), slog.Int("attempt", 3), slog.Group("ctx", slog.String("trace", "abc")))

	if err := h.Handle(context.Background(), r); err != nil {
		t.Fatalf("handle error: %v", err)
	}

	out := strings.TrimSpace(buf.String())
	if !strings.Contains(out, "time=1970") || !strings.Contains(out, "level=Info") || !strings.Contains(out, "msg=\"hello world\"") {
		t.Fatalf("missing core fields: %s", out)
	}
	if !strings.Contains(out, "user=alice") || !strings.Contains(out, "attempt=3") || !strings.Contains(out, "ctx.trace=abc") {
		t.Fatalf("missing attrs: %s", out)
	}
}
