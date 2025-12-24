package tracker

import (
	"context"
	"errors"
	"testing"
	"time"

	"death-level-tracker/internal/config"
	"death-level-tracker/internal/core/domain"
)

func makeWorldContext(world string) *worldContext {
	return &worldContext{
		world:       world,
		guilds:      []domain.GuildConfig{{DiscordGuildID: "guild-1"}},
		dbLevels:    map[string]int{},
		memberships: map[string]map[string]bool{},
	}
}

func makeService(storage *mockServiceStorage, fetcher *mockServiceFetcher, notifier *mockServiceNotifier, cfg *config.Config) *Service {
	if cfg == nil {
		cfg = &config.Config{MinLevelTrack: 100}
	}
	if storage == nil {
		storage = &mockServiceStorage{}
	}
	if fetcher == nil {
		fetcher = &mockServiceFetcher{}
	}
	if notifier == nil {
		notifier = &mockServiceNotifier{}
	}

	return &Service{
		config:       cfg,
		storage:      storage,
		fetcher:      fetcher,
		levelTracker: NewLevelTracker(cfg, storage, notifier),
		deathTracker: NewDeathTracker(notifier),
		guildCache:   make(map[string]GuildCacheItem),
	}
}

func TestFetchPlayerLevels(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		storage := &mockServiceStorage{
			getPlayersLevelsFunc: func(ctx context.Context, world string) (map[string]int, error) {
				return map[string]int{"P1": 100}, nil
			},
		}
		service := &Service{storage: storage}
		levels, err := service.fetchPlayerLevels(context.Background(), "Antica")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(levels) != 1 {
			t.Errorf("expected 1, got %d", len(levels))
		}
	})

	t.Run("error", func(t *testing.T) {
		storage := &mockServiceStorage{
			getPlayersLevelsFunc: func(ctx context.Context, world string) (map[string]int, error) {
				return nil, errors.New("db error")
			},
		}
		service := &Service{storage: storage}
		_, err := service.fetchPlayerLevels(context.Background(), "Antica")
		if err == nil {
			t.Error("expected error")
		}
	})
}

func TestFetchGuildMemberships(t *testing.T) {
	t.Run("fetches unique guilds", func(t *testing.T) {
		fetcher := &mockServiceFetcher{
			fetchGuildMembersFunc: func(ctx context.Context, name string) ([]string, error) {
				return []string{"M1", "M2"}, nil
			},
		}
		guilds := []domain.GuildConfig{{TibiaGuilds: []string{"G1", "G2"}}}
		service := makeService(nil, fetcher, nil, nil)
		memberships := service.fetchGuildMemberships(context.Background(), guilds)
		if len(memberships) != 2 {
			t.Errorf("expected 2, got %d", len(memberships))
		}
	})

	t.Run("handles error", func(t *testing.T) {
		fetcher := &mockServiceFetcher{
			fetchGuildMembersFunc: func(ctx context.Context, name string) ([]string, error) {
				return nil, errors.New("error")
			},
		}
		service := makeService(nil, fetcher, nil, nil)
		memberships := service.fetchGuildMemberships(context.Background(), []domain.GuildConfig{{TibiaGuilds: []string{"G1"}}})
		if len(memberships) != 0 {
			t.Errorf("expected 0, got %d", len(memberships))
		}
	})

	t.Run("uses stale cache on error", func(t *testing.T) {
		callCount := 0
		fetcher := &mockServiceFetcher{
			fetchGuildMembersFunc: func(ctx context.Context, name string) ([]string, error) {
				callCount++
				if callCount == 1 {
					return []string{"M1"}, nil
				}
				return nil, errors.New("error")
			},
		}
		service := makeService(nil, fetcher, nil, nil)
		guilds := []domain.GuildConfig{{TibiaGuilds: []string{"G1"}}}

		// First call: success, populates cache
		memberships := service.fetchGuildMemberships(context.Background(), guilds)
		if len(memberships["G1"]) != 1 {
			t.Errorf("expected 1 member")
		}

		// Force expire cache to trigger fetch?
		// No, if cache is valid it won't fetch.
		// We want to test "fallback if fetch fails".
		// But if cache is valid, we don't fetch.
		// If cache is expired, we fetch. If fetch fails, we use stale cache.

		// Manually expire the cache item
		service.cacheMu.Lock()
		if item, ok := service.guildCache["G1"]; ok {
			item.ExpiresAt = time.Now().Add(-1 * time.Hour)
			service.guildCache["G1"] = item
		}
		service.cacheMu.Unlock()

		// Second call: should try fetch (fail) then use stale cache
		memberships = service.fetchGuildMemberships(context.Background(), guilds)
		if len(memberships["G1"]) != 1 {
			t.Errorf("expected 1 member from stale cache")
		}
		if callCount != 2 {
			t.Errorf("expected 2 calls, got %d", callCount)
		}
	})
}

