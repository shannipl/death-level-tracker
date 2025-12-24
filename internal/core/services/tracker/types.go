package tracker

import "death-level-tracker/internal/core/domain"

type worldContext struct {
	world       string
	guilds      []domain.GuildConfig
	dbLevels    map[string]int
	memberships map[string]map[string]bool
}
