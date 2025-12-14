package tibiadata

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"death-level-tracker/internal/metrics"
)

const BaseURL = "https://api.tibiadata.com/v4"

type Client struct {
	httpClient *http.Client
	baseURL    string
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout:   10 * time.Second,
			Transport: NewMetricsRoundTripper(http.DefaultTransport),
		},
		baseURL: BaseURL,
	}
}

// NewTestClient creates a client with custom base URL for testing
func NewTestClient(baseURL string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL: baseURL,
	}
}

type MetricsRoundTripper struct {
	Proxied http.RoundTripper
}

func NewMetricsRoundTripper(proxied http.RoundTripper) *MetricsRoundTripper {
	if proxied == nil {
		proxied = http.DefaultTransport
	}
	return &MetricsRoundTripper{Proxied: proxied}
}

func (mrt *MetricsRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()
	resp, err := mrt.Proxied.RoundTrip(req)
	duration := time.Since(start).Seconds()

	status := "error"
	if err == nil {
		status = fmt.Sprintf("%d", resp.StatusCode)
	}

	endpoint := "unknown"
	path := req.URL.Path
	if len(path) > 0 {
		// Simple heuristic to grouping endpoints
		// /v4/world/... -> world
		// /v4/character/... -> character
		if canCheck(path, "world") {
			endpoint = "world"
		} else if canCheck(path, "character") {
			endpoint = "character"
		}
	}

	metrics.TibiaDataRequestDuration.WithLabelValues(endpoint, status).Observe(duration)
	metrics.TibiaDataRequests.WithLabelValues(endpoint, status).Inc()

	return resp, err
}

func canCheck(path, segment string) bool {
	// Check if path contains segment surrounded by slashes or at end
	// This is a naive check but sufficient for known TibiaData paths
	// e.g. /v4/world/Name
	for i := 0; i < len(path)-len(segment); i++ {
		if path[i:i+len(segment)] == segment {
			return true
		}
	}
	return false
}

func (c *Client) GetWorld(worldName string) ([]OnlinePlayer, error) {
	u := fmt.Sprintf("%s/world/%s", c.baseURL, url.PathEscape(worldName))
	resp, err := c.httpClient.Get(u)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch world: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var data WorldResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return data.World.OnlinePlayers, nil
}

func (c *Client) GetCharacter(name string) (*CharacterResponse, error) {
	u := fmt.Sprintf("%s/character/%s", c.baseURL, url.PathEscape(name))
	resp, err := c.httpClient.Get(u)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch character: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var data CharacterResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &data, nil
}
