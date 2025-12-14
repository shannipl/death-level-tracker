package tracker

import (
	"errors"
	"sync"
	"testing"

	"death-level-tracker/internal/config"
	"death-level-tracker/internal/tibiadata"
)

// Mock TibiaData client for testing
type mockTibiaDataClient struct {
	getWorldFunc     func(world string) ([]tibiadata.OnlinePlayer, error)
	getCharacterFunc func(name string) (*tibiadata.CharacterResponse, error)
}

func (m *mockTibiaDataClient) GetWorld(world string) ([]tibiadata.OnlinePlayer, error) {
	if m.getWorldFunc != nil {
		return m.getWorldFunc(world)
	}
	return nil, nil
}

func (m *mockTibiaDataClient) GetCharacter(name string) (*tibiadata.CharacterResponse, error) {
	if m.getCharacterFunc != nil {
		return m.getCharacterFunc(name)
	}
	return nil, nil
}

func TestNewFetcher(t *testing.T) {
	client := tibiadata.NewClient()
	cfg := &config.Config{
		WorkerPoolSize: 10,
		MinLevelTrack:  100,
	}

	fetcher := NewFetcher(client, cfg)

	if fetcher == nil {
		t.Fatal("Expected non-nil fetcher")
	}

	if fetcher.client == nil {
		t.Error("Expected client to be set")
	}

	if fetcher.config == nil {
		t.Error("Expected config to be set")
	}
}

func TestFetcher_FetchWorld_Success(t *testing.T) {
	expectedPlayers := []tibiadata.OnlinePlayer{
		{Name: "Player1", Level: 100},
		{Name: "Player2", Level: 200},
	}

	mockClient := &mockTibiaDataClient{
		getWorldFunc: func(world string) ([]tibiadata.OnlinePlayer, error) {
			if world != "Antica" {
				t.Errorf("Expected world 'Antica', got '%s'", world)
			}
			return expectedPlayers, nil
		},
	}

	cfg := &config.Config{}
	fetcher := &Fetcher{
		client: mockClient,
		config: cfg,
	}

	players, err := fetcher.FetchWorld("Antica")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(players) != 2 {
		t.Errorf("Expected 2 players, got %d", len(players))
	}

	if players[0].Name != "Player1" {
		t.Errorf("Expected player 'Player1', got '%s'", players[0].Name)
	}
}

func TestFetcher_FetchWorld_Error(t *testing.T) {
	mockClient := &mockTibiaDataClient{
		getWorldFunc: func(world string) ([]tibiadata.OnlinePlayer, error) {
			return nil, errors.New("API error")
		},
	}

	cfg := &config.Config{}
	fetcher := &Fetcher{
		client: mockClient,
		config: cfg,
	}

	players, err := fetcher.FetchWorld("Antica")

	if err == nil {
		t.Error("Expected error, got nil")
	}

	if players != nil {
		t.Errorf("Expected nil players on error, got %v", players)
	}
}

func TestFetcher_FetchCharacterDetails_Success(t *testing.T) {
	players := []tibiadata.OnlinePlayer{
		{Name: "Player1", Level: 100},
		{Name: "Player2", Level: 150},
	}

	var mu sync.Mutex
	fetchedNames := make(map[string]bool)

	mockClient := &mockTibiaDataClient{
		getCharacterFunc: func(name string) (*tibiadata.CharacterResponse, error) {
			mu.Lock()
			fetchedNames[name] = true
			mu.Unlock()
			return &tibiadata.CharacterResponse{
				Character: struct {
					Character tibiadata.CharacterInfo `json:"character"`
					Deaths    []tibiadata.Death       `json:"deaths"`
				}{
					Character: tibiadata.CharacterInfo{
						Name:  name,
						Level: 100,
						World: "Antica",
					},
				},
			}, nil
		},
	}

	cfg := &config.Config{
		WorkerPoolSize: 2,
		MinLevelTrack:  50,
	}

	fetcher := &Fetcher{
		client: mockClient,
		config: cfg,
	}

	results := fetcher.FetchCharacterDetails(players)

	var count int
	for range results {
		count++
	}

	if count != 2 {
		t.Errorf("Expected 2 character responses, got %d", count)
	}

	mu.Lock()
	if !fetchedNames["Player1"] || !fetchedNames["Player2"] {
		t.Error("Expected both players to be fetched")
	}
	mu.Unlock()
}

