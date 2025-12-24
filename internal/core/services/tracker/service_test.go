package tracker

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"death-level-tracker/internal/config"
	"death-level-tracker/internal/core/domain"
)

func TestNewServiceWithDependencies(t *testing.T) {
	deps := Dependencies{
		Config:   &config.Config{MinLevelTrack: 100},
		Storage:  &mockServiceStorage{},
		Fetcher:  &mockServiceFetcher{},
		Notifier: &mockServiceNotifier{},
	}

	service := NewService(deps)

	if service == nil {
		t.Fatal("expected non-nil")
	}
	if service.config != deps.Config {
		t.Error("expected config to be set")
	}
	if service.storage == nil {
		t.Error("expected storage to be set")
	}
	if service.fetcher == nil {
		t.Error("expected fetcher to be set")
	}
	if service.levelTracker == nil {
		t.Error("expected levelTracker to be set")
	}
	if service.deathTracker == nil {
		t.Error("expected deathTracker to be set")
	}
}

func TestRunLoop(t *testing.T) {
	t.Run("groups by world", func(t *testing.T) {
		storage := &mockServiceStorage{
			getAllGuildConfigsFunc: func(ctx context.Context) ([]domain.GuildConfig, error) {
				return []domain.GuildConfig{
					{DiscordGuildID: "g1", World: "Antica"},
					{DiscordGuildID: "g2", World: "Secura"},
				}, nil
			},
			getPlayersLevelsFunc: func(ctx context.Context, world string) (map[string]int, error) {
				return nil, nil
			},
			deleteOldPlayersFunc: func(ctx context.Context, world string, d time.Duration) (int64, error) {
				return 0, nil
			},
		}

		fetcher := &mockServiceFetcher{
			fetchWorldFunc: func(ctx context.Context, world string) ([]domain.Player, error) {
				return []domain.Player{}, nil
			},
			fetchCharacterDetailsFunc: func(ctx context.Context, names []string) (chan *domain.Player, error) {
				ch := make(chan *domain.Player)
				close(ch)
				return ch, nil
			},
		}

		cfg := &config.Config{}
		service := &Service{
			config:       cfg,
			storage:      storage,
			fetcher:      fetcher,
			levelTracker: NewLevelTracker(cfg, storage, &mockServiceNotifier{}),
			deathTracker: NewDeathTracker(&mockServiceNotifier{}),
		}

		service.runLoop(context.Background())
		time.Sleep(50 * time.Millisecond)
	})

	t.Run("handles error", func(t *testing.T) {
		storage := &mockServiceStorage{
			getAllGuildConfigsFunc: func(ctx context.Context) ([]domain.GuildConfig, error) {
				return nil, errors.New("db error")
			},
		}

		service := &Service{storage: storage}
		service.runLoop(context.Background())
	})
}

func TestStart(t *testing.T) {
	t.Run("stops on cancel", func(t *testing.T) {
		storage := &mockServiceStorage{
			getAllGuildConfigsFunc: func(ctx context.Context) ([]domain.GuildConfig, error) {
				return []domain.GuildConfig{}, nil
			},
		}

		service := &Service{
			config:  &config.Config{TrackerInterval: 100 * time.Millisecond},
			storage: storage,
		}

		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		go func() {
			service.Start(ctx)
			close(done)
		}()

		time.Sleep(50 * time.Millisecond)
		cancel()

		select {
		case <-done:
		case <-time.After(1 * time.Second):
			t.Fatal("did not stop")
		}
	})

	t.Run("runs periodic", func(t *testing.T) {
		var count int64

		storage := &mockServiceStorage{
			getAllGuildConfigsFunc: func(ctx context.Context) ([]domain.GuildConfig, error) {
				atomic.AddInt64(&count, 1)
				return []domain.GuildConfig{}, nil
			},
		}

		service := &Service{
			config:  &config.Config{TrackerInterval: 50 * time.Millisecond},
			storage: storage,
		}

		ctx, cancel := context.WithCancel(context.Background())
		go service.Start(ctx)

		time.Sleep(130 * time.Millisecond)
		cancel()

		if atomic.LoadInt64(&count) < 2 {
			t.Errorf("expected at least 2, got %d", count)
		}
	})
}

func TestGroupConfigsByWorld(t *testing.T) {
	t.Run("groups", func(t *testing.T) {
		configs := []domain.GuildConfig{
			{DiscordGuildID: "g1", World: "Antica"},
			{DiscordGuildID: "g2", World: "Antica"},
			{DiscordGuildID: "g3", World: "Secura"},
		}
		result := groupConfigsByWorld(configs)
		if len(result) != 2 {
			t.Errorf("expected 2, got %d", len(result))
		}
		if len(result["Antica"]) != 2 {
			t.Errorf("expected 2 in Antica, got %d", len(result["Antica"]))
		}
	})

	t.Run("skips empty", func(t *testing.T) {
		configs := []domain.GuildConfig{{DiscordGuildID: "g1", World: ""}}
		result := groupConfigsByWorld(configs)
		if len(result) != 0 {
			t.Errorf("expected 0, got %d", len(result))
		}
	})
}
