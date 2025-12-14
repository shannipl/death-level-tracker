package tracker

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"death-level-tracker/internal/formatting"
	"death-level-tracker/internal/storage"
	"death-level-tracker/internal/tibiadata"
)

type Analytics struct {
	storage        storage.Storage
	notifier       Notifier
	bootTime       time.Time
	seenDeaths     map[string]struct{}
	seenDeathsLock sync.Mutex
}

func NewAnalytics(store storage.Storage, notifier Notifier) *Analytics {
	return &Analytics{
		storage:    store,
		notifier:   notifier,
		bootTime:   time.Now(),
		seenDeaths: make(map[string]struct{}),
	}
}

func (a *Analytics) ProcessCharacter(char *tibiadata.CharacterResponse, guilds []string, dbLevels map[string]int) {
	name := char.Character.Character.Name
	currentLevel := char.Character.Character.Level

	a.checkDeaths(name, char.Character.Deaths, guilds)
	a.checkLevelUp(name, currentLevel, char.Character.Character.World, dbLevels, guilds)
}

func (a *Analytics) checkDeaths(name string, deaths []tibiadata.Death, guilds []string) {
	for _, death := range deaths {
		if a.isOldDeath(death.Time) {
			continue
		}

		if a.isDuplicateDeath(name, death.Time) {
			continue
		}

		a.notifyDeath(guilds, name, death)
	}
}

func (a *Analytics) isOldDeath(t time.Time) bool {
	return t.Before(a.bootTime)
}

func (a *Analytics) isDuplicateDeath(name string, t time.Time) bool {
	key := fmt.Sprintf("%s|%s", name, t.Format(time.RFC3339))

	a.seenDeathsLock.Lock()
	defer a.seenDeathsLock.Unlock()

	if _, exists := a.seenDeaths[key]; exists {
		return true
	}

	a.seenDeaths[key] = struct{}{}
	return false
}

func (a *Analytics) checkLevelUp(name string, currentLevel int, world string, dbLevels map[string]int, guilds []string) {
	savedLevel, exists := dbLevels[name]

	if a.shouldUpdateLevel(exists, savedLevel, currentLevel) {
		if err := a.storage.UpsertPlayerLevel(context.Background(), name, currentLevel, world); err != nil {
			slog.Error("Failed to upsert player level", "name", name, "error", err)
		}
	}

	if a.isLevelUp(exists, savedLevel, currentLevel) {
		slog.Info("Level up detected", "name", name, "old_level", savedLevel, "new_level", currentLevel)
		a.notifyLevelUp(guilds, name, savedLevel, currentLevel)
	}
}

func (a *Analytics) shouldUpdateLevel(exists bool, savedLevel, currentLevel int) bool {
	return !exists || savedLevel != currentLevel
}

func (a *Analytics) isLevelUp(exists bool, savedLevel, currentLevel int) bool {
	return exists && currentLevel > savedLevel
}

func (a *Analytics) notifyDeath(guilds []string, name string, death tibiadata.Death) {
	timeStr := death.Time.Local().Format(formatting.DcLongTimeFormat)
	content := formatting.MsgDeath(name, timeStr, death.Reason)

	for _, guildID := range guilds {
		a.notifier.Send(guildID, "death-level-tracker", content)
	}
}

func (a *Analytics) notifyLevelUp(guilds []string, name string, oldLevel, newLevel int) {
	content := formatting.MsgLevelUp(name, oldLevel, newLevel)

	for _, guildID := range guilds {
		a.notifier.Send(guildID, "level-tracker", content)
	}
}
