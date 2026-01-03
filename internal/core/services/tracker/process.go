package tracker

import (
	"context"
	"log/slog"
	"time"

	"death-level-tracker/internal/core/domain"
)

func (s *Service) processWorld(ctx context.Context, world string, guilds []domain.GuildConfig) {
	wctx := s.initWorldContext(ctx, world, guilds)
	if wctx == nil {
		return
	}
	slog.Info("Processing world", "world", world)
	onlineNames := s.processOnlinePlayers(ctx, wctx)
	s.performMaintenance(ctx, world, onlineNames)
	s.processOfflinePlayers(ctx, wctx, onlineNames)
	slog.Info("Finished processing world", "world", world)
}

func (s *Service) initWorldContext(ctx context.Context, world string, guilds []domain.GuildConfig) *worldContext {
	dbLevels, err := s.fetchPlayerLevels(ctx, world)
	if err != nil {
		return nil
	}

	return &worldContext{
		world:       world,
		guilds:      guilds,
		dbLevels:    dbLevels,
		memberships: s.fetchGuildMemberships(ctx, guilds),
	}
}

func (s *Service) fetchGuildMemberships(ctx context.Context, guilds []domain.GuildConfig) map[string]map[string]bool {
	uniqueGuilds := make(map[string]struct{})
	for _, cfg := range guilds {
		for _, g := range cfg.TibiaGuilds {
			uniqueGuilds[g] = struct{}{}
		}
	}

	memberships := make(map[string]map[string]bool)
	for guildName := range uniqueGuilds {
		members := s.getGuildMembers(ctx, guildName)
		if members == nil {
			continue
		}

		memberMap := make(map[string]bool)
		for _, m := range members {
			memberMap[m] = true
		}
		memberships[guildName] = memberMap
	}

	return memberships
}

func (s *Service) getGuildMembers(ctx context.Context, guildName string) []string {
	s.cacheMu.RLock()
	item, cached := s.guildCache[guildName]
	s.cacheMu.RUnlock()

	now := time.Now()
	if cached && now.Before(item.ExpiresAt) {
		return item.Members
	}

	members, err := s.fetcher.FetchGuildMembers(ctx, guildName)
	if err != nil {
		slog.Warn("Failed to fetch guild members", "guild", guildName, "error", err)
		if cached {
			slog.Info("Using stale cache for guild", "guild", guildName)
			return item.Members
		}
		return nil
	}

	s.cacheMu.Lock()
	s.guildCache[guildName] = GuildCacheItem{
		Members:   members,
		ExpiresAt: now.Add(15 * time.Minute),
	}
	s.cacheMu.Unlock()

	return members
}

func (s *Service) processOnlinePlayers(ctx context.Context, wctx *worldContext) []string {
	if s.config.UseTibiaComForLevels {
		slog.Info("Processing online players via tibia.com", "world", wctx.world)
		return s.processViaTibiaCom(ctx, wctx)
	}
	slog.Info("Processing online players via TibiaData", "world", wctx.world)
	return s.processViaTibiaData(ctx, wctx)
}

func (s *Service) processViaTibiaCom(ctx context.Context, wctx *worldContext) []string {
	levels, err := s.fetcher.FetchWorldFromTibiaCom(ctx, wctx.world)
	if err != nil {
		slog.Warn("Failed to fetch from tibia.com, falling back to TibiaData", "world", wctx.world, "error", err)
		return s.processViaTibiaData(ctx, wctx)
	}

	onlineNames := extractNames(levels)
	slog.Info("Extracted online players", "world", wctx.world, "count", len(onlineNames))

	s.processLevelsFromTibiaCom(ctx, levels, wctx)
	s.performMaintenance(ctx, wctx.world, onlineNames)
	s.processDeathsForOnlinePlayers(ctx, levelsToPlayers(levels), wctx)

	slog.Info("Finished processing online players", "world", wctx.world, "count", len(onlineNames))
	return onlineNames
}

func (s *Service) processViaTibiaData(ctx context.Context, wctx *worldContext) []string {
	players, err := s.fetcher.FetchWorld(ctx, wctx.world)
	if err != nil {
		return nil
	}

	return s.processCharacters(ctx, players, wctx)
}

func (s *Service) processCharacters(ctx context.Context, players []domain.Player, wctx *worldContext) []string {
	filteredNames := s.filterByMinLevel(players)

	results, err := s.fetcher.FetchCharacterDetails(ctx, filteredNames)
	if err != nil {
		slog.Error("Failed to fetch character details", "error", err)
		return nil
	}

	var onlineNames []string
	for char := range results {
		if char.Level < s.config.MinLevelTrack {
			continue
		}
		s.deathTracker.CheckDeaths(char, wctx.guilds, wctx.memberships)
		s.levelTracker.CheckLevelUp(ctx, char.Name, char.Level, char.World, wctx.dbLevels, wctx.guilds, wctx.memberships)
		onlineNames = append(onlineNames, char.Name)
	}
	return onlineNames
}