func TestInitWorldContext(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		storage := &mockServiceStorage{
			getPlayersLevelsFunc: func(ctx context.Context, world string) (map[string]int, error) {
				return map[string]int{"P1": 100}, nil
			},
		}
		fetcher := &mockServiceFetcher{}
		service := &Service{storage: storage, fetcher: fetcher}
		wctx := service.initWorldContext(context.Background(), "Antica", nil)
		if wctx == nil {
			t.Fatal("expected non-nil")
		}
	})

	t.Run("error", func(t *testing.T) {
		storage := &mockServiceStorage{
			getPlayersLevelsFunc: func(ctx context.Context, world string) (map[string]int, error) {
				return nil, errors.New("error")
			},
		}
		service := &Service{storage: storage}
		wctx := service.initWorldContext(context.Background(), "Antica", nil)
		if wctx != nil {
			t.Error("expected nil on error")
		}
	})
}

func TestProcessCharacters(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var upserted bool
		fetcher := &mockServiceFetcher{
			fetchCharacterDetailsFunc: func(ctx context.Context, names []string) (chan *domain.Player, error) {
				ch := make(chan *domain.Player, len(names))
				for _, n := range names {
					ch <- &domain.Player{Name: n, Level: 200, World: "Antica"}
				}
				close(ch)
				return ch, nil
			},
		}
		storage := &mockServiceStorage{
			upsertPlayerLevelFunc: func(ctx context.Context, name string, level int, world string) error {
				upserted = true
				return nil
			},
		}

		service := makeService(storage, fetcher, nil, &config.Config{MinLevelTrack: 100})
		players := []domain.Player{{Name: "P1", Level: 200}} // Will trigger fetch -> then level check
		names := service.processCharacters(context.Background(), players, makeWorldContext("Antica"))

		if len(names) != 1 {
			t.Errorf("expected 1, got %d", len(names))
		}
		if !upserted {
			t.Error("expected upsert (new player logic)")
		}
	})

	t.Run("handles error", func(t *testing.T) {
		fetcher := &mockServiceFetcher{
			fetchCharacterDetailsFunc: func(ctx context.Context, names []string) (chan *domain.Player, error) {
				return nil, errors.New("error")
			},
		}
		service := makeService(nil, fetcher, nil, nil)
		names := service.processCharacters(context.Background(), []domain.Player{{Name: "P1", Level: 200}}, makeWorldContext("Antica"))
		if names != nil {
			t.Error("expected nil on error")
		}
	})
}

