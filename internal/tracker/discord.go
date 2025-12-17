package tracker

import (
	"death-level-tracker/internal/metrics"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"
)

type DiscordNotifier struct {
	session          DiscordSession
	channelCache     map[string]string
	channelCacheLock sync.RWMutex
}

func NewDiscordNotifier(session DiscordSession) *DiscordNotifier {
	return &DiscordNotifier{
		session:      session,
		channelCache: make(map[string]string),
	}
}

func (d *DiscordNotifier) Send(guildID, channelName, content string) {
	channelID, err := d.getChannelID(guildID, channelName)
	if err != nil {
		slog.Error("Failed to get channel ID", "guild_id", guildID, "channel_name", channelName, "error", err)
		return
	}

	channelType := d.getChannelType(channelName)

	if _, err := d.session.ChannelMessageSend(channelID, content); err != nil {
		slog.Error("Failed to send message", "channel_id", channelID, "error", err)
		d.invalidateCache(guildID, channelName)
		metrics.DiscordMessagesSent.WithLabelValues(channelType, "failure").Inc()
	} else {
		metrics.DiscordMessagesSent.WithLabelValues(channelType, "success").Inc()
	}
}

func (d *DiscordNotifier) getChannelID(guildID, channelName string) (string, error) {
	key := d.buildCacheKey(guildID, channelName)

	if id, ok := d.getCachedChannelID(key); ok {
		return id, nil
	}

	id, err := d.fetchChannelID(guildID, channelName)
	if err != nil {
		return "", err
	}

	d.setCachedChannelID(key, id)

	return id, nil
}

func (d *DiscordNotifier) getCachedChannelID(key string) (string, bool) {
	d.channelCacheLock.RLock()
	defer d.channelCacheLock.RUnlock()

	id, ok := d.channelCache[key]
	return id, ok
}

func (d *DiscordNotifier) setCachedChannelID(key, id string) {
	d.channelCacheLock.Lock()
	defer d.channelCacheLock.Unlock()

	d.channelCache[key] = id
}

func (d *DiscordNotifier) fetchChannelID(guildID, channelName string) (string, error) {
	channels, err := d.session.GuildChannels(guildID)
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

func (d *DiscordNotifier) invalidateCache(guildID, channelName string) {
	d.channelCacheLock.Lock()
	defer d.channelCacheLock.Unlock()
	delete(d.channelCache, d.buildCacheKey(guildID, channelName))
}

func (d *DiscordNotifier) buildCacheKey(guildID, channelName string) string {
	return fmt.Sprintf("%s:%s", guildID, channelName)
}

func (d *DiscordNotifier) getChannelType(channelName string) string {
	if strings.Contains(channelName, "death") {
		return "death"
	}
	if strings.Contains(channelName, "level") {
		return "level"
	}
	return "other"
}
