package tracker

import (
	"errors"
	"sync"
	"testing"
	"time"

	"death-level-tracker/internal/core/domain"
)

func TestNewDeathTracker(t *testing.T) {
	notifier := &mockDeathNotifier{}
	tracker := NewDeathTracker(notifier)

	if tracker == nil {
		t.Fatal("expected non-nil tracker")
	}
	if tracker.notifier != notifier {
		t.Error("expected notifier to be set")
	}
	if tracker.seenDeaths == nil {
		t.Error("expected seenDeaths map to be initialized")
	}
	if tracker.ttl != 25*time.Hour {
		t.Errorf("expected TTL 25h, got %v", tracker.ttl)
	}
	if tracker.startTime.IsZero() {
		t.Error("expected startTime to be set")
	}
}

func TestDeathTracker_IsOldDeath(t *testing.T) {
	tracker := &DeathTracker{}

	t.Run("death older than 2h - is old", func(t *testing.T) {
		oldDeath := time.Now().Add(-3 * time.Hour)
		if !tracker.isOldDeath(oldDeath) {
			t.Error("expected true for death older than 2h")
		}
	})

	t.Run("death newer than 2h - is not old", func(t *testing.T) {
		newDeath := time.Now().Add(-1 * time.Hour)
		if tracker.isOldDeath(newDeath) {
			t.Error("expected false for death newer than 2h")
		}
	})

	t.Run("death exactly at 2h boundary - is not old", func(t *testing.T) {
		boundary := time.Now().Add(-2 * time.Hour).Add(1 * time.Second)
		if tracker.isOldDeath(boundary) {
			t.Error("expected false for death just inside 2h window")
		}
	})

	t.Run("death before app start - is old", func(t *testing.T) {
		stTracker := &DeathTracker{startTime: time.Now().Add(-1 * time.Hour)}
		deathBeforeStart := time.Now().Add(-90 * time.Minute)

		if !stTracker.isOldDeath(deathBeforeStart) {
			t.Error("expected true for death before app start")
		}
	})

	t.Run("death after app start - is not old", func(t *testing.T) {
		stTracker := &DeathTracker{startTime: time.Now().Add(-2 * time.Hour)}
		deathAfterStart := time.Now().Add(-1 * time.Hour)

		if stTracker.isOldDeath(deathAfterStart) {
			t.Error("expected false for death after app start")
		}
	})
}

func TestDeathTracker_IsDuplicateDeath(t *testing.T) {
	t.Run("first occurrence - not duplicate", func(t *testing.T) {
		tracker := &DeathTracker{seenDeaths: make(map[string]deathRecord)}
		deathTime := time.Now()

		if tracker.isDuplicateDeath("Player", deathTime) {
			t.Error("expected false for first occurrence")
		}
	})

	t.Run("second occurrence - is duplicate", func(t *testing.T) {
		tracker := &DeathTracker{seenDeaths: make(map[string]deathRecord)}
		deathTime := time.Now()

		tracker.isDuplicateDeath("Player", deathTime)

		if !tracker.isDuplicateDeath("Player", deathTime) {
			t.Error("expected true for second occurrence")
		}
	})

	t.Run("same player different death times - not duplicate", func(t *testing.T) {
		tracker := &DeathTracker{seenDeaths: make(map[string]deathRecord)}

		death1 := time.Now()
		death2 := time.Now().Add(1 * time.Second)

		tracker.isDuplicateDeath("Player", death1)

		if tracker.isDuplicateDeath("Player", death2) {
			t.Error("expected false for different death time")
		}
	})

	t.Run("different players same death time - not duplicate", func(t *testing.T) {
		tracker := &DeathTracker{seenDeaths: make(map[string]deathRecord)}
		deathTime := time.Now()

		tracker.isDuplicateDeath("Player1", deathTime)

		if tracker.isDuplicateDeath("Player2", deathTime) {
			t.Error("expected false for different player")
		}
	})

	t.Run("records are added with timestamp", func(t *testing.T) {
		tracker := &DeathTracker{seenDeaths: make(map[string]deathRecord)}
		before := time.Now()

		tracker.isDuplicateDeath("Player", time.Now())

		if len(tracker.seenDeaths) != 1 {
			t.Fatalf("expected 1 record, got %d", len(tracker.seenDeaths))
		}

		for _, record := range tracker.seenDeaths {
			if record.addedAt.Before(before) {
				t.Error("expected addedAt to be recent")
			}
		}
	})
}

