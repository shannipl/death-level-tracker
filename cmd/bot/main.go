package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"death-level-tracker/internal/config"
)

func main() {
	InitLogger()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()

	app, err := NewApp(ctx, cfg)
	if err != nil {
		slog.Error("Failed to initialize application", "error", err)
		os.Exit(1)
	}

	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := app.Shutdown(shutdownCtx); err != nil {
			slog.Error("Application shutdown error", "error", err)
		}
	}()

	if err := app.Run(); err != nil {
		slog.Error("Failed to start application", "error", err)
		os.Exit(1)
	}

	WaitForShutdown()
}
