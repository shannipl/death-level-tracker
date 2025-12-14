package config

import (
	"errors"
	"fmt"
	"time"
)

// Validation constants define acceptable bounds for configuration values
const (
	// Token validation
	minTokenLength = 50 // Discord tokens are typically 50+ characters

	// TrackerInterval validation
	minTrackerInterval = 1 * time.Minute // Minimum to avoid excessive API calls
	maxTrackerInterval = 24 * time.Hour  // Maximum reasonable interval

	// MinLevelTrack validation
	minLevelTrack = 1 // Minimum valid level

	// WorkerPoolSize validation
	minWorkerPoolSize = 1   // At least one worker needed
	maxWorkerPoolSize = 100 // Prevent resource exhaustion

	// Channel name validation
	minChannelNameLength = 1   // Cannot be empty
	maxChannelNameLength = 100 // Discord limit
)

// Validate checks if the configuration values are valid and within acceptable ranges.
// It returns all validation errors at once using errors.Join for better user experience.
//
// All configuration fields are validated:
//   - Token: Must be at least 50 characters (Discord token format)
//   - TrackerInterval: Must be between 1m and 24h
//   - MinLevelTrack: Must be at least 1 (no upper limit)
//   - WorkerPoolSize: Must be between 1 and 100
//   - Channel names: Must be between 1 and 100 characters
//
// Returns nil if all validations pass, otherwise returns a combined error
// containing all validation failures.
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

// validateToken ensures the Discord token is present and has valid length
func (c *Config) validateToken() error {
	if c.Token == "" {
		return fmt.Errorf("DISCORD_TOKEN is required but not set")
	}

	if len(c.Token) < minTokenLength {
		return fmt.Errorf(
			"DISCORD_TOKEN appears invalid (too short: %d chars, expected %d+)",
			len(c.Token), minTokenLength,
		)
	}

	return nil
}

// validateTrackerInterval ensures the tracker interval is within acceptable bounds
func (c *Config) validateTrackerInterval() error {
	if c.TrackerInterval < minTrackerInterval {
		return fmt.Errorf(
			"TRACKER_INTERVAL must be at least %v to avoid excessive API calls, got %v (hint: recommended range is 2m-15m)",
			minTrackerInterval, c.TrackerInterval,
		)
	}

	if c.TrackerInterval > maxTrackerInterval {
		return fmt.Errorf(
			"TRACKER_INTERVAL must be at most %v, got %v",
			maxTrackerInterval, c.TrackerInterval,
		)
	}

	return nil
}

// validateMinLevelTrack ensures the minimum tracking level is valid
func (c *Config) validateMinLevelTrack() error {
	if c.MinLevelTrack < minLevelTrack {
		return fmt.Errorf(
			"MIN_LEVEL_TRACK must be at least %d, got %d",
			minLevelTrack, c.MinLevelTrack,
		)
	}

	return nil
}

// validateWorkerPoolSize ensures the worker pool size is within safe limits
func (c *Config) validateWorkerPoolSize() error {
	if c.WorkerPoolSize < minWorkerPoolSize {
		return fmt.Errorf(
			"WORKER_POOL_SIZE must be at least %d, got %d",
			minWorkerPoolSize, c.WorkerPoolSize,
		)
	}

	if c.WorkerPoolSize > maxWorkerPoolSize {
		return fmt.Errorf(
			"WORKER_POOL_SIZE must be at most %d to prevent resource exhaustion, got %d (hint: recommended range is 5-25)",
			maxWorkerPoolSize, c.WorkerPoolSize,
		)
	}

	return nil
}

// validateChannelNames ensures both Discord channel names are valid
func (c *Config) validateChannelNames() error {
	var errs []error

	if err := validateChannelName("DISCORD_CHANNEL_DEATH", c.DiscordChannelDeath); err != nil {
		errs = append(errs, err)
	}

	if err := validateChannelName("DISCORD_CHANNEL_LEVEL", c.DiscordChannelLevel); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// validateChannelName validates a single channel name
func validateChannelName(fieldName, channelName string) error {
	if channelName == "" {
		return fmt.Errorf("%s cannot be empty", fieldName)
	}

	if len(channelName) > maxChannelNameLength {
		return fmt.Errorf(
			"%s must be at most %d characters (Discord limit), got %d",
			fieldName, maxChannelNameLength, len(channelName),
		)
	}

	return nil
}
