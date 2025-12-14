package tracker

import (
	"context"
	"errors"
	"testing"
	"time"

	"death-level-tracker/internal/tibiadata"
)

func TestService_FetchPlayerLevels_Success(t *testing.T) {
	expectedLevels := map[string]int{
		"Player1": 100,
		"Player2": 200,
	}

	mockStorage := &mockServiceStorage{
		getPlayersLevelsFunc: func(ctx context.Context, world string) (map[string]int, error) {
			return expectedLevels, nil
		},
	}

	service := &Service{
		storage: mockStorage,
	}

	levels, err := service.fetchPlayerLevels("Antica")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(levels) != 2 {
		t.Errorf("Expected 2 player levels, got %d", len(levels))
	}

	if levels["Player1"] != 100 {
		t.Errorf("Expected Player1 level 100, got %d", levels["Player1"])
	}
}

func TestService_FetchPlayerLevels_Error(t *testing.T) {
	mockStorage := &mockServiceStorage{
		getPlayersLevelsFunc: func(ctx context.Context, world string) (map[string]int, error) {
			return nil, errors.New("database error")
		},
	}

	service := &Service{
		storage: mockStorage,
	}

	levels, err := service.fetchPlayerLevels("Antica")

	if err == nil {
		t.Error("Expected error, got nil")
	}

	if levels != nil {
		t.Errorf("Expected nil levels on error, got %v", levels)
	}
}

func TestService_ProcessCharacters(t *testing.T) {
	players := []tibiadata.OnlinePlayer{
		{Name: "Player1", Level: 100},
		{Name: "Player2", Level: 150},
	}

	processedCharacters := make(map[string]bool)

	mockFetcher := &mockServiceFetcher{
		fetchCharacterDetailsFunc: func(players []tibiadata.OnlinePlayer) <-chan *tibiadata.CharacterResponse {
			results := make(chan *tibiadata.CharacterResponse, len(players))
			for _, p := range players {
				results <- &tibiadata.CharacterResponse{
					Character: struct {
						Character tibiadata.CharacterInfo `json:"character"`
						Deaths    []tibiadata.Death       `json:"deaths"`
					}{
						Character: tibiadata.CharacterInfo{
							Name:  p.Name,
							Level: p.Level,
							World: "Antica",
						},
					},
				}
			}
			close(results)
			return results
		},
	}

	mockAnalytics := &mockServiceAnalytics{
		processCharacterFunc: func(char *tibiadata.CharacterResponse, guilds []string, dbLevels map[string]int) {
			processedCharacters[char.Character.Character.Name] = true
		},
	}

	service := &Service{
		fetcher:   mockFetcher,
		analytics: mockAnalytics,
	}

	guilds := []string{"guild-1"}
	dbLevels := map[string]int{}

	onlineNames := service.processCharacters(players, guilds, dbLevels)

	if len(onlineNames) != 2 {
		t.Errorf("Expected 2 online names, got %d", len(onlineNames))
	}

	if !processedCharacters["Player1"] || !processedCharacters["Player2"] {
		t.Error("Expected both characters to be processed")
	}
}

func TestService_ProcessCharacters_Empty(t *testing.T) {
	players := []tibiadata.OnlinePlayer{}

	mockFetcher := &mockServiceFetcher{
		fetchCharacterDetailsFunc: func(players []tibiadata.OnlinePlayer) <-chan *tibiadata.CharacterResponse {
			results := make(chan *tibiadata.CharacterResponse)
			close(results)
			return results
		},
	}

	mockAnalytics := &mockServiceAnalytics{
		processCharacterFunc: func(char *tibiadata.CharacterResponse, guilds []string, dbLevels map[string]int) {
			t.Error("Should not process any characters")
		},
	}

	service := &Service{
		fetcher:   mockFetcher,
		analytics: mockAnalytics,
	}

	onlineNames := service.processCharacters(players, []string{"guild-1"}, map[string]int{})

	if len(onlineNames) != 0 {
		t.Errorf("Expected 0 online names, got %d", len(onlineNames))
	}
}

