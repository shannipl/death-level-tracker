package api

import "time"

type WorldResponse struct {
	World struct {
		OnlinePlayers []OnlinePlayer `json:"online_players"`
	} `json:"world"`
}

type OnlinePlayer struct {
	Name     string `json:"name"`
	Level    int    `json:"level"`
	Vocation string `json:"vocation"`
}

type CharacterResponse struct {
	Character struct {
		Character CharacterInfo `json:"character"`
		Deaths    []Death       `json:"deaths"`
	} `json:"character"`
}

type CharacterInfo struct {
	Name     string `json:"name"`
	Level    int    `json:"level"`
	Vocation string `json:"vocation"`
	World    string `json:"world"`
}

type Death struct {
	Time   time.Time `json:"time"`
	Level  int       `json:"level"`
	Reason string    `json:"reason"`
}

type GuildResponse struct {
	Guild GuildInfo `json:"guild"`
}

type GuildInfo struct {
	Name    string        `json:"name"`
	Members []GuildMember `json:"members"`
}

type GuildMember struct {
	Name     string `json:"name"`
	Level    int    `json:"level"`
	Vocation string `json:"vocation"`
	Rank     string `json:"rank"`
	Status   string `json:"status"`
}
