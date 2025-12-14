package storage

import (
	"context"
	"fmt"
	"time"

	"death-level-tracker/internal/storage/db"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Storage interface {
	SaveGuildWorld(ctx context.Context, guildID, world string) error
	GetWorldsMap(ctx context.Context) (map[string][]string, error)
	GetPlayersLevels(ctx context.Context, world string) (map[string]int, error)
	GetOfflinePlayers(ctx context.Context, world string, onlineNames []string) ([]OfflinePlayer, error)
	UpsertPlayerLevel(ctx context.Context, name string, level int, world string) error
	BatchTouchPlayers(ctx context.Context, names []string) error
	DeleteOldPlayers(ctx context.Context, world string, threshold time.Duration) (int64, error)
	DeleteGuildConfig(ctx context.Context, guildID string) error
	Close()
}

type OfflinePlayer struct {
	Name  string
	Level int
}

type PostgresStore struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

func NewPostgresStore(ctx context.Context, connString string) (*PostgresStore, error) {
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	store := &PostgresStore{
		pool: pool,
		q:    db.New(pool),
	}

	return store, nil
}

func (s *PostgresStore) SaveGuildWorld(ctx context.Context, guildID, world string) error {
	return s.q.SaveGuildWorld(ctx, db.SaveGuildWorldParams{
		GuildID: guildID,
		World:   world,
	})
}

func (s *PostgresStore) GetWorldsMap(ctx context.Context) (map[string][]string, error) {
	rows, err := s.q.GetWorldsMap(ctx)
	if err != nil {
		return nil, err
	}

	result := make(map[string][]string)
	for _, row := range rows {
		result[row.World] = append(result[row.World], row.GuildID)
	}

	return result, nil
}

func (s *PostgresStore) GetPlayersLevels(ctx context.Context, world string) (map[string]int, error) {
	rows, err := s.q.GetPlayersLevels(ctx, world)
	if err != nil {
		return nil, err
	}

	result := make(map[string]int)
	for _, row := range rows {
		result[row.Name] = int(row.Level)
	}
	return result, nil
}

func (s *PostgresStore) UpsertPlayerLevel(ctx context.Context, name string, level int, world string) error {
	return s.q.UpsertPlayerLevel(ctx, db.UpsertPlayerLevelParams{
		Name:  name,
		Level: int32(level),
		World: world,
	})
}

func (s *PostgresStore) BatchTouchPlayers(ctx context.Context, names []string) error {
	if len(names) == 0 {
		return nil
	}
	// sqlc expects []string for text[]
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
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func (s *PostgresStore) GetOfflinePlayers(ctx context.Context, world string, onlineNames []string) ([]OfflinePlayer, error) {
	rows, err := s.q.GetOfflinePlayers(ctx, db.GetOfflinePlayersParams{
		World:       world,
		OnlineNames: onlineNames,
	})
	if err != nil {
		return nil, err
	}

	result := make([]OfflinePlayer, 0, len(rows))
	for _, row := range rows {
		result = append(result, OfflinePlayer{
			Name:  row.Name,
			Level: int(row.Level),
		})
	}
	return result, nil
}

func (s *PostgresStore) DeleteGuildConfig(ctx context.Context, guildID string) error {
	return s.q.DeleteGuildConfig(ctx, guildID)
}

func (s *PostgresStore) Close() {
	s.pool.Close()
}
