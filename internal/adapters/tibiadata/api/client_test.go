package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	client := NewClient()

	if client == nil {
		t.Fatal("Expected NewClient to return non-nil client")
	}
	if client.httpClient == nil {
		t.Error("Expected httpClient to be initialized")
	}
	if client.httpClient.Timeout != 10*time.Second {
		t.Errorf("Expected timeout 10s, got %v", client.httpClient.Timeout)
	}
	if client.baseURL != DefaultBaseURL {
		t.Errorf("Expected baseURL '%s', got '%s'", DefaultBaseURL, client.baseURL)
	}
}

func TestClient_GetWorld(t *testing.T) {
	tests := []struct {
		name          string
		worldName     string
		mockHandler   func(w http.ResponseWriter, r *http.Request)
		expectError   bool
		errorContains string
		validate      func(t *testing.T, players []OnlinePlayer)
	}{
		{
			name:      "Success - Standard World",
			worldName: "Antica",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/world/Antica" {
					w.WriteHeader(http.StatusNotFound)
					return
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
			},
			expectError: false,
			validate: func(t *testing.T, players []OnlinePlayer) {
				if len(players) != 2 {
					t.Errorf("Expected 2 players, got %d", len(players))
				}
				if players[0].Name != "Player One" {
					t.Errorf("Expected first player 'Player One', got '%s'", players[0].Name)
				}
			},
		},
		{
			name:      "Success - Encoded Names",
			worldName: "Antica",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				// Simulating API returning URL encoded special characters
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{
					"world": {
						"online_players": [
							{"name": "Hell%27Draco", "level": 100, "vocation": "Knight"}
						]
					}
				}`))
			},
			expectError: false,
			validate: func(t *testing.T, players []OnlinePlayer) {
				if len(players) != 1 {
					t.Fatalf("Expected 1 player, got %d", len(players))
				}
				// Verify decoding logic (Hell%27Draco -> Hell'Draco)
				if players[0].Name != "Hell'Draco" {
					t.Errorf("Expected decoded name \"Hell'Draco\", got '%s'", players[0].Name)
				}
			},
		},
		{
			name:      "Success - Empty World",
			worldName: "EmptyWorld",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"world": {"online_players": []}}`))
			},
			expectError: false,
			validate: func(t *testing.T, players []OnlinePlayer) {
				if len(players) != 0 {
					t.Errorf("Expected 0 players, got %d", len(players))
				}
			},
		},
		{
			name:      "Error - 404 Not Found",
			worldName: "Unknown",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			expectError:   true,
			errorContains: "unexpected status code: 404",
		},
		{
			name:      "Error - 500 Internal Server Error",
			worldName: "Antica",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			expectError:   true,
			errorContains: "unexpected status code: 500",
		},
		{
			name:      "Error - Invalid JSON",
			worldName: "Antica",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"world": {broken_json`))
			},
			expectError:   true,
			errorContains: "decode response",
		},
		{
			name:      "Edge Case - Special Characters in World Name",
			worldName: "Dolera (Retro)",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				// Verify URL encoding using RequestURI which preserves encoding
				if !strings.Contains(r.RequestURI, "/world/Dolera%20%28Retro%29") {
					t.Errorf("Expected encoded path '/world/Dolera%%20%%28Retro%%29', got '%s'", r.RequestURI)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"world": {"online_players": []}}`))
			},
			expectError: false,
			validate: func(t *testing.T, players []OnlinePlayer) {
				if len(players) != 0 {
					t.Errorf("Expected 0 players")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.mockHandler))
			defer server.Close()

			client := NewTestClient(server.URL)
			players, err := client.GetWorld(tt.worldName)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				} else if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if tt.validate != nil {
					tt.validate(t, players)
				}
			}
		})
	}
}

