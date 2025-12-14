package handlers

import (
	"testing"

	"death-level-tracker/internal/formatting"

	"github.com/bwmarrin/discordgo"
)

func TestWithAdmin_AdminUser_Success(t *testing.T) {
	var handlerCalled bool
	mockSession := &mockDiscordSession{}

	// Create a handler that will be wrapped
	nextHandler := func(s DiscordSession, i *discordgo.InteractionCreate) {
		handlerCalled = true
	}

	// Wrap with admin middleware
	wrappedHandler := WithAdmin(nextHandler)

	// Create interaction with admin permissions
	interaction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type:    discordgo.InteractionApplicationCommand,
			GuildID: "test-guild",
			Member: &discordgo.Member{
				Permissions: discordgo.PermissionAdministrator,
			},
		},
	}

	// Call the wrapped handler
	wrappedHandler(mockSession, interaction)

	// Verify the next handler was called
	if !handlerCalled {
		t.Error("Expected handler to be called for admin user")
	}

	// Verify no error response was sent
	if mockSession.lastInteractionResponse != nil {
		t.Error("Expected no error response for admin user")
	}
}

func TestWithAdmin_NonAdminUser_Blocked(t *testing.T) {
	var handlerCalled bool
	mockSession := &mockDiscordSession{}

	nextHandler := func(s DiscordSession, i *discordgo.InteractionCreate) {
		handlerCalled = true
	}

	wrappedHandler := WithAdmin(nextHandler)

	// Create interaction WITHOUT admin permissions
	interaction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type:    discordgo.InteractionApplicationCommand,
			GuildID: "test-guild",
			Member: &discordgo.Member{
				Permissions: 0, // No permissions
			},
		},
	}

	wrappedHandler(mockSession, interaction)

	// Verify the next handler was NOT called
	if handlerCalled {
		t.Error("Expected handler NOT to be called for non-admin user")
	}

	// Verify error response was sent
	if mockSession.lastInteractionResponse == nil {
		t.Fatal("Expected error response to be sent")
	}

	if mockSession.lastInteractionResponse.Data.Content != formatting.MsgAdminRequired {
		t.Errorf("Expected message '%s', got '%s'",
			formatting.MsgAdminRequired,
			mockSession.lastInteractionResponse.Data.Content)
	}

	if mockSession.lastInteractionResponse.Data.Flags != discordgo.MessageFlagsEphemeral {
		t.Error("Expected ephemeral error message")
	}
}

func TestWithAdmin_MissingMember_Blocked(t *testing.T) {
	var handlerCalled bool
	mockSession := &mockDiscordSession{}

	nextHandler := func(s DiscordSession, i *discordgo.InteractionCreate) {
		handlerCalled = true
	}

	wrappedHandler := WithAdmin(nextHandler)

	// Create interaction with nil Member
	interaction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type:    discordgo.InteractionApplicationCommand,
			GuildID: "test-guild",
			Member:  nil, // Missing member info
		},
	}

	wrappedHandler(mockSession, interaction)

	// Verify the next handler was NOT called
	if handlerCalled {
		t.Error("Expected handler NOT to be called when member is nil")
	}

	// Verify error response was sent
	if mockSession.lastInteractionResponse == nil {
		t.Fatal("Expected error response to be sent")
	}

	if mockSession.lastInteractionResponse.Data.Content != formatting.MsgAdminRequired {
		t.Errorf("Expected message '%s', got '%s'",
			formatting.MsgAdminRequired,
			mockSession.lastInteractionResponse.Data.Content)
	}
}

func TestWithAdmin_PartialAdminPermissions_Blocked(t *testing.T) {
	var handlerCalled bool
	mockSession := &mockDiscordSession{}

	nextHandler := func(s DiscordSession, i *discordgo.InteractionCreate) {
		handlerCalled = true
	}

	wrappedHandler := WithAdmin(nextHandler)

	// Create interaction with some permissions but NOT admin
	interaction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type:    discordgo.InteractionApplicationCommand,
			GuildID: "test-guild",
			Member: &discordgo.Member{
				Permissions: discordgo.PermissionManageMessages | discordgo.PermissionKickMembers,
			},
		},
	}

	wrappedHandler(mockSession, interaction)

	// Verify the next handler was NOT called
	if handlerCalled {
		t.Error("Expected handler NOT to be called without admin permission")
	}

	// Verify error response was sent
	if mockSession.lastInteractionResponse != nil {
		if mockSession.lastInteractionResponse.Data.Content != formatting.MsgAdminRequired {
			t.Errorf("Expected admin required message")
		}
	}
}

func TestWithAdmin_AdminPlusOtherPermissions_Success(t *testing.T) {
	var handlerCalled bool
	mockSession := &mockDiscordSession{}

	nextHandler := func(s DiscordSession, i *discordgo.InteractionCreate) {
		handlerCalled = true
	}

	wrappedHandler := WithAdmin(nextHandler)

	// Create interaction with admin + other permissions
	interaction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type:    discordgo.InteractionApplicationCommand,
			GuildID: "test-guild",
			Member: &discordgo.Member{
				Permissions: discordgo.PermissionAdministrator | discordgo.PermissionManageServer,
			},
		},
	}

	wrappedHandler(mockSession, interaction)

	// Verify the next handler WAS called
	if !handlerCalled {
		t.Error("Expected handler to be called when admin permission is present")
	}

	// Verify no error response was sent
	if mockSession.lastInteractionResponse != nil {
		t.Error("Expected no error response for admin user")
	}
}
