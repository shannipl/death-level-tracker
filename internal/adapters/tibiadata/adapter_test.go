package tibiadata

import (
	"testing"
	"time"

	"death-level-tracker/internal/adapters/tibiadata/api"
	"death-level-tracker/internal/config"
)

func TestNewAdapter(t *testing.T) {
	client := api.NewClient()
	cfg := &config.Config{
		WorkerPoolSize: 10,
	}

	adapter := NewAdapter(client, cfg)

	if adapter == nil {
		t.Fatal("Expected non-nil adapter")
	}

	if adapter.client != client {
		t.Error("Expected client to be referenced correctly")
	}

	if adapter.config != cfg {
		t.Error("Expected config to be referenced correctly")
	}

	// Verify internal HTTP client initialization
	if adapter.tibiaComClient == nil {
		t.Error("Expected internal tibiaComClient to be initialized")
	} else {
		if adapter.tibiaComClient.Timeout != 30*time.Second {
			t.Errorf("Expected tibiaComClient timeout to be 30s, got %v", adapter.tibiaComClient.Timeout)
		}
	}
}
