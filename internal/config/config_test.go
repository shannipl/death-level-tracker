package config

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestLoad_Success(t *testing.T) {
	setEnv(map[string]string{
		"DISCORD_TOKEN":           strings.Repeat("x", 60),
		"DATABASE_URL":            "postgres://user:pass@localhost:5432/db",
		"TRACKER_INTERVAL":        "3m",
		"MIN_LEVEL_TRACK":         "600",
		"DISCORD_CHANNEL_DEATH":   "custom-death",
		"DISCORD_CHANNEL_LEVEL":   "custom-level",
		"WORKER_POOL_SIZE":        "20",
		"USE_TIBIACOM_FOR_LEVELS": "false",
		"DISCORD_GUILD_ID":        "123456",
	})
	defer clearEnv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEqual(t, "Token", strings.Repeat("x", 60), cfg.Token)
	assertEqual(t, "DatabaseURL", "postgres://user:pass@localhost:5432/db", cfg.DatabaseURL)
	assertEqual(t, "TrackerInterval", 3*time.Minute, cfg.TrackerInterval)
	assertEqual(t, "MinLevelTrack", 600, cfg.MinLevelTrack)
	assertEqual(t, "DiscordChannelDeath", "custom-death", cfg.DiscordChannelDeath)
	assertEqual(t, "DiscordChannelLevel", "custom-level", cfg.DiscordChannelLevel)
	assertEqual(t, "WorkerPoolSize", 20, cfg.WorkerPoolSize)
	assertEqual(t, "UseTibiaComForLevels", false, cfg.UseTibiaComForLevels)
	assertEqual(t, "DiscordGuildID", "123456", cfg.DiscordGuildID)
}

func TestLoad_Defaults(t *testing.T) {
	setEnv(map[string]string{
		"DISCORD_TOKEN": strings.Repeat("x", 60),
		"DATABASE_URL":  "postgres://localhost:5432/db",
	})
	defer clearEnv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEqual(t, "TrackerInterval", 5*time.Minute, cfg.TrackerInterval)
	assertEqual(t, "MinLevelTrack", 500, cfg.MinLevelTrack)
	assertEqual(t, "DiscordChannelDeath", "death-tracker", cfg.DiscordChannelDeath)
	assertEqual(t, "DiscordChannelLevel", "level-tracker", cfg.DiscordChannelLevel)
	assertEqual(t, "WorkerPoolSize", 10, cfg.WorkerPoolSize)
	assertEqual(t, "UseTibiaComForLevels", true, cfg.UseTibiaComForLevels)
}

func TestLoad_MissingToken(t *testing.T) {
	clearEnv()

	cfg, err := Load()
	if err == nil {
		t.Fatal("expected error for missing token")
	}
	if cfg != nil {
		t.Error("config should be nil on error")
	}
	assertContains(t, err.Error(), "DISCORD_TOKEN is not set")
}

func TestLoad_InvalidConfig(t *testing.T) {
	setEnv(map[string]string{
		"DISCORD_TOKEN":    strings.Repeat("x", 30),
		"DATABASE_URL":     "postgres://localhost:5432/db",
		"WORKER_POOL_SIZE": "200",
	})
	defer clearEnv()

	_, err := Load()
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestReadSecret(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir := secretsDir
	secretsDir = tmpDir + "/"
	defer func() { secretsDir = originalDir }()

	t.Run("reads existing secret", func(t *testing.T) {
		os.WriteFile(tmpDir+"/test_secret", []byte("  secret-value  \n"), 0600)
		result := readSecret("test_secret")
		assertEqual(t, "secret", "secret-value", result)
	})

	t.Run("returns empty for missing secret", func(t *testing.T) {
		result := readSecret("nonexistent")
		assertEqual(t, "secret", "", result)
	})
}

func TestEnvString(t *testing.T) {
	tests := []struct {
		name     string
		envVal   string
		fallback string
		expected string
	}{
		{"env set", "custom", "default", "custom"},
		{"env empty", "", "default", "default"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := "TEST_ENV_STRING"
			if tt.envVal != "" {
				os.Setenv(key, tt.envVal)
				defer os.Unsetenv(key)
			}
			result := envString(key, tt.fallback)
			assertEqual(t, "result", tt.expected, result)
		})
	}
}

func TestEnvInt(t *testing.T) {
	tests := []struct {
		name     string
		envVal   string
		fallback int
		expected int
	}{
		{"valid int", "42", 100, 42},
		{"invalid int", "abc", 100, 100},
		{"negative", "-10", 100, -10},
		{"zero", "0", 100, 0},
		{"empty", "", 100, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := "TEST_ENV_INT"
			if tt.envVal != "" {
				os.Setenv(key, tt.envVal)
				defer os.Unsetenv(key)
			}
			result := envInt(key, tt.fallback)
			assertEqual(t, "result", tt.expected, result)
		})
	}
}

func TestEnvDuration(t *testing.T) {
	tests := []struct {
		name     string
		envVal   string
		fallback time.Duration
		expected time.Duration
	}{
		{"valid duration", "10m", time.Minute, 10 * time.Minute},
		{"complex duration", "1h30m", time.Minute, 90 * time.Minute},
		{"invalid duration", "invalid", time.Minute, time.Minute},
		{"empty", "", time.Minute, time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := "TEST_ENV_DURATION"
			if tt.envVal != "" {
				os.Setenv(key, tt.envVal)
				defer os.Unsetenv(key)
			}
			result := envDuration(key, tt.fallback)
			assertEqual(t, "result", tt.expected, result)
		})
	}
}

func TestEnvBool(t *testing.T) {
	tests := []struct {
		name     string
		envVal   string
		fallback bool
		expected bool
	}{
		{"true", "true", false, true},
		{"false", "false", true, false},
		{"1", "1", false, true},
		{"0", "0", true, false},
		{"invalid", "maybe", false, false},
		{"empty", "", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := "TEST_ENV_BOOL"
			if tt.envVal != "" {
				os.Setenv(key, tt.envVal)
				defer os.Unsetenv(key)
			}
			result := envBool(key, tt.fallback)
			assertEqual(t, "result", tt.expected, result)
		})
	}
}

func setEnv(vars map[string]string) {
	for k, v := range vars {
		os.Setenv(k, v)
	}
}

func clearEnv() {
	keys := []string{
		"DISCORD_TOKEN", "TRACKER_INTERVAL", "MIN_LEVEL_TRACK",
		"DISCORD_CHANNEL_DEATH", "DISCORD_CHANNEL_LEVEL",
		"WORKER_POOL_SIZE", "USE_TIBIACOM_FOR_LEVELS", "DISCORD_GUILD_ID",
	}
	for _, k := range keys {
		os.Unsetenv(k)
	}
}

func assertEqual[T comparable](t *testing.T, name string, expected, actual T) {
	t.Helper()
	if expected != actual {
		t.Errorf("%s: expected %v, got %v", name, expected, actual)
	}
}

func assertContains(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Errorf("expected %q to contain %q", s, substr)
	}
}
