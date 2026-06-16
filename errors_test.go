package slog

import (
	"fmt"
	"testing"
)

func TestSlogErrorHelpersSupportWrappedErrors(t *testing.T) {
	base := NewProcessingError("writer", "rotate", fmt.Errorf("disk full"))
	err := fmt.Errorf("outer: %w", base)

	if !IsErrorType(err, ErrorTypeProcessing) {
		t.Fatalf("IsErrorType should match wrapped SlogError")
	}
	if got := GetErrorComponent(err); got != "writer" {
		t.Fatalf("GetErrorComponent() = %q, want writer", got)
	}
	if got := GetErrorOperation(err); got != "rotate" {
		t.Fatalf("GetErrorOperation() = %q, want rotate", got)
	}
}
