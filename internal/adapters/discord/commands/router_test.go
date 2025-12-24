package commands

import (
	"testing"

	"github.com/bwmarrin/discordgo"
)

type mockSession struct{}

func (m *mockSession) GuildChannels(guildID string, opts ...discordgo.RequestOption) ([]*discordgo.Channel, error) {
	return nil, nil
}

func (m *mockSession) GuildChannelCreate(guildID, name string, ctype discordgo.ChannelType, opts ...discordgo.RequestOption) (*discordgo.Channel, error) {
	return nil, nil
}

func (m *mockSession) InteractionRespond(i *discordgo.Interaction, resp *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
	return nil
}

func TestNewRouter(t *testing.T) {
	router := NewRouter()

	if router == nil {
		t.Fatal("expected non-nil router")
	}
	if router.routes == nil {
		t.Fatal("expected routes map to be initialized")
	}
	if len(router.routes) != 0 {
		t.Errorf("expected empty routes, got %d", len(router.routes))
	}
}

func TestRouter_Register(t *testing.T) {
	router := NewRouter()

	called := false
	router.Register("test-cmd", func(s DiscordSession, i *discordgo.InteractionCreate) {
		called = true
	})

	if len(router.routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(router.routes))
	}

	router.routes["test-cmd"](nil, nil)
	if !called {
		t.Error("registered handler should be callable")
	}
}

func TestRouter_Register_OverwritesPrevious(t *testing.T) {
	router := NewRouter()

	firstCalled := false
	secondCalled := false

	router.Register("cmd", func(s DiscordSession, i *discordgo.InteractionCreate) {
		firstCalled = true
	})
	router.Register("cmd", func(s DiscordSession, i *discordgo.InteractionCreate) {
		secondCalled = true
	})

	router.routes["cmd"](nil, nil)

	if firstCalled {
		t.Error("first handler should be overwritten")
	}
	if !secondCalled {
		t.Error("second handler should be called")
	}
}

func TestRouter_Handle_DispatchesToCorrectHandler(t *testing.T) {
	router := NewRouter()
	session := &mockSession{}

	var called string
	router.Register("cmd-a", func(s DiscordSession, i *discordgo.InteractionCreate) { called = "a" })
	router.Register("cmd-b", func(s DiscordSession, i *discordgo.InteractionCreate) { called = "b" })

	router.Handle(session, makeInteraction("cmd-a", discordgo.InteractionApplicationCommand))
	if called != "a" {
		t.Errorf("expected 'a', got %q", called)
	}

	router.Handle(session, makeInteraction("cmd-b", discordgo.InteractionApplicationCommand))
	if called != "b" {
		t.Errorf("expected 'b', got %q", called)
	}
}

func TestRouter_Handle_SupportsAutocomplete(t *testing.T) {
	router := NewRouter()
	session := &mockSession{}

	called := false
	router.Register("autocomplete-cmd", func(s DiscordSession, i *discordgo.InteractionCreate) {
		called = true
	})

	router.Handle(session, makeInteraction("autocomplete-cmd", discordgo.InteractionApplicationCommandAutocomplete))

	if !called {
		t.Error("handler should be called for autocomplete interactions")
	}
}

func TestRouter_Handle_IgnoresOtherInteractionTypes(t *testing.T) {
	ignoredTypes := []discordgo.InteractionType{
		discordgo.InteractionPing,
		discordgo.InteractionMessageComponent,
		discordgo.InteractionModalSubmit,
	}

	for _, iType := range ignoredTypes {
		t.Run(iType.String(), func(t *testing.T) {
			router := NewRouter()
			session := &mockSession{}

			called := false
			router.Register("test-cmd", func(s DiscordSession, i *discordgo.InteractionCreate) {
				called = true
			})

			router.Handle(session, makeInteraction("test-cmd", iType))

			if called {
				t.Errorf("handler should NOT be called for %s", iType)
			}
		})
	}
}

func TestRouter_Handle_UnregisteredCommand(t *testing.T) {
	router := NewRouter()
	session := &mockSession{}

	called := false
	router.Register("registered", func(s DiscordSession, i *discordgo.InteractionCreate) {
		called = true
	})

	router.Handle(session, makeInteraction("unknown", discordgo.InteractionApplicationCommand))

	if called {
		t.Error("handler should NOT be called for unregistered command")
	}
}

func TestRouter_Handle_EmptyRouter(t *testing.T) {
	router := NewRouter()
	session := &mockSession{}

	// Should not panic
	router.Handle(session, makeInteraction("any", discordgo.InteractionApplicationCommand))
}

func TestRouter_Handle_PassesSessionAndInteraction(t *testing.T) {
	router := NewRouter()
	session := &mockSession{}
	interaction := makeInteraction("test", discordgo.InteractionApplicationCommand)

	var gotSession DiscordSession
	var gotInteraction *discordgo.InteractionCreate

	router.Register("test", func(s DiscordSession, i *discordgo.InteractionCreate) {
		gotSession = s
		gotInteraction = i
	})

	router.Handle(session, interaction)

	if gotSession != session {
		t.Error("handler should receive the session")
	}
	if gotInteraction != interaction {
		t.Error("handler should receive the interaction")
	}
}

func TestRouter_HandleFunc_ReturnsCompatibleHandler(t *testing.T) {
	router := NewRouter()

	called := false
	router.Register("test", func(s DiscordSession, i *discordgo.InteractionCreate) {
		called = true
	})

	handler := router.HandleFunc()
	if handler == nil {
		t.Fatal("expected non-nil handler function")
	}

	session := &discordgo.Session{}
	interaction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type: discordgo.InteractionApplicationCommand,
			Data: discordgo.ApplicationCommandInteractionData{Name: "test"},
		},
	}

	handler(session, interaction)

	if !called {
		t.Error("HandleFunc should dispatch to registered handler")
	}
}

func makeInteraction(name string, iType discordgo.InteractionType) *discordgo.InteractionCreate {
	return &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type: iType,
			Data: discordgo.ApplicationCommandInteractionData{Name: name},
		},
	}
}
