package commands

import (
	"log/slog"

	"github.com/bwmarrin/discordgo"
)

type CommandHandler func(s DiscordSession, i *discordgo.InteractionCreate)

type Router struct {
	routes map[string]CommandHandler
}

func NewRouter() *Router {
	slog.Info("Router initialized")
	return &Router{
		routes: make(map[string]CommandHandler),
	}
}

func (r *Router) Register(name string, handler CommandHandler) {
	r.routes[name] = handler
}

func (r *Router) Handle(s DiscordSession, i *discordgo.InteractionCreate) {
	if !isCommandInteraction(i.Type) {
		return
	}

	name := i.ApplicationCommandData().Name
	slog.Info("Router received interaction", "type", i.Type, "name", name)

	handler, ok := r.routes[name]
	if !ok {
		slog.Warn("No handler found for command", "name", name)
		return
	}

	handler(s, i)
}

func (r *Router) HandleFunc() func(*discordgo.Session, *discordgo.InteractionCreate) {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		r.Handle(s, i)
	}
}

func isCommandInteraction(t discordgo.InteractionType) bool {
	return t == discordgo.InteractionApplicationCommand ||
		t == discordgo.InteractionApplicationCommandAutocomplete
}
