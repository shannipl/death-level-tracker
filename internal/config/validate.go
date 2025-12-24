package config

import (
	"errors"
	"fmt"
	"time"
)

const (
	minTokenLength     = 50
	minTrackerInterval = 1 * time.Minute
	maxTrackerInterval = 24 * time.Hour
	minLevelTrack      = 1
	minWorkerPoolSize  = 1
	maxWorkerPoolSize  = 100
	maxChannelNameLen  = 100
)

func (c *Config) Validate() error {
	var errs []error

	if err := c.validateToken(); err != nil {
		errs = append(errs, err)
	}
	if err := c.validateTrackerInterval(); err != nil {
		errs = append(errs, err)
	}
	if err := c.validateMinLevelTrack(); err != nil {
		errs = append(errs, err)
	}
	if err := c.validateWorkerPoolSize(); err != nil {
		errs = append(errs, err)
	}
	if err := c.validateChannelNames(); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("configuration validation failed:\n  %w", errors.Join(errs...))
	}
	return nil
}

func (c *Config) validateToken() error {
	if c.Token == "" {
		return fmt.Errorf("DISCORD_TOKEN is required")
	}
	if len(c.Token) < minTokenLength {
		return fmt.Errorf("DISCORD_TOKEN too short: %d chars, expected %d+", len(c.Token), minTokenLength)
	}
	return nil
}

func (c *Config) validateTrackerInterval() error {
	if c.TrackerInterval < minTrackerInterval {
		return fmt.Errorf("TRACKER_INTERVAL must be at least %v, got %v", minTrackerInterval, c.TrackerInterval)
	}
	if c.TrackerInterval > maxTrackerInterval {
		return fmt.Errorf("TRACKER_INTERVAL must be at most %v, got %v", maxTrackerInterval, c.TrackerInterval)
	}
	return nil
}

func (c *Config) validateMinLevelTrack() error {
	if c.MinLevelTrack < minLevelTrack {
		return fmt.Errorf("MIN_LEVEL_TRACK must be at least %d, got %d", minLevelTrack, c.MinLevelTrack)
	}
	return nil
}

func (c *Config) validateWorkerPoolSize() error {
	if c.WorkerPoolSize < minWorkerPoolSize {
		return fmt.Errorf("WORKER_POOL_SIZE must be at least %d, got %d", minWorkerPoolSize, c.WorkerPoolSize)
	}
	if c.WorkerPoolSize > maxWorkerPoolSize {
		return fmt.Errorf("WORKER_POOL_SIZE must be at most %d, got %d", maxWorkerPoolSize, c.WorkerPoolSize)
	}
	return nil
}

func (c *Config) validateChannelNames() error {
	var errs []error

	if err := validateChannel("DISCORD_CHANNEL_DEATH", c.DiscordChannelDeath); err != nil {
		errs = append(errs, err)
	}
	if err := validateChannel("DISCORD_CHANNEL_LEVEL", c.DiscordChannelLevel); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func validateChannel(name, value string) error {
	if value == "" {
		return fmt.Errorf("%s cannot be empty", name)
	}
	if len(value) > maxChannelNameLen {
		return fmt.Errorf("%s exceeds %d characters: %d", name, maxChannelNameLen, len(value))
	}
	return nil
}
