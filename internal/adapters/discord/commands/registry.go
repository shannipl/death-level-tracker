package commands

import (
	"log/slog"

	"github.com/bwmarrin/discordgo"
)

var adminPerms = int64(discordgo.PermissionAdministrator)

func GetApplicationCommands() []*discordgo.ApplicationCommand {
	return []*discordgo.ApplicationCommand{
		{
			Name:                     "track-world",
			Description:              "Set the Tibia world to track for this server",
			DefaultMemberPermissions: &adminPerms,
			Options: []*discordgo.ApplicationCommandOption{
				stringOption("name", "Name of the Tibia world", true, false),
			},
		},
		{
			Name:                     "stop-tracking",
			Description:              "Stop tracking kills",
			DefaultMemberPermissions: &adminPerms,
		},
		{
			Name:                     "add-guild",
			Description:              "Add a Tibia guild to the tracking whitelist",
			DefaultMemberPermissions: &adminPerms,
			Options: []*discordgo.ApplicationCommandOption{
				stringOption("name", "Name of the Tibia guild", true, false),
			},
		},
		{
			Name:                     "unset-guild",
			Description:              "Remove a Tibia guild from the tracking whitelist",
			DefaultMemberPermissions: &adminPerms,
			Options: []*discordgo.ApplicationCommandOption{
				stringOption("name", "Name of the Tibia guild", true, true),
			},
		},
		{
			Name:                     "list-guilds",
			Description:              "List all tracked Tibia guilds",
			DefaultMemberPermissions: &adminPerms,
		},
	}
}

func stringOption(name, description string, required, autocomplete bool) *discordgo.ApplicationCommandOption {
	return &discordgo.ApplicationCommandOption{
		Type:         discordgo.ApplicationCommandOptionString,
		Name:         name,
		Description:  description,
		Required:     required,
		Autocomplete: autocomplete,
	}
}

func RegisterCommands(session CommandSession, commands []*discordgo.ApplicationCommand, userID, guildID string) []*discordgo.ApplicationCommand {
	registered := make([]*discordgo.ApplicationCommand, len(commands))

	for i, cmd := range commands {
		result, err := session.ApplicationCommandCreate(userID, guildID, cmd)
		if err != nil {
			slog.Error("Cannot create command", "name", cmd.Name, "error", err)
			continue
		}
		registered[i] = result
		slog.Info("Registered command", "name", cmd.Name, "guild", guildID)
	}

	return registered
}

func CleanupCommands(session CommandSession, commands []*discordgo.ApplicationCommand, userID, guildID string) {
	for _, cmd := range commands {
		if cmd == nil {
			continue
		}
		if err := session.ApplicationCommandDelete(userID, guildID, cmd.ID); err != nil {
			slog.Error("Cannot delete command", "name", cmd.Name, "error", err)
		}
	}
}
