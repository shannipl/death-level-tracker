package domain

import "time"

type World struct {
	Name string
}

type Guild struct {
	Name string
}

type Player struct {
	Name     string
	Level    int
	Vocation string
	World    string
	Deaths   []Kill
}

type Kill struct {
	ID       string
	Time     time.Time
	Level    int
	Reason   string
	Involved []Killer
}

type Killer struct {
	Name     string
	IsPlayer bool
	IsSummon bool
}

type LevelUp struct {
	PlayerName string
	OldLevel   int
	NewLevel   int
	World      string
}

type GuildConfig struct {
	DiscordGuildID string
	World          string
	TibiaGuilds    []string
}