func TestProcessOfflinePlayers(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var upserted bool
		storage := &mockServiceStorage{
			getOfflinePlayersFunc: func(ctx context.Context, world string, online []string) ([]domain.Player, error) {
				return []domain.Player{{Name: "Off1"}}, nil
			},
			upsertPlayerLevelFunc: func(ctx context.Context, name string, level int, world string) error {
				upserted = true
				return nil
			},
		}
		fetcher := &mockServiceFetcher{
			fetchCharacterDetailsFunc: func(ctx context.Context, names []string) (chan *domain.Player, error) {
				ch := make(chan *domain.Player, 1)
				ch <- &domain.Player{Name: "Off1", Level: 150}
				close(ch)
				return ch, nil
			},
		}
		service := makeService(storage, fetcher, nil, nil)
		service.processOfflinePlayers(context.Background(), makeWorldContext("Antica"), []string{})
		if !upserted {
			t.Error("expected upsert for offline player check")
		}
	})

	t.Run("no offline", func(t *testing.T) {
		var fetchCalled bool
		storage := &mockServiceStorage{
			getOfflinePlayersFunc: func(ctx context.Context, world string, online []string) ([]domain.Player, error) {
				return []domain.Player{}, nil
			},
		}
		fetcher := &mockServiceFetcher{
			fetchCharacterDetailsFunc: func(ctx context.Context, names []string) (chan *domain.Player, error) {
				fetchCalled = true
				return nil, nil
			},
		}
		service := makeService(storage, fetcher, nil, nil)
		service.processOfflinePlayers(context.Background(), makeWorldContext("Antica"), []string{})
		if fetchCalled {
			t.Error("expected no fetch")
		}
	})

	t.Run("get offline error", func(t *testing.T) {
		storage := &mockServiceStorage{
			getOfflinePlayersFunc: func(ctx context.Context, world string, online []string) ([]domain.Player, error) {
				return nil, errors.New("error")
			},
		}
		service := makeService(storage, nil, nil, nil)
		service.processOfflinePlayers(context.Background(), makeWorldContext("Antica"), []string{})
	})

	t.Run("fetch details error", func(t *testing.T) {
		storage := &mockServiceStorage{
			getOfflinePlayersFunc: func(ctx context.Context, world string, online []string) ([]domain.Player, error) {
				return []domain.Player{{Name: "Off1"}}, nil
			},
		}
		fetcher := &mockServiceFetcher{
			fetchCharacterDetailsFunc: func(ctx context.Context, names []string) (chan *domain.Player, error) {
				return nil, errors.New("error")
			},
		}
		service := makeService(storage, fetcher, nil, nil)
		service.processOfflinePlayers(context.Background(), makeWorldContext("Antica"), []string{})
	})
}

func TestPerformMaintenance(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var touched, deleted bool
		storage := &mockServiceStorage{
			batchTouchPlayersFunc: func(ctx context.Context, names []string) error {
				touched = true
				return nil
			},
			deleteOldPlayersFunc: func(ctx context.Context, world string, d time.Duration) (int64, error) {
				deleted = true
				return 1, nil
			},
		}
		service := &Service{storage: storage}
		service.performMaintenance(context.Background(), "Antica", []string{"P1"})
		if !touched || !deleted {
			t.Error("expected touch and delete")
		}
	})

	t.Run("empty names", func(t *testing.T) {
		var touchCalled bool
		storage := &mockServiceStorage{
			batchTouchPlayersFunc: func(ctx context.Context, names []string) error {
				touchCalled = true
				return nil
			},
			deleteOldPlayersFunc: func(ctx context.Context, world string, d time.Duration) (int64, error) {
				return 0, nil
			},
		}
		service := &Service{storage: storage}
		service.performMaintenance(context.Background(), "Antica", []string{})
		if touchCalled {
			t.Error("expected no touch for empty")
		}
	})

	t.Run("maintenance errors", func(t *testing.T) {
		storage := &mockServiceStorage{
			batchTouchPlayersFunc: func(ctx context.Context, names []string) error {
				return errors.New("touch error")
			},
			deleteOldPlayersFunc: func(ctx context.Context, world string, d time.Duration) (int64, error) {
				return 0, errors.New("delete error")
			},
		}
		service := &Service{storage: storage}
		// Should log error but not panic
		service.performMaintenance(context.Background(), "Antica", []string{"P1"})
	})
}

func TestProcessWorld(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		storage := &mockServiceStorage{
			getPlayersLevelsFunc: func(ctx context.Context, world string) (map[string]int, error) {
				return map[string]int{}, nil
			},
			getOfflinePlayersFunc: func(ctx context.Context, world string, online []string) ([]domain.Player, error) {
				return []domain.Player{}, nil
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
		service := makeService(storage, fetcher, nil, &config.Config{})
		service.processWorld(context.Background(), "Antica", []domain.GuildConfig{})
	})

	t.Run("init fail", func(t *testing.T) {
		storage := &mockServiceStorage{
			getPlayersLevelsFunc: func(ctx context.Context, world string) (map[string]int, error) {
				return nil, errors.New("db error")
			},
		}
		service := &Service{storage: storage}
		// Should return early
		service.processWorld(context.Background(), "Antica", []domain.GuildConfig{})
	})
}

