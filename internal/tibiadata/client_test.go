package tibiadata

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewClient(t *testing.T) {
	client := NewClient()

	if client == nil {
		t.Fatal("Expected NewClient to return non-nil client")
	}

	if client.httpClient == nil {
		t.Error("Expected httpClient to be initialized")
	}

	if client.httpClient.Timeout == 0 {
		t.Error("Expected timeout to be set")
	}

	if client.baseURL != BaseURL {
		t.Errorf("Expected baseURL '%s', got '%s'", BaseURL, client.baseURL)
	}
}

func TestClient_GetWorld_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/world/Antica") {
			t.Errorf("Expected path to contain '/world/Antica', got '%s'", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"world": {
				"online_players": [
					{"name": "Player One", "level": 100, "vocation": "Knight"},
					{"name": "Player Two", "level": 200, "vocation": "Sorcerer"}
				]
			}
		}`))
	}))
	defer server.Close()

	client := NewTestClient(server.URL)
	players, err := client.GetWorld("Antica")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(players) != 2 {
		t.Errorf("Expected 2 players, got %d", len(players))
	}

	if players[0].Name != "Player One" {
		t.Errorf("Expected first player 'Player One', got '%s'", players[0].Name)
	}

	if players[0].Level != 100 {
		t.Errorf("Expected first player level 100, got %d", players[0].Level)
	}
}

func TestClient_GetWorld_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewTestClient(server.URL)
	_, err := client.GetWorld("Test")
	if err == nil {
		t.Error("Expected error for non-200 status code")
	}

	if !strings.Contains(err.Error(), "unexpected status code") {
		t.Errorf("Expected 'unexpected status code' error, got: %v", err)
	}
}

func TestClient_GetWorld_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{invalid json`))
	}))
	defer server.Close()

	client := NewTestClient(server.URL)
	_, err := client.GetWorld("Test")
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}

	if !strings.Contains(err.Error(), "failed to decode response") {
		t.Errorf("Expected 'failed to decode response' error, got: %v", err)
	}
}

func TestClient_GetWorld_EmptyPlayers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"world": {"online_players": []}}`))
	}))
	defer server.Close()

	client := NewTestClient(server.URL)
	players, err := client.GetWorld("Empty")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(players) != 0 {
		t.Errorf("Expected 0 players, got %d", len(players))
	}
}

func TestClient_GetCharacter_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/character/TestPlayer") {
			t.Errorf("Expected path to contain '/character/TestPlayer', got '%s'", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"character": {
				"character": {
					"name": "TestPlayer",
					"level": 150,
					"world": "Antica"
				},
				"deaths": []
			}
		}`))
	}))
	defer server.Close()

	client := NewTestClient(server.URL)
	charResp, err := client.GetCharacter("TestPlayer")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if charResp == nil {
		t.Fatal("Expected non-nil character response")
	}

	if charResp.Character.Character.Name != "TestPlayer" {
		t.Errorf("Expected character name 'TestPlayer', got '%s'", charResp.Character.Character.Name)
	}

	if charResp.Character.Character.Level != 150 {
		t.Errorf("Expected level 150, got %d", charResp.Character.Character.Level)
	}
}

func TestClient_GetCharacter_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewTestClient(server.URL)
	_, err := client.GetCharacter("NonExistent")
	if err == nil {
		t.Error("Expected error for 404 status code")
	}

	if !strings.Contains(err.Error(), "unexpected status code") {
		t.Errorf("Expected 'unexpected status code' error, got: %v", err)
	}
}

func TestClient_GetCharacter_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`not valid json`))
	}))
	defer server.Close()

	client := NewTestClient(server.URL)
	_, err := client.GetCharacter("Test")
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}

	if !strings.Contains(err.Error(), "failed to decode response") {
		t.Errorf("Expected 'failed to decode response' error, got: %v", err)
	}
}

// Helper function extracted from client logic
func decodeJSON(resp *http.Response, v interface{}) error {
	return json.NewDecoder(resp.Body).Decode(v)
}

func TestBaseURL_Constant(t *testing.T) {
	expectedURL := "https://api.tibiadata.com/v4"
	if BaseURL != expectedURL {
		t.Errorf("Expected BaseURL '%s', got '%s'", expectedURL, BaseURL)
	}
}

func TestClient_GetCharacter_WithQuote(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify that the URL path contains the raw single quote, not %27
		if !strings.Contains(r.URL.Path, "/character/Hell'Draco") {
			t.Errorf("Expected path to contain \"/character/Hell'Draco\", got '%s'", r.URL.Path)
		}
		if strings.Contains(r.URL.Path, "%27") {
			t.Errorf("Path should not contain encoded quote %%27, got '%s'", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"character": {
				"character": {
					"name": "Hell'Draco",
					"level": 150,
					"world": "Antica"
				},
				"deaths": []
			}
		}`))
	}))
	defer server.Close()

	client := NewTestClient(server.URL)
	charResp, err := client.GetCharacter("Hell'Draco")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if charResp.Character.Character.Name != "Hell'Draco" {
		t.Errorf("Expected character name \"Hell'Draco\", got '%s'", charResp.Character.Character.Name)
	}
}

func TestClient_GetWorld_WithEncodedName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Simulate TibiaData returning an encoded name in JSON
		w.Write([]byte(`{
			"world": {
				"online_players": [
					{"name": "Hell%27Draco", "level": 123, "vocation": "Knight"},
					{"name": "Normal Player", "level": 456, "vocation": "Druid"}
				]
			}
		}`))
	}))
	defer server.Close()

	client := NewTestClient(server.URL)
	players, err := client.GetWorld("Antica")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	found := false
	for _, p := range players {
		if p.Name == "Hell'Draco" {
			found = true
			break
		}
		if p.Name == "Hell%27Draco" {
			t.Fatal("Found encoded name 'Hell%27Draco' in results, expected it to be decoded")
		}
	}

	if !found {
		t.Error("Did not find decoded player 'Hell'Draco' in results")
	}
}
