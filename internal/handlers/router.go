package handlers

import "github.com/bwmarrin/discordgo"

type CommandHandler func(s DiscordSession, i *discordgo.InteractionCreate)

type Router struct {
	routes map[string]CommandHandler
}

func NewRouter() *Router {
	return &Router{
		routes: make(map[string]CommandHandler),
	}
}

func (r *Router) Register(command string, handler CommandHandler) {
	r.routes[command] = handler
}

// Handle processes interactions using the DiscordSession interface (for testing).
func (r *Router) Handle(s DiscordSession, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	data := i.ApplicationCommandData()
	if handler, ok := r.routes[data.Name]; ok {
		handler(s, i)
	}
}

// HandleFunc returns a function compatible with discordgo.AddHandler.
func (r *Router) HandleFunc() func(*discordgo.Session, *discordgo.InteractionCreate) {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		r.Handle(s, i)
	}
}