func TestFilterByMinLevel(t *testing.T) {
	service := &Service{config: &config.Config{MinLevelTrack: 100}}
	players := []domain.Player{{Name: "Low", Level: 50}, {Name: "High", Level: 200}}
	names := service.filterByMinLevel(players)
	if len(names) != 1 || names[0] != "High" {
		t.Errorf("got %v", names)
	}
}

func TestHelperFunctions(t *testing.T) {
	t.Run("extractNames", func(t *testing.T) {
		names := extractNames(map[string]int{"A": 1, "B": 2})
		if len(names) != 2 {
			t.Errorf("expected 2, got %d", len(names))
		}
	})

	t.Run("levelsToPlayers", func(t *testing.T) {
		players := levelsToPlayers(map[string]int{"A": 100})
		if len(players) != 1 {
			t.Errorf("expected 1, got %d", len(players))
		}
	})

	t.Run("playerNames", func(t *testing.T) {
		names := playerNames([]domain.Player{{Name: "A"}})
		if len(names) != 1 || names[0] != "A" {
			t.Errorf("got %v", names)
		}
	})
}

func TestProcessLevelsFromTibiaCom(t *testing.T) {
	t.Run("upserts", func(t *testing.T) {
		var upserted bool
		storage := &mockServiceStorage{
			upsertPlayerLevelFunc: func(ctx context.Context, name string, level int, world string) error {
				upserted = true
				return nil
			},
		}
		service := makeService(storage, nil, nil, &config.Config{MinLevelTrack: 100})
		wctx := &worldContext{world: "Antica", dbLevels: map[string]int{}}
		service.processLevelsFromTibiaCom(context.Background(), map[string]int{"P1": 200}, wctx)
		if !upserted {
			t.Error("expected upsert")
		}
	})

	t.Run("level up", func(t *testing.T) {
		var notified bool
		notifier := &mockServiceNotifier{
			sendLevelUpFunc: func(guildID string, levelUp domain.LevelUp) error {
				notified = true
				return nil
			},
		}

		storage := &mockServiceStorage{
			upsertPlayerLevelFunc: func(ctx context.Context, name string, level int, world string) error {
				return nil
			},
		}
		wctx := &worldContext{
			world:       "Antica",
			dbLevels:    map[string]int{"P1": 100},
			guilds:      []domain.GuildConfig{{DiscordGuildID: "G1", TibiaGuilds: []string{}}},
			memberships: map[string]map[string]bool{}, // No membership constraint = notify all
		}
		service := makeService(storage, nil, notifier, &config.Config{MinLevelTrack: 100})
		service.processLevelsFromTibiaCom(context.Background(), map[string]int{"P1": 150}, wctx)
		if !notified {
			t.Error("expected notification")
		}
	})

	t.Run("upsert error", func(t *testing.T) {
		storage := &mockServiceStorage{
			upsertPlayerLevelFunc: func(ctx context.Context, name string, level int, world string) error {
				return errors.New("db error")
			},
		}
		service := makeService(storage, nil, nil, &config.Config{MinLevelTrack: 100})
		wctx := &worldContext{world: "Antica", dbLevels: map[string]int{}}
		service.processLevelsFromTibiaCom(context.Background(), map[string]int{"P1": 200}, wctx)
	})
}

