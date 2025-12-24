package ports

import (
	"context"
	"death-level-tracker/internal/core/domain"
	"time"
)

type Repository interface {
	SaveGuildWorld(ctx context.Context, discordGuildID, world string) error
	GetGuildConfig(ctx context.Context, discordGuildID string) (*domain.GuildConfig, error)
	GetAllGuildConfigs(ctx context.Context) ([]domain.GuildConfig, error)
	DeleteGuildConfig(ctx context.Context, discordGuildID string) error
	AddGuildToConfig(ctx context.Context, discordGuildID, guildName string) error
	RemoveGuildFromConfig(ctx context.Context, discordGuildID, guildName string) error

	UpsertPlayerLevel(ctx context.Context, name string, level int, world string) error
	GetPlayersLevels(ctx context.Context, world string) (map[string]int, error)
	GetOfflinePlayers(ctx context.Context, world string, onlineNames []string) ([]domain.Player, error)

	BatchTouchPlayers(ctx context.Context, names []string) error
	DeleteOldPlayers(ctx context.Context, world string, maxAge time.Duration) (int64, error)
	Close()
}

type TibiaFetcher interface {
	FetchWorld(ctx context.Context, world string) ([]domain.Player, error)
	FetchGuildMembers(ctx context.Context, guildName string) ([]string, error)
	FetchCharacterDetails(ctx context.Context, names []string) (chan *domain.Player, error)
	FetchCharacter(ctx context.Context, name string) (*domain.Player, error)
	FetchWorldFromTibiaCom(ctx context.Context, world string) (map[string]int, error)
}

type NotificationService interface {
	SendLevelUpNotification(guildID string, levelUp domain.LevelUp) error
	SendDeathNotification(guildID string, playerName string, kill domain.Kill) error
	SendGenericMessage(guildID string, channelName string, message string) error
}
