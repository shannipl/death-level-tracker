package postgres

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"death-level-tracker/internal/adapters/storage/postgres/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestPostgresStore_SaveGuildWorld(t *testing.T) {
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		mockDB := &MockDB{
			ExecFunc: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
				if len(args) != 2 {
					return pgconn.CommandTag{}, fmt.Errorf("expected 2 args, got %d", len(args))
				}
				if args[0] != "guild123" || args[1] != "Antica" {
					return pgconn.CommandTag{}, fmt.Errorf("unexpected args: %v", args)
				}
				return pgconn.NewCommandTag("INSERT 1"), nil
			},
		}

		store := &PostgresStore{q: db.New(mockDB)}
		err := store.SaveGuildWorld(ctx, "guild123", "Antica")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
	})

	t.Run("Error", func(t *testing.T) {
		mockDB := &MockDB{
			ExecFunc: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
				return pgconn.CommandTag{}, errors.New("db error")
			},
		}

		store := &PostgresStore{q: db.New(mockDB)}
		err := store.SaveGuildWorld(ctx, "guild123", "Antica")
		if err == nil {
			t.Fatal("Expected error, got nil")
		}
	})
}

func TestPostgresStore_GetGuildConfig(t *testing.T) {
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		mockDB := &MockDB{
			QueryRowFunc: func(ctx context.Context, sql string, args ...any) pgx.Row {
				return &MockRow{
					ScanFunc: func(dest ...any) error {
						// dest[0] = guild_id, dest[1] = world, dest[2] = tibia_guilds
						if len(dest) < 3 {
							return fmt.Errorf("scan expected 3 args")
						}
						// Assign values to pointers in dest
						*dest[0].(*string) = "guild123"
						*dest[1].(*string) = "Antica"
						*dest[2].(*[]string) = []string{"Red Rose"}
						return nil
					},
				}
			},
		}

		store := &PostgresStore{q: db.New(mockDB)}
		cfg, err := store.GetGuildConfig(ctx, "guild123")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if cfg.DiscordGuildID != "guild123" {
			t.Errorf("Expected guild123, got %s", cfg.DiscordGuildID)
		}
		if len(cfg.TibiaGuilds) != 1 || cfg.TibiaGuilds[0] != "Red Rose" {
			t.Errorf("Unexpected tibia guilds: %v", cfg.TibiaGuilds)
		}
	})

	t.Run("Not Found", func(t *testing.T) {
		mockDB := &MockDB{
			QueryRowFunc: func(ctx context.Context, sql string, args ...any) pgx.Row {
				return &MockRow{
					ScanFunc: func(dest ...any) error {
						return errors.New("no rows in result set") // mimicking pgx.ErrNoRows behavior conceptually
					},
				}
			},
		}

		store := &PostgresStore{q: db.New(mockDB)}
		_, err := store.GetGuildConfig(ctx, "unknown")
		if err == nil {
			t.Fatal("Expected error, got nil")
		}
	})
}

func TestPostgresStore_UpsertPlayerLevel(t *testing.T) {
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		mockDB := &MockDB{
			ExecFunc: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
				return pgconn.NewCommandTag("INSERT 1"), nil
			},
		}

		store := &PostgresStore{q: db.New(mockDB)}
		err := store.UpsertPlayerLevel(ctx, "Player1", 100, "Antica")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
	})
}

func TestPostgresStore_GetAllGuildConfigs(t *testing.T) {
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		mockDB := &MockDB{
			QueryFunc: func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
				count := 0
				return &MockRows{
					NextFunc: func() bool {
						count++
						return count <= 2
					},
					ScanFunc: func(dest ...any) error {
						// Assuming schema: guild_id, world, tibia_guilds
						*dest[0].(*string) = fmt.Sprintf("guild%d", count)
						*dest[1].(*string) = "Antica"
						*dest[2].(*[]string) = []string{}
						return nil
					},
				}, nil
			},
		}

		store := &PostgresStore{q: db.New(mockDB)}
		configs, err := store.GetAllGuildConfigs(ctx)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if len(configs) != 2 {
			t.Errorf("Expected 2 configs, got %d", len(configs))
		}
	})

	t.Run("Error", func(t *testing.T) {
		mockDB := &MockDB{
			QueryFunc: func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
				return nil, errors.New("db error")
			},
		}
		store := &PostgresStore{q: db.New(mockDB)}
		_, err := store.GetAllGuildConfigs(ctx)
		if err == nil {
			t.Fatal("Expected error, got nil")
		}
	})
}

