package tracker

import (
	"context"
	"errors"
	"testing"
	"time"

	"death-level-tracker/internal/config"
	"death-level-tracker/internal/core/domain"
)

func TestNewLevelTracker(t *testing.T) {
	cfg := &config.Config{MinLevelTrack: 100}
	store := &mockLevelStorage{}
	notifier := &mockLevelNotifier{}

	tracker := NewLevelTracker(cfg, store, notifier)

	if tracker == nil {
		t.Fatal("expected non-nil tracker")
	}
	if tracker.config != cfg {
		t.Error("expected config to be set")
	}
	if tracker.storage != store {
		t.Error("expected storage to be set")
	}
	if tracker.notifier != notifier {
		t.Error("expected notifier to be set")
	}
}

func TestLevelTracker_ShouldUpdateLevel(t *testing.T) {
	tracker := &LevelTracker{}

	tests := []struct {
		name     string
		exists   bool
		saved    int
		current  int
		expected bool
	}{
		{"new player - should upsert", false, 0, 100, true},
		{"same level - no update needed", true, 100, 100, false},
		{"level increased - should update", true, 100, 150, true},
		{"level decreased - protect against stale data", true, 150, 100, false},
		{"new player at level 1", false, 0, 1, true},
		{"large level jump", true, 100, 500, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tracker.shouldUpdateLevel(tt.exists, tt.saved, tt.current)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestLevelTracker_IsLevelUp(t *testing.T) {
	tracker := &LevelTracker{}

	tests := []struct {
		name     string
		exists   bool
		saved    int
		current  int
		expected bool
	}{
		{"new player - not a level up", false, 0, 100, false},
		{"level increased - is level up", true, 100, 150, true},
		{"same level - not a level up", true, 100, 100, false},
		{"level decreased - not a level up", true, 150, 100, false},
		{"single level up", true, 99, 100, true},
		{"large jump", true, 100, 1000, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tracker.isLevelUp(tt.exists, tt.saved, tt.current)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestLevelTracker_CheckLevelUp(t *testing.T) {
	t.Run("new player - upserts without notification", func(t *testing.T) {
		var upserted bool
		var notified bool

		storage := &mockLevelStorage{
			upsertFunc: func(ctx context.Context, name string, level int, world string) error {
				upserted = true
				if name != "NewPlayer" || level != 100 || world != "Antica" {
					t.Errorf("unexpected upsert args: %s, %d, %s", name, level, world)
				}
				return nil
			},
		}
		notifier := &mockLevelNotifier{
			onNotify: func() { notified = true },
		}

		tracker := &LevelTracker{storage: storage, notifier: notifier}
		tracker.CheckLevelUp(context.Background(), "NewPlayer", 100, "Antica", map[string]int{}, nil, nil)

		if !upserted {
			t.Error("expected upsert for new player")
		}
		if notified {
			t.Error("expected no notification for new player")
		}
	})

	t.Run("level up - upserts and notifies", func(t *testing.T) {
		var upserted bool
		var notifiedGuilds []string

		storage := &mockLevelStorage{
			upsertFunc: func(ctx context.Context, name string, level int, world string) error {
				upserted = true
				return nil
			},
		}
		notifier := &mockLevelNotifier{
			sendLevelUpFunc: func(guildID string, levelUp domain.LevelUp) error {
				notifiedGuilds = append(notifiedGuilds, guildID)
				if levelUp.OldLevel != 100 || levelUp.NewLevel != 150 {
					t.Errorf("unexpected levels: %d -> %d", levelUp.OldLevel, levelUp.NewLevel)
				}
				return nil
			},
		}

		guilds := []domain.GuildConfig{{DiscordGuildID: "guild-1"}, {DiscordGuildID: "guild-2"}}
		dbLevels := map[string]int{"Player": 100}

		tracker := &LevelTracker{storage: storage, notifier: notifier}
		tracker.CheckLevelUp(context.Background(), "Player", 150, "Antica", dbLevels, guilds, nil)

		if !upserted {
			t.Error("expected upsert")
		}
		if len(notifiedGuilds) != 2 {
			t.Errorf("expected 2 notifications, got %d", len(notifiedGuilds))
		}
	})

	t.Run("same level - no action", func(t *testing.T) {
		var upserted bool
		var notified bool

		storage := &mockLevelStorage{
			upsertFunc: func(ctx context.Context, name string, level int, world string) error {
				upserted = true
				return nil
			},
		}
		notifier := &mockLevelNotifier{
			onNotify: func() { notified = true },
		}

		dbLevels := map[string]int{"Player": 100}
		tracker := &LevelTracker{storage: storage, notifier: notifier}
		tracker.CheckLevelUp(context.Background(), "Player", 100, "Antica", dbLevels, nil, nil)

		if upserted {
			t.Error("expected no upsert for same level")
		}
		if notified {
			t.Error("expected no notification for same level")
		}
	})

	t.Run("level down - no action (stale data protection)", func(t *testing.T) {
		var upserted bool
		var notified bool

		storage := &mockLevelStorage{
			upsertFunc: func(ctx context.Context, name string, level int, world string) error {
				upserted = true
				return nil
			},
		}
		notifier := &mockLevelNotifier{
			onNotify: func() { notified = true },
		}

		dbLevels := map[string]int{"Player": 150}
		tracker := &LevelTracker{storage: storage, notifier: notifier}
		tracker.CheckLevelUp(context.Background(), "Player", 100, "Antica", dbLevels, nil, nil)

		if upserted {
			t.Error("expected no upsert for level down")
		}
		if notified {
			t.Error("expected no notification for level down")
		}
	})

	t.Run("upsert error - continues gracefully", func(t *testing.T) {
		storage := &mockLevelStorage{
			upsertFunc: func(ctx context.Context, name string, level int, world string) error {
				return errors.New("db error")
			},
		}

		tracker := &LevelTracker{storage: storage, notifier: &mockLevelNotifier{}}
		tracker.CheckLevelUp(context.Background(), "Player", 100, "Antica", map[string]int{}, nil, nil)
	})

	t.Run("notification error - continues gracefully", func(t *testing.T) {
		storage := &mockLevelStorage{
			upsertFunc: func(ctx context.Context, name string, level int, world string) error {
				return nil
			},
		}
		notifier := &mockLevelNotifier{
			sendLevelUpFunc: func(guildID string, levelUp domain.LevelUp) error {
				return errors.New("discord error")
			},
		}

		guilds := []domain.GuildConfig{{DiscordGuildID: "guild-1"}}
		dbLevels := map[string]int{"Player": 100}

		tracker := &LevelTracker{storage: storage, notifier: notifier}
		tracker.CheckLevelUp(context.Background(), "Player", 150, "Antica", dbLevels, guilds, nil)
	})
}

func TestLevelTracker_NotifyLevelUp_GuildFiltering(t *testing.T) {
	t.Run("notifies all guilds when no filter", func(t *testing.T) {
		var notifiedGuilds []string
		notifier := &mockLevelNotifier{
			sendLevelUpFunc: func(guildID string, levelUp domain.LevelUp) error {
				notifiedGuilds = append(notifiedGuilds, guildID)
				return nil
			},
		}

		guilds := []domain.GuildConfig{
			{DiscordGuildID: "g1", TibiaGuilds: []string{}},
			{DiscordGuildID: "g2", TibiaGuilds: []string{}},
		}

		tracker := &LevelTracker{notifier: notifier}
		tracker.notifyLevelUp(guilds, "Player", 100, 150, "Antica", nil)

		if len(notifiedGuilds) != 2 {
			t.Errorf("expected 2, got %d", len(notifiedGuilds))
		}
	})

	t.Run("filters by guild membership", func(t *testing.T) {
		var notifiedGuilds []string
		notifier := &mockLevelNotifier{
			sendLevelUpFunc: func(guildID string, levelUp domain.LevelUp) error {
				notifiedGuilds = append(notifiedGuilds, guildID)
				return nil
			},
		}

		guilds := []domain.GuildConfig{
			{DiscordGuildID: "g1", TibiaGuilds: []string{"MyGuild"}},
			{DiscordGuildID: "g2", TibiaGuilds: []string{"OtherGuild"}},
		}
		memberships := map[string]map[string]bool{
			"MyGuild":    {"Player": true},
			"OtherGuild": {"Someone": true},
		}

		tracker := &LevelTracker{notifier: notifier}
		tracker.notifyLevelUp(guilds, "Player", 100, 150, "Antica", memberships)

		if len(notifiedGuilds) != 1 || notifiedGuilds[0] != "g1" {
			t.Errorf("expected only g1, got %v", notifiedGuilds)
		}
	})

	t.Run("no notifications if not a member of any guild", func(t *testing.T) {
		var notifyCount int
		notifier := &mockLevelNotifier{
			sendLevelUpFunc: func(guildID string, levelUp domain.LevelUp) error {
				notifyCount++
				return nil
			},
		}

		guilds := []domain.GuildConfig{
			{DiscordGuildID: "g1", TibiaGuilds: []string{"SomeGuild"}},
		}
		memberships := map[string]map[string]bool{
			"SomeGuild": {"OtherPlayer": true},
		}

		tracker := &LevelTracker{notifier: notifier}
		tracker.notifyLevelUp(guilds, "Player", 100, 150, "Antica", memberships)

		if notifyCount != 0 {
			t.Errorf("expected 0, got %d", notifyCount)
		}
	})
}

func TestShouldNotifyGuild(t *testing.T) {
	t.Run("empty TibiaGuilds - always notify", func(t *testing.T) {
		guild := domain.GuildConfig{TibiaGuilds: []string{}}
		if !shouldNotifyGuild("AnyPlayer", guild, nil) {
			t.Error("expected true")
		}
	})

	t.Run("player is member - notify", func(t *testing.T) {
		guild := domain.GuildConfig{TibiaGuilds: []string{"Guild1", "Guild2"}}
		memberships := map[string]map[string]bool{
			"Guild2": {"Player": true},
		}
		if !shouldNotifyGuild("Player", guild, memberships) {
			t.Error("expected true")
		}
	})

	t.Run("player not member - no notify", func(t *testing.T) {
		guild := domain.GuildConfig{TibiaGuilds: []string{"Guild1"}}
		memberships := map[string]map[string]bool{
			"Guild1": {"Other": true},
		}
		if shouldNotifyGuild("Player", guild, memberships) {
			t.Error("expected false")
		}
	})

	t.Run("guild not in memberships - no notify", func(t *testing.T) {
		guild := domain.GuildConfig{TibiaGuilds: []string{"NonExistent"}}
		memberships := map[string]map[string]bool{}
		if shouldNotifyGuild("Player", guild, memberships) {
			t.Error("expected false")
		}
	})

	t.Run("multiple guilds - member of one", func(t *testing.T) {
		guild := domain.GuildConfig{TibiaGuilds: []string{"Guild1", "Guild2", "Guild3"}}
		memberships := map[string]map[string]bool{
			"Guild1": {"Other1": true},
			"Guild2": {"Other2": true},
			"Guild3": {"Player": true},
		}
		if !shouldNotifyGuild("Player", guild, memberships) {
			t.Error("expected true")
		}
	})
}

type mockLevelStorage struct {
	upsertFunc func(ctx context.Context, name string, level int, world string) error
}

func (m *mockLevelStorage) UpsertPlayerLevel(ctx context.Context, name string, level int, world string) error {
	if m.upsertFunc != nil {
		return m.upsertFunc(ctx, name, level, world)
	}
	return nil
}

func (m *mockLevelStorage) GetPlayersLevels(ctx context.Context, world string) (map[string]int, error) {
	return nil, nil
}
func (m *mockLevelStorage) GetAllGuildConfigs(ctx context.Context) ([]domain.GuildConfig, error) {
	return nil, nil
}
func (m *mockLevelStorage) SaveGuildWorld(ctx context.Context, guildID, world string) error {
	return nil
}
func (m *mockLevelStorage) DeleteGuildConfig(ctx context.Context, guildID string) error { return nil }
func (m *mockLevelStorage) AddGuildToConfig(ctx context.Context, guildID, guild string) error {
	return nil
}
func (m *mockLevelStorage) RemoveGuildFromConfig(ctx context.Context, guildID, guild string) error {
	return nil
}
func (m *mockLevelStorage) GetGuildConfig(ctx context.Context, guildID string) (*domain.GuildConfig, error) {
	return nil, nil
}
func (m *mockLevelStorage) GetOfflinePlayers(ctx context.Context, world string, onlineNames []string) ([]domain.Player, error) {
	return nil, nil
}
func (m *mockLevelStorage) BatchTouchPlayers(ctx context.Context, names []string) error { return nil }
func (m *mockLevelStorage) DeleteOldPlayers(ctx context.Context, world string, threshold time.Duration) (int64, error) {
	return 0, nil
}
func (m *mockLevelStorage) Close() {}

type mockLevelNotifier struct {
	onNotify        func()
	sendLevelUpFunc func(guildID string, levelUp domain.LevelUp) error
}

func (m *mockLevelNotifier) SendLevelUpNotification(guildID string, levelUp domain.LevelUp) error {
	if m.onNotify != nil {
		m.onNotify()
	}
	if m.sendLevelUpFunc != nil {
		return m.sendLevelUpFunc(guildID, levelUp)
	}
	return nil
}

func (m *mockLevelNotifier) SendDeathNotification(guildID string, playerName string, kill domain.Kill) error {
	return nil
}

func (m *mockLevelNotifier) SendGenericMessage(guildID, channelName, message string) error {
	return nil
}
