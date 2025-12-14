package main

import (
	"log/slog"

	"death-level-tracker/internal/config"

	"github.com/bwmarrin/discordgo"
)

func NewDiscordSession(cfg *config.Config) (*discordgo.Session, error) {
	discord, err := discordgo.New("Bot " + cfg.Token)
	if err != nil {
		slog.Error("Failed to create discord session", "error", err)
		return nil, err
	}

	discord.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessages

	return discord, nil
}
