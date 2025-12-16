package slog

import "testing"

func TestApplyRuntimeOption(t *testing.T) {
	originalLevel := GetLevel()
	originalText, originalJSON := textEnabled, jsonEnabled
	defer func() {
		if originalText {
			EnableTextLogger()
		} else {
			DisableTextLogger()
		}
		if originalJSON {
			EnableJSONLogger()
		} else {
			DisableJSONLogger()
		}
		_ = SetLevel(originalLevel)
	}()

	if _, err := ApplyRuntimeOption("level", "warn"); err != nil {
		t.Fatalf("set level failed: %v", err)
	}
	if GetLevel() != LevelWarn {
		t.Fatalf("level not updated")
	}

	snap, _ := ApplyRuntimeOption("text", "off")
	if snap.TextEnabled {
		t.Fatalf("expected text disabled")
	}

	snap, _ = ApplyRuntimeOption("json", "on")
	if !snap.JSONEnabled {
		t.Fatalf("expected json enabled")
	}

	if _, err := ApplyRuntimeOption("unknown", "x"); err == nil {
		t.Fatalf("expected error for unknown option")
	}
}
