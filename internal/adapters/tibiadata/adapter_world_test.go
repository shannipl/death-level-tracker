package tibiadata

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"death-level-tracker/internal/adapters/tibiadata/api"
	"death-level-tracker/internal/config"
	"death-level-tracker/internal/core/domain"
)

func TestAdapter_FetchWorld(t *testing.T) {
	tests := []struct {
		name         string
		worldName    string
		mockResponse string
		mockStatus   int
		wantErr      bool
		errContains  string
		validate     func(t *testing.T, players []domain.Player)
	}{
		{
			name:       "Success - Standard World",
			worldName:  "Antica",
			mockStatus: http.StatusOK,
			mockResponse: `{
				"world": {
					"world_information": {
						"online_players": 2
					},
					"online_players": [
						{"name": "Player One", "level": 100, "vocation": "Knight"},
						{"name": "Player Two", "level": 200, "vocation": "Druid"}
					]
				}
			}`,
			wantErr: false,
			validate: func(t *testing.T, players []domain.Player) {
				if len(players) != 2 {
					t.Fatalf("Expected 2 players, got %d", len(players))
				}
				if players[0].Name != "Player One" || players[0].Level != 100 {
					t.Errorf("Unexpected player 0: %v", players[0])
				}
				if players[0].World != "Antica" {
					t.Errorf("Expected World 'Antica', got %s", players[0].World)
				}
			},
		},
		{
			name:       "Success - Empty World",
			worldName:  "EmptyWorld",
			mockStatus: http.StatusOK,
			mockResponse: `{
				"world": {
					"online_players": []
				}
			}`,
			wantErr: false,
			validate: func(t *testing.T, players []domain.Player) {
				if len(players) != 0 {
					t.Errorf("Expected 0 players, got %d", len(players))
				}
			},
		},
		{
			name:        "Error - 500 Internal Error",
			worldName:   "BrokenWorld",
			mockStatus:  http.StatusInternalServerError,
			wantErr:     true,
			errContains: "unexpected status code: 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.mockStatus)
				if tt.mockResponse != "" {
					w.Write([]byte(tt.mockResponse))
				}
			}))
			defer server.Close()

			client := api.NewTestClient(server.URL)
			adapter := NewAdapter(client, &config.Config{})

			players, err := adapter.FetchWorld(context.Background(), tt.worldName)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Error %q does not contain %q", err.Error(), tt.errContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if tt.validate != nil {
				tt.validate(t, players)
			}
		})
	}
}

func TestAdapter_FetchWorldFromTibiaCom(t *testing.T) {
	htmlWithPlayers := `
		<html><body><table>
		<tr class="Odd"><td><a href="?name=One">One</a></td><td>100</td></tr>
		<tr class="Even"><td><a href="?name=Two">Two</a></td><td>200</td></tr>
		</table></body></html>`

	tests := []struct {
		name        string
		worldName   string
		mockHTML    string
		mockStatus  int
		wantErr     bool
		errContains string
		validate    func(t *testing.T, players map[string]int)
	}{
		{
			name:       "Success - Scrape Valid World",
			worldName:  "Antica",
			mockStatus: http.StatusOK,
			mockHTML:   htmlWithPlayers,
			wantErr:    false,
			validate: func(t *testing.T, players map[string]int) {
				if len(players) != 2 {
					t.Fatalf("Expected 2 players, got %d", len(players))
				}
				if players["One"] != 100 {
					t.Errorf("Expected One: 100, got %d", players["One"])
				}
				if players["Two"] != 200 {
					t.Errorf("Expected Two: 200, got %d", players["Two"])
				}
			},
		},
		{
			name:        "Error - 404 Not Found (Tibia.com not reachable)",
			worldName:   "Unknown",
			mockStatus:  http.StatusNotFound,
			wantErr:     true,
			errContains: "unexpected status code: 404",
		},
		{
			name:        "Error - 503 Maintenance",
			worldName:   "Maintenance",
			mockStatus:  http.StatusServiceUnavailable,
			wantErr:     true,
			errContains: "unexpected status code: 503",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if !strings.Contains(r.URL.RawQuery, "world="+tt.worldName) {
					t.Logf("Warning: Expected world query param for %s", tt.worldName)
				}
				w.WriteHeader(tt.mockStatus)
				w.Write([]byte(tt.mockHTML))
			}))
			defer server.Close()

			client := api.NewClient()
			adapter := NewAdapter(client, &config.Config{})

			// Inject custom transport to hijack requests to tibia.com and redirect to mock server
			adapter.tibiaComClient = &http.Client{
				Timeout:   1 * time.Second,
				Transport: &hijackTransport{target: server.URL},
			}

			players, err := adapter.FetchWorldFromTibiaCom(context.Background(), tt.worldName)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Error %q does not contain %q", err.Error(), tt.errContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if tt.validate != nil {
				tt.validate(t, players)
			}
		})
	}
}

type hijackTransport struct {
	target string
}

func (t *hijackTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Rewrite request to go to target server
	u, err := url.Parse(t.target)
	if err != nil {
		return nil, err
	}

	req.URL.Scheme = u.Scheme
	req.URL.Host = u.Host
	// Keep Path and Query as original

	return http.DefaultTransport.RoundTrip(req)
}