func TestProcessDeathsForOnlinePlayers(t *testing.T) {
	t.Run("checks deaths", func(t *testing.T) {
		var notified bool
		fetcher := &mockServiceFetcher{
			fetchCharacterDetailsFunc: func(ctx context.Context, names []string) (chan *domain.Player, error) {
				ch := make(chan *domain.Player, 1)
				// Create a death that is recent (after boot time)
				recentDeath := domain.Kill{Time: time.Now()}
				ch <- &domain.Player{Name: "P1", Deaths: []domain.Kill{recentDeath}}
				close(ch)
				return ch, nil
			},
		}
		notifier := &mockServiceNotifier{
			sendDeathFunc: func(guildID string, playerName string, kill domain.Kill) error {
				notified = true
				return nil
			},
		}

		service := makeService(nil, fetcher, notifier, &config.Config{MinLevelTrack: 100})

		wctx := &worldContext{
			world:       "Antica",
			guilds:      []domain.GuildConfig{{DiscordGuildID: "G1", TibiaGuilds: []string{}}},
			memberships: map[string]map[string]bool{},
		}

		time.Sleep(1 * time.Millisecond) // Ensure boot time is strictly before death time

		service.processDeathsForOnlinePlayers(context.Background(), []domain.Player{{Name: "P1", Level: 200}}, wctx)

		if !notified {
			t.Error("expected death notification")
		}
	})

	t.Run("fetch error", func(t *testing.T) {
		fetcher := &mockServiceFetcher{
			fetchCharacterDetailsFunc: func(ctx context.Context, names []string) (chan *domain.Player, error) {
				return nil, errors.New("error")
			},
		}
		service := makeService(nil, fetcher, nil, &config.Config{MinLevelTrack: 100})
		service.processDeathsForOnlinePlayers(context.Background(), []domain.Player{{Name: "P1", Level: 200}}, nil)
	})
}

func TestProcessViaTibiaCom(t *testing.T) {
	t.Run("fallback on error", func(t *testing.T) {
		var tibiaDataCalled bool
		fetcher := &mockServiceFetcher{
			fetchWorldFromTibiaComFunc: func(ctx context.Context, world string) (map[string]int, error) {
				return nil, errors.New("error")
			},
			fetchWorldFunc: func(ctx context.Context, world string) ([]domain.Player, error) {
				tibiaDataCalled = true
				return []domain.Player{}, nil
			},
			fetchCharacterDetailsFunc: func(ctx context.Context, names []string) (chan *domain.Player, error) {
				ch := make(chan *domain.Player)
				close(ch)
				return ch, nil
			},
		}
		service := makeService(nil, fetcher, nil, &config.Config{MinLevelTrack: 100})
		service.processViaTibiaCom(context.Background(), makeWorldContext("Antica"))
		if !tibiaDataCalled {
			t.Error("expected fallback")
		}
	})
}

func TestProcessOnlinePlayers(t *testing.T) {
	t.Run("uses TibiaCom", func(t *testing.T) {
		var called bool
		fetcher := &mockServiceFetcher{
			fetchWorldFromTibiaComFunc: func(ctx context.Context, world string) (map[string]int, error) {
				called = true
				return map[string]int{}, nil
			},
			fetchCharacterDetailsFunc: func(ctx context.Context, names []string) (chan *domain.Player, error) {
				ch := make(chan *domain.Player)
				close(ch)
				return ch, nil
			},
		}
		service := makeService(nil, fetcher, nil, &config.Config{UseTibiaComForLevels: true, MinLevelTrack: 100})
		service.processOnlinePlayers(context.Background(), makeWorldContext("Antica"))
		if !called {
			t.Error("expected TibiaCom")
		}
	})

	t.Run("TibiaData error", func(t *testing.T) {
		fetcher := &mockServiceFetcher{
			fetchWorldFunc: func(ctx context.Context, world string) ([]domain.Player, error) {
				return nil, errors.New("error")
			},
		}
		service := makeService(nil, fetcher, nil, &config.Config{UseTibiaComForLevels: false, MinLevelTrack: 100})
		// processOnlinePlayers -> processViaTibiaData
		names := service.processOnlinePlayers(context.Background(), makeWorldContext("Antica"))
		if names != nil {
			t.Error("expected nil on error")
		}
	})
}

func TestProcessLevelsFromTibiaCom_MinLevel(t *testing.T) {
	t.Run("ignores low levels", func(t *testing.T) {
		var upserted bool
		storage := &mockServiceStorage{
			upsertPlayerLevelFunc: func(ctx context.Context, name string, level int, world string) error {
				upserted = true
				return nil
			},
		}
		service := makeService(storage, nil, nil, &config.Config{MinLevelTrack: 100})
		wctx := &worldContext{world: "Antica", dbLevels: map[string]int{}}
		service.processLevelsFromTibiaCom(context.Background(), map[string]int{"LowP": 50}, wctx)
		if upserted {
			t.Error("expected no upsert for low level")
		}
	})
}
