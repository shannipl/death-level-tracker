package tracker

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"death-level-tracker/internal/storage"
	"death-level-tracker/internal/tibiadata"
)

// Mock storage for testing
type mockAnalyticsStorage struct {
	upsertPlayerLevelFunc func(ctx context.Context, name string, level int, world string) error
}

func (m *mockAnalyticsStorage) UpsertPlayerLevel(ctx context.Context, name string, level int, world string) error {
	if m.upsertPlayerLevelFunc != nil {
		return m.upsertPlayerLevelFunc(ctx, name, level, world)
	}
	return nil
}

func (m *mockAnalyticsStorage) SaveGuildWorld(ctx context.Context, guildID, world string) error {
	return nil
}

func (m *mockAnalyticsStorage) GetWorldsMap(ctx context.Context) (map[string][]string, error) {
	return nil, nil
}

func (m *mockAnalyticsStorage) GetPlayersLevels(ctx context.Context, world string) (map[string]int, error) {
	return nil, nil
}

func (m *mockAnalyticsStorage) BatchTouchPlayers(ctx context.Context, names []string) error {
	return nil
}

func (m *mockAnalyticsStorage) DeleteOldPlayers(ctx context.Context, world string, threshold time.Duration) (int64, error) {
	return 0, nil
}

func (m *mockAnalyticsStorage) DeleteGuildConfig(ctx context.Context, guildID string) error {
	return nil
}

func (m *mockAnalyticsStorage) GetOfflinePlayers(ctx context.Context, world string, onlineNames []string) ([]storage.OfflinePlayer, error) {
	return nil, nil
}

func (m *mockAnalyticsStorage) Close() {}

// Mock notifier for testing
type mockNotifier struct {
	messages []mockMessage
}

type mockMessage struct {
	guildID string
	channel string
	content string
}

func (m *mockNotifier) Send(guildID, channel, content string) {
	m.messages = append(m.messages, mockMessage{
		guildID: guildID,
		channel: channel,
		content: content,
	})
}

func TestNewAnalytics(t *testing.T) {
	storage := &mockAnalyticsStorage{}
	notifier := &mockNotifier{}

	analytics := NewAnalytics(storage, notifier)

	if analytics == nil {
		t.Fatal("Expected non-nil analytics")
	}

	if analytics.storage == nil {
		t.Error("Expected storage to be set")
	}

	if analytics.notifier == nil {
		t.Error("Expected notifier to be set")
	}

	if analytics.seenDeaths == nil {
		t.Error("Expected seenDeaths map to be initialized")
	}

	if analytics.bootTime.IsZero() {
		t.Error("Expected bootTime to be set")
	}
}

func TestAnalytics_IsOldDeath(t *testing.T) {
	analytics := NewAnalytics(&mockAnalyticsStorage{}, &mockNotifier{})

	// Death before boot time
	oldDeath := analytics.bootTime.Add(-1 * time.Hour)
	if !analytics.isOldDeath(oldDeath) {
		t.Error("Expected death before boot time to be old")
	}

	// Death after boot time
	recentDeath := analytics.bootTime.Add(1 * time.Hour)
	if analytics.isOldDeath(recentDeath) {
		t.Error("Expected death after boot time to not be old")
	}

	// Death exactly at boot time
	if analytics.isOldDeath(analytics.bootTime) {
		t.Error("Expected death at boot time to not be old")
	}
}

func TestAnalytics_IsDuplicateDeath(t *testing.T) {
	analytics := NewAnalytics(&mockAnalyticsStorage{}, &mockNotifier{})

	name := "TestPlayer"
	deathTime := time.Now()

	// First occurrence - should not be duplicate
	if analytics.isDuplicateDeath(name, deathTime) {
		t.Error("Expected first occurrence to not be duplicate")
	}

	// Second occurrence - should be duplicate
	if !analytics.isDuplicateDeath(name, deathTime) {
		t.Error("Expected second occurrence to be duplicate")
	}

	// Different player, same time - should not be duplicate
	if analytics.isDuplicateDeath("OtherPlayer", deathTime) {
		t.Error("Expected different player to not be duplicate")
	}

	// Same player, different time - should not be duplicate
	differentTime := deathTime.Add(1 * time.Hour)
	if analytics.isDuplicateDeath(name, differentTime) {
		t.Error("Expected different time to not be duplicate")
	}
}