func TestPostgresStore_GetPlayersLevels(t *testing.T) {
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		mockDB := &MockDB{
			QueryFunc: func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
				count := 0
				return &MockRows{
					NextFunc: func() bool {
						count++
						return count <= 1
					},
					ScanFunc: func(dest ...any) error {
						*dest[0].(*string) = "Player1"
						*dest[1].(*int32) = 150
						return nil
					},
				}, nil
			},
		}

		store := &PostgresStore{q: db.New(mockDB)}
		levels, err := store.GetPlayersLevels(ctx, "Antica")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if len(levels) != 1 {
			t.Errorf("Expected 1 player, got %d", len(levels))
		}
		if levels["Player1"] != 150 {
			t.Errorf("Expected level 150, got %d", levels["Player1"])
		}
	})

	t.Run("Error", func(t *testing.T) {
		mockDB := &MockDB{
			QueryFunc: func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
				return nil, errors.New("db error")
			},
		}
		store := &PostgresStore{q: db.New(mockDB)}
		_, err := store.GetPlayersLevels(ctx, "Antica")
		if err == nil {
			t.Fatal("Expected error, got nil")
		}
	})
}

func TestPostgresStore_DeleteOldPlayers(t *testing.T) {
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		mockDB := &MockDB{
			ExecFunc: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
				return pgconn.NewCommandTag("DELETE 5"), nil
			},
		}

		store := &PostgresStore{q: db.New(mockDB)}
		deleted, err := store.DeleteOldPlayers(ctx, "Antica", 24*time.Hour)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if deleted != 5 {
			t.Errorf("Expected 5 deleted rows, got %d", deleted)
		}
	})

	t.Run("Error", func(t *testing.T) {
		mockDB := &MockDB{
			ExecFunc: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
				return pgconn.CommandTag{}, errors.New("db error")
			},
		}
		store := &PostgresStore{q: db.New(mockDB)}
		_, err := store.DeleteOldPlayers(ctx, "Antica", 24*time.Hour)
		if err == nil {
			t.Fatal("Expected error, got nil")
		}
	})
}

func TestPostgresStore_BatchTouchPlayers(t *testing.T) {
	ctx := context.Background()

	t.Run("Empty", func(t *testing.T) {
		store := &PostgresStore{q: db.New(&MockDB{})}
		err := store.BatchTouchPlayers(ctx, []string{})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
	})

	t.Run("Success", func(t *testing.T) {
		mockDB := &MockDB{
			ExecFunc: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
				return pgconn.NewCommandTag("UPDATE 10"), nil
			},
		}

		store := &PostgresStore{q: db.New(mockDB)}
		err := store.BatchTouchPlayers(ctx, []string{"A", "B"})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
	})
}

func TestPostgresStore_ManageGuildConfig(t *testing.T) {
	ctx := context.Background()

	t.Run("DeleteGuildConfig", func(t *testing.T) {
		mockDB := &MockDB{
			ExecFunc: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
				if len(args) != 1 || args[0] != "guild1" {
					return pgconn.CommandTag{}, fmt.Errorf("unexpected args")
				}
				return pgconn.NewCommandTag("DELETE 1"), nil
			},
		}
		store := &PostgresStore{q: db.New(mockDB)}
		if err := store.DeleteGuildConfig(ctx, "guild1"); err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	t.Run("AddGuildToConfig", func(t *testing.T) {
		mockDB := &MockDB{
			ExecFunc: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
				if len(args) != 2 {
					return pgconn.CommandTag{}, fmt.Errorf("unexpected args count")
				}
				return pgconn.NewCommandTag("INSERT 1"), nil
			},
		}
		store := &PostgresStore{q: db.New(mockDB)}
		if err := store.AddGuildToConfig(ctx, "guild1", "Red Rose"); err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	t.Run("RemoveGuildFromConfig", func(t *testing.T) {
		mockDB := &MockDB{
			ExecFunc: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
				return pgconn.NewCommandTag("DELETE 1"), nil
			},
		}
		store := &PostgresStore{q: db.New(mockDB)}
		if err := store.RemoveGuildFromConfig(ctx, "guild1", "Red Rose"); err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})
}

func TestPostgresStore_GetOfflinePlayers(t *testing.T) {
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		mockDB := &MockDB{
			QueryFunc: func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
				count := 0
				return &MockRows{
					NextFunc: func() bool {
						count++
						return count <= 1
					},
					ScanFunc: func(dest ...any) error {
						// name, level
						*dest[0].(*string) = "OfflinePlayer"
						*dest[1].(*int32) = 50
						return nil
					},
				}, nil
			},
		}

		store := &PostgresStore{q: db.New(mockDB)}
		players, err := store.GetOfflinePlayers(ctx, "Antica", []string{"OnlinePlayer"})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if len(players) != 1 {
			t.Errorf("Expected 1 player, got %d", len(players))
		}
		if players[0].Name != "OfflinePlayer" {
			t.Errorf("Expected OfflinePlayer, got %s", players[0].Name)
		}
	})

	t.Run("Error", func(t *testing.T) {
		mockDB := &MockDB{
			QueryFunc: func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
				return nil, errors.New("db error")
			},
		}
		store := &PostgresStore{q: db.New(mockDB)}
		_, err := store.GetOfflinePlayers(ctx, "Antica", []string{"OnlinePlayer"})
		if err == nil {
			t.Fatal("Expected error, got nil")
		}
	})
}
