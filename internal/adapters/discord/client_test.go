package discord

import (
	"strings"
	"testing"
	"time"

	"death-level-tracker/internal/config"
	"death-level-tracker/internal/core/domain"

	"github.com/bwmarrin/discordgo"
)

type mockDiscordSession struct {
	guildChannelsFunc      func(guildID string, options ...discordgo.RequestOption) ([]*discordgo.Channel, error)
	channelMessageSendFunc func(channelID, content string, options ...discordgo.RequestOption) (*discordgo.Message, error)
}

func (m *mockDiscordSession) GuildChannels(guildID string, options ...discordgo.RequestOption) ([]*discordgo.Channel, error) {
	if m.guildChannelsFunc != nil {
		return m.guildChannelsFunc(guildID, options...)
	}
	return nil, nil
}

func (m *mockDiscordSession) ChannelMessageSend(channelID, content string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
	if m.channelMessageSendFunc != nil {
		return m.channelMessageSendFunc(channelID, content, options...)
	}
	return &discordgo.Message{}, nil
}

var testConfig = &config.Config{
	DiscordChannelDeath: "death-tracker",
	DiscordChannelLevel: "level-tracker",
}

func TestNewAdapter(t *testing.T) {
	session := &mockDiscordSession{}
	adapter := NewAdapter(session, testConfig)

	if adapter == nil {
		t.Fatal("Expected non-nil adapter")
	}

	if adapter.session == nil {
		t.Error("Expected session to be set")
	}

	if adapter.cache == nil {
		t.Error("Expected channel cache to be initialized")
	}
}

func TestAdapter_SendLevelUpNotification(t *testing.T) {
	var sentChannelID, sentContent string

	session := &mockDiscordSession{
		guildChannelsFunc: func(guildID string, options ...discordgo.RequestOption) ([]*discordgo.Channel, error) {
			return []*discordgo.Channel{
				{ID: "channel-level-123", Name: "level-tracker", Type: discordgo.ChannelTypeGuildText},
			}, nil
		},
		channelMessageSendFunc: func(channelID, content string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
			sentChannelID = channelID
			sentContent = content
			return &discordgo.Message{ID: "msg-123"}, nil
		},
	}

	adapter := NewAdapter(session, testConfig)
	levelUp := domain.LevelUp{
		PlayerName: "Hero",
		OldLevel:   100,
		NewLevel:   101,
	}

	err := adapter.SendLevelUpNotification("guild-1", levelUp)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if sentChannelID != "channel-level-123" {
		t.Errorf("Expected channel ID 'channel-level-123', got '%s'", sentChannelID)
	}

	if !strings.Contains(sentContent, "Hero") || !strings.Contains(sentContent, "100") || !strings.Contains(sentContent, "101") {
		t.Errorf("Expected content to contain info, got '%s'", sentContent)
	}
}

func TestAdapter_SendDeathNotification(t *testing.T) {
	var sentChannelID, sentContent string

	session := &mockDiscordSession{
		guildChannelsFunc: func(guildID string, options ...discordgo.RequestOption) ([]*discordgo.Channel, error) {
			return []*discordgo.Channel{
				{ID: "channel-death-123", Name: "death-tracker", Type: discordgo.ChannelTypeGuildText},
			}, nil
		},
		channelMessageSendFunc: func(channelID, content string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
			sentChannelID = channelID
			sentContent = content
			return &discordgo.Message{ID: "msg-123"}, nil
		},
	}

	adapter := NewAdapter(session, testConfig)
	kill := domain.Kill{
		Time:   time.Now(),
		Reason: "Dragon",
	}

	err := adapter.SendDeathNotification("guild-1", "Hero", kill)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if sentChannelID != "channel-death-123" {
		t.Errorf("Expected channel ID 'channel-death-123', got '%s'", sentChannelID)
	}

	if !strings.Contains(sentContent, "Hero") || !strings.Contains(sentContent, "Dragon") {
		t.Errorf("Expected content to contain info, got '%s'", sentContent)
	}
}

func TestAdapter_SendGenericMessage_CacheRequests(t *testing.T) {
	guildChannelsCalled := 0

	session := &mockDiscordSession{
		guildChannelsFunc: func(guildID string, options ...discordgo.RequestOption) ([]*discordgo.Channel, error) {
			guildChannelsCalled++
			return []*discordgo.Channel{
				{ID: "channel-gen", Name: "general", Type: discordgo.ChannelTypeGuildText},
			}, nil
		},
		channelMessageSendFunc: func(channelID, content string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
			return &discordgo.Message{}, nil
		},
	}

	adapter := NewAdapter(session, testConfig)

	// First call - should fetch from API
	adapter.SendGenericMessage("guild-1", "general", "Message 1")
	if guildChannelsCalled != 1 {
		t.Errorf("Expected GuildChannels to be called once, got %d", guildChannelsCalled)
	}

	// Second call - should use cache
	adapter.SendGenericMessage("guild-1", "general", "Message 2")
	if guildChannelsCalled != 1 {
		t.Errorf("Expected GuildChannels to still be 1 (cached), got %d", guildChannelsCalled)
	}
}

func TestChannelCache_Invalidate(t *testing.T) {
	cache := newChannelCache()

	cache.Set("guild-1", "channel-a", "id-1")
	cache.Set("guild-1", "channel-b", "id-2")

	if _, ok := cache.Get("guild-1", "channel-a"); !ok {
		t.Fatal("expected cache hit before invalidate")
	}

	cache.Invalidate("guild-1", "channel-a")

	if _, ok := cache.Get("guild-1", "channel-a"); ok {
		t.Error("expected cache miss after invalidate")
	}
	if _, ok := cache.Get("guild-1", "channel-b"); !ok {
		t.Error("expected cache hit for non-invalidated key")
	}
}
