package main

import (
	"context"
	"log/slog"
	"os"

	"death-level-tracker/internal/config"
	"death-level-tracker/internal/handlers"
	"death-level-tracker/internal/storage"
	"death-level-tracker/internal/tracker"

	"github.com/bwmarrin/discordgo"
)

type App struct {
	config             *config.Config
	store              *storage.PostgresStore
	discord            *discordgo.Session
	trackerService     *tracker.Service
	router             *handlers.Router
	trackerCtx         context.Context
	trackerCancel      context.CancelFunc
	registeredCommands []*discordgo.ApplicationCommand
}

func NewApp(ctx context.Context) (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		return nil, err
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		slog.Error("DATABASE_URL is not set")
		return nil, err
	}

	store, err := storage.NewPostgresStore(ctx, dbURL)
	if err != nil {
		slog.Error("Failed to connect to storage", "error", err)
		return nil, err
	}

	discord, err := NewDiscordSession(cfg)
	if err != nil {
		return nil, err
	}

	trackerService := tracker.NewService(cfg, store, discord)

	botHandlers := &handlers.BotHandler{Config: cfg, Store: store}
	router := handlers.NewRouter()
	router.Register("track-world", handlers.WithAdmin(botHandlers.TrackWorld))
	router.Register("stop-tracking", handlers.WithAdmin(botHandlers.StopTracking))

	discord.AddHandler(handlers.ReadyHandler)
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
	err := a.discord.Open()
	if err != nil {
		slog.Error("Failed to open discord session", "error", err)
		return err
	}

	commands := GetApplicationCommands()
	CleanupCommands(a.discord, a.registeredCommands, a.discord.State.User.ID)
	a.registeredCommands = RegisterCommands(a.discord, commands, a.discord.State.User.ID)

	slog.Info("Players Tracker is online!")

	a.trackerCtx, a.trackerCancel = context.WithCancel(context.Background())
	go a.trackerService.Start(a.trackerCtx)

	return nil
}

func (a *App) Shutdown() {
	slog.Info("Shutting down...")

	if a.trackerCancel != nil {
		a.trackerCancel()
	}

	if a.discord != nil {
		a.discord.Close()
	}

	if a.store != nil {
		a.store.Close()
	}
}
