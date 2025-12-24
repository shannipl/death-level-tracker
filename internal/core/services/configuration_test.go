package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"death-level-tracker/internal/core/domain"
)

type mockRepository struct {
	saveGuildWorldFunc        func(ctx context.Context, guildID, world string) error
	deleteGuildConfigFunc     func(ctx context.Context, guildID string) error
	getGuildConfigFunc        func(ctx context.Context, guildID string) (*domain.GuildConfig, error)
	addGuildToConfigFunc      func(ctx context.Context, guildID, guildName string) error
	removeGuildFromConfigFunc func(ctx context.Context, guildID, guildName string) error
}

func (m *mockRepository) SaveGuildWorld(ctx context.Context, guildID, world string) error {
	if m.saveGuildWorldFunc != nil {
		return m.saveGuildWorldFunc(ctx, guildID, world)
	}
	return nil
}

func (m *mockRepository) DeleteGuildConfig(ctx context.Context, guildID string) error {
	if m.deleteGuildConfigFunc != nil {
		return m.deleteGuildConfigFunc(ctx, guildID)
	}
	return nil
}

func (m *mockRepository) GetGuildConfig(ctx context.Context, guildID string) (*domain.GuildConfig, error) {
	if m.getGuildConfigFunc != nil {
		return m.getGuildConfigFunc(ctx, guildID)
	}
	return nil, nil
}

func (m *mockRepository) AddGuildToConfig(ctx context.Context, guildID, guildName string) error {
	if m.addGuildToConfigFunc != nil {
		return m.addGuildToConfigFunc(ctx, guildID, guildName)
	}
	return nil
}

func (m *mockRepository) RemoveGuildFromConfig(ctx context.Context, guildID, guildName string) error {
	if m.removeGuildFromConfigFunc != nil {
		return m.removeGuildFromConfigFunc(ctx, guildID, guildName)
	}
	return nil
}

func (m *mockRepository) GetAllGuildConfigs(ctx context.Context) ([]domain.GuildConfig, error) {
	return nil, nil
}

func (m *mockRepository) UpsertPlayerLevel(ctx context.Context, name string, level int, world string) error {
	return nil
}

func (m *mockRepository) GetPlayersLevels(ctx context.Context, world string) (map[string]int, error) {
	return nil, nil
}

func (m *mockRepository) GetOfflinePlayers(ctx context.Context, world string, onlineNames []string) ([]domain.Player, error) {
	return nil, nil
}

func (m *mockRepository) BatchTouchPlayers(ctx context.Context, names []string) error {
	return nil
}

func (m *mockRepository) DeleteOldPlayers(ctx context.Context, world string, maxAge time.Duration) (int64, error) {
	return 0, nil
}

func (m *mockRepository) Close() {}

func TestSetWorld_Success(t *testing.T) {
	var savedWorld string
	repo := &mockRepository{
		saveGuildWorldFunc: func(ctx context.Context, guildID, world string) error {
			savedWorld = world
			return nil
		},
	}

	svc := NewConfigurationService(repo)
	result, err := svc.SetWorld(context.Background(), "guild-1", "antica")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Antica" {
		t.Errorf("expected 'Antica', got '%s'", result)
	}
	if savedWorld != "Antica" {
		t.Errorf("expected saved 'Antica', got '%s'", savedWorld)
	}
}

