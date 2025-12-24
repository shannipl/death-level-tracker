package tracker

import (
	"context"
	"time"

	"death-level-tracker/internal/core/domain"
)

type mockServiceStorage struct {
	getAllGuildConfigsFunc func(ctx context.Context) ([]domain.GuildConfig, error)
	getPlayersLevelsFunc   func(ctx context.Context, world string) (map[string]int, error)
	batchTouchPlayersFunc  func(ctx context.Context, names []string) error
	upsertPlayerLevelFunc  func(ctx context.Context, name string, level int, world string) error
	deleteOldPlayersFunc   func(ctx context.Context, world string, threshold time.Duration) (int64, error)
	getOfflinePlayersFunc  func(ctx context.Context, world string, onlineNames []string) ([]domain.Player, error)
}

func (m *mockServiceStorage) GetAllGuildConfigs(ctx context.Context) ([]domain.GuildConfig, error) {
	if m.getAllGuildConfigsFunc != nil {
		return m.getAllGuildConfigsFunc(ctx)
	}
	return nil, nil
}

func (m *mockServiceStorage) GetPlayersLevels(ctx context.Context, world string) (map[string]int, error) {
	if m.getPlayersLevelsFunc != nil {
		return m.getPlayersLevelsFunc(ctx, world)
	}
	return nil, nil
}

func (m *mockServiceStorage) BatchTouchPlayers(ctx context.Context, names []string) error {
	if m.batchTouchPlayersFunc != nil {
		return m.batchTouchPlayersFunc(ctx, names)
	}
	return nil
}

func (m *mockServiceStorage) DeleteOldPlayers(ctx context.Context, world string, threshold time.Duration) (int64, error) {
	if m.deleteOldPlayersFunc != nil {
		return m.deleteOldPlayersFunc(ctx, world, threshold)
	}
	return 0, nil
}

func (m *mockServiceStorage) SaveGuildWorld(ctx context.Context, guildID, world string) error {
	return nil
}
func (m *mockServiceStorage) UpsertPlayerLevel(ctx context.Context, name string, level int, world string) error {
	if m.upsertPlayerLevelFunc != nil {
		return m.upsertPlayerLevelFunc(ctx, name, level, world)
	}
	return nil
}
func (m *mockServiceStorage) DeleteGuildConfig(ctx context.Context, guildID string) error { return nil }
func (m *mockServiceStorage) AddGuildToConfig(ctx context.Context, guildID, guild string) error {
	return nil
}
func (m *mockServiceStorage) RemoveGuildFromConfig(ctx context.Context, guildID, guild string) error {
	return nil
}
func (m *mockServiceStorage) GetGuildConfig(ctx context.Context, guildID string) (*domain.GuildConfig, error) {
	return nil, nil
}
func (m *mockServiceStorage) GetOfflinePlayers(ctx context.Context, world string, onlineNames []string) ([]domain.Player, error) {
	if m.getOfflinePlayersFunc != nil {
		return m.getOfflinePlayersFunc(ctx, world, onlineNames)
	}
	return nil, nil
}
func (m *mockServiceStorage) Close() {}

type mockServiceFetcher struct {
	fetchWorldFunc             func(ctx context.Context, world string) ([]domain.Player, error)
	fetchCharacterDetailsFunc  func(ctx context.Context, names []string) (chan *domain.Player, error)
	fetchWorldFromTibiaComFunc func(ctx context.Context, world string) (map[string]int, error)
	fetchGuildMembersFunc      func(ctx context.Context, name string) ([]string, error)
	fetchCharacterFunc         func(ctx context.Context, name string) (*domain.Player, error)
}

func (m *mockServiceFetcher) FetchGuildMembers(ctx context.Context, name string) ([]string, error) {
	if m.fetchGuildMembersFunc != nil {
		return m.fetchGuildMembersFunc(ctx, name)
	}
	return nil, nil
}

func (m *mockServiceFetcher) FetchWorld(ctx context.Context, world string) ([]domain.Player, error) {
	if m.fetchWorldFunc != nil {
		return m.fetchWorldFunc(ctx, world)
	}
	return nil, nil
}

func (m *mockServiceFetcher) FetchWorldFromTibiaCom(ctx context.Context, world string) (map[string]int, error) {
	if m.fetchWorldFromTibiaComFunc != nil {
		return m.fetchWorldFromTibiaComFunc(ctx, world)
	}
	return make(map[string]int), nil
}

func (m *mockServiceFetcher) FetchCharacterDetails(ctx context.Context, names []string) (chan *domain.Player, error) {
	if m.fetchCharacterDetailsFunc != nil {
		return m.fetchCharacterDetailsFunc(ctx, names)
	}
	ch := make(chan *domain.Player)
	close(ch)
	return ch, nil
}

func (m *mockServiceFetcher) FetchCharacter(ctx context.Context, name string) (*domain.Player, error) {
	if m.fetchCharacterFunc != nil {
		return m.fetchCharacterFunc(ctx, name)
	}
	return nil, nil
}

type mockServiceNotifier struct {
	sendLevelUpFunc func(guildID string, levelUp domain.LevelUp) error
	sendDeathFunc   func(guildID string, playerName string, kill domain.Kill) error
}

func (m *mockServiceNotifier) SendLevelUpNotification(guildID string, levelUp domain.LevelUp) error {
	if m.sendLevelUpFunc != nil {
		return m.sendLevelUpFunc(guildID, levelUp)
	}
	return nil
}

func (m *mockServiceNotifier) SendDeathNotification(guildID string, playerName string, kill domain.Kill) error {
	if m.sendDeathFunc != nil {
		return m.sendDeathFunc(guildID, playerName, kill)
	}
	return nil
}

func (m *mockServiceNotifier) SendGenericMessage(guildID, channelName, message string) error {
	return nil
}
