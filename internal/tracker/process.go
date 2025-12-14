package tracker

import (
	"context"
	"death-level-tracker/internal/tibiadata"
	"log/slog"
	"time"
)

func (s *Service) processWorld(world string, guilds []string) {
	players, err := s.fetcher.FetchWorld(world)
	if err != nil {
		return
	}

	dbLevels, err := s.fetchPlayerLevels(world)
	if err != nil {
		return
	}

	onlineNames := s.processCharacters(players, guilds, dbLevels)

	s.processOfflinePlayers(world, onlineNames, guilds, dbLevels)

	s.performMaintenance(world, onlineNames)
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
	results := s.fetcher.FetchCharacterDetails(players)

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
