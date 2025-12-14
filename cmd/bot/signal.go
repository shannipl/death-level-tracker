package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func WaitForShutdown() {
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
	slog.Info("Shutdown signal received")
}
