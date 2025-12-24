package config

import (
	"strings"
	"testing"
	"time"
)

func validConfig() *Config {
	return &Config{
		Token:               strings.Repeat("x", 50),
		TrackerInterval:     5 * time.Minute,
		MinLevelTrack:       500,
		WorkerPoolSize:      10,
		DiscordChannelDeath: "death-tracker",
		DiscordChannelLevel: "level-tracker",
	}
}

func TestValidate_ValidConfig(t *testing.T) {
	if err := validConfig().Validate(); err != nil {
		t.Errorf("valid config should pass: %v", err)
	}
}

func TestValidate_Token(t *testing.T) {
	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{"valid", strings.Repeat("x", 50), false},
		{"exactly min", strings.Repeat("x", 50), false},
		{"too short", strings.Repeat("x", 49), true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			cfg.Token = tt.token
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Token=%q: error=%v, wantErr=%v", tt.token, err, tt.wantErr)
			}
		})
	}
}

func TestValidate_TrackerInterval(t *testing.T) {
	tests := []struct {
		name     string
		interval time.Duration
		wantErr  bool
	}{
		{"min valid", time.Minute, false},
		{"below min", 59 * time.Second, true},
		{"normal", 5 * time.Minute, false},
		{"max valid", 24 * time.Hour, false},
		{"above max", 25 * time.Hour, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			cfg.TrackerInterval = tt.interval
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("TrackerInterval=%v: error=%v, wantErr=%v", tt.interval, err, tt.wantErr)
			}
		})
	}
}

func TestValidate_MinLevelTrack(t *testing.T) {
	tests := []struct {
		name    string
		level   int
		wantErr bool
	}{
		{"min valid", 1, false},
		{"zero", 0, true},
		{"negative", -1, true},
		{"normal", 500, false},
		{"high", 5000, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			cfg.MinLevelTrack = tt.level
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("MinLevelTrack=%d: error=%v, wantErr=%v", tt.level, err, tt.wantErr)
			}
		})
	}
}

func TestValidate_WorkerPoolSize(t *testing.T) {
	tests := []struct {
		name    string
		size    int
		wantErr bool
	}{
		{"min valid", 1, false},
		{"zero", 0, true},
		{"negative", -1, true},
		{"normal", 10, false},
		{"max valid", 100, false},
		{"above max", 101, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			cfg.WorkerPoolSize = tt.size
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("WorkerPoolSize=%d: error=%v, wantErr=%v", tt.size, err, tt.wantErr)
			}
		})
	}
}

func TestValidate_ChannelNames(t *testing.T) {
	tests := []struct {
		name      string
		deathChan string
		levelChan string
		wantErr   bool
	}{
		{"both valid", "death", "level", false},
		{"death empty", "", "level", true},
		{"level empty", "death", "", true},
		{"both empty", "", "", true},
		{"death too long", strings.Repeat("x", 101), "level", true},
		{"level too long", "death", strings.Repeat("x", 101), true},
		{"max length", strings.Repeat("x", 100), strings.Repeat("y", 100), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			cfg.DiscordChannelDeath = tt.deathChan
			cfg.DiscordChannelLevel = tt.levelChan
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("channels=%q/%q: error=%v, wantErr=%v", tt.deathChan, tt.levelChan, err, tt.wantErr)
			}
		})
	}
}

func TestValidate_MultipleErrors(t *testing.T) {
	cfg := &Config{
		Token:               "",
		TrackerInterval:     30 * time.Second,
		MinLevelTrack:       0,
		WorkerPoolSize:      0,
		DiscordChannelDeath: "",
		DiscordChannelLevel: "",
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation errors")
	}

	errMsg := err.Error()
	mustContain := []string{
		"DISCORD_TOKEN",
		"TRACKER_INTERVAL",
		"MIN_LEVEL_TRACK",
		"WORKER_POOL_SIZE",
		"DISCORD_CHANNEL_DEATH",
		"DISCORD_CHANNEL_LEVEL",
	}

	for _, s := range mustContain {
		if !strings.Contains(errMsg, s) {
			t.Errorf("error should contain %q: %s", s, errMsg)
		}
	}
}
