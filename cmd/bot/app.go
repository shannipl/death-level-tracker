package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	discordadapter "death-level-tracker/internal/adapters/discord"
	"death-level-tracker/internal/adapters/discord/commands"
	"death-level-tracker/internal/adapters/storage/postgres"
	"death-level-tracker/internal/adapters/tibiadata"
	"death-level-tracker/internal/adapters/tibiadata/api"
	"death-level-tracker/internal/config"
	"death-level-tracker/internal/core/ports"
	"death-level-tracker/internal/core/services"
	"death-level-tracker/internal/core/services/tracker"

	"github.com/bwmarrin/discordgo"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type App struct {
	config         *config.Config
	store          ports.Repository
	discord        *discordgo.Session
	trackerService *tracker.Service
	router         *commands.Router

	metricsServer *http.Server

	trackerCtx    context.Context
	trackerCancel context.CancelFunc

	registeredCommands []*discordgo.ApplicationCommand
}

func NewApp(ctx context.Context, cfg *config.Config) (*App, error) {
	store, err := postgres.NewPostgresStore(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("Failed to connect to storage", "error", err)
		return nil, err
	}

	discord, err := discordadapter.NewSession(cfg)
	if err != nil {
		return nil, err
	}

	client := api.NewClient()
	fetcher := tibiadata.NewAdapter(client, cfg)
	notifier := discordadapter.NewAdapter(discord, cfg)

	trackerService := tracker.NewService(tracker.Dependencies{
		Config:   cfg,
		Storage:  store,
		Fetcher:  fetcher,
		Notifier: notifier,
	})

	configService := services.NewConfigurationService(store)
	botHandlers := &commands.BotHandler{Config: cfg, Service: configService}

	router := commands.NewRouter()
	router.Register("track-world", commands.WithAdmin(botHandlers.TrackWorld))
	router.Register("stop-tracking", commands.WithAdmin(botHandlers.StopTracking))
	router.Register("add-guild", commands.WithAdmin(botHandlers.AddGuild))
	router.Register("unset-guild", commands.WithAdmin(botHandlers.UnsetGuild))
	router.Register("list-guilds", commands.WithAdmin(botHandlers.ListGuilds))

	discord.AddHandler(commands.ReadyHandler)
	discord.AddHandler(router.HandleFunc())

	return &App{
		config:         cfg,
		store:          store,
		discord:        discord,
		trackerService: trackerService,
		router:         router,
	}, nil
}

func (a *App) Run() error {
	a.startMetricsServer()

	if err := a.discord.Open(); err != nil {
		slog.Error("Failed to open discord session", "error", err)
		return err
	}

	cmds := commands.GetApplicationCommands()
	commands.CleanupCommands(a.discord, a.registeredCommands, a.discord.State.User.ID, a.config.DiscordGuildID)
	a.registeredCommands = commands.RegisterCommands(a.discord, cmds, a.discord.State.User.ID, a.config.DiscordGuildID)

	slog.Info("Players Tracker is online!")

	a.trackerCtx, a.trackerCancel = context.WithCancel(context.Background())
	go a.trackerService.Start(a.trackerCtx)

	return nil
}

func (a *App) Shutdown(ctx context.Context) error {
	slog.Info("Shutting down application...")

	if a.trackerCancel != nil {
		a.trackerCancel()
	}

	if a.metricsServer != nil {
		if err := a.metricsServer.Shutdown(ctx); err != nil {
			slog.Error("Failed to shutdown metrics server", "error", err)
		}
	}

	if a.discord != nil {
		if err := a.discord.Close(); err != nil {
			slog.Error("Failed to close discord session", "error", err)
		}
	}

	if a.store != nil {
		a.store.Close()
	}

	slog.Info("Shutdown complete")
	return nil
}

func (a *App) startMetricsServer() {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	a.metricsServer = &http.Server{
		Addr:    ":2112",
		Handler: mux,
	}

	go func() {
		slog.Info("Starting metrics server on :2112")
		if err := a.metricsServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Metrics server failed", "error", err)
		}
	}()
}
