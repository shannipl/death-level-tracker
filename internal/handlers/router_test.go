package handlers

import (
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestNewRouter(t *testing.T) {
	router := NewRouter()

	if router == nil {
		t.Fatal("Expected NewRouter to return non-nil router")
	}

	if router.routes == nil {
		t.Error("Expected routes map to be initialized")
	}

	if len(router.routes) != 0 {
		t.Errorf("Expected empty routes map, got %d routes", len(router.routes))
	}
}

func TestRouter_Register(t *testing.T) {
	router := NewRouter()

	var handlerCalled bool
	handler := func(s DiscordSession, i *discordgo.InteractionCreate) {
		handlerCalled = true
	}

	// Register a command
	router.Register("test-command", handler)

	// Verify handler was registered
	if len(router.routes) != 1 {
		t.Fatalf("Expected 1 route, got %d", len(router.routes))
	}

	if _, exists := router.routes["test-command"]; !exists {
		t.Error("Expected test-command to be registered")
	}

	// Call the registered handler to verify it works
	router.routes["test-command"](nil, nil)
	if !handlerCalled {
		t.Error("Expected registered handler to be callable")
	}
}

func TestRouter_Register_Multiple(t *testing.T) {
	router := NewRouter()

	handler1Called := false
	handler2Called := false

	handler1 := func(s DiscordSession, i *discordgo.InteractionCreate) {
		handler1Called = true
	}

	handler2 := func(s DiscordSession, i *discordgo.InteractionCreate) {
		handler2Called = true
	}

	router.Register("command1", handler1)
	router.Register("command2", handler2)

	if len(router.routes) != 2 {
		t.Fatalf("Expected 2 routes, got %d", len(router.routes))
	}

	// Verify both handlers are registered
	router.routes["command1"](nil, nil)
	router.routes["command2"](nil, nil)

	if !handler1Called {
		t.Error("Expected handler1 to be called")
	}
	if !handler2Called {
		t.Error("Expected handler2 to be called")
	}
}

func TestRouter_Handle_DispatchesToCorrectHandler(t *testing.T) {
	router := NewRouter()
	mockSession := &mockDiscordSession{}

	var calledCommand string
	handler1 := func(s DiscordSession, i *discordgo.InteractionCreate) {
		calledCommand = "command1"
	}
	handler2 := func(s DiscordSession, i *discordgo.InteractionCreate) {
		calledCommand = "command2"
	}

	router.Register("command1", handler1)
	router.Register("command2", handler2)

	// Create interaction for command1
	interaction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type: discordgo.InteractionApplicationCommand,
			Data: discordgo.ApplicationCommandInteractionData{
				Name: "command1",
			},
		},
	}

	router.Handle(mockSession, interaction)

	if calledCommand != "command1" {
		t.Errorf("Expected command1 to be called, got %s", calledCommand)
	}

	// Test command2
	calledCommand = ""
	interaction.Data = discordgo.ApplicationCommandInteractionData{
		Name: "command2",
	}

	router.Handle(mockSession, interaction)

	if calledCommand != "command2" {
		t.Errorf("Expected command2 to be called, got %s", calledCommand)
	}
}

func TestRouter_Handle_IgnoresNonCommandInteractions(t *testing.T) {
	router := NewRouter()
	mockSession := &mockDiscordSession{}

	var handlerCalled bool
	handler := func(s DiscordSession, i *discordgo.InteractionCreate) {
		handlerCalled = true
	}

	router.Register("test-command", handler)

	// Test with different interaction types
	testCases := []struct {
		name            string
		interactionType discordgo.InteractionType
	}{
		{"Ping", discordgo.InteractionPing},
		{"MessageComponent", discordgo.InteractionMessageComponent},
		{"ApplicationCommandAutocomplete", discordgo.InteractionApplicationCommandAutocomplete},
		{"ModalSubmit", discordgo.InteractionModalSubmit},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handlerCalled = false

			interaction := &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					Type: tc.interactionType,
					Data: discordgo.ApplicationCommandInteractionData{
						Name: "test-command",
					},
				},
			}

			router.Handle(mockSession, interaction)

			if handlerCalled {
				t.Errorf("Expected handler NOT to be called for %s interaction", tc.name)
			}
		})
	}
}

func TestRouter_Handle_UnregisteredCommand(t *testing.T) {
	router := NewRouter()
	mockSession := &mockDiscordSession{}

	var handlerCalled bool
	handler := func(s DiscordSession, i *discordgo.InteractionCreate) {
		handlerCalled = true
	}

	router.Register("registered-command", handler)

	// Try to handle an unregistered command
	interaction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type: discordgo.InteractionApplicationCommand,
			Data: discordgo.ApplicationCommandInteractionData{
				Name: "unregistered-command",
			},
		},
	}

	router.Handle(mockSession, interaction)

	if handlerCalled {
		t.Error("Expected handler NOT to be called for unregistered command")
	}
}

func TestRouter_Handle_WithRealSession(t *testing.T) {
	router := NewRouter()

	var receivedSession DiscordSession
	var receivedInteraction *discordgo.InteractionCreate

	handler := func(s DiscordSession, i *discordgo.InteractionCreate) {
		receivedSession = s
		receivedInteraction = i
	}

	router.Register("test-cmd", handler)

	mockSession := &mockDiscordSession{}
	interaction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type: discordgo.InteractionApplicationCommand,
			Data: discordgo.ApplicationCommandInteractionData{
				Name: "test-cmd",
			},
		},
	}

	router.Handle(mockSession, interaction)

	if receivedSession != mockSession {
		t.Error("Expected handler to receive the session")
	}

	if receivedInteraction != interaction {
		t.Error("Expected handler to receive the interaction")
	}
}

func TestRouter_Handle_EmptyRoutes(t *testing.T) {
	router := NewRouter()
	mockSession := &mockDiscordSession{}

	// Handle interaction with no registered routes (should not panic)
	interaction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type: discordgo.InteractionApplicationCommand,
			Data: discordgo.ApplicationCommandInteractionData{
				Name: "any-command",
			},
		},
	}

	// Should not panic
	router.Handle(mockSession, interaction)
}
