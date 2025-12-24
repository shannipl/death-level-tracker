package commands

import (
	"errors"
	"testing"

	"github.com/bwmarrin/discordgo"
)

type mockCommandSession struct {
	createFunc func(appID, guildID string, cmd *discordgo.ApplicationCommand) (*discordgo.ApplicationCommand, error)
	deleteFunc func(appID, guildID, cmdID string) error
}

func (m *mockCommandSession) ApplicationCommandCreate(appID, guildID string, cmd *discordgo.ApplicationCommand, opts ...discordgo.RequestOption) (*discordgo.ApplicationCommand, error) {
	if m.createFunc != nil {
		return m.createFunc(appID, guildID, cmd)
	}
	return &discordgo.ApplicationCommand{ID: "id-" + cmd.Name, Name: cmd.Name}, nil
}

func (m *mockCommandSession) ApplicationCommandDelete(appID, guildID, cmdID string, opts ...discordgo.RequestOption) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(appID, guildID, cmdID)
	}
	return nil
}

func TestGetApplicationCommands(t *testing.T) {
	commands := GetApplicationCommands()

	if len(commands) != 5 {
		t.Fatalf("expected 5 commands, got %d", len(commands))
	}

	expectedNames := []string{"track-world", "stop-tracking", "add-guild", "unset-guild", "list-guilds"}
	for i, cmd := range commands {
		if cmd.Name != expectedNames[i] {
			t.Errorf("command %d: expected name %q, got %q", i, expectedNames[i], cmd.Name)
		}
	}
}

func TestGetApplicationCommands_AllRequireAdminPermissions(t *testing.T) {
	for _, cmd := range GetApplicationCommands() {
		if cmd.DefaultMemberPermissions == nil {
			t.Errorf("command %q: missing DefaultMemberPermissions", cmd.Name)
			continue
		}
		if *cmd.DefaultMemberPermissions != int64(discordgo.PermissionAdministrator) {
			t.Errorf("command %q: expected Administrator permission", cmd.Name)
		}
	}
}

func TestGetApplicationCommands_AllHaveDescriptions(t *testing.T) {
	for _, cmd := range GetApplicationCommands() {
		if cmd.Description == "" {
			t.Errorf("command %q: missing description", cmd.Name)
		}
	}
}

func TestGetApplicationCommands_OptionsConfiguration(t *testing.T) {
	tests := []struct {
		name         string
		cmdIndex     int
		wantOptions  int
		autocomplete bool
	}{
		{"track-world has required name option", 0, 1, false},
		{"stop-tracking has no options", 1, 0, false},
		{"add-guild has required name option", 2, 1, false},
		{"unset-guild has autocomplete option", 3, 1, true},
		{"list-guilds has no options", 4, 0, false},
	}

	commands := GetApplicationCommands()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := commands[tt.cmdIndex]
			if len(cmd.Options) != tt.wantOptions {
				t.Fatalf("expected %d options, got %d", tt.wantOptions, len(cmd.Options))
			}
			if tt.wantOptions > 0 {
				opt := cmd.Options[0]
				if opt.Type != discordgo.ApplicationCommandOptionString {
					t.Errorf("expected string option type")
				}
				if opt.Name != "name" {
					t.Errorf("expected option name 'name', got %q", opt.Name)
				}
				if !opt.Required {
					t.Errorf("expected option to be required")
				}
				if opt.Autocomplete != tt.autocomplete {
					t.Errorf("expected autocomplete=%v, got %v", tt.autocomplete, opt.Autocomplete)
				}
			}
		})
	}
}

func TestGetApplicationCommands_IsIdempotent(t *testing.T) {
	first := GetApplicationCommands()
	second := GetApplicationCommands()

	if len(first) != len(second) {
		t.Fatal("successive calls return different lengths")
	}

	for i := range first {
		if first[i].Name != second[i].Name {
			t.Errorf("command %d differs: %q vs %q", i, first[i].Name, second[i].Name)
		}
	}
}

