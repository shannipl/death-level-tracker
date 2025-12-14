package main

import "github.com/bwmarrin/discordgo"

// CommandSession defines the Discord session operations needed for command management
type CommandSession interface {
	ApplicationCommandCreate(appID, guildID string, cmd *discordgo.ApplicationCommand, options ...discordgo.RequestOption) (*discordgo.ApplicationCommand, error)
	ApplicationCommandDelete(appID, guildID, cmdID string, options ...discordgo.RequestOption) error
}
