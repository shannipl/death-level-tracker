package handlers

import "github.com/bwmarrin/discordgo"

// DiscordSession defines the interface for Discord API operations needed by handlers.
// This interface allows for testing with mocked Discord sessions.
type DiscordSession interface {
	GuildChannels(guildID string, options ...discordgo.RequestOption) ([]*discordgo.Channel, error)
	GuildChannelCreate(guildID, name string, ctype discordgo.ChannelType, options ...discordgo.RequestOption) (*discordgo.Channel, error)
	InteractionRespond(interaction *discordgo.Interaction, resp *discordgo.InteractionResponse, options ...discordgo.RequestOption) error
}