func TestDeathTracker_EvictOld(t *testing.T) {
	t.Run("evicts entries older than TTL", func(t *testing.T) {
		tracker := &DeathTracker{
			seenDeaths: map[string]deathRecord{
				"old1":   {addedAt: time.Now().Add(-30 * time.Hour)},
				"old2":   {addedAt: time.Now().Add(-26 * time.Hour)},
				"recent": {addedAt: time.Now().Add(-1 * time.Hour)},
			},
			ttl: 25 * time.Hour,
		}

		tracker.evictOld()

		if len(tracker.seenDeaths) != 1 {
			t.Errorf("expected 1 remaining, got %d", len(tracker.seenDeaths))
		}
		if _, ok := tracker.seenDeaths["recent"]; !ok {
			t.Error("expected 'recent' to remain")
		}
	})

	t.Run("keeps all if none expired", func(t *testing.T) {
		tracker := &DeathTracker{
			seenDeaths: map[string]deathRecord{
				"a": {addedAt: time.Now()},
				"b": {addedAt: time.Now().Add(-1 * time.Hour)},
			},
			ttl: 25 * time.Hour,
		}

		tracker.evictOld()

		if len(tracker.seenDeaths) != 2 {
			t.Errorf("expected 2, got %d", len(tracker.seenDeaths))
		}
	})

	t.Run("handles empty map", func(t *testing.T) {
		tracker := &DeathTracker{
			seenDeaths: make(map[string]deathRecord),
			ttl:        25 * time.Hour,
		}

		tracker.evictOld()

		if len(tracker.seenDeaths) != 0 {
			t.Errorf("expected 0, got %d", len(tracker.seenDeaths))
		}
	})

	t.Run("evicts exactly at boundary", func(t *testing.T) {
		ttl := 25 * time.Hour
		tracker := &DeathTracker{
			seenDeaths: map[string]deathRecord{
				"boundary": {addedAt: time.Now().Add(-ttl - 1*time.Second)},
			},
			ttl: ttl,
		}

		tracker.evictOld()

		if len(tracker.seenDeaths) != 0 {
			t.Error("expected boundary entry to be evicted")
		}
	})
}

