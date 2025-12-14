package tracker

import (
	"death-level-tracker/internal/tibiadata"

	"github.com/bwmarrin/discordgo"
)

// External API Interfaces - abstractions for external dependencies

// TibiaDataClient defines the TibiaData API methods used by Fetcher
type TibiaDataClient interface {
	GetWorld(world string) ([]tibiadata.OnlinePlayer, error)
	GetCharacter(name string) (*tibiadata.CharacterResponse, error)
}

// DiscordSession defines the Discord API methods used by DiscordNotif ier
type DiscordSession interface {
	GuildChannels(guildID string, options ...discordgo.RequestOption) ([]*discordgo.Channel, error)
	ChannelMessageSend(channelID, content string, options ...discordgo.RequestOption) (*discordgo.Message, error)
}

// Internal Component Interfaces - abstractions for internal components

// Notifier sends notifications to Discord channels
type Notifier interface {
	Send(guildID, channel, content string)
}

// FetcherInterface defines methods for fetching world and character data
type FetcherInterface interface {
	FetchWorld(world string) ([]tibiadata.OnlinePlayer, error)
	FetchWorldFromTibiaCom(world string) (map[string]int, error)
	FetchCharacterDetails(players []tibiadata.OnlinePlayer) <-chan *tibiadata.CharacterResponse
}

// AnalyticsInterface defines methods for processing character data
type AnalyticsInterface interface {
	ProcessCharacter(char *tibiadata.CharacterResponse, guilds []string, dbLevels map[string]int)
}