func TestService_PerformMaintenance_Success(t *testing.T) {
	var touchedPlayers []string
	var deleteCalled bool

	mockStorage := &mockServiceStorage{
		batchTouchPlayersFunc: func(ctx context.Context, names []string) error {
			touchedPlayers = names
			return nil
		},
		deleteOldPlayersFunc: func(ctx context.Context, world string, threshold time.Duration) (int64, error) {
			deleteCalled = true
			if threshold != 30*time.Minute {
				t.Errorf("Expected threshold 30 minutes, got %v", threshold)
			}
			return 5, nil
		},
	}

	service := &Service{
		storage: mockStorage,
	}

	onlineNames := []string{"Player1", "Player2", "Player3"}
	service.performMaintenance("Antica", onlineNames)

	if len(touchedPlayers) != 3 {
		t.Errorf("Expected 3 touched players, got %d", len(touchedPlayers))
	}

	if !deleteCalled {
		t.Error("Expected DeleteOldPlayers to be called")
	}
}

func TestService_PerformMaintenance_EmptyPlayers(t *testing.T) {
	var touchCalled bool

	mockStorage := &mockServiceStorage{
		batchTouchPlayersFunc: func(ctx context.Context, names []string) error {
			touchCalled = true
			return nil
		},
		deleteOldPlayersFunc: func(ctx context.Context, world string, threshold time.Duration) (int64, error) {
			return 0, nil
		},
	}

	service := &Service{
		storage: mockStorage,
	}

	service.performMaintenance("Antica", []string{})

	if touchCalled {
		t.Error("Expected BatchTouchPlayers NOT to be called for empty names")
	}
}

func TestService_PerformMaintenance_TouchError(t *testing.T) {
	mockStorage := &mockServiceStorage{
		batchTouchPlayersFunc: func(ctx context.Context, names []string) error {
			return errors.New("database error")
		},
		deleteOldPlayersFunc: func(ctx context.Context, world string, threshold time.Duration) (int64, error) {
			return 0, nil
		},
	}

	service := &Service{
		storage: mockStorage,
	}

	// Should not panic on error
	service.performMaintenance("Antica", []string{"Player1"})
}

func TestService_PerformMaintenance_DeleteError(t *testing.T) {
	mockStorage := &mockServiceStorage{
		batchTouchPlayersFunc: func(ctx context.Context, names []string) error {
			return nil
		},
		deleteOldPlayersFunc: func(ctx context.Context, world string, threshold time.Duration) (int64, error) {
			return 0, errors.New("delete error")
		},
	}

	service := &Service{
		storage: mockStorage,
	}

	// Should not panic on delete error
	service.performMaintenance("Antica", []string{"Player1"})
}

func TestService_ProcessWorld_Success(t *testing.T) {
	players := []tibiadata.OnlinePlayer{
		{Name: "Player1", Level: 100},
	}

	var fetchWorldCalled bool
	var fetchLevelsCalled bool
	var maintenanceCalled bool

	mockFetcher := &mockServiceFetcher{
		fetchWorldFunc: func(world string) ([]tibiadata.OnlinePlayer, error) {
			fetchWorldCalled = true
			return players, nil
		},
		fetchCharacterDetailsFunc: func(players []tibiadata.OnlinePlayer) <-chan *tibiadata.CharacterResponse {
			results := make(chan *tibiadata.CharacterResponse, len(players))
			for _, p := range players {
				results <- &tibiadata.CharacterResponse{
					Character: struct {
						Character tibiadata.CharacterInfo `json:"character"`
						Deaths    []tibiadata.Death       `json:"deaths"`
					}{
						Character: tibiadata.CharacterInfo{Name: p.Name},
					},
				}
			}
			close(results)
			return results
		},
	}

	mockStorage := &mockServiceStorage{
		getPlayersLevelsFunc: func(ctx context.Context, world string) (map[string]int, error) {
			fetchLevelsCalled = true
			return map[string]int{}, nil
		},
		batchTouchPlayersFunc: func(ctx context.Context, names []string) error {
			maintenanceCalled = true
			return nil
		},
		deleteOldPlayersFunc: func(ctx context.Context, world string, threshold time.Duration) (int64, error) {
			return 0, nil
		},
	}

	mockAnalytics := &mockServiceAnalytics{
		processCharacterFunc: func(char *tibiadata.CharacterResponse, guilds []string, dbLevels map[string]int) {},
	}

	service := &Service{
		fetcher:   mockFetcher,
		storage:   mockStorage,
		analytics: mockAnalytics,
	}

	service.processWorld("Antica", []string{"guild-1"})

	if !fetchWorldCalled {
		t.Error("Expected FetchWorld to be called")
	}

	if !fetchLevelsCalled {
		t.Error("Expected GetPlayersLevels to be called")
	}

	if !maintenanceCalled {
		t.Error("Expected maintenance to be called")
	}
}