func TestDeathTracker_CheckDeaths(t *testing.T) {
	t.Run("ignores old deaths", func(t *testing.T) {
		var notified bool
		notifier := &mockDeathNotifier{onNotify: func() { notified = true }}

		tracker := &DeathTracker{
			notifier:   notifier,
			seenDeaths: make(map[string]deathRecord),
			ttl:        25 * time.Hour,
		}

		oldDeath := domain.Kill{Time: time.Now().Add(-3 * time.Hour)}
		player := &domain.Player{Name: "P1", Deaths: []domain.Kill{oldDeath}}

		tracker.CheckDeaths(player, []domain.GuildConfig{{DiscordGuildID: "g1"}}, nil)

		if notified {
			t.Error("expected no notification for old death")
		}
	})

	t.Run("notifies new death", func(t *testing.T) {
		var notified bool
		notifier := &mockDeathNotifier{onNotify: func() { notified = true }}

		tracker := &DeathTracker{
			notifier:   notifier,
			seenDeaths: make(map[string]deathRecord),
			ttl:        25 * time.Hour,
		}

		newDeath := domain.Kill{Time: time.Now()}
		player := &domain.Player{Name: "P1", Deaths: []domain.Kill{newDeath}}

		tracker.CheckDeaths(player, []domain.GuildConfig{{DiscordGuildID: "g1"}}, nil)

		if !notified {
			t.Error("expected notification for new death")
		}
	})

	t.Run("ignores duplicate deaths", func(t *testing.T) {
		var notifyCount int
		notifier := &mockDeathNotifier{onNotify: func() { notifyCount++ }}

		tracker := &DeathTracker{
			notifier:   notifier,
			seenDeaths: make(map[string]deathRecord),
			ttl:        25 * time.Hour,
		}

		death := domain.Kill{Time: time.Now()}
		player := &domain.Player{Name: "P1", Deaths: []domain.Kill{death}}

		tracker.CheckDeaths(player, []domain.GuildConfig{{DiscordGuildID: "g1"}}, nil)
		tracker.CheckDeaths(player, []domain.GuildConfig{{DiscordGuildID: "g1"}}, nil)
		tracker.CheckDeaths(player, []domain.GuildConfig{{DiscordGuildID: "g1"}}, nil)

		if notifyCount != 1 {
			t.Errorf("expected 1, got %d", notifyCount)
		}
	})

	t.Run("processes multiple deaths for one player", func(t *testing.T) {
		var notifyCount int
		notifier := &mockDeathNotifier{onNotify: func() { notifyCount++ }}

		tracker := &DeathTracker{
			notifier:   notifier,
			seenDeaths: make(map[string]deathRecord),
			ttl:        25 * time.Hour,
		}

		deaths := []domain.Kill{
			{Time: time.Now()},
			{Time: time.Now().Add(1 * time.Second)},
			{Time: time.Now().Add(2 * time.Second)},
		}
		player := &domain.Player{Name: "P1", Deaths: deaths}

		tracker.CheckDeaths(player, []domain.GuildConfig{{DiscordGuildID: "g1"}}, nil)

		if notifyCount != 3 {
			t.Errorf("expected 3, got %d", notifyCount)
		}
	})

	t.Run("filters mixed old and new deaths", func(t *testing.T) {
		var notifyCount int
		notifier := &mockDeathNotifier{onNotify: func() { notifyCount++ }}

		tracker := &DeathTracker{
			notifier:   notifier,
			seenDeaths: make(map[string]deathRecord),
			ttl:        25 * time.Hour,
		}

		now := time.Now()
		deaths := []domain.Kill{
			{Time: now.Add(-3 * time.Hour)},    // Old (> 2h)
			{Time: now.Add(-1 * time.Hour)},    // New (< 2h)
			{Time: now.Add(-10 * time.Minute)}, // New (< 2h)
			{Time: now.Add(-4 * time.Hour)},    // Old (> 2h)
		}
		player := &domain.Player{Name: "P1", Deaths: deaths}

		tracker.CheckDeaths(player, []domain.GuildConfig{{DiscordGuildID: "g1"}}, nil)

		if notifyCount != 2 {
			t.Errorf("expected 2 (only new deaths), got %d", notifyCount)
		}
	})

	t.Run("calls evictOld on each check", func(t *testing.T) {
		tracker := &DeathTracker{
			notifier:   &mockDeathNotifier{},
			ttl:        1 * time.Millisecond,
			seenDeaths: make(map[string]deathRecord),
		}

		death := domain.Kill{Time: time.Now()}
		player := &domain.Player{Name: "P1", Deaths: []domain.Kill{death}}
		tracker.CheckDeaths(player, []domain.GuildConfig{{DiscordGuildID: "g1"}}, nil)

		time.Sleep(5 * time.Millisecond)

		player2 := &domain.Player{Name: "P2", Deaths: []domain.Kill{}}
		tracker.CheckDeaths(player2, []domain.GuildConfig{{DiscordGuildID: "g1"}}, nil)

		if len(tracker.seenDeaths) != 0 {
			t.Errorf("expected eviction, got %d entries", len(tracker.seenDeaths))
		}
	})
}

