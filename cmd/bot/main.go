package main

import (
	"context"
	"log/slog"
	"os"
)

func main() {
	InitLogger()

	ctx := context.Background()

	app, err := NewApp(ctx)
	if err != nil {
		slog.Error("Failed to initialize application", "error", err)
		os.Exit(1)
	}
	defer app.Shutdown()

	if err := app.Run(); err != nil {
		slog.Error("Failed to start application", "error", err)
		os.Exit(1)
	}

	WaitForShutdown()
}
