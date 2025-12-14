package main

import (
	"os"
	"syscall"
	"testing"
	"time"
)

func TestWaitForShutdown_SIGINT(t *testing.T) {
	// Start WaitForShutdown in a goroutine
	done := make(chan bool)
	go func() {
		WaitForShutdown()
		done <- true
	}()

	// Give the signal handler time to set up
	time.Sleep(50 * time.Millisecond)

	// Send SIGINT
	currentProcess, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatalf("Failed to find current process: %v", err)
	}

	if err := currentProcess.Signal(syscall.SIGINT); err != nil {
		t.Fatalf("Failed to send SIGINT: %v", err)
	}

	// Wait for WaitForShutdown to complete
	select {
	case <-done:
		// Success - function returned after receiving signal
	case <-time.After(1 * time.Second):
		t.Fatal("WaitForShutdown did not return after receiving SIGINT")
	}
}

func TestWaitForShutdown_SIGTERM(t *testing.T) {
	// Start WaitForShutdown in a goroutine
	done := make(chan bool)
	go func() {
		WaitForShutdown()
		done <- true
	}()

	// Give the signal handler time to set up
	time.Sleep(50 * time.Millisecond)

	// Send SIGTERM
	currentProcess, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatalf("Failed to find current process: %v", err)
	}

	if err := currentProcess.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("Failed to send SIGTERM: %v", err)
	}

	// Wait for WaitForShutdown to complete
	select {
	case <-done:
		// Success - function returned after receiving signal
	case <-time.After(1 * time.Second):
		t.Fatal("WaitForShutdown did not return after receiving SIGTERM")
	}
}

func TestWaitForShutdown_Interrupt(t *testing.T) {
	// Start WaitForShutdown in a goroutine
	done := make(chan bool)
	go func() {
		WaitForShutdown()
		done <- true
	}()

	// Give the signal handler time to set up
	time.Sleep(50 * time.Millisecond)

	// Send os.Interrupt
	currentProcess, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatalf("Failed to find current process: %v", err)
	}

	if err := currentProcess.Signal(os.Interrupt); err != nil {
		t.Fatalf("Failed to send interrupt: %v", err)
	}

	// Wait for WaitForShutdown to complete
	select {
	case <-done:
		// Success - function returned after receiving signal
	case <-time.After(1 * time.Second):
		t.Fatal("WaitForShutdown did not return after receiving interrupt signal")
	}
}

func TestWaitForShutdown_DoesNotReturnWithoutSignal(t *testing.T) {
	// Start WaitForShutdown in a goroutine
	done := make(chan bool)
	go func() {
		WaitForShutdown()
		done <- true
	}()

	// Wait a short time WITHOUT sending a signal
	select {
	case <-done:
		t.Fatal("WaitForShutdown returned without receiving a signal")
	case <-time.After(200 * time.Millisecond):
		// Expected - function should still be waiting

		// Clean up by sending a signal so the goroutine doesn't leak
		currentProcess, _ := os.FindProcess(os.Getpid())
		_ = currentProcess.Signal(syscall.SIGINT)
		<-done // Wait for it to finish
	}
}