func TestFetcher_FetchCharacterDetails_FiltersByLevel(t *testing.T) {
	players := []tibiadata.OnlinePlayer{
		{Name: "LowLevel", Level: 50},
		{Name: "HighLevel", Level: 200},
	}

	var mu sync.Mutex
	fetchedNames := make(map[string]bool)

	mockClient := &mockTibiaDataClient{
		getCharacterFunc: func(name string) (*tibiadata.CharacterResponse, error) {
			mu.Lock()
			fetchedNames[name] = true
			mu.Unlock()
			return &tibiadata.CharacterResponse{}, nil
		},
	}

	cfg := &config.Config{
		WorkerPoolSize: 2,
		MinLevelTrack:  100, // Only fetch players >= 100
	}

	fetcher := &Fetcher{
		client: mockClient,
		config: cfg,
	}

	results := fetcher.FetchCharacterDetails(players)

	var count int
	for range results {
		count++
	}

	if count != 1 {
		t.Errorf("Expected 1 character response (filtered), got %d", count)
	}

	mu.Lock()
	defer mu.Unlock()
	if fetchedNames["LowLevel"] {
		t.Error("Expected LowLevel player to be filtered out")
	}

	if !fetchedNames["HighLevel"] {
		t.Error("Expected HighLevel player to be fetched")
	}
}

func TestFetcher_FetchCharacterDetails_HandlesErrors(t *testing.T) {
	players := []tibiadata.OnlinePlayer{
		{Name: "Player1", Level: 100},
		{Name: "FailPlayer", Level: 150},
		{Name: "Player3", Level: 200},
	}

	mockClient := &mockTibiaDataClient{
		getCharacterFunc: func(name string) (*tibiadata.CharacterResponse, error) {
			if name == "FailPlayer" {
				return nil, errors.New("character not found")
			}
			return &tibiadata.CharacterResponse{
				Character: struct {
					Character tibiadata.CharacterInfo `json:"character"`
					Deaths    []tibiadata.Death       `json:"deaths"`
				}{
					Character: tibiadata.CharacterInfo{Name: name},
				},
			}, nil
		},
	}

	cfg := &config.Config{
		WorkerPoolSize: 3,
		MinLevelTrack:  50,
	}

	fetcher := &Fetcher{
		client: mockClient,
		config: cfg,
	}

	results := fetcher.FetchCharacterDetails(players)

	var count int
	for char := range results {
		if char.Character.Character.Name == "FailPlayer" {
			t.Error("Expected failed player to be skipped")
		}
		count++
	}

	// Should get 2 results (FailPlayer is skipped due to error)
	if count != 2 {
		t.Errorf("Expected 2 successful responses, got %d", count)
	}
}

func TestFetcher_FetchCharacterDetails_EmptyPlayers(t *testing.T) {
	players := []tibiadata.OnlinePlayer{}

	mockClient := &mockTibiaDataClient{
		getCharacterFunc: func(name string) (*tibiadata.CharacterResponse, error) {
			t.Error("Should not fetch any characters for empty player list")
			return nil, nil
		},
	}

	cfg := &config.Config{
		WorkerPoolSize: 2,
		MinLevelTrack:  50,
	}

	fetcher := &Fetcher{
		client: mockClient,
		config: cfg,
	}

	results := fetcher.FetchCharacterDetails(players)

	var count int
	for range results {
		count++
	}

	if count != 0 {
		t.Errorf("Expected 0 results for empty players, got %d", count)
	}
}

func TestFetcher_FetchCharacterDetails_WorkerCount(t *testing.T) {
	// Create enough players to test worker pool
	players := make([]tibiadata.OnlinePlayer, 20)
	for i := range players {
		players[i] = tibiadata.OnlinePlayer{
			Name:  "Player" + string(rune('A'+i)),
			Level: 100,
		}
	}

	mockClient := &mockTibiaDataClient{
		getCharacterFunc: func(name string) (*tibiadata.CharacterResponse, error) {
			return &tibiadata.CharacterResponse{
				Character: struct {
					Character tibiadata.CharacterInfo `json:"character"`
					Deaths    []tibiadata.Death       `json:"deaths"`
				}{
					Character: tibiadata.CharacterInfo{Name: name},
				},
			}, nil
		},
	}

	cfg := &config.Config{
		WorkerPoolSize: 5,
		MinLevelTrack:  50,
	}

	fetcher := &Fetcher{
		client: mockClient,
		config: cfg,
	}

	results := fetcher.FetchCharacterDetails(players)

	var count int
	for range results {
		count++
	}

	// All 20 players should be processed
	if count != 20 {
		t.Errorf("Expected 20 results, got %d", count)
	}
}