func TestAnalytics_ShouldUpdateLevel(t *testing.T) {
	analytics := NewAnalytics(&mockAnalyticsStorage{}, &mockNotifier{})

	testCases := []struct {
		name         string
		exists       bool
		savedLevel   int
		currentLevel int
		expected     bool
	}{
		{"New player", false, 0, 100, true},
		{"Same level", true, 100, 100, false},
		{"Level up", true, 100, 101, true},
		{"Level down", true, 100, 99, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := analytics.shouldUpdateLevel(tc.exists, tc.savedLevel, tc.currentLevel)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestAnalytics_IsLevelUp(t *testing.T) {
	analytics := NewAnalytics(&mockAnalyticsStorage{}, &mockNotifier{})

	testCases := []struct {
		name         string
		exists       bool
		savedLevel   int
		currentLevel int
		expected     bool
	}{
		{"New player", false, 0, 100, false},
		{"Level up", true, 100, 101, true},
		{"Same level", true, 100, 100, false},
		{"Level down", true, 100, 99, false},
		{"Big level jump", true, 50, 150, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := analytics.isLevelUp(tc.exists, tc.savedLevel, tc.currentLevel)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestAnalytics_NotifyDeath(t *testing.T) {
	storage := &mockAnalyticsStorage{}
	mockNotif := &mockNotifier{}

	analytics := NewAnalytics(storage, mockNotif)

	death := tibiadata.Death{
		Time:   time.Now(),
		Level:  100,
		Reason: "Killed by a dragon",
	}

	guilds := []string{"guild-1", "guild-2"}
	analytics.notifyDeath(guilds, "TestPlayer", death)

	if len(mockNotif.messages) != 2 {
		t.Fatalf("Expected 2 notifications, got %d", len(mockNotif.messages))
	}

	for i, msg := range mockNotif.messages {
		if msg.guildID != guilds[i] {
			t.Errorf("Expected guild ID '%s', got '%s'", guilds[i], msg.guildID)
		}
		if msg.channel != "death-level-tracker" {
			t.Errorf("Expected channel 'death-level-tracker', got '%s'", msg.channel)
		}
		if !strings.Contains(msg.content, "TestPlayer") {
			t.Errorf("Expected content to contain player name")
		}
	}
}

func TestAnalytics_NotifyLevelUp(t *testing.T) {
	storage := &mockAnalyticsStorage{}
	mockNotif := &mockNotifier{}

	analytics := NewAnalytics(storage, mockNotif)

	guilds := []string{"guild-1", "guild-2"}
	analytics.notifyLevelUp(guilds, "TestPlayer", 100, 101)

	if len(mockNotif.messages) != 2 {
		t.Fatalf("Expected 2 notifications, got %d", len(mockNotif.messages))
	}

	for i, msg := range mockNotif.messages {
		if msg.guildID != guilds[i] {
			t.Errorf("Expected guild ID '%s', got '%s'", guilds[i], msg.guildID)
		}
		if msg.channel != "level-tracker" {
			t.Errorf("Expected channel 'level-tracker', got '%s'", msg.channel)
		}
		if !strings.Contains(msg.content, "TestPlayer") {
			t.Errorf("Expected content to contain player name")
		}
	}
}

func TestAnalytics_CheckDeaths_OldDeathsIgnored(t *testing.T) {
	storage := &mockAnalyticsStorage{}
	mockNotif := &mockNotifier{}

	analytics := NewAnalytics(storage, mockNotif)

	// Create death before boot time
	oldDeath := tibiadata.Death{
		Time:   analytics.bootTime.Add(-1 * time.Hour),
		Level:  100,
		Reason: "Old death",
	}

	analytics.checkDeaths("TestPlayer", []tibiadata.Death{oldDeath}, []string{"guild-1"})

	if len(mockNotif.messages) != 0 {
		t.Errorf("Expected no notifications for old death, got %d", len(mockNotif.messages))
	}
}

func TestAnalytics_CheckDeaths_DuplicateIgnored(t *testing.T) {
	storage := &mockAnalyticsStorage{}
	mockNotif := &mockNotifier{}

	analytics := NewAnalytics(storage, mockNotif)

	death := tibiadata.Death{
		Time:   analytics.bootTime.Add(1 * time.Hour),
		Level:  100,
		Reason: "Test death",
	}

	// First check - should notify
	analytics.checkDeaths("TestPlayer", []tibiadata.Death{death}, []string{"guild-1"})
	if len(mockNotif.messages) != 1 {
		t.Fatalf("Expected 1 notification for first death, got %d", len(mockNotif.messages))
	}

	// Second check with same death - should not notify
	analytics.checkDeaths("TestPlayer", []tibiadata.Death{death}, []string{"guild-1"})
	if len(mockNotif.messages) != 1 {
		t.Errorf("Expected still 1 notification (duplicate ignored), got %d", len(mockNotif.messages))
	}
}

func TestAnalytics_CheckLevelUp_NewPlayer(t *testing.T) {
	var upsertCalled bool
	storage := &mockAnalyticsStorage{
		upsertPlayerLevelFunc: func(ctx context.Context, name string, level int, world string) error {
			upsertCalled = true
			if name != "NewPlayer" {
				t.Errorf("Expected name 'NewPlayer', got '%s'", name)
			}
			if level != 100 {
				t.Errorf("Expected level 100, got %d", level)
			}
			if world != "Antica" {
				t.Errorf("Expected world 'Antica', got '%s'", world)
			}
			return nil
		},
	}
	mockNotif := &mockNotifier{}

	analytics := NewAnalytics(storage, mockNotif)

	dbLevels := make(map[string]int) // Empty - new player
	analytics.checkLevelUp("NewPlayer", 100, "Antica", dbLevels, []string{"guild-1"})

	if !upsertCalled {
		t.Error("Expected UpsertPlayerLevel to be called for new player")
	}

	if len(mockNotif.messages) != 0 {
		t.Errorf("Expected no level-up notification for new player, got %d", len(mockNotif.messages))
	}
}

func TestAnalytics_CheckLevelUp_LevelIncrease(t *testing.T) {
	var upsertCalled bool
	storage := &mockAnalyticsStorage{
		upsertPlayerLevelFunc: func(ctx context.Context, name string, level int, world string) error {
			upsertCalled = true
			return nil
		},
	}
	mockNotif := &mockNotifier{}

	analytics := NewAnalytics(storage, mockNotif)

	dbLevels := map[string]int{"TestPlayer": 100}
	analytics.checkLevelUp("TestPlayer", 101, "Antica", dbLevels, []string{"guild-1"})

	if !upsertCalled {
		t.Error("Expected UpsertPlayerLevel to be called")
	}

	if len(mockNotif.messages) != 1 {
		t.Fatalf("Expected 1 level-up notification, got %d", len(mockNotif.messages))
	}

	msg := mockNotif.messages[0]
	if msg.channel != "level-tracker" {
		t.Errorf("Expected 'level-tracker' channel, got '%s'", msg.channel)
	}
}

func TestAnalytics_CheckLevelUp_SameLevel(t *testing.T) {
	var upsertCalled bool
	storage := &mockAnalyticsStorage{
		upsertPlayerLevelFunc: func(ctx context.Context, name string, level int, world string) error {
			upsertCalled = true
			return nil
		},
	}
	mockNotif := &mockNotifier{}

	analytics := NewAnalytics(storage, mockNotif)

	dbLevels := map[string]int{"TestPlayer": 100}
	analytics.checkLevelUp("TestPlayer", 100, "Antica", dbLevels, []string{"guild-1"})

	if upsertCalled {
		t.Error("Expected UpsertPlayerLevel NOT to be called for same level")
	}

	if len(mockNotif.messages) != 0 {
		t.Errorf("Expected no notification for same level, got %d", len(mockNotif.messages))
	}
}

func TestAnalytics_CheckLevelUp_StorageError(t *testing.T) {
	var upsertCalled bool
	storage := &mockAnalyticsStorage{
		upsertPlayerLevelFunc: func(ctx context.Context, name string, level int, world string) error {
			upsertCalled = true
			return errors.New("database connection failed")
		},
	}
	mockNotif := &mockNotifier{}

	analytics := NewAnalytics(storage, mockNotif)

	dbLevels := map[string]int{"TestPlayer": 100}
	analytics.checkLevelUp("TestPlayer", 101, "Antica", dbLevels, []string{"guild-1"})

	if !upsertCalled {
		t.Error("Expected UpsertPlayerLevel to be called even with error")
	}

	// Should still send level-up notification despite storage error
	if len(mockNotif.messages) != 1 {
		t.Fatalf("Expected 1 level-up notification, got %d", len(mockNotif.messages))
	}

	msg := mockNotif.messages[0]
	if msg.channel != "level-tracker" {
		t.Errorf("Expected 'level-tracker' channel, got '%s'", msg.channel)
	}
}

func TestAnalytics_ProcessCharacter(t *testing.T) {
	storage := &mockAnalyticsStorage{
		upsertPlayerLevelFunc: func(ctx context.Context, name string, level int, world string) error {
			return nil
		},
	}
	mockNotif := &mockNotifier{}

	analytics := NewAnalytics(storage, mockNotif)

	char := &tibiadata.CharacterResponse{
		Character: struct {
			Character tibiadata.CharacterInfo `json:"character"`
			Deaths    []tibiadata.Death       `json:"deaths"`
		}{
			Character: tibiadata.CharacterInfo{
				Name:  "TestPlayer",
				Level: 101,
				World: "Antica",
			},
			Deaths: []tibiadata.Death{
				{
					Time:   analytics.bootTime.Add(1 * time.Hour),
					Level:  100,
					Reason: "Killed by a dragon",
				},
			},
		},
	}

	guilds := []string{"guild-1"}
	dbLevels := map[string]int{"TestPlayer": 100}

	analytics.ProcessCharacter(char, guilds, dbLevels)

	// Should get 1 death notification + 1 level-up notification
	if len(mockNotif.messages) != 2 {
		t.Errorf("Expected 2 notifications (death + level-up), got %d", len(mockNotif.messages))
	}
}
