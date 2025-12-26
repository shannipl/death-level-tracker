package tracker

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"death-level-tracker/internal/adapters/metrics"
	"death-level-tracker/internal/core/domain"
	"death-level-tracker/internal/core/ports"
)

const deathCacheTTL = 25 * time.Hour

type deathRecord struct {
	addedAt time.Time
}

type DeathTracker struct {
	notifier   ports.NotificationService
	seenDeaths map[string]deathRecord
	ttl        time.Duration
	mu         sync.Mutex
}

func NewDeathTracker(notifier ports.NotificationService) *DeathTracker {
	return &DeathTracker{
		notifier:   notifier,
		seenDeaths: make(map[string]deathRecord),
		ttl:        deathCacheTTL,
	}
}

func (d *DeathTracker) CheckDeaths(player *domain.Player, guilds []domain.GuildConfig, memberships map[string]map[string]bool) {
	d.evictOld()

	for _, death := range player.Deaths {
		if d.isOldDeath(death.Time) {
			continue
		}

		if d.isDuplicateDeath(player.Name, death.Time) {
			continue
		}

		d.notifyDeath(guilds, player.Name, death, memberships)
	}
}

func (d *DeathTracker) evictOld() {
	d.mu.Lock()
	defer d.mu.Unlock()

	cutoff := time.Now().Add(-d.ttl)
	for key, record := range d.seenDeaths {
		if record.addedAt.Before(cutoff) {
			delete(d.seenDeaths, key)
		}
	}
}

func (d *DeathTracker) isOldDeath(t time.Time) bool {
	return t.Before(time.Now().Add(-2 * time.Hour))
}

func (d *DeathTracker) isDuplicateDeath(name string, t time.Time) bool {
	key := fmt.Sprintf("%s|%s", name, t.Format(time.RFC3339))

	d.mu.Lock()
	defer d.mu.Unlock()

	if _, exists := d.seenDeaths[key]; exists {
		return true
	}

	d.seenDeaths[key] = deathRecord{addedAt: time.Now()}
	return false
}

func (d *DeathTracker) notifyDeath(guilds []domain.GuildConfig, name string, death domain.Kill, memberships map[string]map[string]bool) {
	for _, guild := range guilds {
		if shouldNotifyGuild(name, guild, memberships) {
			if err := d.notifier.SendDeathNotification(guild.DiscordGuildID, name, death); err != nil {
				slog.Error("Failed to send death notification", "guild_id", guild.DiscordGuildID, "error", err)
			}
		}
	}

	metrics.TrackedDeaths.Inc()
}
