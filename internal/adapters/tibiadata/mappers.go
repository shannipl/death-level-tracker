package tibiadata

import (
	"death-level-tracker/internal/adapters/tibiadata/api"
	"death-level-tracker/internal/core/domain"
)

func (a *Adapter) mapCharacter(char *api.CharacterResponse) *domain.Player {
	if char == nil || char.Character.Character.Name == "" {
		return nil
	}

	c := char.Character.Character

	var deaths []domain.Kill
	for _, d := range char.Character.Deaths {
		deaths = append(deaths, domain.Kill{
			Time:   d.Time,
			Level:  d.Level,
			Reason: d.Reason,
		})
	}

	return &domain.Player{
		Name:     c.Name,
		Level:    c.Level,
		World:    c.World,
		Vocation: c.Vocation,
		Deaths:   deaths,
	}
}
