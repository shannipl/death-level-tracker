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
	Token               string
	TrackerInterval     time.Duration
	MinLevelTrack       int
	DiscordChannelDeath string
	DiscordChannelLevel string
	WorkerPoolSize      int
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	// Try Docker secret first, then environment variable
	token := getSecret("discord_token")
	if token == "" {
		token = os.Getenv("DISCORD_TOKEN")
	}
	if token == "" {
		return nil, fmt.Errorf("DISCORD_TOKEN is not set (via secret or env var)")
	}

	cfg := &Config{
		Token:               token,
		TrackerInterval:     getEnvDuration("TRACKER_INTERVAL", 5*time.Minute),
		MinLevelTrack:       getEnvInt("MIN_LEVEL_TRACK", 500),
		DiscordChannelDeath: getEnvString("DISCORD_CHANNEL_DEATH", "death-level-tracker"),
		DiscordChannelLevel: getEnvString("DISCORD_CHANNEL_LEVEL", "level-tracker"),
		WorkerPoolSize:      getEnvInt("WORKER_POOL_SIZE", 10),
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func getSecret(name string) string {
	path := "/run/secrets/" + name
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func getEnvString(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
