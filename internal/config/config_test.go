package config

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestLoad_Success(t *testing.T) {
	os.Setenv("DISCORD_TOKEN", strings.Repeat("a", 59)+"test-token-123")
	os.Setenv("TRACKER_INTERVAL", "3m")
	os.Setenv("MIN_LEVEL_TRACK", "600")
	os.Setenv("DISCORD_CHANNEL_DEATH", "custom-death")
	os.Setenv("DISCORD_CHANNEL_LEVEL", "custom-level")
	os.Setenv("WORKER_POOL_SIZE", "20")
	defer clearEnv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if cfg.Token != strings.Repeat("a", 59)+"test-token-123" {
		t.Errorf("Expected Token='%s', got '%s'", strings.Repeat("a", 59)+"test-token-123", cfg.Token)
	}
	if cfg.TrackerInterval != 3*time.Minute {
		t.Errorf("Expected TrackerInterval=3m, got %v", cfg.TrackerInterval)
	}
	if cfg.MinLevelTrack != 600 {
		t.Errorf("Expected MinLevelTrack=600, got %d", cfg.MinLevelTrack)
	}
	if cfg.DiscordChannelDeath != "custom-death" {
		t.Errorf("Expected DiscordChannelDeath='custom-death', got '%s'", cfg.DiscordChannelDeath)
	}
	if cfg.DiscordChannelLevel != "custom-level" {
		t.Errorf("Expected DiscordChannelLevel='custom-level', got '%s'", cfg.DiscordChannelLevel)
	}
	if cfg.WorkerPoolSize != 20 {
		t.Errorf("Expected WorkerPoolSize=20, got %d", cfg.WorkerPoolSize)
	}
}

func TestLoad_WithDefaults(t *testing.T) {
	os.Setenv("DISCORD_TOKEN", strings.Repeat("b", 50)+"test-token")
	defer clearEnv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if cfg.TrackerInterval != 5*time.Minute {
		t.Errorf("Expected default TrackerInterval=5m, got %v", cfg.TrackerInterval)
	}
	if cfg.MinLevelTrack != 500 {
		t.Errorf("Expected default MinLevelTrack=500, got %d", cfg.MinLevelTrack)
	}
	if cfg.DiscordChannelDeath != "death-tracker" {
		t.Errorf("Expected default DiscordChannelDeath='death-tracker', got '%s'", cfg.DiscordChannelDeath)
	}
	if cfg.DiscordChannelLevel != "level-tracker" {
		t.Errorf("Expected default DiscordChannelLevel='level-tracker', got '%s'", cfg.DiscordChannelLevel)
	}
	if cfg.WorkerPoolSize != 10 {
		t.Errorf("Expected default WorkerPoolSize=10, got %d", cfg.WorkerPoolSize)
	}
}

func TestLoad_MissingToken(t *testing.T) {
	clearEnv()

	cfg, err := Load()
	if err == nil {
		t.Fatalf("expected error containing 'DISCORD_TOKEN is not set', got: %v", err)
	}
	if cfg != nil {
		t.Error("Expected nil config when error occurs")
	}
	expectedErr := "DISCORD_TOKEN is not set (via secret or env var)"
	if err.Error() != expectedErr {
		t.Errorf("Expected error message '%s', got '%s'", expectedErr, err.Error())
	}
}

func TestGetEnvString(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    string
		fallback string
		expected string
	}{
		{
			name:     "returns environment value when set",
			key:      "TEST_STRING",
			value:    "custom-value",
			fallback: "default-value",
			expected: "custom-value",
		},
		{
			name:     "returns fallback when env not set",
			key:      "MISSING_KEY",
			value:    "",
			fallback: "default-value",
			expected: "default-value",
		},
		{
			name:     "returns fallback when env is empty string",
			key:      "EMPTY_STRING",
			value:    "",
			fallback: "default-value",
			expected: "default-value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != "" {
				os.Setenv(tt.key, tt.value)
				defer os.Unsetenv(tt.key)
			}

			result := getEnvString(tt.key, tt.fallback)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestGetEnvInt(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    string
		fallback int
		expected int
	}{
		{
			name:     "returns parsed int when valid",
			key:      "TEST_INT",
			value:    "42",
			fallback: 100,
			expected: 42,
		},
		{
			name:     "returns fallback when env not set",
			key:      "MISSING_INT",
			value:    "",
			fallback: 100,
			expected: 100,
		},
		{
			name:     "returns fallback when value is invalid",
			key:      "INVALID_INT",
			value:    "not-a-number",
			fallback: 100,
			expected: 100,
		},
		{
			name:     "handles negative numbers",
			key:      "NEGATIVE_INT",
			value:    "-50",
			fallback: 100,
			expected: -50,
		},
		{
			name:     "handles zero",
			key:      "ZERO_INT",
			value:    "0",
			fallback: 100,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != "" {
				os.Setenv(tt.key, tt.value)
				defer os.Unsetenv(tt.key)
			}

			result := getEnvInt(tt.key, tt.fallback)
			if result != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestGetEnvDuration(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    string
		fallback time.Duration
		expected time.Duration
	}{
		{
			name:     "returns parsed duration when valid",
			key:      "TEST_DURATION",
			value:    "10m",
			fallback: 5 * time.Minute,
			expected: 10 * time.Minute,
		},
		{
			name:     "handles seconds",
			key:      "TEST_SECONDS",
			value:    "30s",
			fallback: 1 * time.Minute,
			expected: 30 * time.Second,
		},
		{
			name:     "handles hours",
			key:      "TEST_HOURS",
			value:    "2h",
			fallback: 1 * time.Hour,
			expected: 2 * time.Hour,
		},
		{
			name:     "handles complex duration",
			key:      "TEST_COMPLEX",
			value:    "1h30m45s",
			fallback: 1 * time.Minute,
			expected: 1*time.Hour + 30*time.Minute + 45*time.Second,
		},
		{
			name:     "returns fallback when env not set",
			key:      "MISSING_DURATION",
			value:    "",
			fallback: 5 * time.Minute,
			expected: 5 * time.Minute,
		},
		{
			name:     "returns fallback when value is invalid",
			key:      "INVALID_DURATION",
			value:    "invalid",
			fallback: 5 * time.Minute,
			expected: 5 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != "" {
				os.Setenv(tt.key, tt.value)
				defer os.Unsetenv(tt.key)
			}

			result := getEnvDuration(tt.key, tt.fallback)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func clearEnv() {
	os.Unsetenv("DISCORD_TOKEN")
	os.Unsetenv("TRACKER_INTERVAL")
	os.Unsetenv("MIN_LEVEL_TRACK")
	os.Unsetenv("DISCORD_CHANNEL_DEATH")
	os.Unsetenv("DISCORD_CHANNEL_LEVEL")
	os.Unsetenv("WORKER_POOL_SIZE")
}
