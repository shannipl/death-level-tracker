package commands

import (
	"errors"
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestRespond(t *testing.T) {
	t.Run("ephemeral message", func(t *testing.T) {
		session := &mockDiscordSession{}
		interaction := &discordgo.InteractionCreate{
			Interaction: &discordgo.Interaction{},
		}

		respond(session, interaction, "test message", true)

		if session.lastInteractionResponse == nil {
			t.Fatal("expected response to be sent")
		}
		if session.lastInteractionResponse.Data.Content != "test message" {
			t.Errorf("expected 'test message', got '%s'", session.lastInteractionResponse.Data.Content)
		}
		if session.lastInteractionResponse.Data.Flags != discordgo.MessageFlagsEphemeral {
			t.Error("expected ephemeral flag")
		}
	})

	t.Run("non-ephemeral message", func(t *testing.T) {
		session := &mockDiscordSession{}
		interaction := &discordgo.InteractionCreate{
			Interaction: &discordgo.Interaction{},
		}

		respond(session, interaction, "public message", false)

		if session.lastInteractionResponse.Data.Flags != 0 {
			t.Error("expected no flags for non-ephemeral message")
		}
	})
}

func TestRespondAutocomplete(t *testing.T) {
	t.Run("returns choices", func(t *testing.T) {
		session := &mockDiscordSession{}
		interaction := &discordgo.InteractionCreate{
			Interaction: &discordgo.Interaction{},
		}

		choices := []*discordgo.ApplicationCommandOptionChoice{
			{Name: "Option 1", Value: "opt1"},
			{Name: "Option 2", Value: "opt2"},
		}

		err := respondAutocomplete(session, interaction, choices)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if session.lastInteractionResponse.Type != discordgo.InteractionApplicationCommandAutocompleteResult {
			t.Error("expected autocomplete response type")
		}
		if len(session.lastInteractionResponse.Data.Choices) != 2 {
			t.Errorf("expected 2 choices, got %d", len(session.lastInteractionResponse.Data.Choices))
		}
	})

	t.Run("returns error from session", func(t *testing.T) {
		session := &mockDiscordSession{
			interactionRespondFunc: func(i *discordgo.Interaction, r *discordgo.InteractionResponse) error {
				return errors.New("discord error")
			},
		}
		interaction := &discordgo.InteractionCreate{
			Interaction: &discordgo.Interaction{},
		}

		err := respondAutocomplete(session, interaction, nil)

		if err == nil || err.Error() != "discord error" {
			t.Errorf("expected 'discord error', got %v", err)
		}
	})
}

func TestEnsureChannel(t *testing.T) {
	t.Run("finds existing channel", func(t *testing.T) {
		session := &mockDiscordSession{
			guildChannelsFunc: func(guildID string) ([]*discordgo.Channel, error) {
				return []*discordgo.Channel{
					{ID: "ch-123", Name: "target-channel", Type: discordgo.ChannelTypeGuildText},
				}, nil
			},
		}

		id, err := ensureChannel(session, "guild-1", "target-channel")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != "ch-123" {
			t.Errorf("expected 'ch-123', got '%s'", id)
		}
	})

	t.Run("creates channel if not found", func(t *testing.T) {
		var createdName string
		session := &mockDiscordSession{
			guildChannelsFunc: func(guildID string) ([]*discordgo.Channel, error) {
				return []*discordgo.Channel{}, nil
			},
			guildChannelCreateFunc: func(guildID, name string, ctype discordgo.ChannelType) (*discordgo.Channel, error) {
				createdName = name
				return &discordgo.Channel{ID: "new-ch", Name: name, Type: ctype}, nil
			},
		}

		id, err := ensureChannel(session, "guild-1", "new-channel")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != "new-ch" {
			t.Errorf("expected 'new-ch', got '%s'", id)
		}
		if createdName != "new-channel" {
			t.Errorf("expected to create 'new-channel', got '%s'", createdName)
		}
	})

	t.Run("ignores non-text channels with same name", func(t *testing.T) {
		var created bool
		session := &mockDiscordSession{
			guildChannelsFunc: func(guildID string) ([]*discordgo.Channel, error) {
				return []*discordgo.Channel{
					{ID: "voice-ch", Name: "target", Type: discordgo.ChannelTypeGuildVoice},
				}, nil
			},
			guildChannelCreateFunc: func(guildID, name string, ctype discordgo.ChannelType) (*discordgo.Channel, error) {
				created = true
				return &discordgo.Channel{ID: "new-text", Name: name, Type: ctype}, nil
			},
		}

		id, _ := ensureChannel(session, "guild-1", "target")

		if !created {
			t.Error("expected channel to be created")
		}
		if id != "new-text" {
			t.Errorf("expected 'new-text', got '%s'", id)
		}
	})

	t.Run("returns error on GuildChannels failure", func(t *testing.T) {
		session := &mockDiscordSession{
			guildChannelsFunc: func(guildID string) ([]*discordgo.Channel, error) {
				return nil, errors.New("api error")
			},
		}

		_, err := ensureChannel(session, "guild-1", "channel")

		if err == nil || err.Error() != "api error" {
			t.Errorf("expected 'api error', got %v", err)
		}
	})

	t.Run("returns error on GuildChannelCreate failure", func(t *testing.T) {
		session := &mockDiscordSession{
			guildChannelsFunc: func(guildID string) ([]*discordgo.Channel, error) {
				return []*discordgo.Channel{}, nil
			},
			guildChannelCreateFunc: func(guildID, name string, ctype discordgo.ChannelType) (*discordgo.Channel, error) {
				return nil, errors.New("permission denied")
			},
		}

		_, err := ensureChannel(session, "guild-1", "channel")

		if err == nil || err.Error() != "permission denied" {
			t.Errorf("expected 'permission denied', got %v", err)
		}
	})
}

func TestGetStringOption(t *testing.T) {
	opts := []*discordgo.ApplicationCommandInteractionDataOption{
		{Name: "world", Type: discordgo.ApplicationCommandOptionString, Value: "Antica"},
		{Name: "guild", Type: discordgo.ApplicationCommandOptionString, Value: "Red Rose"},
	}

	t.Run("finds existing option", func(t *testing.T) {
		if result := getStringOption(opts, "world"); result != "Antica" {
			t.Errorf("expected 'Antica', got '%s'", result)
		}
	})

	t.Run("returns empty for missing option", func(t *testing.T) {
		if result := getStringOption(opts, "missing"); result != "" {
			t.Errorf("expected empty, got '%s'", result)
		}
	})

	t.Run("handles nil slice", func(t *testing.T) {
		if result := getStringOption(nil, "world"); result != "" {
			t.Errorf("expected empty, got '%s'", result)
		}
	})

	t.Run("handles empty slice", func(t *testing.T) {
		if result := getStringOption([]*discordgo.ApplicationCommandInteractionDataOption{}, "world"); result != "" {
			t.Errorf("expected empty, got '%s'", result)
		}
	})
}

func TestGetFocusedOption(t *testing.T) {
	t.Run("finds focused option", func(t *testing.T) {
		opts := []*discordgo.ApplicationCommandInteractionDataOption{
			{Name: "first", Type: discordgo.ApplicationCommandOptionString, Value: "val1", Focused: false},
			{Name: "second", Type: discordgo.ApplicationCommandOptionString, Value: "val2", Focused: true},
			{Name: "third", Type: discordgo.ApplicationCommandOptionString, Value: "val3", Focused: false},
		}

		if result := getFocusedOption(opts); result != "val2" {
			t.Errorf("expected 'val2', got '%s'", result)
		}
	})

	t.Run("returns empty when no focused option", func(t *testing.T) {
		opts := []*discordgo.ApplicationCommandInteractionDataOption{
			{Name: "first", Type: discordgo.ApplicationCommandOptionString, Value: "val1", Focused: false},
		}

		if result := getFocusedOption(opts); result != "" {
			t.Errorf("expected empty, got '%s'", result)
		}
	})

	t.Run("handles empty slice", func(t *testing.T) {
		if result := getFocusedOption([]*discordgo.ApplicationCommandInteractionDataOption{}); result != "" {
			t.Errorf("expected empty, got '%s'", result)
		}
	})

	t.Run("handles nil slice", func(t *testing.T) {
		if result := getFocusedOption(nil); result != "" {
			t.Errorf("expected empty, got '%s'", result)
		}
	})
}