func TestSetWorld_Formatting(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"antica", "Antica"},
		{"SECURA", "Secura"},
		{"beLaBonA", "Belabona"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			svc := NewConfigurationService(&mockRepository{})
			result, _ := svc.SetWorld(context.Background(), "guild-1", tt.input)

			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestSetWorld_Error(t *testing.T) {
	repo := &mockRepository{
		saveGuildWorldFunc: func(ctx context.Context, guildID, world string) error {
			return errors.New("db error")
		},
	}

	svc := NewConfigurationService(repo)
	_, err := svc.SetWorld(context.Background(), "guild-1", "antica")

	if err == nil {
		t.Error("expected error")
	}
}

func TestStopTracking_Success(t *testing.T) {
	var deletedGuildID string
	repo := &mockRepository{
		deleteGuildConfigFunc: func(ctx context.Context, guildID string) error {
			deletedGuildID = guildID
			return nil
		},
	}

	svc := NewConfigurationService(repo)
	err := svc.StopTracking(context.Background(), "guild-123")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deletedGuildID != "guild-123" {
		t.Errorf("expected 'guild-123', got '%s'", deletedGuildID)
	}
}

func TestStopTracking_Error(t *testing.T) {
	repo := &mockRepository{
		deleteGuildConfigFunc: func(ctx context.Context, guildID string) error {
			return errors.New("db error")
		},
	}

	svc := NewConfigurationService(repo)
	err := svc.StopTracking(context.Background(), "guild-1")

	if err == nil {
		t.Error("expected error")
	}
}

func TestAddGuildToTrack_Success(t *testing.T) {
	var addedGuild string
	repo := &mockRepository{
		addGuildToConfigFunc: func(ctx context.Context, guildID, guildName string) error {
			addedGuild = guildName
			return nil
		},
	}

	svc := NewConfigurationService(repo)
	err := svc.AddGuildToTrack(context.Background(), "guild-1", "Red Rose")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if addedGuild != "Red Rose" {
		t.Errorf("expected 'Red Rose', got '%s'", addedGuild)
	}
}

func TestAddGuildToTrack_Error(t *testing.T) {
	repo := &mockRepository{
		addGuildToConfigFunc: func(ctx context.Context, guildID, guildName string) error {
			return errors.New("db error")
		},
	}

	svc := NewConfigurationService(repo)
	err := svc.AddGuildToTrack(context.Background(), "guild-1", "Test")

	if err == nil {
		t.Error("expected error")
	}
}

func TestRemoveGuildFromTrack_Success(t *testing.T) {
	var removedGuild string
	repo := &mockRepository{
		removeGuildFromConfigFunc: func(ctx context.Context, guildID, guildName string) error {
			removedGuild = guildName
			return nil
		},
	}

	svc := NewConfigurationService(repo)
	err := svc.RemoveGuildFromTrack(context.Background(), "guild-1", "Red Rose")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if removedGuild != "Red Rose" {
		t.Errorf("expected 'Red Rose', got '%s'", removedGuild)
	}
}

func TestRemoveGuildFromTrack_Error(t *testing.T) {
	repo := &mockRepository{
		removeGuildFromConfigFunc: func(ctx context.Context, guildID, guildName string) error {
			return errors.New("db error")
		},
	}

	svc := NewConfigurationService(repo)
	err := svc.RemoveGuildFromTrack(context.Background(), "guild-1", "Test")

	if err == nil {
		t.Error("expected error")
	}
}

func TestGetGuildConfig_Success(t *testing.T) {
	expected := &domain.GuildConfig{
		DiscordGuildID: "guild-1",
		World:          "Antica",
		TibiaGuilds:    []string{"Red Rose"},
	}

	repo := &mockRepository{
		getGuildConfigFunc: func(ctx context.Context, guildID string) (*domain.GuildConfig, error) {
			return expected, nil
		},
	}

	svc := NewConfigurationService(repo)
	result, err := svc.GetGuildConfig(context.Background(), "guild-1")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.World != "Antica" {
		t.Errorf("expected 'Antica', got '%s'", result.World)
	}
}

func TestGetGuildConfig_NotFound(t *testing.T) {
	repo := &mockRepository{
		getGuildConfigFunc: func(ctx context.Context, guildID string) (*domain.GuildConfig, error) {
			return nil, nil
		},
	}

	svc := NewConfigurationService(repo)
	result, err := svc.GetGuildConfig(context.Background(), "guild-1")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Error("expected nil result")
	}
}

func TestGetGuildConfig_Error(t *testing.T) {
	repo := &mockRepository{
		getGuildConfigFunc: func(ctx context.Context, guildID string) (*domain.GuildConfig, error) {
			return nil, errors.New("db error")
		},
	}

	svc := NewConfigurationService(repo)
	_, err := svc.GetGuildConfig(context.Background(), "guild-1")

	if err == nil {
		t.Error("expected error")
	}
}
