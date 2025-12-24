package tibiadata

import (
	"context"
)

// FetchGuildMembers gets all members of a guild.
func (a *Adapter) FetchGuildMembers(ctx context.Context, name string) ([]string, error) {
	guild, err := a.client.GetGuild(name)
	if err != nil {
		return nil, err
	}

	members := make([]string, len(guild.Guild.Members))
	for i, m := range guild.Guild.Members {
		members[i] = m.Name
	}
	return members, nil
}
