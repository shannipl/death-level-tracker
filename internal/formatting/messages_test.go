package formatting

import "testing"

func TestConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{
			name:     "MsgAdminRequired",
			constant: MsgAdminRequired,
			expected: "You need Administrator permissions to use this command.",
		},
		{
			name:     "MsgWorldRequired",
			constant: MsgWorldRequired,
			expected: "World name is required.",
		},
		{
			name:     "MsgSaveError",
			constant: MsgSaveError,
			expected: "Failed to save configuration.",
		},
		{
			name:     "MsgStopError",
			constant: MsgStopError,
			expected: "Failed to stop tracking.",
		},
		{
			name:     "MsgStopSuccess",
			constant: MsgStopSuccess,
			expected: "Tracking stopped. Configuration removed.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, tt.constant)
			}
		})
	}
}

func TestMsgDeath(t *testing.T) {
	tests := []struct {
		name     string
		charName string
		timeStr  string
		reason   string
		expected string
	}{
		{
			name:     "standard death message",
			charName: "Lord Paulistinha",
			timeStr:  "2024-12-13 10:30:00",
			reason:   "Killed by a dragon",
			expected: "Lord Paulistinha - 2024-12-13 10:30:00 - Killed by a dragon",
		},
		{
			name:     "death with special characters",
			charName: "Sir O'Malley",
			timeStr:  "2024-12-13 15:45:30",
			reason:   "Died at level 100 by a demon",
			expected: "Sir O'Malley - 2024-12-13 15:45:30 - Died at level 100 by a demon",
		},
		{
			name:     "death with empty reason",
			charName: "Player One",
			timeStr:  "2024-12-13 12:00:00",
			reason:   "",
			expected: "Player One - 2024-12-13 12:00:00 - ",
		},
		{
			name:     "death with unicode characters",
			charName: "Señor José",
			timeStr:  "2024-12-13 09:15:22",
			reason:   "Killed by a lich 💀",
			expected: "Señor José - 2024-12-13 09:15:22 - Killed by a lich 💀",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MsgDeath(tt.charName, tt.timeStr, tt.reason)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestMsgLevelUp(t *testing.T) {
	tests := []struct {
		name     string
		charName string
		oldLevel int
		newLevel int
		expected string
	}{
		{
			name:     "standard level up",
			charName: "Knight Bob",
			oldLevel: 100,
			newLevel: 101,
			expected: "Knight Bob advanced from level 100 to 101",
		},
		{
			name:     "large level jump",
			charName: "Mage Alice",
			oldLevel: 500,
			newLevel: 550,
			expected: "Mage Alice advanced from level 500 to 550",
		},
		{
			name:     "low level advancement",
			charName: "Newbie",
			oldLevel: 8,
			newLevel: 9,
			expected: "Newbie advanced from level 8 to 9",
		},
		{
			name:     "high level player",
			charName: "Epic Druid",
			oldLevel: 999,
			newLevel: 1000,
			expected: "Epic Druid advanced from level 999 to 1000",
		},
		{
			name:     "level up with special characters in name",
			charName: "Dragon-Slayer",
			oldLevel: 250,
			newLevel: 251,
			expected: "Dragon-Slayer advanced from level 250 to 251",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MsgLevelUp(tt.charName, tt.oldLevel, tt.newLevel)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestMsgChannelError(t *testing.T) {
	tests := []struct {
		name        string
		channelName string
		expected    string
	}{
		{
			name:        "standard channel name",
			channelName: "death-level-tracker",
			expected:    "Failed to create or find #death-level-tracker channel.",
		},
		{
			name:        "channel with numbers",
			channelName: "tracker-123",
			expected:    "Failed to create or find #tracker-123 channel.",
		},
		{
			name:        "short channel name",
			channelName: "log",
			expected:    "Failed to create or find #log channel.",
		},
		{
			name:        "channel with underscores",
			channelName: "level_up_notifications",
			expected:    "Failed to create or find #level_up_notifications channel.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MsgChannelError(tt.channelName)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestMsgTrackSuccess(t *testing.T) {
	tests := []struct {
		name      string
		world     string
		deathChan string
		levelChan string
		expected  string
	}{
		{
			name:      "standard tracking message",
			world:     "Antica",
			deathChan: "death-level-tracker",
			levelChan: "level-tracker",
			expected:  "Tracking world **Antica** configured! Notifications will appear in #death-level-tracker and #level-tracker.",
		},
		{
			name:      "different world",
			world:     "Secura",
			deathChan: "deaths",
			levelChan: "levels",
			expected:  "Tracking world **Secura** configured! Notifications will appear in #deaths and #levels.",
		},
		{
			name:      "world with special characters",
			world:     "Premia",
			deathChan: "tibia-deaths",
			levelChan: "tibia-levels",
			expected:  "Tracking world **Premia** configured! Notifications will appear in #tibia-deaths and #tibia-levels.",
		},
		{
			name:      "same channel for both",
			world:     "Pacera",
			deathChan: "notifications",
			levelChan: "notifications",
			expected:  "Tracking world **Pacera** configured! Notifications will appear in #notifications and #notifications.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MsgTrackSuccess(tt.world, tt.deathChan, tt.levelChan)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}