func TestDeathTracker_NotifyDeath(t *testing.T) {
	t.Run("notifies all matching guilds", func(t *testing.T) {
		var notifiedGuilds []string
		notifier := &mockDeathNotifier{
			sendDeathFunc: func(guildID, name string, death domain.Kill) error {
				notifiedGuilds = append(notifiedGuilds, guildID)
				return nil
			},
		}

		guilds := []domain.GuildConfig{
			{DiscordGuildID: "g1", TibiaGuilds: []string{}},
			{DiscordGuildID: "g2", TibiaGuilds: []string{}},
		}

		tracker := &DeathTracker{notifier: notifier}
		tracker.notifyDeath(guilds, "Player", domain.Kill{}, nil)

		if len(notifiedGuilds) != 2 {
			t.Errorf("expected 2, got %d", len(notifiedGuilds))
		}
	})

	t.Run("handles notification error gracefully", func(t *testing.T) {
		notifier := &mockDeathNotifier{
			sendDeathFunc: func(guildID, name string, death domain.Kill) error {
				return errors.New("discord error")
			},
		}

		guilds := []domain.GuildConfig{{DiscordGuildID: "g1"}}
		tracker := &DeathTracker{notifier: notifier}
		tracker.notifyDeath(guilds, "Player", domain.Kill{}, nil)
	})

	t.Run("filters by guild membership", func(t *testing.T) {
		var notifiedGuilds []string
		notifier := &mockDeathNotifier{
			sendDeathFunc: func(guildID, name string, death domain.Kill) error {
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

		tracker := &DeathTracker{notifier: notifier}
		tracker.notifyDeath(guilds, "Player", domain.Kill{}, memberships)

		if len(notifiedGuilds) != 1 || notifiedGuilds[0] != "g1" {
			t.Errorf("expected only g1, got %v", notifiedGuilds)
		}
	})
}

func TestDeathTracker_Concurrency(t *testing.T) {
	t.Run("concurrent isDuplicateDeath is thread-safe", func(t *testing.T) {
		tracker := &DeathTracker{
			seenDeaths: make(map[string]deathRecord),
		}

		var wg sync.WaitGroup
		concurrency := 100

		baseTime := time.Now()
		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			deathTime := baseTime.Add(time.Duration(i) * time.Hour)
			go func(dt time.Time) {
				defer wg.Done()
				tracker.isDuplicateDeath("Player", dt)
			}(deathTime)
		}

		wg.Wait()

		if len(tracker.seenDeaths) != concurrency {
			t.Errorf("expected %d entries, got %d", concurrency, len(tracker.seenDeaths))
		}
	})

	t.Run("concurrent evictOld is thread-safe", func(t *testing.T) {
		tracker := &DeathTracker{
			seenDeaths: make(map[string]deathRecord),
			ttl:        25 * time.Hour,
		}

		for i := 0; i < 100; i++ {
			tracker.seenDeaths[time.Now().String()] = deathRecord{addedAt: time.Now()}
		}

		var wg sync.WaitGroup
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				tracker.evictOld()
			}()
		}

		wg.Wait()
	})
}

type mockDeathNotifier struct {
	onNotify      func()
	sendDeathFunc func(guildID, name string, death domain.Kill) error
}

func (m *mockDeathNotifier) SendDeathNotification(guildID string, playerName string, kill domain.Kill) error {
	if m.onNotify != nil {
		m.onNotify()
	}
	if m.sendDeathFunc != nil {
		return m.sendDeathFunc(guildID, playerName, kill)
	}
	return nil
}

func (m *mockDeathNotifier) SendLevelUpNotification(guildID string, levelUp domain.LevelUp) error {
	return nil
}

func (m *mockDeathNotifier) SendGenericMessage(guildID, channelName, message string) error {
	return nil
}
