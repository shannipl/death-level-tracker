package tibiadata

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
	Name  string `json:"name"`
	Level int    `json:"level"`
	World string `json:"world"`
}

type Death struct {
	Time   time.Time `json:"time"`
	Level  int       `json:"level"`
	Reason string    `json:"reason"`
}