func TestRegisterCommands_AllSucceed(t *testing.T) {
	var created []string
	session := &mockCommandSession{
		createFunc: func(appID, guildID string, cmd *discordgo.ApplicationCommand) (*discordgo.ApplicationCommand, error) {
			created = append(created, cmd.Name)
			return &discordgo.ApplicationCommand{ID: "id-" + cmd.Name, Name: cmd.Name}, nil
		},
	}

	commands := []*discordgo.ApplicationCommand{{Name: "cmd-1"}, {Name: "cmd-2"}, {Name: "cmd-3"}}
	result := RegisterCommands(session, commands, "bot-id", "guild-id")

	if len(created) != 3 {
		t.Errorf("expected 3 creates, got %d", len(created))
	}

	for i, cmd := range result {
		if cmd == nil {
			t.Errorf("result[%d] is nil", i)
			continue
		}
		if cmd.Name != commands[i].Name {
			t.Errorf("result[%d]: expected name %q, got %q", i, commands[i].Name, cmd.Name)
		}
	}
}

func TestRegisterCommands_PartialFailure(t *testing.T) {
	session := &mockCommandSession{
		createFunc: func(appID, guildID string, cmd *discordgo.ApplicationCommand) (*discordgo.ApplicationCommand, error) {
			if cmd.Name == "failing" {
				return nil, errors.New("api error")
			}
			return &discordgo.ApplicationCommand{ID: "id-" + cmd.Name, Name: cmd.Name}, nil
		},
	}

	commands := []*discordgo.ApplicationCommand{{Name: "ok-1"}, {Name: "failing"}, {Name: "ok-2"}}
	result := RegisterCommands(session, commands, "bot-id", "guild-id")

	if len(result) != 3 {
		t.Fatalf("expected 3 results (with nil for failure), got %d", len(result))
	}

	if result[0] == nil || result[0].Name != "ok-1" {
		t.Error("result[0] should be ok-1")
	}
	if result[1] != nil {
		t.Error("result[1] should be nil (failed)")
	}
	if result[2] == nil || result[2].Name != "ok-2" {
		t.Error("result[2] should be ok-2")
	}
}

func TestRegisterCommands_AllFail(t *testing.T) {
	session := &mockCommandSession{
		createFunc: func(appID, guildID string, cmd *discordgo.ApplicationCommand) (*discordgo.ApplicationCommand, error) {
			return nil, errors.New("api unavailable")
		},
	}

	commands := []*discordgo.ApplicationCommand{{Name: "cmd-1"}, {Name: "cmd-2"}}
	result := RegisterCommands(session, commands, "bot-id", "guild-id")

	for i, cmd := range result {
		if cmd != nil {
			t.Errorf("result[%d] should be nil, got %v", i, cmd)
		}
	}
}

func TestRegisterCommands_EmptySlice(t *testing.T) {
	session := &mockCommandSession{
		createFunc: func(appID, guildID string, cmd *discordgo.ApplicationCommand) (*discordgo.ApplicationCommand, error) {
			t.Error("should not be called for empty slice")
			return nil, nil
		},
	}

	result := RegisterCommands(session, []*discordgo.ApplicationCommand{}, "bot-id", "guild-id")

	if len(result) != 0 {
		t.Errorf("expected empty result, got %d", len(result))
	}
}

func TestRegisterCommands_PassesCorrectParameters(t *testing.T) {
	var gotAppID, gotGuildID string

	session := &mockCommandSession{
		createFunc: func(appID, guildID string, cmd *discordgo.ApplicationCommand) (*discordgo.ApplicationCommand, error) {
			gotAppID = appID
			gotGuildID = guildID
			return &discordgo.ApplicationCommand{Name: cmd.Name}, nil
		},
	}

	RegisterCommands(session, []*discordgo.ApplicationCommand{{Name: "test"}}, "my-bot-id", "my-guild-id")

	if gotAppID != "my-bot-id" {
		t.Errorf("expected appID 'my-bot-id', got %q", gotAppID)
	}
	if gotGuildID != "my-guild-id" {
		t.Errorf("expected guildID 'my-guild-id', got %q", gotGuildID)
	}
}

