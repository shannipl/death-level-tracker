package commands

import "github.com/bwmarrin/discordgo"

type DiscordSession interface {
	GuildChannels(guildID string, options ...discordgo.RequestOption) ([]*discordgo.Channel, error)
	GuildChannelCreate(guildID, name string, ctype discordgo.ChannelType, options ...discordgo.RequestOption) (*discordgo.Channel, error)
	InteractionRespond(interaction *discordgo.Interaction, resp *discordgo.InteractionResponse, options ...discordgo.RequestOption) error
}

type CommandSession interface {
	ApplicationCommandCreate(appID, guildID string, cmd *discordgo.ApplicationCommand, options ...discordgo.RequestOption) (*discordgo.ApplicationCommand, error)
	ApplicationCommandDelete(appID, guildID, cmdID string, options ...discordgo.RequestOption) error
}
