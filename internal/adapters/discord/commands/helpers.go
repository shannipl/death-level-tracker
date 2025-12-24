package commands

import "github.com/bwmarrin/discordgo"

func respond(s DiscordSession, i *discordgo.InteractionCreate, msg string, ephemeral bool) {
	var flags discordgo.MessageFlags
	if ephemeral {
		flags = discordgo.MessageFlagsEphemeral
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg,
			Flags:   flags,
		},
	})
}

func respondAutocomplete(s DiscordSession, i *discordgo.InteractionCreate, choices []*discordgo.ApplicationCommandOptionChoice) error {
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{
			Choices: choices,
		},
	})
}

func ensureChannel(s DiscordSession, guildID, name string) (string, error) {
	channels, err := s.GuildChannels(guildID)
	if err != nil {
		return "", err
	}

	for _, ch := range channels {
		if ch.Name == name && ch.Type == discordgo.ChannelTypeGuildText {
			return ch.ID, nil
		}
	}

	ch, err := s.GuildChannelCreate(guildID, name, discordgo.ChannelTypeGuildText)
	if err != nil {
		return "", err
	}
	return ch.ID, nil
}

func getStringOption(opts []*discordgo.ApplicationCommandInteractionDataOption, name string) string {
	for _, opt := range opts {
		if opt.Name == name {
			return opt.StringValue()
		}
	}
	return ""
}

func getFocusedOption(opts []*discordgo.ApplicationCommandInteractionDataOption) string {
	for _, opt := range opts {
		if opt.Focused {
			return opt.StringValue()
		}
	}
	return ""
}
