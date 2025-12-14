package tracker

import (
	"errors"
	"testing"

	"github.com/bwmarrin/discordgo"
)

// Mock Discord session for testing
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

func TestNewDiscordNotifier(t *testing.T) {
	session := &mockDiscordSession{}
	notifier := NewDiscordNotifier(session)

	if notifier == nil {
		t.Fatal("Expected non-nil notifier")
	}

	if notifier.session == nil {
		t.Error("Expected session to be set")
	}

	if notifier.channelCache == nil {
		t.Error("Expected channel cache to be initialized")
	}
}

func TestDiscordNotifier_Send_Success(t *testing.T) {
	var sentChannelID, sentContent string

	session := &mockDiscordSession{
		guildChannelsFunc: func(guildID string, options ...discordgo.RequestOption) ([]*discordgo.Channel, error) {
			return []*discordgo.Channel{
				{ID: "channel-123", Name: "death-tracker", Type: discordgo.ChannelTypeGuildText},
			}, nil
		},
		channelMessageSendFunc: func(channelID, content string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
			sentChannelID = channelID
			sentContent = content
			return &discordgo.Message{ID: "msg-123"}, nil
		},
	}

	notifier := NewDiscordNotifier(session)
	notifier.Send("guild-1", "death-tracker", "Test death message")

	if sentChannelID != "channel-123" {
		t.Errorf("Expected channel ID 'channel-123', got '%s'", sentChannelID)
	}

	if sentContent != "Test death message" {
		t.Errorf("Expected content 'Test death message', got '%s'", sentContent)
	}
}

func TestDiscordNotifier_Send_CachedChannel(t *testing.T) {
	guildChannelsCalled := 0

	session := &mockDiscordSession{
		guildChannelsFunc: func(guildID string, options ...discordgo.RequestOption) ([]*discordgo.Channel, error) {
			guildChannelsCalled++
			return []*discordgo.Channel{
				{ID: "channel-123", Name: "level-tracker", Type: discordgo.ChannelTypeGuildText},
			}, nil
		},
		channelMessageSendFunc: func(channelID, content string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
			return &discordgo.Message{}, nil
		},
	}

	notifier := NewDiscordNotifier(session)

	// First call - should fetch from API
	notifier.Send("guild-1", "level-tracker", "Message 1")
	if guildChannelsCalled != 1 {
		t.Errorf("Expected GuildChannels to be called once, got %d", guildChannelsCalled)
	}

	// Second call - should use cache
	notifier.Send("guild-1", "level-tracker", "Message 2")
	if guildChannelsCalled != 1 {
		t.Errorf("Expected GuildChannels to still be 1 (cached), got %d", guildChannelsCalled)
	}
}

func TestDiscordNotifier_Send_GetChannelError(t *testing.T) {
	session := &mockDiscordSession{
		guildChannelsFunc: func(guildID string, options ...discordgo.RequestOption) ([]*discordgo.Channel, error) {
			return nil, errors.New("API error")
		},
	}

	notifier := NewDiscordNotifier(session)

	// Should not panic, just log error
	notifier.Send("guild-1", "death-tracker", "Test message")
}

func TestDiscordNotifier_Send_SendMessageError_InvalidatesCache(t *testing.T) {
	guildChannelsCalled := 0

	session := &mockDiscordSession{
		guildChannelsFunc: func(guildID string, options ...discordgo.RequestOption) ([]*discordgo.Channel, error) {
			guildChannelsCalled++
			return []*discordgo.Channel{
				{ID: "channel-123", Name: "test-channel", Type: discordgo.ChannelTypeGuildText},
			}, nil
		},
		channelMessageSendFunc: func(channelID, content string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
			if content == "fail" {
				return nil, errors.New("send failed")
			}
			return &discordgo.Message{}, nil
		},
	}

	notifier := NewDiscordNotifier(session)

	// First send succeeds
	notifier.Send("guild-1", "test-channel", "success")
	if guildChannelsCalled != 1 {
		t.Fatalf("Expected 1 GuildChannels call, got %d", guildChannelsCalled)
	}

	// Second send fails - should invalidate cache
	notifier.Send("guild-1", "test-channel", "fail")

	// Third send should re-fetch channel (cache was invalidated)
	notifier.Send("guild-1", "test-channel", "success again")
	if guildChannelsCalled != 2 {
		t.Errorf("Expected 2 GuildChannels calls (cache invalidated), got %d", guildChannelsCalled)
	}
}

