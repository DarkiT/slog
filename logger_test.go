package slog

import (
	"bytes"
	"strings"
	"testing"
)

func TestLoggerWithValue(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&buf, false, false)

	logger.WithValue("test", "value").Info("test message")

	output := buf.String()
	if !strings.Contains(output, "test=value") {
		t.Errorf("Expected output to contain context value, got: %s", output)
	}
}
