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

func TestAdapter_FetchCharacter(t *testing.T) {
	tests := []struct {
		name         string
		charName     string
		mockResponse string
		mockStatus   int
		wantErr      bool
		errContains  string
		validate     func(t *testing.T, p *domain.Player)
	}{
		{
			name:       "Success - Standard Character",
			charName:   "Bubble",
			mockStatus: http.StatusOK,
			mockResponse: `{
				"character": {
					"character": {
						"name": "Bubble",
						"level": 100,
						"world": "Antica",
						"vocation": "Knight"
					},
					"deaths": []
				}
			}`,
			wantErr: false,
			validate: func(t *testing.T, p *domain.Player) {
				if p.Name != "Bubble" {
					t.Errorf("Expected Name Bubble, got %s", p.Name)
				}
				if p.Level != 100 {
					t.Errorf("Expected Level 100, got %d", p.Level)
				}
				if p.World != "Antica" {
					t.Errorf("Expected World Antica, got %s", p.World)
				}
			},
		},
		{
			name:       "Success - Character with Deaths",
			charName:   "Dead Player",
			mockStatus: http.StatusOK,
			mockResponse: `{
				"character": {
					"character": {
						"name": "Dead Player",
						"level": 50,
						"world": "Antica",
						"vocation": "Druid"
					},
					"deaths": [
						{"time": "2023-01-01T12:00:00Z", "level": 49, "reason": "Died by a rat"}
					]
				}
			}`,
			wantErr: false,
			validate: func(t *testing.T, p *domain.Player) {
				if len(p.Deaths) != 1 {
					t.Fatalf("Expected 1 death, got %d", len(p.Deaths))
				}
				if p.Deaths[0].Level != 49 {
					t.Errorf("Expected death level 49, got %d", p.Deaths[0].Level)
				}
			},
		},
		{
			name:        "Error - 404 Not Found",
			charName:    "Unknown",
			mockStatus:  http.StatusNotFound,
			wantErr:     true,
			errContains: "unexpected status code: 404",
		},
		{
			name:        "Error - API Failure (500)",
			charName:    "Broken",
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

			p, err := adapter.FetchCharacter(context.Background(), tt.charName)

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
				tt.validate(t, p)
			}
		})
	}
}

func TestAdapter_FetchCharacterDetails_Batch(t *testing.T) {
	responses := map[string]string{
		"Player1":      `{"character": {"character": {"name": "Player1", "level": 10}, "deaths": []}}`,
		"Player2":      `{"character": {"character": {"name": "Player2", "level": 20}, "deaths": []}}`,
		"Player3":      `{"character": {"character": {"name": "Player3", "level": 30}, "deaths": []}}`,
		"Encoded Name": `{"character": {"character": {"name": "Encoded Name", "level": 40}, "deaths": []}}`,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) < 3 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		rawName := parts[len(parts)-1]

		if response, ok := responses[rawName]; ok {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(response))
			return
		}

		decoded, err := url.QueryUnescape(rawName)
		if err == nil {
			if resp, ok := responses[decoded]; ok {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(resp))
				return
			}
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := api.NewTestClient(server.URL)
	cfg := &config.Config{WorkerPoolSize: 2}
	adapter := NewAdapter(client, cfg)

	names := []string{"Player1", "Player2", "Player3", "Encoded Name"}
	resultsChan, err := adapter.FetchCharacterDetails(context.Background(), names)
	if err != nil {
		t.Fatalf("Failed to start fetch: %v", err)
	}

	results := make(map[string]*domain.Player)
	for p := range resultsChan {
		results[p.Name] = p
	}

	if len(results) != 4 {
		t.Errorf("Expected 4 results, got %d", len(results))
	}

	if results["Player1"].Level != 10 {
		t.Errorf("Expected Player1 level 10, got %d", results["Player1"].Level)
	}
	if results["Encoded Name"].Level != 40 {
		t.Errorf("Expected Encoded Name level 40, got %d", results["Encoded Name"].Level)
	}
}

func TestAdapter_FetchCharacterDetails_PartialErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "Fail") {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"character": {"character": {"name": "Success", "level": 10}, "deaths": []}}`))
		}
	}))
	defer server.Close()

	client := api.NewTestClient(server.URL)
	cfg := &config.Config{WorkerPoolSize: 5}
	adapter := NewAdapter(client, cfg)

	names := []string{"Success1", "Fail1", "Success2"}
	resultsChan, _ := adapter.FetchCharacterDetails(context.Background(), names)

	var count int
	for range resultsChan {
		count++
	}

	if count != 2 {
		t.Errorf("Expected 2 successful results (skipping 1 failure), got %d", count)
	}
}

func TestAdapter_FetchCharacterDetails_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"character": {"character": {"name": "Player", "level": 10}, "deaths": []}}`))
	}))
	defer server.Close()

	client := api.NewTestClient(server.URL)
	cfg := &config.Config{WorkerPoolSize: 1}
	adapter := NewAdapter(client, cfg)

	ctx, cancel := context.WithCancel(context.Background())

	names := []string{"A", "B", "C", "D"}
	resultsChan, _ := adapter.FetchCharacterDetails(ctx, names)

	cancel()

	var count int
	for range resultsChan {
		count++
	}

	// Cancellation should stop processing.
	// Since we sleep 50ms per request and have 1 worker,
	// cancelling immediately should result in at most 1 (the one already picked up) or 0.
	if count >= len(names) {
		t.Errorf("Expected cancellation to stop processing, but got %d results", count)
	}
}
