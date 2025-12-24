package discord

import (
	"fmt"
	"log/slog"
	"strings"

	"death-level-tracker/internal/adapters/discord/formatting"
	"death-level-tracker/internal/adapters/metrics"
	"death-level-tracker/internal/config"
	"death-level-tracker/internal/core/domain"

	"github.com/bwmarrin/discordgo"
)

type DiscordSession interface {
	GuildChannels(guildID string, options ...discordgo.RequestOption) ([]*discordgo.Channel, error)
	ChannelMessageSend(channelID, content string, options ...discordgo.RequestOption) (*discordgo.Message, error)
}

type Adapter struct {
	session DiscordSession
	config  *config.Config
	cache   *channelCache
}

func NewAdapter(session DiscordSession, cfg *config.Config) *Adapter {
	return &Adapter{
		session: session,
		config:  cfg,
		cache:   newChannelCache(),
	}
}

func (a *Adapter) SendLevelUpNotification(guildID string, levelUp domain.LevelUp) error {
	content := formatting.MsgLevelUp(levelUp.PlayerName, levelUp.OldLevel, levelUp.NewLevel)
	return a.SendGenericMessage(guildID, a.config.DiscordChannelLevel, content)
}

func (a *Adapter) SendDeathNotification(guildID string, playerName string, kill domain.Kill) error {
	timeStr := kill.Time.Local().Format(formatting.DcLongTimeFormat)
	content := formatting.MsgDeath(playerName, timeStr, kill.Reason)
	return a.SendGenericMessage(guildID, a.config.DiscordChannelDeath, content)
}

func (a *Adapter) SendGenericMessage(guildID, channelName, message string) error {
	channelID, err := a.resolveChannelID(guildID, channelName)
	if err != nil {
		slog.Error("Failed to get channel ID", "guild_id", guildID, "channel_name", channelName, "error", err)
		return err
	}

	if _, err := a.session.ChannelMessageSend(channelID, message); err != nil {
		slog.Error("Failed to send message", "channel_id", channelID, "error", err)
		a.cache.Invalidate(guildID, channelName)
		metrics.DiscordMessagesSent.WithLabelValues(channelType(channelName), "failure").Inc()
		return err
	}

	metrics.DiscordMessagesSent.WithLabelValues(channelType(channelName), "success").Inc()
	return nil
}

func (a *Adapter) resolveChannelID(guildID, channelName string) (string, error) {
	if id, ok := a.cache.Get(guildID, channelName); ok {
		return id, nil
	}

	id, err := a.fetchChannelID(guildID, channelName)
	if err != nil {
		return "", err
	}

	a.cache.Set(guildID, channelName, id)
	return id, nil
}

func (a *Adapter) fetchChannelID(guildID, channelName string) (string, error) {
	channels, err := a.session.GuildChannels(guildID)
	if err != nil {
		slog.Error("Failed to fetch guild channels", "guild_id", guildID, "error", err)
		return "", err
	}

	for _, ch := range channels {
		if ch.Name == channelName && ch.Type == discordgo.ChannelTypeGuildText {
			return ch.ID, nil
		}
	}

	return "", fmt.Errorf("channel %s not found", channelName)
}

func channelType(name string) string {
	switch {
	case strings.Contains(name, "death"):
		return "death"
	case strings.Contains(name, "level"):
		return "level"
	default:
		return "other"
	}
}
