package commands

import (
	"testing"

	"death-level-tracker/internal/adapters/discord/formatting"

	"github.com/bwmarrin/discordgo"
)

func TestWithAdmin_AllowsAdminUser(t *testing.T) {
	session := &mockDiscordSession{}
	called := false

	handler := WithAdmin(func(s DiscordSession, i *discordgo.InteractionCreate) {
		called = true
	})

	handler(session, interactionWithPermissions(discordgo.PermissionAdministrator))

	if !called {
		t.Error("handler should be called for admin user")
	}
	if session.lastInteractionResponse != nil {
		t.Error("no error response should be sent for admin user")
	}
}

func TestWithAdmin_AllowsAdminWithOtherPermissions(t *testing.T) {
	session := &mockDiscordSession{}
	called := false

	handler := WithAdmin(func(s DiscordSession, i *discordgo.InteractionCreate) {
		called = true
	})

	perms := int64(discordgo.PermissionAdministrator | discordgo.PermissionManageServer | discordgo.PermissionManageMessages)
	handler(session, interactionWithPermissions(perms))

	if !called {
		t.Error("handler should be called when admin permission is present")
	}
}

func TestWithAdmin_BlocksNoPermissions(t *testing.T) {
	session := &mockDiscordSession{}
	called := false

	handler := WithAdmin(func(s DiscordSession, i *discordgo.InteractionCreate) {
		called = true
	})

	handler(session, interactionWithPermissions(0))

	if called {
		t.Error("handler should NOT be called for user with no permissions")
	}
	assertAdminRequiredResponse(t, session)
}

func TestWithAdmin_BlocksNonAdminPermissions(t *testing.T) {
	nonAdminPerms := []int64{
		discordgo.PermissionManageMessages,
		discordgo.PermissionKickMembers,
		discordgo.PermissionBanMembers,
		discordgo.PermissionManageServer,
		discordgo.PermissionManageChannels,
		discordgo.PermissionManageMessages | discordgo.PermissionKickMembers | discordgo.PermissionBanMembers,
	}

	for _, perm := range nonAdminPerms {
		session := &mockDiscordSession{}
		called := false

		handler := WithAdmin(func(s DiscordSession, i *discordgo.InteractionCreate) {
			called = true
		})

		handler(session, interactionWithPermissions(perm))

		if called {
			t.Errorf("handler should NOT be called for permission %d", perm)
		}
		assertAdminRequiredResponse(t, session)
	}
}

func TestWithAdmin_BlocksNilMember(t *testing.T) {
	session := &mockDiscordSession{}
	called := false

	handler := WithAdmin(func(s DiscordSession, i *discordgo.InteractionCreate) {
		called = true
	})

	interaction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type:   discordgo.InteractionApplicationCommand,
			Member: nil,
		},
	}

	handler(session, interaction)

	if called {
		t.Error("handler should NOT be called when member is nil")
	}
	assertAdminRequiredResponse(t, session)
}

func TestWithAdmin_PassesSessionAndInteraction(t *testing.T) {
	session := &mockDiscordSession{}
	interaction := interactionWithPermissions(discordgo.PermissionAdministrator)

	var gotSession DiscordSession
	var gotInteraction *discordgo.InteractionCreate

	handler := WithAdmin(func(s DiscordSession, i *discordgo.InteractionCreate) {
		gotSession = s
		gotInteraction = i
	})

	handler(session, interaction)

	if gotSession != session {
		t.Error("handler should receive the session")
	}
	if gotInteraction != interaction {
		t.Error("handler should receive the interaction")
	}
}

func TestWithAdmin_ResponseIsEphemeral(t *testing.T) {
	session := &mockDiscordSession{}

	handler := WithAdmin(func(s DiscordSession, i *discordgo.InteractionCreate) {})
	handler(session, interactionWithPermissions(0))

	if session.lastInteractionResponse == nil {
		t.Fatal("expected response to be sent")
	}
	if session.lastInteractionResponse.Data.Flags != discordgo.MessageFlagsEphemeral {
		t.Error("error response should be ephemeral")
	}
}

func TestMiddleware_TypeSignature(t *testing.T) {
	var _ Middleware = WithAdmin
}

func interactionWithPermissions(perms int64) *discordgo.InteractionCreate {
	return &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type:    discordgo.InteractionApplicationCommand,
			GuildID: "test-guild",
			Member: &discordgo.Member{
				Permissions: perms,
			},
		},
	}
}

func assertAdminRequiredResponse(t *testing.T, session *mockDiscordSession) {
	t.Helper()
	if session.lastInteractionResponse == nil {
		t.Fatal("expected error response to be sent")
	}
	if session.lastInteractionResponse.Data.Content != formatting.MsgAdminRequired {
		t.Errorf("expected message %q, got %q",
			formatting.MsgAdminRequired,
			session.lastInteractionResponse.Data.Content)
	}
}
