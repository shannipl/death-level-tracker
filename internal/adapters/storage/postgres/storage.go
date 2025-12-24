package postgres

import (
	"context"
	"fmt"
	"time"

	"death-level-tracker/internal/adapters/storage/postgres/db"
	"death-level-tracker/internal/core/domain"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

func NewPostgresStore(ctx context.Context, connString string) (*PostgresStore, error) {
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return &PostgresStore{
		pool: pool,
		q:    db.New(pool),
	}, nil
}

func (s *PostgresStore) Close() {
	s.pool.Close()
}

// -- Guild Configuration Methods --

func (s *PostgresStore) SaveGuildWorld(ctx context.Context, guildID, world string) error {
	return s.q.SaveGuildWorld(ctx, db.SaveGuildWorldParams{
		GuildID: guildID,
		World:   world,
	})
}

func (s *PostgresStore) GetGuildConfig(ctx context.Context, guildID string) (*domain.GuildConfig, error) {
	row, err := s.q.GetGuildConfig(ctx, guildID)
	if err != nil {
		return nil, fmt.Errorf("get guild config: %w", err)
	}

	return &domain.GuildConfig{
		DiscordGuildID: row.GuildID,
		World:          row.World,
		TibiaGuilds:    row.TibiaGuilds,
	}, nil
}

func (s *PostgresStore) GetAllGuildConfigs(ctx context.Context) ([]domain.GuildConfig, error) {
	rows, err := s.q.GetWorldsMap(ctx)
	if err != nil {
		return nil, fmt.Errorf("get all guild configs: %w", err)
	}

	result := make([]domain.GuildConfig, 0, len(rows))
	for _, row := range rows {
		result = append(result, domain.GuildConfig{
			DiscordGuildID: row.GuildID,
			World:          row.World,
			TibiaGuilds:    row.TibiaGuilds,
		})
	}
	return result, nil
}

func (s *PostgresStore) DeleteGuildConfig(ctx context.Context, guildID string) error {
	return s.q.DeleteGuildConfig(ctx, guildID)
}

func (s *PostgresStore) AddGuildToConfig(ctx context.Context, guildID, tibiaGuild string) error {
	return s.q.AddGuildToConfig(ctx, db.AddGuildToConfigParams{
		GuildID:    guildID,
		TibiaGuild: tibiaGuild,
	})
}

func (s *PostgresStore) RemoveGuildFromConfig(ctx context.Context, guildID, tibiaGuild string) error {
	return s.q.RemoveGuildFromConfig(ctx, db.RemoveGuildFromConfigParams{
		GuildID:    guildID,
		TibiaGuild: tibiaGuild,
	})
}

// -- Player & Level Management Methods --

func (s *PostgresStore) UpsertPlayerLevel(ctx context.Context, name string, level int, world string) error {
	return s.q.UpsertPlayerLevel(ctx, db.UpsertPlayerLevelParams{
		Name:  name,
		Level: int32(level),
		World: world,
	})
}

func (s *PostgresStore) GetPlayersLevels(ctx context.Context, world string) (map[string]int, error) {
	rows, err := s.q.GetPlayersLevels(ctx, world)
	if err != nil {
		return nil, fmt.Errorf("get players levels: %w", err)
	}

	result := make(map[string]int, len(rows))
	for _, row := range rows {
		result[row.Name] = int(row.Level)
	}
	return result, nil
}

func (s *PostgresStore) BatchTouchPlayers(ctx context.Context, names []string) error {
	if len(names) == 0 {
		return nil
	}
	return s.q.BatchTouchPlayers(ctx, names)
}

func (s *PostgresStore) DeleteOldPlayers(ctx context.Context, world string, threshold time.Duration) (int64, error) {
	tag, err := s.q.DeleteOldPlayers(ctx, db.DeleteOldPlayersParams{
		World: world,
		Threshold: pgtype.Interval{
			Microseconds: int64(threshold.Microseconds()),
			Valid:        true,
		},
	})
	if err != nil {
		return 0, fmt.Errorf("delete old players: %w", err)
	}
	return tag.RowsAffected(), nil
}

func (s *PostgresStore) GetOfflinePlayers(ctx context.Context, world string, onlineNames []string) ([]domain.Player, error) {
	rows, err := s.q.GetOfflinePlayers(ctx, db.GetOfflinePlayersParams{
		World:       world,
		OnlineNames: onlineNames,
	})
	if err != nil {
		return nil, fmt.Errorf("get offline players: %w", err)
	}

	result := make([]domain.Player, 0, len(rows))
	for _, row := range rows {
		result = append(result, domain.Player{
			Name:  row.Name,
			Level: int(row.Level),
			World: world,
		})
	}
	return result, nil
}
