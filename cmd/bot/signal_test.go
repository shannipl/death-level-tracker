package main

import (
	"os"
	"syscall"
	"testing"
	"time"
)

func TestWaitForShutdown_SIGINT(t *testing.T) {
	done := make(chan bool)
	go func() {
		WaitForShutdown()
		done <- true
	}()

	time.Sleep(50 * time.Millisecond)

	currentProcess, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatalf("Failed to find current process: %v", err)
	}

	if err := currentProcess.Signal(syscall.SIGINT); err != nil {
		t.Fatalf("Failed to send SIGINT: %v", err)
	}

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("WaitForShutdown did not return after receiving SIGINT")
	}
}

func TestWaitForShutdown_SIGTERM(t *testing.T) {
	done := make(chan bool)
	go func() {
		WaitForShutdown()
		done <- true
	}()

	time.Sleep(50 * time.Millisecond)

	currentProcess, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatalf("Failed to find current process: %v", err)
	}

	if err := currentProcess.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("Failed to send SIGTERM: %v", err)
	}

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("WaitForShutdown did not return after receiving SIGTERM")
	}
}

func TestWaitForShutdown_Interrupt(t *testing.T) {
	done := make(chan bool)
	go func() {
		WaitForShutdown()
		done <- true
	}()

	time.Sleep(50 * time.Millisecond)

	currentProcess, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatalf("Failed to find current process: %v", err)
	}

	if err := currentProcess.Signal(os.Interrupt); err != nil {
		t.Fatalf("Failed to send interrupt: %v", err)
	}

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("WaitForShutdown did not return after receiving interrupt signal")
	}
}

func TestWaitForShutdown_DoesNotReturnWithoutSignal(t *testing.T) {
	done := make(chan bool)
	go func() {
		WaitForShutdown()
		done <- true
	}()

	select {
	case <-done:
		t.Fatal("WaitForShutdown returned without receiving a signal")
	case <-time.After(200 * time.Millisecond):
		currentProcess, _ := os.FindProcess(os.Getpid())
		_ = currentProcess.Signal(syscall.SIGINT)
		<-done
	}
}
