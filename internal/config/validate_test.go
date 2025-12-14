package config

import (
	"strings"
	"testing"
	"time"
)

func TestConfig_Validate_ValidConfig(t *testing.T) {
	cfg := &Config{
		Token:               strings.Repeat("a", 50),
		TrackerInterval:     5 * time.Minute,
		MinLevelTrack:       500,
		WorkerPoolSize:      10,
		DiscordChannelDeath: "death-tracker",
		DiscordChannelLevel: "level-tracker",
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("Valid config should not produce error: %v", err)
	}
}

func TestConfig_Validate_Token(t *testing.T) {
	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{"valid token", strings.Repeat("a", 50), false},
		{"too short", strings.Repeat("a", 49), true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Token:               tt.token,
				TrackerInterval:     5 * time.Minute,
				MinLevelTrack:       500,
				WorkerPoolSize:      10,
				DiscordChannelDeath: "death-tracker",
				DiscordChannelLevel: "level-tracker",
			}

			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Token validation error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_Validate_TrackerInterval(t *testing.T) {
	tests := []struct {
		name     string
		interval time.Duration
		wantErr  bool
	}{
		{"minimum valid", 1 * time.Minute, false},
		{"below minimum", 59 * time.Second, true},
		{"normal", 5 * time.Minute, false},
		{"maximum valid", 24 * time.Hour, false},
		{"too large", 25 * time.Hour, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Token:               strings.Repeat("a", 50),
				TrackerInterval:     tt.interval,
				MinLevelTrack:       500,
				WorkerPoolSize:      10,
				DiscordChannelDeath: "death-tracker",
				DiscordChannelLevel: "level-tracker",
			}

			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("TrackerInterval validation error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_Validate_MinLevelTrack(t *testing.T) {
	tests := []struct {
		name    string
		level   int
		wantErr bool
	}{
		{"minimum valid", 1, false},
		{"too small", 0, true},
		{"negative", -1, true},
		{"normal", 500, false},
		{"high level", 2000, false},
		{"very high level", 5000, false}, // No upper limit now
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Token:               strings.Repeat("a", 50),
				TrackerInterval:     5 * time.Minute,
				MinLevelTrack:       tt.level,
				WorkerPoolSize:      10,
				DiscordChannelDeath: "death-tracker",
				DiscordChannelLevel: "level-tracker",
			}

			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("MinLevelTrack validation error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_Validate_WorkerPoolSize(t *testing.T) {
	tests := []struct {
		name    string
		size    int
		wantErr bool
	}{
		{"minimum valid", 1, false},
		{"too small", 0, true},
		{"negative", -1, true},
		{"normal", 10, false},
		{"maximum valid", 100, false},
		{"too large", 101, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Token:               strings.Repeat("a", 50),
				TrackerInterval:     5 * time.Minute,
				MinLevelTrack:       500,
				WorkerPoolSize:      tt.size,
				DiscordChannelDeath: "death-tracker",
				DiscordChannelLevel: "level-tracker",
			}

			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("WorkerPoolSize validation error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_Validate_ChannelNames(t *testing.T) {
	tests := []struct {
		name      string
		deathChan string
		levelChan string
		wantErr   bool
	}{
		{"both valid", "death-tracker", "level-tracker", false},
		{"empty death channel", "", "level-tracker", true},
		{"empty level channel", "death-tracker", "", true},
		{"both empty", "", "", true},
		{"death too long", strings.Repeat("a", 101), "level-tracker", true},
		{"level too long", "death-tracker", strings.Repeat("a", 101), true},
		{"max length valid", strings.Repeat("a", 100), strings.Repeat("b", 100), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Token:               strings.Repeat("a", 50),
				TrackerInterval:     5 * time.Minute,
				MinLevelTrack:       500,
				WorkerPoolSize:      10,
				DiscordChannelDeath: tt.deathChan,
				DiscordChannelLevel: tt.levelChan,
			}

			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Channel validation error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_Validate_MultipleErrors(t *testing.T) {
	// Config with multiple invalid fields
	cfg := &Config{
		Token:               "",               // invalid: empty
		TrackerInterval:     30 * time.Second, // invalid: too small
		MinLevelTrack:       0,                // invalid: too small
		WorkerPoolSize:      -1,               // invalid: negative
		DiscordChannelDeath: "",               // invalid: empty
		DiscordChannelLevel: "",               // invalid: empty
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Expected validation error for invalid config")
	}

	// Should contain multiple error messages
	errMsg := err.Error()
	expectedSubstrings := []string{
		"DISCORD_TOKEN",
		"TRACKER_INTERVAL",
		"MIN_LEVEL_TRACK",
		"WORKER_POOL_SIZE",
		"DISCORD_CHANNEL_DEATH",
		"DISCORD_CHANNEL_LEVEL",
	}

	for _, substr := range expectedSubstrings {
		if !strings.Contains(errMsg, substr) {
			t.Errorf("Error message should contain %q, got: %s", substr, errMsg)
		}
	}
}
