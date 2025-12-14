package main

import (
	"context"
	"testing"
	"time"

	"death-level-tracker/internal/config"
)

// TestApp_Shutdown tests the Shutdown method
func TestApp_Shutdown(t *testing.T) {
	// Create a real app structure but with minimal setup
	app := &App{
		config: &config.Config{},
	}

	// Set up tracker context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	app.trackerCtx = ctx
	app.trackerCancel = cancel

	// Should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Shutdown panicked: %v", r)
		}
	}()

	app.Shutdown()

	// Verify context was cancelled
	select {
	case <-app.trackerCtx.Done():
		// Expected - context should be cancelled
	case <-time.After(100 * time.Millisecond):
		t.Error("Context was not cancelled")
	}
}

func TestApp_Shutdown_NilCancel(t *testing.T) {
	app := &App{
		config:        &config.Config{},
		trackerCancel: nil, // nil cancel function
	}

	// Should not panic with nil cancel
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Shutdown panicked with nil cancel: %v", r)
		}
	}()

	app.Shutdown()
}

func TestApp_Shutdown_NilDiscord(t *testing.T) {
	app := &App{
		discord: nil,
	}

	// Should not panic with nil discord
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Shutdown panicked with nil discord: %v", r)
		}
	}()

	app.Shutdown()
}

func TestApp_Shutdown_NilStore(t *testing.T) {
	app := &App{
		store: nil,
	}

	// Should not panic with nil store
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Shutdown panicked with nil store: %v", r)
		}
	}()

	app.Shutdown()
}

// Note: Testing NewApp requires a full environment setup (DATABASE_URL, valid config, etc.)
// and is better suited for integration tests.

// Note: Testing Run requires a real Discord session which would make actual API calls.
// This is better suited for integration/E2E tests. The Run method's logic is straightforward:
// 1. Opens Discord session
// 2. Registers commands
// 3. Starts tracker service
// These are tested individually in their respective unit tests.
