package discord

import (
	"testing"

	"death-level-tracker/internal/config"

	"github.com/bwmarrin/discordgo"
)

func TestNewSession_Success(t *testing.T) {
	cfg := &config.Config{
		Token: "MTk.test.token",
	}

	session, err := NewSession(cfg)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expectedIntents := discordgo.IntentsGuilds | discordgo.IntentsGuildMessages
	if session.Identify.Intents != expectedIntents {
		t.Errorf("Expected intents %d, got %d", expectedIntents, session.Identify.Intents)
	}
}

func TestNewSession_EmptyToken(t *testing.T) {
	cfg := &config.Config{
		Token: "",
	}

	session, err := NewSession(cfg)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if session == nil {
		t.Error("Expected session to be created")
	}
}

func TestNewSession_VariousTokenFormats(t *testing.T) {
	testCases := []struct {
		name  string
		token string
	}{
		{"standard format", "MTk.test.token"},
		{"short token", "test"},
		{"empty", ""},
		{"with special chars", "test-token_123"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.Config{
				Token: tc.token,
			}

			session, err := NewSession(cfg)
			if err != nil {
				t.Fatalf("Unexpected error creating session: %v", err)
			}

			if session.Identify.Intents == 0 {
				t.Error("Expected intents to be set")
			}
		})
	}
}

func TestNewSession_IntentsConfiguration(t *testing.T) {
	cfg := &config.Config{
		Token: "test-token",
	}

	session, err := NewSession(cfg)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expectedIntents := discordgo.IntentsGuilds | discordgo.IntentsGuildMessages
	if session.Identify.Intents != expectedIntents {
		t.Errorf("Expected intents to be %d (Guilds|GuildMessages), got %d",
			expectedIntents, session.Identify.Intents)
	}
}

func TestNewSession_TokenPrefixing(t *testing.T) {
	cfg := &config.Config{
		Token: "my-token-123",
	}

	session, err := NewSession(cfg)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expectedToken := "Bot my-token-123"
	if session.Token != expectedToken {
		t.Errorf("Expected token '%s', got '%s'", expectedToken, session.Token)
	}
}
