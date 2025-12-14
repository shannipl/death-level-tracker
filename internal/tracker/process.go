package tracker

import (
	"context"
	"death-level-tracker/internal/tibiadata"
	"log/slog"
	"time"
)

func (s *Service) processWorld(world string, guilds []string) {

	dbLevels, err := s.fetchPlayerLevels(world)
	if err != nil {
		return
	}

	var onlineNames []string
	var tibiaComLevels map[string]int

	if s.config.UseTibiaComForLevels {
		tibiaComLevels, err = s.fetcher.FetchWorldFromTibiaCom(world)
		if err != nil {
			slog.Warn("Failed to fetch from tibia.com, falling back to TibiaData", "world", world, "error", err)
			// Fallback to TibiaData if tibia.com fails
			s.processWorldUsingTibiaData(world, guilds, dbLevels)
			return
		}

		for name := range tibiaComLevels {
			onlineNames = append(onlineNames, name)
		}

		s.processLevelsFromTibiaCom(tibiaComLevels, world, guilds, dbLevels)

		players := make([]tibiadata.OnlinePlayer, 0, len(tibiaComLevels))
		for name, level := range tibiaComLevels {
			players = append(players, tibiadata.OnlinePlayer{
				Name:  name,
				Level: level,
			})
		}
		s.processDeathsForOnlinePlayers(players, guilds)
	} else {
		onlineNames = s.processWorldUsingTibiaData(world, guilds, dbLevels)
	}

	s.processOfflinePlayers(world, onlineNames, guilds, dbLevels)
	s.performMaintenance(world, onlineNames)
}

func (s *Service) processWorldUsingTibiaData(world string, guilds []string, dbLevels map[string]int) []string {
	players, err := s.fetcher.FetchWorld(world)
	if err != nil {
		return nil
	}

	return s.processCharacters(players, guilds, dbLevels)
}

func (s *Service) fetchPlayerLevels(world string) (map[string]int, error) {
	dbLevels, err := s.storage.GetPlayersLevels(context.Background(), world)
	if err != nil {
		slog.Error("Failed to fetch player levels from DB", "world", world, "error", err)
		return nil, err
	}
	return dbLevels, nil
}

func (s *Service) processCharacters(players []tibiadata.OnlinePlayer, guilds []string, dbLevels map[string]int) []string {
	var filteredPlayers []tibiadata.OnlinePlayer
	for _, p := range players {
		if p.Level >= s.config.MinLevelTrack {
			filteredPlayers = append(filteredPlayers, p)
		}
	}

	results := s.fetcher.FetchCharacterDetails(filteredPlayers)

	var onlineNames []string
	for char := range results {
		s.analytics.ProcessCharacter(char, guilds, dbLevels)
		onlineNames = append(onlineNames, char.Character.Character.Name)
	}
	return onlineNames
}

func (s *Service) processOfflinePlayers(world string, onlineNames, guilds []string, dbLevels map[string]int) {
	offlinePlayers, err := s.storage.GetOfflinePlayers(context.Background(), world, onlineNames)
	if err != nil {
		slog.Error("Failed to get offline players", "world", world, "error", err)
		return
	}

	if len(offlinePlayers) == 0 {
		return
	}

	slog.Info("Checking offline players", "world", world, "count", len(offlinePlayers))

	players := make([]tibiadata.OnlinePlayer, len(offlinePlayers))
	for i, p := range offlinePlayers {
		players[i] = tibiadata.OnlinePlayer{
			Name:  p.Name,
			Level: p.Level,
		}
	}

	results := s.fetcher.FetchCharacterDetails(players)
	for char := range results {
		s.analytics.ProcessCharacter(char, guilds, dbLevels)
	}
}

func (s *Service) performMaintenance(world string, onlineNames []string) {
	ctx := context.Background()

	if len(onlineNames) > 0 {
		if err := s.storage.BatchTouchPlayers(ctx, onlineNames); err != nil {
			slog.Error("Failed to touch players", "world", world, "error", err)
		}
	}

	deletedCount, err := s.storage.DeleteOldPlayers(ctx, world, 30*time.Minute)
	if err != nil {
		slog.Error("Failed to prune old players", "world", world, "error", err)
	} else if deletedCount > 0 {
		slog.Info("Pruned old players", "world", world, "count", deletedCount)
	}
}

func (s *Service) processLevelsFromTibiaCom(tibiaComLevels map[string]int, world string, guilds []string, dbLevels map[string]int) {
	for name, currentLevel := range tibiaComLevels {
		// Skip players below minimum level
		if currentLevel < s.config.MinLevelTrack {
			continue
		}

		savedLevel, exists := dbLevels[name]

		if !exists || savedLevel != currentLevel {
			if err := s.storage.UpsertPlayerLevel(context.Background(), name, currentLevel, world); err != nil {
				slog.Error("Failed to upsert player level", "name", name, "error", err)
			}
			// Update in-memory map to prevent stale data if processed again in this cycle
			dbLevels[name] = currentLevel
		}

		if exists && currentLevel > savedLevel {
			slog.Info("Level up detected", "name", name, "old_level", savedLevel, "new_level", currentLevel)
			s.analytics.(*Analytics).notifyLevelUp(guilds, name, savedLevel, currentLevel)
		}
	}
}

func (s *Service) processDeathsForOnlinePlayers(players []tibiadata.OnlinePlayer, guilds []string) {
	var filteredPlayers []tibiadata.OnlinePlayer
	for _, p := range players {
		if p.Level >= s.config.MinLevelTrack {
			filteredPlayers = append(filteredPlayers, p)
		}
	}

	if len(filteredPlayers) == 0 {
		return
	}

	results := s.fetcher.FetchCharacterDetails(filteredPlayers)

	for char := range results {
		name := char.Character.Character.Name
		s.analytics.(*Analytics).checkDeaths(name, char.Character.Deaths, guilds)
	}
}
