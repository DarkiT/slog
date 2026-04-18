package webhook

import (
	"testing"

	"github.com/darkit/slog/modules"
)

func TestWebhookAdapter_ConfigureWithCodec(t *testing.T) {
	adapter := NewWebhookAdapter()
	err := adapter.Configure(modules.Config{
		"endpoint": "http://localhost",
		"codec":    "json",
	})
	if err != nil {
		t.Fatalf("configure: %v", err)
	}
	if adapter.Handler() == nil {
		t.Fatal("expected handler")
	}
}

func TestWebhookAdapter_ConfigureInvalidCodec(t *testing.T) {
	adapter := NewWebhookAdapter()
	err := adapter.Configure(modules.Config{
		"endpoint": "http://localhost",
		"codec":    "invalid",
	})
	if err == nil {
		t.Fatal("expected invalid codec error")
	}
}
