package main

import (
	"testing"

	"death-level-tracker/internal/config"

	"github.com/bwmarrin/discordgo"
)

func TestNewDiscordSession_Success(t *testing.T) {
	// Test with a valid token format (even if not real)
	// Discord tokens are typically in format: prefix.payload.signature
	cfg := &config.Config{
		Token: "MTk.test.token",
	}

	session, err := NewDiscordSession(cfg)

	// Note: This will still fail authentication with Discord,
	// but it tests that the session is created with proper intents
	// Real authentication requires a valid Discord bot token
	if err != nil {
		// Expected to fail with invalid token, but session should be created
		t.Logf("Expected error with test token: %v", err)
	}

	if session == nil {
		t.Error("Expected session to be created even with invalid token")
	}

	// Verify intents are set correctly if session was created
	if session != nil {
		expectedIntents := discordgo.Intent(1 | 512) // IntentsGuilds (1) | IntentsGuildMessages (512)
		if session.Identify.Intents != expectedIntents {
			t.Errorf("Expected intents %d, got %d", expectedIntents, session.Identify.Intents)
		}
	}
}

func TestNewDiscordSession_EmptyToken(t *testing.T) {
	cfg := &config.Config{
		Token: "",
	}

	session, err := NewDiscordSession(cfg)

	// Should create session but with empty token
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if session == nil {
		t.Error("Expected session to be created")
	}
}

func TestNewDiscordSession_VariousTokenFormats(t *testing.T) {
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

			session, err := NewDiscordSession(cfg)

			// Session creation should succeed
			// (authentication will fail later when connecting)
			if err != nil {
				t.Errorf("Unexpected error creating session: %v", err)
			}

			if session == nil {
				t.Error("Expected session to be created")
			}

			// Verify intents are always set
			if session != nil {
				if session.Identify.Intents == 0 {
					t.Error("Expected intents to be set")
				}
			}
		})
	}
}

func TestNewDiscordSession_IntentsConfiguration(t *testing.T) {
	cfg := &config.Config{
		Token: "test-token",
	}

	session, err := NewDiscordSession(cfg)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if session == nil {
		t.Fatal("Expected session to be created")
	}

	// Verify both required intents are set
	// IntentsGuilds (1) and IntentsGuildMessages (512)
	expectedIntents := discordgo.Intent(1 | 512)

	if session.Identify.Intents != expectedIntents {
		t.Errorf("Expected intents to be %d (Guilds|GuildMessages), got %d",
			expectedIntents, session.Identify.Intents)
	}
}

func TestNewDiscordSession_TokenPrefixing(t *testing.T) {
	// Verify that "Bot " prefix is added
	cfg := &config.Config{
		Token: "my-token-123",
	}

	session, err := NewDiscordSession(cfg)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if session == nil {
		t.Fatal("Expected session to be created")
	}

	// The token should have "Bot " prefix
	expectedToken := "Bot my-token-123"
	if session.Token != expectedToken {
		t.Errorf("Expected token '%s', got '%s'", expectedToken, session.Token)
	}
}