func TestService_ProcessWorld_FetchWorldError(t *testing.T) {
	var levelsFetchCalled bool

	mockFetcher := &mockServiceFetcher{
		fetchWorldFunc: func(world string) ([]tibiadata.OnlinePlayer, error) {
			return nil, errors.New("API error")
		},
	}

	mockStorage := &mockServiceStorage{
		getPlayersLevelsFunc: func(ctx context.Context, world string) (map[string]int, error) {
			levelsFetchCalled = true
			return nil, nil
		},
	}

	service := &Service{
		fetcher: mockFetcher,
		storage: mockStorage,
	}

	service.processWorld("Antica", []string{"guild-1"})

	if levelsFetchCalled {
		t.Error("Expected GetPlayersLevels NOT to be called after fetch error")
	}
}

func TestService_ProcessWorld_FetchLevelsError(t *testing.T) {
	var processCalled bool

	mockFetcher := &mockServiceFetcher{
		fetchWorldFunc: func(world string) ([]tibiadata.OnlinePlayer, error) {
			return []tibiadata.OnlinePlayer{{Name: "Player1", Level: 100}}, nil
		},
		fetchCharacterDetailsFunc: func(players []tibiadata.OnlinePlayer) <-chan *tibiadata.CharacterResponse {
			processCalled = true
			results := make(chan *tibiadata.CharacterResponse)
			close(results)
			return results
		},
	}

	mockStorage := &mockServiceStorage{
		getPlayersLevelsFunc: func(ctx context.Context, world string) (map[string]int, error) {
			return nil, errors.New("database error")
		},
	}

	service := &Service{
		fetcher: mockFetcher,
		storage: mockStorage,
	}

	service.processWorld("Antica", []string{"guild-1"})

	if processCalled {
		t.Error("Expected character processing NOT to happen after levels fetch error")
	}
}

// Mock types for testing Service methods
type mockServiceStorage struct {
	getWorldsMapFunc      func(ctx context.Context) (map[string][]string, error)
	getPlayersLevelsFunc  func(ctx context.Context, world string) (map[string]int, error)
	batchTouchPlayersFunc func(ctx context.Context, names []string) error
	deleteOldPlayersFunc  func(ctx context.Context, world string, threshold time.Duration) (int64, error)
}

func (m *mockServiceStorage) GetWorldsMap(ctx context.Context) (map[string][]string, error) {
	if m.getWorldsMapFunc != nil {
		return m.getWorldsMapFunc(ctx)
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
	return nil
}

func (m *mockServiceStorage) DeleteGuildConfig(ctx context.Context, guildID string) error {
	return nil
}

func (m *mockServiceStorage) Close() {}

type mockServiceFetcher struct {
	fetchWorldFunc            func(world string) ([]tibiadata.OnlinePlayer, error)
	fetchCharacterDetailsFunc func(players []tibiadata.OnlinePlayer) <-chan *tibiadata.CharacterResponse
}

func (m *mockServiceFetcher) FetchWorld(world string) ([]tibiadata.OnlinePlayer, error) {
	if m.fetchWorldFunc != nil {
		return m.fetchWorldFunc(world)
	}
	return nil, nil
}

func (m *mockServiceFetcher) FetchCharacterDetails(players []tibiadata.OnlinePlayer) <-chan *tibiadata.CharacterResponse {
	if m.fetchCharacterDetailsFunc != nil {
		return m.fetchCharacterDetailsFunc(players)
	}
	results := make(chan *tibiadata.CharacterResponse)
	close(results)
	return results
}

type mockServiceAnalytics struct {
	processCharacterFunc func(char *tibiadata.CharacterResponse, guilds []string, dbLevels map[string]int)
}

func (m *mockServiceAnalytics) ProcessCharacter(char *tibiadata.CharacterResponse, guilds []string, dbLevels map[string]int) {
	if m.processCharacterFunc != nil {
		m.processCharacterFunc(char, guilds, dbLevels)
	}
}
