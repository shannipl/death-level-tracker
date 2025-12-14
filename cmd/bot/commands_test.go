package main

import (
	"errors"
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestGetApplicationCommands(t *testing.T) {
	commands := GetApplicationCommands()

	if commands == nil {
		t.Fatal("Expected non-nil commands slice")
	}

	if len(commands) != 2 {
		t.Fatalf("Expected 2 commands, got %d", len(commands))
	}

	// Test track-world command
	trackCmd := commands[0]
	if trackCmd.Name != "track-world" {
		t.Errorf("Expected command name 'track-world', got '%s'", trackCmd.Name)
	}

	if trackCmd.Description != "Set the Tibia world to track for this server" {
		t.Errorf("Unexpected track-world description: %s", trackCmd.Description)
	}

	if len(trackCmd.Options) != 1 {
		t.Fatalf("Expected 1 option for track-world, got %d", len(trackCmd.Options))
	}

	option := trackCmd.Options[0]
	if option.Name != "name" {
		t.Errorf("Expected option name 'name', got '%s'", option.Name)
	}

	if option.Type != discordgo.ApplicationCommandOptionString {
		t.Errorf("Expected option type String, got %v", option.Type)
	}

	if option.Description != "Name of the Tibia world" {
		t.Errorf("Unexpected option description: %s", option.Description)
	}

	if !option.Required {
		t.Error("Expected 'name' option to be required")
	}

	// Test stop-tracking command
	stopCmd := commands[1]
	if stopCmd.Name != "stop-tracking" {
		t.Errorf("Expected command name 'stop-tracking', got '%s'", stopCmd.Name)
	}

	if stopCmd.Description != "Stop tracking kills" {
		t.Errorf("Unexpected stop-tracking description: %s", stopCmd.Description)
	}

	if len(stopCmd.Options) != 0 {
		t.Errorf("Expected 0 options for stop-tracking, got %d", len(stopCmd.Options))
	}
}

func TestGetApplicationCommands_TrackWorldOption(t *testing.T) {
	commands := GetApplicationCommands()
	trackCmd := commands[0]

	if len(trackCmd.Options) < 1 {
		t.Fatal("Track-world command should have at least 1 option")
	}

	nameOption := trackCmd.Options[0]

	// Verify all properties of the name option
	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"option name", nameOption.Name, "name"},
		{"option type", nameOption.Type, discordgo.ApplicationCommandOptionString},
		{"option description", nameOption.Description, "Name of the Tibia world"},
		{"option required", nameOption.Required, true},
	}

	for _, tt := range tests {
		if tt.got != tt.expected {
			t.Errorf("%s: expected %v, got %v", tt.name, tt.expected, tt.got)
		}
	}
}

func TestGetApplicationCommands_Consistency(t *testing.T) {
	// Verify each call returns consistent data
	commands1 := GetApplicationCommands()
	commands2 := GetApplicationCommands()

	if len(commands1) != len(commands2) {
		t.Fatal("Commands returned different lengths on successive calls")
	}

	for i := range commands1 {
		if commands1[i].Name != commands2[i].Name {
			t.Errorf("Command %d has different name: '%s' vs '%s'",
				i, commands1[i].Name, commands2[i].Name)
		}

		if commands1[i].Description != commands2[i].Description {
			t.Errorf("Command %d has different description", i)
		}

		if len(commands1[i].Options) != len(commands2[i].Options) {
			t.Errorf("Command %d has different number of options", i)
		}
	}
}

func TestGetApplicationCommands_AllCommandsValid(t *testing.T) {
	commands := GetApplicationCommands()

	for i, cmd := range commands {
		if cmd == nil {
			t.Errorf("Command %d is nil", i)
			continue
		}

		if cmd.Name == "" {
			t.Errorf("Command %d has empty name", i)
		}

		if cmd.Description == "" {
			t.Errorf("Command %d has empty description", i)
		}
	}
}

// Tests for RegisterCommands
func TestRegisterCommands_Success(t *testing.T) {
	registeredCount := 0
	mockSession := &mockCommandSession{
		applicationCommandCreateFunc: func(appID, guildID string, cmd *discordgo.ApplicationCommand) (*discordgo.ApplicationCommand, error) {
			registeredCount++
			return &discordgo.ApplicationCommand{
				ID:   "cmd-" + cmd.Name,
				Name: cmd.Name,
			}, nil
		},
	}

	commands := []*discordgo.ApplicationCommand{
		{Name: "test-cmd-1"},
		{Name: "test-cmd-2"},
	}

	result := RegisterCommands(mockSession, commands, "bot-123")

	if registeredCount != 2 {
		t.Errorf("Expected 2 commands to be registered, got %d", registeredCount)
	}

	if len(result) != 2 {
		t.Fatalf("Expected 2 commands in result, got %d", len(result))
	}

	for i, cmd := range result {
		if cmd.Name != commands[i].Name {
			t.Errorf("Command %d: expected name '%s', got '%s'", i, commands[i].Name, cmd.Name)
		}
	}
}