func TestDiscordNotifier_FetchChannelID_Success(t *testing.T) {
	session := &mockDiscordSession{
		guildChannelsFunc: func(guildID string, options ...discordgo.RequestOption) ([]*discordgo.Channel, error) {
			return []*discordgo.Channel{
				{ID: "voice-1", Name: "voice", Type: discordgo.ChannelTypeGuildVoice},
				{ID: "text-1", Name: "general", Type: discordgo.ChannelTypeGuildText},
				{ID: "text-2", Name: "death-tracker", Type: discordgo.ChannelTypeGuildText},
			}, nil
		},
	}

	notifier := NewDiscordNotifier(session)
	channelID, err := notifier.fetchChannelID("guild-1", "death-tracker")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if channelID != "text-2" {
		t.Errorf("Expected channel ID 'text-2', got '%s'", channelID)
	}
}

func TestDiscordNotifier_FetchChannelID_NotFound(t *testing.T) {
	session := &mockDiscordSession{
		guildChannelsFunc: func(guildID string, options ...discordgo.RequestOption) ([]*discordgo.Channel, error) {
			return []*discordgo.Channel{
				{ID: "text-1", Name: "general", Type: discordgo.ChannelTypeGuildText},
			}, nil
		},
	}

	notifier := NewDiscordNotifier(session)
	_, err := notifier.fetchChannelID("guild-1", "nonexistent")

	if err == nil {
		t.Error("Expected error for nonexistent channel")
	}

	if err.Error() != "channel nonexistent not found" {
		t.Errorf("Expected 'channel nonexistent not found' error, got: %v", err)
	}
}

func TestDiscordNotifier_FetchChannelID_APIError(t *testing.T) {
	session := &mockDiscordSession{
		guildChannelsFunc: func(guildID string, options ...discordgo.RequestOption) ([]*discordgo.Channel, error) {
			return nil, errors.New("Discord API unavailable")
		},
	}

	notifier := NewDiscordNotifier(session)
	_, err := notifier.fetchChannelID("guild-1", "test")

	if err == nil {
		t.Error("Expected error from API")
	}
}

func TestDiscordNotifier_FetchChannelID_IgnoresNonTextChannels(t *testing.T) {
	session := &mockDiscordSession{
		guildChannelsFunc: func(guildID string, options ...discordgo.RequestOption) ([]*discordgo.Channel, error) {
			return []*discordgo.Channel{
				{ID: "voice-1", Name: "death-tracker", Type: discordgo.ChannelTypeGuildVoice},
				{ID: "text-1", Name: "death-tracker", Type: discordgo.ChannelTypeGuildText},
			}, nil
		},
	}

	notifier := NewDiscordNotifier(session)
	channelID, err := notifier.fetchChannelID("guild-1", "death-tracker")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should find the text channel, not the voice channel
	if channelID != "text-1" {
		t.Errorf("Expected text channel 'text-1', got '%s'", channelID)
	}
}

func TestDiscordNotifier_BuildCacheKey(t *testing.T) {
	notifier := NewDiscordNotifier(&mockDiscordSession{})

	testCases := []struct {
		guildID     string
		channelName string
		expected    string
	}{
		{"guild-1", "death-tracker", "guild-1:death-tracker"},
		{"guild-2", "level-tracker", "guild-2:level-tracker"},
		{"", "channel", ":channel"},
	}

	for _, tc := range testCases {
		result := notifier.buildCacheKey(tc.guildID, tc.channelName)
		if result != tc.expected {
			t.Errorf("For %s/%s expected '%s', got '%s'", tc.guildID, tc.channelName, tc.expected, result)
		}
	}
}

func TestDiscordNotifier_ConcurrentAccess(t *testing.T) {
	session := &mockDiscordSession{
		guildChannelsFunc: func(guildID string, options ...discordgo.RequestOption) ([]*discordgo.Channel, error) {
			return []*discordgo.Channel{
				{ID: "channel-1", Name: "test", Type: discordgo.ChannelTypeGuildText},
			}, nil
		},
		channelMessageSendFunc: func(channelID, content string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
			return &discordgo.Message{}, nil
		},
	}

	notifier := NewDiscordNotifier(session)

	// Test concurrent sends to ensure no race conditions
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			notifier.Send("guild-1", "test", "concurrent message")
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
