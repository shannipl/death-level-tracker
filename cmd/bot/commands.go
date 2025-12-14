package main

import (
	"log/slog"

	"github.com/bwmarrin/discordgo"
)

func GetApplicationCommands() []*discordgo.ApplicationCommand {
	return []*discordgo.ApplicationCommand{
		{
			Name:        "track-world",
			Description: "Set the Tibia world to track for this server",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "name",
					Description: "Name of the Tibia world",
					Required:    true,
				},
			},
		},
		{
			Name:        "stop-tracking",
			Description: "Stop tracking kills",
		},
	}
}

func RegisterCommands(session CommandSession, commands []*discordgo.ApplicationCommand, userID string) []*discordgo.ApplicationCommand {
	registeredCommands := make([]*discordgo.ApplicationCommand, len(commands))

	for i, cmd := range commands {
		registered, err := session.ApplicationCommandCreate(userID, "", cmd)
		if err != nil {
			slog.Error("Cannot create command", "name", cmd.Name, "error", err)
			continue
		}
		registeredCommands[i] = registered
		slog.Info("Registered command", "name", cmd.Name)
	}

	return registeredCommands
}

func CleanupCommands(session CommandSession, commands []*discordgo.ApplicationCommand, userID string) {
	for _, cmd := range commands {
		if cmd == nil {
			continue
		}
		err := session.ApplicationCommandDelete(userID, "", cmd.ID)
		if err != nil {
			slog.Error("Cannot delete command", "name", cmd.Name, "error", err)
		}
	}
}
