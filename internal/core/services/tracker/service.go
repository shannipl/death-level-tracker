package tracker

import (
	"context"
	"log/slog"
	"time"

	"death-level-tracker/internal/config"
	"death-level-tracker/internal/core/domain"
	"death-level-tracker/internal/core/ports"
)

type Dependencies struct {
	Config   *config.Config
	Storage  ports.Repository
	Fetcher  ports.TibiaFetcher
	Notifier ports.NotificationService
}

type Service struct {
	config       *config.Config
	storage      ports.Repository
	fetcher      ports.TibiaFetcher
	levelTracker *LevelTracker
	deathTracker *DeathTracker
}

func NewService(deps Dependencies) *Service {
	return &Service{
		config:       deps.Config,
		storage:      deps.Storage,
		fetcher:      deps.Fetcher,
		levelTracker: NewLevelTracker(deps.Config, deps.Storage, deps.Notifier),
		deathTracker: NewDeathTracker(deps.Notifier),
	}
}

func (s *Service) Start(ctx context.Context) {
	ticker := time.NewTicker(s.config.TrackerInterval)
	defer ticker.Stop()

	slog.Info("Tracker service started", "interval", s.config.TrackerInterval)

	s.runLoop(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runLoop(ctx)
		}
	}
}

func (s *Service) runLoop(ctx context.Context) {
	configs, err := s.storage.GetAllGuildConfigs(ctx)
	if err != nil {
		slog.Error("Failed to fetch guild configs", "error", err)
		return
	}

	worlds := groupConfigsByWorld(configs)

	for world, guilds := range worlds {
		slog.Info("Processing world", "world", world, "guilds_count", len(guilds))
		go s.processWorld(ctx, world, guilds)
	}
}

func groupConfigsByWorld(configs []domain.GuildConfig) map[string][]domain.GuildConfig {
	worlds := make(map[string][]domain.GuildConfig)
	for _, cfg := range configs {
		if cfg.World == "" {
			continue
		}
		worlds[cfg.World] = append(worlds[cfg.World], cfg)
	}
	return worlds
}
