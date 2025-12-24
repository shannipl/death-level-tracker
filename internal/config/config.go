package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Token                string
	TrackerInterval      time.Duration
	MinLevelTrack        int
	DiscordChannelDeath  string
	DiscordChannelLevel  string
	WorkerPoolSize       int
	UseTibiaComForLevels bool
	DiscordGuildID       string
	DatabaseURL          string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	token := readSecret("discord_token")
	if token == "" {
		token = os.Getenv("DISCORD_TOKEN")
	}
	if token == "" {
		return nil, fmt.Errorf("DISCORD_TOKEN is not set (via secret or env var)")
	}

	dbURL := readSecret("database_url")
	if dbURL == "" {
		dbURL = os.Getenv("DATABASE_URL")
	}
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is not set (via secret or env var)")
	}

	cfg := &Config{
		Token:                token,
		TrackerInterval:      envDuration("TRACKER_INTERVAL", 5*time.Minute),
		MinLevelTrack:        envInt("MIN_LEVEL_TRACK", 500),
		DiscordChannelDeath:  envString("DISCORD_CHANNEL_DEATH", "death-tracker"),
		DiscordChannelLevel:  envString("DISCORD_CHANNEL_LEVEL", "level-tracker"),
		WorkerPoolSize:       envInt("WORKER_POOL_SIZE", 10),
		UseTibiaComForLevels: envBool("USE_TIBIACOM_FOR_LEVELS", true),
		DiscordGuildID:       envString("DISCORD_GUILD_ID", ""),
		DatabaseURL:          dbURL,
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

var secretsDir = "/run/secrets/"

func readSecret(name string) string {
	data, err := os.ReadFile(secretsDir + name)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func envString(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}