func TestRegisterCommands_WithErrors(t *testing.T) {
	successCount := 0
	mockSession := &mockCommandSession{
		applicationCommandCreateFunc: func(appID, guildID string, cmd *discordgo.ApplicationCommand) (*discordgo.ApplicationCommand, error) {
			if cmd.Name == "failing-cmd" {
				return nil, errors.New("registration failed")
			}
			successCount++
			return &discordgo.ApplicationCommand{
				ID:   "cmd-" + cmd.Name,
				Name: cmd.Name,
			}, nil
		},
	}

	commands := []*discordgo.ApplicationCommand{
		{Name: "good-cmd"},
		{Name: "failing-cmd"},
		{Name: "another-good-cmd"},
	}

	result := RegisterCommands(mockSession, commands, "bot-123")

	if successCount != 2 {
		t.Errorf("Expected 2 successful registrations, got %d", successCount)
	}

	if len(result) != 3 {
		t.Errorf("Expected 3 elements in result, got %d", len(result))
	}

	// Check that failing command is nil
	if result[1] != nil {
		t.Error("Expected nil for failing command")
	}

	// Check successful commands
	if result[0] == nil || result[0].Name != "good-cmd" {
		t.Error("First command should be 'good-cmd'")
	}

	if result[2] == nil || result[2].Name != "another-good-cmd" {
		t.Error("Third command should be 'another-good-cmd'")
	}
}

func TestRegisterCommands_EmptySlice(t *testing.T) {
	mockSession := &mockCommandSession{
		applicationCommandCreateFunc: func(appID, guildID string, cmd *discordgo.ApplicationCommand) (*discordgo.ApplicationCommand, error) {
			t.Error("Should not call ApplicationCommandCreate for empty slice")
			return nil, nil
		},
	}

	result := RegisterCommands(mockSession, []*discordgo.ApplicationCommand{}, "bot-123")

	if len(result) != 0 {
		t.Errorf("Expected empty result, got %d commands", len(result))
	}
}

// Tests for CleanupCommands
func TestCleanupCommands_Success(t *testing.T) {
	deletedCommands := make(map[string]bool)
	mockSession := &mockCommandSession{
		applicationCommandDeleteFunc: func(appID, guildID, cmdID string) error {
			deletedCommands[cmdID] = true
			return nil
		},
	}

	commands := []*discordgo.ApplicationCommand{
		{ID: "cmd-1", Name: "command-1"},
		{ID: "cmd-2", Name: "command-2"},
	}

	CleanupCommands(mockSession, commands, "bot-123")

	if len(deletedCommands) != 2 {
		t.Errorf("Expected 2 commands to be deleted, got %d", len(deletedCommands))
	}

	if !deletedCommands["cmd-1"] || !deletedCommands["cmd-2"] {
		t.Error("Not all commands were deleted")
	}
}

func TestCleanupCommands_WithNilCommands(t *testing.T) {
	deleteCount := 0
	mockSession := &mockCommandSession{
		applicationCommandDeleteFunc: func(appID, guildID, cmdID string) error {
			deleteCount++
			return nil
		},
	}

	commands := []*discordgo.ApplicationCommand{
		{ID: "cmd-1", Name: "command-1"},
		nil, // Should be skipped
		{ID: "cmd-3", Name: "command-3"},
		nil, // Should be skipped
	}

	CleanupCommands(mockSession, commands, "bot-123")

	if deleteCount != 2 {
		t.Errorf("Expected 2 commands to be deleted (skipping nils), got %d", deleteCount)
	}
}

func TestCleanupCommands_WithErrors(t *testing.T) {
	attemptedDeletes := 0
	mockSession := &mockCommandSession{
		applicationCommandDeleteFunc: func(appID, guildID, cmdID string) error {
			attemptedDeletes++
			if cmdID == "cmd-2" {
				return errors.New("delete failed")
			}
			return nil
		},
	}

	commands := []*discordgo.ApplicationCommand{
		{ID: "cmd-1", Name: "command-1"},
		{ID: "cmd-2", Name: "command-2"},
		{ID: "cmd-3", Name: "command-3"},
	}

	CleanupCommands(mockSession, commands, "bot-123")

	if attemptedDeletes != 3 {
		t.Errorf("Expected 3 deletion attempts, got %d", attemptedDeletes)
	}
}

func TestCleanupCommands_EmptySlice(t *testing.T) {
	mockSession := &mockCommandSession{
		applicationCommandDeleteFunc: func(appID, guildID, cmdID string) error {
			t.Error("Should not call ApplicationCommandDelete for empty slice")
			return nil
		},
	}

	CleanupCommands(mockSession, []*discordgo.ApplicationCommand{}, "bot-123")
}

// Mock session for command testing
type mockCommandSession struct {
	applicationCommandCreateFunc func(appID, guildID string, cmd *discordgo.ApplicationCommand) (*discordgo.ApplicationCommand, error)
	applicationCommandDeleteFunc func(appID, guildID, cmdID string) error
}

func (m *mockCommandSession) ApplicationCommandCreate(appID, guildID string, cmd *discordgo.ApplicationCommand, options ...discordgo.RequestOption) (*discordgo.ApplicationCommand, error) {
	if m.applicationCommandCreateFunc != nil {
		return m.applicationCommandCreateFunc(appID, guildID, cmd)
	}
	return nil, nil
}

func (m *mockCommandSession) ApplicationCommandDelete(appID, guildID, cmdID string, options ...discordgo.RequestOption) error {
	if m.applicationCommandDeleteFunc != nil {
		return m.applicationCommandDeleteFunc(appID, guildID, cmdID)
	}
	return nil
}
