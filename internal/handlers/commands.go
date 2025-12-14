package handlers

import (
	"context"
	"log/slog"
	"strings"

	"death-level-tracker/internal/config"
	"death-level-tracker/internal/formatting"
	"death-level-tracker/internal/storage"

	"github.com/bwmarrin/discordgo"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type BotHandler struct {
	Config *config.Config
	Store  storage.Storage
}

func ReadyHandler(session *discordgo.Session, ready *discordgo.Ready) {
	slog.Info("Death Level Tracker is online!", "user", session.State.User.Username, "discriminator", session.State.User.Discriminator)
}

func (h *BotHandler) TrackWorld(s DiscordSession, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options

	var worldName string
	for _, opt := range options {
		if opt.Name == "name" {
			worldName = opt.StringValue()
		}
	}

	if worldName == "" {
		respond(s, i, formatting.MsgWorldRequired, true)
		return
	}

	if _, err := ensureChannel(s, i.GuildID, h.Config.DiscordChannelDeath); err != nil {
		slog.Error("Failed to ensure death-level-tracker channel", "error", err)
		respond(s, i, formatting.MsgChannelError(h.Config.DiscordChannelDeath), true)
		return
	}

	if _, err := ensureChannel(s, i.GuildID, h.Config.DiscordChannelLevel); err != nil {
		slog.Error("Failed to ensure level-tracker channel", "error", err)
		respond(s, i, formatting.MsgChannelError(h.Config.DiscordChannelLevel), true)
		return
	}

	formattedWorld := cases.Title(language.English).String(strings.ToLower(worldName))

	if err := h.Store.SaveGuildWorld(context.Background(), i.GuildID, formattedWorld); err != nil {
		slog.Error("Failed to save world", "error", err)
		respond(s, i, formatting.MsgSaveError, true)
		return
	}

	respond(s, i, formatting.MsgTrackSuccess(formattedWorld, h.Config.DiscordChannelDeath, h.Config.DiscordChannelLevel), false)
}

func (h *BotHandler) StopTracking(s DiscordSession, i *discordgo.InteractionCreate) {
	if err := h.Store.DeleteGuildConfig(context.Background(), i.GuildID); err != nil {
		slog.Error("Failed to delete guild config", "guild_id", i.GuildID, "error", err)
		respond(s, i, formatting.MsgStopError, true)
		return
	}

	respond(s, i, formatting.MsgStopSuccess, false)
}

func ensureChannel(s DiscordSession, guildID, name string) (string, error) {
	channels, err := s.GuildChannels(guildID)
	if err != nil {
		return "", err
	}

	for _, ch := range channels {
		if ch.Name == name && ch.Type == discordgo.ChannelTypeGuildText {
			return ch.ID, nil
		}
	}

	ch, err := s.GuildChannelCreate(guildID, name, discordgo.ChannelTypeGuildText)
	if err != nil {
		return "", err
	}
	return ch.ID, nil
}

func respond(s DiscordSession, i *discordgo.InteractionCreate, msg string, ephemeral bool) {
	var flags discordgo.MessageFlags
	if ephemeral {
		flags = discordgo.MessageFlagsEphemeral
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg,
			Flags:   flags,
		},
	})
}
