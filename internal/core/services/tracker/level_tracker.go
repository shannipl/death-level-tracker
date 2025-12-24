package tracker

import (
	"context"
	"log/slog"

	"death-level-tracker/internal/adapters/metrics"
	"death-level-tracker/internal/config"
	"death-level-tracker/internal/core/domain"
	"death-level-tracker/internal/core/ports"
)

type LevelTracker struct {
	config   *config.Config
	storage  ports.Repository
	notifier ports.NotificationService
}

func NewLevelTracker(cfg *config.Config, store ports.Repository, notifier ports.NotificationService) *LevelTracker {
	return &LevelTracker{
		config:   cfg,
		storage:  store,
		notifier: notifier,
	}
}

func (l *LevelTracker) CheckLevelUp(ctx context.Context, name string, currentLevel int, world string, dbLevels map[string]int, guilds []domain.GuildConfig, memberships map[string]map[string]bool) {
	savedLevel, exists := dbLevels[name]

	if l.shouldUpdateLevel(exists, savedLevel, currentLevel) {
		if err := l.storage.UpsertPlayerLevel(ctx, name, currentLevel, world); err != nil {
			slog.Error("Failed to upsert player level", "name", name, "error", err)
		}
	}

	if l.isLevelUp(exists, savedLevel, currentLevel) {
		slog.Info("Level up detected", "name", name, "old_level", savedLevel, "new_level", currentLevel)
		l.notifyLevelUp(guilds, name, savedLevel, currentLevel, world, memberships)
	}
}

func (l *LevelTracker) shouldUpdateLevel(exists bool, savedLevel, currentLevel int) bool {
	if exists && currentLevel < savedLevel {
		return false
	}
	return !exists || savedLevel != currentLevel
}

func (l *LevelTracker) isLevelUp(exists bool, savedLevel, currentLevel int) bool {
	return exists && currentLevel > savedLevel
}

func (l *LevelTracker) notifyLevelUp(guilds []domain.GuildConfig, name string, oldLevel, newLevel int, world string, memberships map[string]map[string]bool) {
	levelUp := domain.LevelUp{
		PlayerName: name,
		OldLevel:   oldLevel,
		NewLevel:   newLevel,
		World:      world,
	}

	for _, guild := range guilds {
		if shouldNotifyGuild(name, guild, memberships) {
			if err := l.notifier.SendLevelUpNotification(guild.DiscordGuildID, levelUp); err != nil {
				slog.Error("Failed to send level up notification", "guild_id", guild.DiscordGuildID, "error", err)
			}
		}
	}

	metrics.TrackedLevelUps.Inc()
}

func shouldNotifyGuild(characterName string, guild domain.GuildConfig, memberships map[string]map[string]bool) bool {
	if len(guild.TibiaGuilds) == 0 {
		return true
	}

	for _, tibiaGuild := range guild.TibiaGuilds {
		if members, ok := memberships[tibiaGuild]; ok {
			if members[characterName] {
				return true
			}
		}
	}

	return false
}
