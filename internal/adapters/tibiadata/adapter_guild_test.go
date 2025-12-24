package tibiadata

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"death-level-tracker/internal/adapters/tibiadata/api"
	"death-level-tracker/internal/config"
)

func TestAdapter_FetchGuildMembers(t *testing.T) {
	tests := []struct {
		name         string
		guildName    string
		mockResponse string
		mockStatus   int
		wantErr      bool
		errContains  string
		validate     func(t *testing.T, members []string)
	}{
		{
			name:       "Success - Standard Guild",
			guildName:  "Red Rose",
			mockStatus: http.StatusOK,
			mockResponse: `{
				"guild": {
					"members": [
						{"name": "Player One", "rank": "Leader"},
						{"name": "Player Two", "rank": "Member"}
					]
				}
			}`,
			wantErr: false,
			validate: func(t *testing.T, members []string) {
				if len(members) != 2 {
					t.Fatalf("Expected 2 members, got %d", len(members))
				}
				if members[0] != "Player One" || members[1] != "Player Two" {
					t.Errorf("Unexpected members: %v", members)
				}
			},
		},
		{
			name:       "Success - Guild with Special Chars",
			guildName:  "Hell's Angels",
			mockStatus: http.StatusOK,
			mockResponse: `{
				"guild": {
					"members": [
						{"name": "Biker One", "rank": "Leader"}
					]
				}
			}`,
			wantErr: false,
			validate: func(t *testing.T, members []string) {
				if len(members) != 1 {
					t.Fatalf("Expected 1 member, got %d", len(members))
				}
				if members[0] != "Biker One" {
					t.Errorf("Expected Biker One, got %s", members[0])
				}
			},
		},
		{
			name:       "Success - Empty Guild Members",
			guildName:  "Empty Guild",
			mockStatus: http.StatusOK,
			mockResponse: `{
				"guild": {
					"members": []
				}
			}`,
			wantErr: false,
			validate: func(t *testing.T, members []string) {
				if len(members) != 0 {
					t.Errorf("Expected 0 members, got %d", len(members))
				}
			},
		},
		{
			name:        "Error - 404 Not Found",
			guildName:   "Unknown Guild",
			mockStatus:  http.StatusNotFound,
			wantErr:     true,
			errContains: "unexpected status code: 404",
		},
		{
			name:        "Error - 500 Internal Error",
			guildName:   "Broken Guild",
			mockStatus:  http.StatusInternalServerError,
			wantErr:     true,
			errContains: "unexpected status code: 500",
		},
		{
			name:         "Error - Invalid JSON",
			guildName:    "Bad Data",
			mockStatus:   http.StatusOK,
			mockResponse: `{"guild": { bad json }}`,
			wantErr:      true,
			errContains:  "decode response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify URL path contains the guild name (encoded or safe version)
				// For Hell's Angels, client sends Hell's%20Angels (with raw ') or similar.
				// We don't need strict URL verification here as that is covered in api/client_test.go
				// We just need to return the mock response.

				w.WriteHeader(tt.mockStatus)
				if tt.mockResponse != "" {
					w.Write([]byte(tt.mockResponse))
				}
			}))
			defer server.Close()

			client := api.NewTestClient(server.URL)
			adapter := NewAdapter(client, &config.Config{})

			members, err := adapter.FetchGuildMembers(context.Background(), tt.guildName)

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
				tt.validate(t, members)
			}
		})
	}
}