func (s *Service) filterByMinLevel(players []domain.Player) []string {
	var names []string
	for _, p := range players {
		if p.Level >= s.config.MinLevelTrack {
			names = append(names, p.Name)
		}
	}
	return names
}

func (s *Service) processOfflinePlayers(ctx context.Context, wctx *worldContext, onlineNames []string) {
	offlinePlayers, err := s.storage.GetOfflinePlayers(ctx, wctx.world, onlineNames)
	slog.Info("Found offline players", "world", wctx.world, "count", len(offlinePlayers))
	if err != nil {
		slog.Error("Failed to get offline players", "world", wctx.world, "error", err)
		return
	}

	if len(offlinePlayers) == 0 {
		return
	}

	slog.Info("Checking offline players", "world", wctx.world, "count", len(offlinePlayers))

	names := playerNames(offlinePlayers)
	results, err := s.fetcher.FetchCharacterDetails(ctx, names)
	if err != nil {
		slog.Error("Failed to fetch character details for offline players", "error", err)
		return
	}
	slog.Info("Fetched details for offline players from TibiaData", "world", wctx.world, "count", len(results))

	for char := range results {
		if char.Level < s.config.MinLevelTrack {
			continue
		}
		s.deathTracker.CheckDeaths(char, wctx.guilds, wctx.memberships)
		s.levelTracker.CheckLevelUp(ctx, char.Name, char.Level, char.World, wctx.dbLevels, wctx.guilds, wctx.memberships)
	}
	slog.Info("Finished checking offline players", "world", wctx.world, "count", len(offlinePlayers))
}

func (s *Service) performMaintenance(ctx context.Context, world string, onlineNames []string) {
	slog.Info("Performing maintenance", "world", world, "online_count", len(onlineNames))
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

func (s *Service) fetchPlayerLevels(ctx context.Context, world string) (map[string]int, error) {
	dbLevels, err := s.storage.GetPlayersLevels(ctx, world)
	if err != nil {
		slog.Error("Failed to fetch player levels from DB", "world", world, "error", err)
		return nil, err
	}
	return dbLevels, nil
}

func (s *Service) processLevelsFromTibiaCom(ctx context.Context, levels map[string]int, wctx *worldContext) {
	for name, currentLevel := range levels {
		if currentLevel < s.config.MinLevelTrack {
			continue
		}

		savedLevel, exists := wctx.dbLevels[name]

		if !exists || savedLevel != currentLevel {
			if err := s.storage.UpsertPlayerLevel(ctx, name, currentLevel, wctx.world); err != nil {
				slog.Error("Failed to upsert player level", "name", name, "error", err)
			}
			wctx.dbLevels[name] = currentLevel
		}

		if exists && currentLevel > savedLevel {
			slog.Info("Level up detected", "name", name, "old_level", savedLevel, "new_level", currentLevel)
			s.levelTracker.notifyLevelUp(wctx.guilds, name, savedLevel, currentLevel, wctx.world, wctx.memberships)
		}
	}
	slog.Info("Finished processing players from tibia.com", "world", wctx.world, "count", len(levels))
}

func (s *Service) processDeathsForOnlinePlayers(ctx context.Context, players []domain.Player, wctx *worldContext) {
	filteredNames := s.filterByMinLevel(players)
	if len(filteredNames) == 0 {
		return
	}

	slog.Info("Processing deaths for online players", "world", wctx.world, "count", len(filteredNames))
	results, err := s.fetcher.FetchCharacterDetails(ctx, filteredNames)
	slog.Info("Fetched details for online players from TibiaData", "world", wctx.world, "count", len(results))
	if err != nil {
		slog.Error("Failed to fetch character details for deaths", "error", err)
		return
	}

	slog.Info("Checking deaths for online players", "world", wctx.world, "count", len(results))
	for char := range results {
		s.deathTracker.CheckDeaths(char, wctx.guilds, wctx.memberships)
	}
	slog.Info("Finished checking deaths for online players", "world", wctx.world, "count", len(results))
}

func extractNames(levels map[string]int) []string {
	names := make([]string, 0, len(levels))
	for name := range levels {
		names = append(names, name)
	}
	return names
}

func levelsToPlayers(levels map[string]int) []domain.Player {
	players := make([]domain.Player, 0, len(levels))
	for name, level := range levels {
		players = append(players, domain.Player{Name: name, Level: level})
	}
	return players
}

func playerNames(players []domain.Player) []string {
	names := make([]string, len(players))
	for i, p := range players {
		names[i] = p.Name
	}
	return names
}
