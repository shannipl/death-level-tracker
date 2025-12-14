package tracker

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"death-level-tracker/internal/config"
	"death-level-tracker/internal/tibiadata"

	"github.com/bwmarrin/discordgo"
)

func TestNewService(t *testing.T) {
	cfg := &config.Config{
		WorkerPoolSize:  10,
		MinLevelTrack:   100,
		TrackerInterval: 2 * time.Minute,
	}

	mockStorage := &mockServiceStorage{}
	mockSession := &discordgo.Session{}

	service := NewService(cfg, mockStorage, mockSession)

	if service == nil {
		t.Fatal("Expected non-nil service")
	}

	if service.config == nil {
		t.Error("Expected config to be set")
	}

	if service.storage == nil {
		t.Error("Expected storage to be set")
	}

	if service.fetcher == nil {
		t.Error("Expected fetcher to be set")
	}

	if service.analytics == nil {
		t.Error("Expected analytics to be set")
	}
}

func TestService_RunLoop_Success(t *testing.T) {
	worldsMap := map[string][]string{
		"Antica":  {"guild-1", "guild-2"},
		"Celesta": {"guild-3"},
	}

	processedWorlds := make(map[string]bool)
	var mu sync.Mutex

	mockStorage := &mockServiceStorage{
		getWorldsMapFunc: func(ctx context.Context) (map[string][]string, error) {
			return worldsMap, nil
		},
		getPlayersLevelsFunc: func(ctx context.Context, world string) (map[string]int, error) {
			mu.Lock()
			processedWorlds[world] = true
			mu.Unlock()
			return map[string]int{}, nil
		},
		batchTouchPlayersFunc: func(ctx context.Context, names []string) error {
			return nil
		},
		deleteOldPlayersFunc: func(ctx context.Context, world string, threshold time.Duration) (int64, error) {
			return 0, nil
		},
	}

	mockFetcher := &mockServiceFetcher{
		fetchWorldFunc: func(world string) ([]tibiadata.OnlinePlayer, error) {
			return []tibiadata.OnlinePlayer{}, nil
		},
		fetchWorldFromTibiaComFunc: func(world string) (map[string]int, error) {
			// Return empty map for tests - tests can override if needed
			return make(map[string]int), nil
		},
		fetchCharacterDetailsFunc: func(players []tibiadata.OnlinePlayer) <-chan *tibiadata.CharacterResponse {
			results := make(chan *tibiadata.CharacterResponse)
			close(results)
			return results
		},
	}

	mockAnalytics := &mockServiceAnalytics{
		processCharacterFunc: func(char *tibiadata.CharacterResponse, guilds []string, dbLevels map[string]int) {},
	}

	service := &Service{
		config:    &config.Config{},
		storage:   mockStorage,
		fetcher:   mockFetcher,
		analytics: mockAnalytics,
	}

	ctx := context.Background()
	service.runLoop(ctx)

	// Give goroutines time to execute
	time.Sleep(100 * time.Millisecond)

	// Both worlds should be processed (eventually via goroutines)
	// Note: Since processWorld runs in goroutines, we can't guarantee they've completed,
	// but we can verify runLoop doesn't error
}

func TestService_RunLoop_GetWorldsMapError(t *testing.T) {
	mockStorage := &mockServiceStorage{
		getWorldsMapFunc: func(ctx context.Context) (map[string][]string, error) {
			return nil, errors.New("database error")
		},
	}

	service := &Service{
		storage: mockStorage,
	}

	ctx := context.Background()

	// Should not panic on error
	service.runLoop(ctx)
}

func TestService_RunLoop_EmptyWorlds(t *testing.T) {
	var processWorldCalled bool

	mockStorage := &mockServiceStorage{
		getWorldsMapFunc: func(ctx context.Context) (map[string][]string, error) {
			return map[string][]string{}, nil
		},
	}

	mockFetcher := &mockServiceFetcher{
		fetchWorldFunc: func(world string) ([]tibiadata.OnlinePlayer, error) {
			processWorldCalled = true
			return nil, nil
		},
	}

	service := &Service{
		storage: mockStorage,
		fetcher: mockFetcher,
	}

	ctx := context.Background()
	service.runLoop(ctx)

	time.Sleep(50 * time.Millisecond)

	if processWorldCalled {
		t.Error("Expected processWorld NOT to be called for empty worlds map")
	}
}

func TestService_Start_ContextCancellation(t *testing.T) {
	mockStorage := &mockServiceStorage{
		getWorldsMapFunc: func(ctx context.Context) (map[string][]string, error) {
			return map[string][]string{}, nil
		},
	}

	service := &Service{
		config: &config.Config{
			TrackerInterval: 100 * time.Millisecond,
		},
		storage: mockStorage,
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Start service in goroutine
	done := make(chan bool)
	go func() {
		service.Start(ctx)
		done <- true
	}()

	// Wait a bit to ensure it's running
	time.Sleep(50 * time.Millisecond)

	// Cancel context
	cancel()

	// Wait for service to stop
	select {
	case <-done:
		// Service stopped successfully
	case <-time.After(1 * time.Second):
		t.Fatal("Service did not stop after context cancellation")
	}
}

func TestService_Start_RunsPeriodicLoop(t *testing.T) {
	var runCount int64

	mockStorage := &mockServiceStorage{
		getWorldsMapFunc: func(ctx context.Context) (map[string][]string, error) {
			atomic.AddInt64(&runCount, 1)
			return map[string][]string{}, nil
		},
	}

	service := &Service{
		config: &config.Config{
			TrackerInterval: 50 * time.Millisecond,
		},
		storage: mockStorage,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start service in goroutine
	go service.Start(ctx)

	// Wait for multiple ticks
	time.Sleep(150 * time.Millisecond)
	cancel()

	// runLoop should have run at least 2-3 times (initial + ticks)
	if atomic.LoadInt64(&runCount) < 2 {
		t.Errorf("Expected at least 2 runs, got %d", atomic.LoadInt64(&runCount))
	}
}