func TestCleanupCommands_AllSucceed(t *testing.T) {
	var deleted []string
	session := &mockCommandSession{
		deleteFunc: func(appID, guildID, cmdID string) error {
			deleted = append(deleted, cmdID)
			return nil
		},
	}

	commands := []*discordgo.ApplicationCommand{
		{ID: "id-1", Name: "cmd-1"},
		{ID: "id-2", Name: "cmd-2"},
	}
	CleanupCommands(session, commands, "bot-id", "guild-id")

	if len(deleted) != 2 {
		t.Errorf("expected 2 deletions, got %d", len(deleted))
	}
}

func TestCleanupCommands_SkipsNilEntries(t *testing.T) {
	var deleted []string
	session := &mockCommandSession{
		deleteFunc: func(appID, guildID, cmdID string) error {
			deleted = append(deleted, cmdID)
			return nil
		},
	}

	commands := []*discordgo.ApplicationCommand{
		{ID: "id-1", Name: "cmd-1"},
		nil,
		{ID: "id-3", Name: "cmd-3"},
		nil,
	}
	CleanupCommands(session, commands, "bot-id", "guild-id")

	if len(deleted) != 2 {
		t.Errorf("expected 2 deletions (skipping nils), got %d", len(deleted))
	}
}

func TestCleanupCommands_ContinuesOnError(t *testing.T) {
	var attempts int
	session := &mockCommandSession{
		deleteFunc: func(appID, guildID, cmdID string) error {
			attempts++
			if cmdID == "id-2" {
				return errors.New("delete failed")
			}
			return nil
		},
	}

	commands := []*discordgo.ApplicationCommand{
		{ID: "id-1", Name: "cmd-1"},
		{ID: "id-2", Name: "cmd-2"},
		{ID: "id-3", Name: "cmd-3"},
	}
	CleanupCommands(session, commands, "bot-id", "guild-id")

	if attempts != 3 {
		t.Errorf("expected 3 attempts (continues on error), got %d", attempts)
	}
}

func TestCleanupCommands_EmptySlice(t *testing.T) {
	session := &mockCommandSession{
		deleteFunc: func(appID, guildID, cmdID string) error {
			t.Error("should not be called for empty slice")
			return nil
		},
	}

	CleanupCommands(session, []*discordgo.ApplicationCommand{}, "bot-id", "guild-id")
}

func TestCleanupCommands_PassesCorrectParameters(t *testing.T) {
	var gotAppID, gotGuildID, gotCmdID string

	session := &mockCommandSession{
		deleteFunc: func(appID, guildID, cmdID string) error {
			gotAppID = appID
			gotGuildID = guildID
			gotCmdID = cmdID
			return nil
		},
	}

	CleanupCommands(session, []*discordgo.ApplicationCommand{{ID: "cmd-123", Name: "test"}}, "my-bot-id", "my-guild-id")

	if gotAppID != "my-bot-id" {
		t.Errorf("expected appID 'my-bot-id', got %q", gotAppID)
	}
	if gotGuildID != "my-guild-id" {
		t.Errorf("expected guildID 'my-guild-id', got %q", gotGuildID)
	}
	if gotCmdID != "cmd-123" {
		t.Errorf("expected cmdID 'cmd-123', got %q", gotCmdID)
	}
}

func TestStringOption(t *testing.T) {
	tests := []struct {
		name         string
		optName      string
		description  string
		required     bool
		autocomplete bool
	}{
		{"required without autocomplete", "world", "World name", true, false},
		{"required with autocomplete", "guild", "Guild name", true, true},
		{"optional without autocomplete", "filter", "Optional filter", false, false},
		{"optional with autocomplete", "search", "Search term", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opt := stringOption(tt.optName, tt.description, tt.required, tt.autocomplete)

			if opt.Type != discordgo.ApplicationCommandOptionString {
				t.Error("expected string type")
			}
			if opt.Name != tt.optName {
				t.Errorf("expected name %q, got %q", tt.optName, opt.Name)
			}
			if opt.Description != tt.description {
				t.Errorf("expected description %q, got %q", tt.description, opt.Description)
			}
			if opt.Required != tt.required {
				t.Errorf("expected required=%v", tt.required)
			}
			if opt.Autocomplete != tt.autocomplete {
				t.Errorf("expected autocomplete=%v", tt.autocomplete)
			}
		})
	}
}
