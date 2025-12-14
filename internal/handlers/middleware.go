package handlers

import (
	"death-level-tracker/internal/formatting"

	"github.com/bwmarrin/discordgo"
)

type Middleware func(CommandHandler) CommandHandler

func WithAdmin(next CommandHandler) CommandHandler {
	return func(s DiscordSession, i *discordgo.InteractionCreate) {
		if i.Member == nil || i.Member.Permissions&discordgo.PermissionAdministrator == 0 {
			respond(s, i, formatting.MsgAdminRequired, true)
			return
		}
		next(s, i)
	}
}
