package tracker

import (
	"context"
	"log/slog"
	"time"

	"death-level-tracker/internal/config"
	"death-level-tracker/internal/storage"
	"death-level-tracker/internal/tibiadata"

	"github.com/bwmarrin/discordgo"
)

type Service struct {
	config    *config.Config
	storage   storage.Storage
	fetcher   FetcherInterface
	analytics AnalyticsInterface
}

func NewService(cfg *config.Config, store storage.Storage, discord *discordgo.Session) *Service {
	notifier := NewDiscordNotifier(discord)
	client := tibiadata.NewClient()

	return &Service{
		config:    cfg,
		storage:   store,
		fetcher:   NewFetcher(client, cfg),
		analytics: NewAnalytics(store, notifier),
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
	worlds, err := s.storage.GetWorldsMap(ctx)
	if err != nil {
		slog.Error("Failed to fetch worlds map", "error", err)
		return
	}

	for world, guilds := range worlds {
		slog.Info("Processing world", "world", world, "guilds_count", len(guilds))
		go s.processWorld(world, guilds)
	}
}
