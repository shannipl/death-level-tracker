package main

import (
	"context"
	"net/http"
	"testing"
	"time"

	"death-level-tracker/internal/config"
	"death-level-tracker/internal/core/ports"
)

type mockStore struct {
	ports.Repository
	closed bool
}

func (m *mockStore) Close() {
	m.closed = true
}

func TestApp_Shutdown(t *testing.T) {
	cfg := &config.Config{}
	store := &mockStore{}

	trackerCtx, trackerCancel := context.WithCancel(context.Background())

	metricsServer := &http.Server{Addr: ":0"}
	go func() {
		_ = metricsServer.ListenAndServe()
	}()
	time.Sleep(10 * time.Millisecond)

	app := &App{
		config:        cfg,
		store:         store,
		metricsServer: metricsServer,
		trackerCtx:    trackerCtx,
		trackerCancel: trackerCancel,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if err := app.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	if !store.closed {
		t.Error("Store was not closed")
	}

	scan := false
	select {
	case <-trackerCtx.Done():
		scan = true
	default:
	}
	if !scan {
		t.Error("Tracker context was not cancelled")
	}
}

func TestApp_Shutdown_NilComponents(t *testing.T) {
	app := &App{
		config: &config.Config{},
	}

	ctx := context.Background()
	if err := app.Shutdown(ctx); err != nil {
		t.Errorf("Shutdown failed with nil components: %v", err)
	}
}

func TestStartMetricsServer(t *testing.T) {
	app := &App{
		config: &config.Config{},
	}

	app.startMetricsServer()

	if app.metricsServer == nil {
		t.Error("Metrics server not initialized")
	}

	_ = app.metricsServer.Close()
}