func TestClient_GetCharacter(t *testing.T) {
	tests := []struct {
		name          string
		charName      string
		mockHandler   func(w http.ResponseWriter, r *http.Request)
		expectError   bool
		errorContains string
		validate      func(t *testing.T, char *CharacterResponse)
	}{
		{
			name:     "Success - Simple Name",
			charName: "Bubble",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/character/Bubble" {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				w.WriteHeader(http.StatusOK)
				// Returning minimal valid structure
				json.NewEncoder(w).Encode(CharacterResponse{
					Character: struct {
						Character CharacterInfo `json:"character"`
						Deaths    []Death       `json:"deaths"`
					}{
						Character: CharacterInfo{Name: "Bubble", Level: 100, World: "Antica", Vocation: "Knight"},
					},
				})
			},
			expectError: false,
			validate: func(t *testing.T, char *CharacterResponse) {
				if char.Character.Character.Name != "Bubble" {
					t.Errorf("Expected name Bubble, got %s", char.Character.Character.Name)
				}
			},
		},
		{
			name:     "Success - Name with Single Quote",
			charName: "Hell'Draco",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				// Verify custom handling of single quotes (raw ' instead of %27) using RequestURI
				if !strings.Contains(r.RequestURI, "Hell'Draco") {
					t.Errorf("RequestURI should contain literal quote: %s", r.RequestURI)
				}
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(CharacterResponse{
					Character: struct {
						Character CharacterInfo `json:"character"`
						Deaths    []Death       `json:"deaths"`
					}{
						Character: CharacterInfo{Name: "Hell'Draco", Level: 200},
					},
				})
			},
			expectError: false,
			validate: func(t *testing.T, char *CharacterResponse) {
				if char.Character.Character.Name != "Hell'Draco" {
					t.Errorf("Expected name Hell'Draco, got %s", char.Character.Character.Name)
				}
			},
		},
		{
			name:     "Success - Name with Space",
			charName: "Eternal Oblivion",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				if !strings.Contains(r.RequestURI, "Eternal%20Oblivion") {
					t.Errorf("Expected encoded space in path, got: %s", r.RequestURI)
				}
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(CharacterResponse{
					Character: struct {
						Character CharacterInfo `json:"character"`
						Deaths    []Death       `json:"deaths"`
					}{
						Character: CharacterInfo{Name: "Eternal Oblivion"},
					},
				})
			},
			expectError: false,
		},
		{
			name:     "Error - 404 Not Found",
			charName: "NonExistent",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			expectError:   true,
			errorContains: "unexpected status code: 404",
		},
		{
			name:     "Error - Invalid JSON",
			charName: "Bubble",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("invalid json"))
			},
			expectError:   true,
			errorContains: "decode response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.mockHandler))
			defer server.Close()

			client := NewTestClient(server.URL)
			char, err := client.GetCharacter(tt.charName)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				} else if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if tt.validate != nil {
					tt.validate(t, char)
				}
			}
		})
	}
}

func TestClient_GetGuild(t *testing.T) {
	tests := []struct {
		name          string
		guildName     string
		mockHandler   func(w http.ResponseWriter, r *http.Request)
		expectError   bool
		errorContains string
		validate      func(t *testing.T, guild *GuildResponse)
	}{
		{
			name:      "Success - Standard Guild",
			guildName: "Red Rose",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				if !strings.Contains(r.RequestURI, "Red%20Rose") {
					t.Errorf("Expected path to contain encoded space, got %s", r.RequestURI)
				}
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(GuildResponse{
					Guild: GuildInfo{Name: "Red Rose"},
				})
			},
			expectError: false,
			validate: func(t *testing.T, guild *GuildResponse) {
				if guild.Guild.Name != "Red Rose" {
					t.Errorf("Expected guild name 'Red Rose', got '%s'", guild.Guild.Name)
				}
			},
		},
		{
			name:      "Success - Guild with Quote",
			guildName: "Hell's Angels",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				if !strings.Contains(r.RequestURI, "Hell's%20Angels") {
					t.Errorf("Expected path to contain raw quote, got %s", r.RequestURI)
				}
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(GuildResponse{
					Guild: GuildInfo{Name: "Hell's Angels"},
				})
			},
			expectError: false,
		},
		{
			name:      "Error - 500",
			guildName: "MyGuild",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			expectError:   true,
			errorContains: "unexpected status code: 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.mockHandler))
			defer server.Close()

			client := NewTestClient(server.URL)
			guild, err := client.GetGuild(tt.guildName)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				} else if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if tt.validate != nil {
					tt.validate(t, guild)
				}
			}
		})
	}
}
