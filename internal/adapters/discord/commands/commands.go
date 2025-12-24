package commands

import (
	"context"
	"log/slog"
	"strings"

	"death-level-tracker/internal/adapters/discord/formatting"
	"death-level-tracker/internal/config"
	"death-level-tracker/internal/core/domain"
	"death-level-tracker/internal/core/services"

	"github.com/bwmarrin/discordgo"
)

type BotHandler struct {
	Config  *config.Config
	Service *services.ConfigurationService
}

func ReadyHandler(session *discordgo.Session, ready *discordgo.Ready) {
	slog.Info("Death Level Tracker is online!", "user", session.State.User.Username, "discriminator", session.State.User.Discriminator)
}

func (h *BotHandler) TrackWorld(s DiscordSession, i *discordgo.InteractionCreate) {
	worldName := getStringOption(i.ApplicationCommandData().Options, "name")
	if worldName == "" {
		respond(s, i, formatting.MsgWorldRequired, true)
		return
	}

	if _, err := ensureChannel(s, i.GuildID, h.Config.DiscordChannelDeath); err != nil {
		slog.Error("Failed to ensure death-tracker channel", "error", err)
		respond(s, i, formatting.MsgChannelError(h.Config.DiscordChannelDeath), true)
		return
	}

	if _, err := ensureChannel(s, i.GuildID, h.Config.DiscordChannelLevel); err != nil {
		slog.Error("Failed to ensure level-tracker channel", "error", err)
		respond(s, i, formatting.MsgChannelError(h.Config.DiscordChannelLevel), true)
		return
	}

	formattedWorld, err := h.Service.SetWorld(context.Background(), i.GuildID, worldName)
	if err != nil {
		slog.Error("Failed to save world", "error", err)
		respond(s, i, formatting.MsgSaveError, true)
		return
	}

	respond(s, i, formatting.MsgTrackSuccess(formattedWorld, h.Config.DiscordChannelDeath, h.Config.DiscordChannelLevel), false)
}

func (h *BotHandler) StopTracking(s DiscordSession, i *discordgo.InteractionCreate) {
	if err := h.Service.StopTracking(context.Background(), i.GuildID); err != nil {
		slog.Error("Failed to delete guild config", "guild_id", i.GuildID, "error", err)
		respond(s, i, formatting.MsgStopError, true)
		return
	}

	respond(s, i, formatting.MsgStopSuccess, false)
}

func (h *BotHandler) AddGuild(s DiscordSession, i *discordgo.InteractionCreate) {
	guildName := getStringOption(i.ApplicationCommandData().Options, "name")
	if guildName == "" {
		respond(s, i, formatting.MsgGuildNameRequired, true)
		return
	}

	if err := h.Service.AddGuildToTrack(context.Background(), i.GuildID, guildName); err != nil {
		slog.Error("Failed to add guild", "error", err)
		respond(s, i, formatting.MsgSaveError, true)
		return
	}

	respond(s, i, formatting.MsgGuildAdded(guildName), false)
}

func (h *BotHandler) UnsetGuild(s DiscordSession, i *discordgo.InteractionCreate) {
	if i.Type == discordgo.InteractionApplicationCommandAutocomplete {
		h.handleGuildAutocomplete(s, i)
		return
	}

	guildName := getStringOption(i.ApplicationCommandData().Options, "name")
	if guildName == "" {
		respond(s, i, formatting.MsgGuildNameRequired, true)
		return
	}

	if err := h.Service.RemoveGuildFromTrack(context.Background(), i.GuildID, guildName); err != nil {
		slog.Error("Failed to remove guild", "error", err)
		respond(s, i, formatting.MsgSaveError, true)
		return
	}

	respond(s, i, formatting.MsgGuildRemoved(guildName), false)
}

func (h *BotHandler) handleGuildAutocomplete(s DiscordSession, i *discordgo.InteractionCreate) {
	query := getFocusedOption(i.ApplicationCommandData().Options)

	cfg, err := h.Service.GetGuildConfig(context.Background(), i.GuildID)
	if err != nil {
		slog.Error("Failed to fetch guild config for autocomplete", "error", err)
		return
	}

	choices := buildGuildChoices(cfg, query)
	if err := respondAutocomplete(s, i, choices); err != nil {
		slog.Error("Failed to send autocomplete response", "error", err)
	}
}

func (h *BotHandler) ListGuilds(s DiscordSession, i *discordgo.InteractionCreate) {
	cfg, err := h.Service.GetGuildConfig(context.Background(), i.GuildID)
	if err != nil {
		slog.Error("Failed to get guild config", "error", err)
		respond(s, i, formatting.MsgConfigError, true)
		return
	}

	if cfg == nil || len(cfg.TibiaGuilds) == 0 {
		respond(s, i, formatting.MsgNoGuildsTracked, false)
		return
	}

	respond(s, i, formatting.MsgGuildsList(cfg.TibiaGuilds), false)
}

func buildGuildChoices(cfg *domain.GuildConfig, query string) []*discordgo.ApplicationCommandOptionChoice {
	if cfg == nil {
		return nil
	}

	var choices []*discordgo.ApplicationCommandOptionChoice
	for _, g := range cfg.TibiaGuilds {
		if strings.Contains(strings.ToLower(g), strings.ToLower(query)) {
			choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
				Name:  g,
				Value: g,
			})
		}
		if len(choices) >= 25 {
			break
		}
	}
	return choices
}
