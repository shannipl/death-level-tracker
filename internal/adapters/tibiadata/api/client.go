package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"death-level-tracker/internal/adapters/metrics"
)

const DefaultBaseURL = "https://api.tibiadata.com/v4"

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
		baseURL: DefaultBaseURL,
	}
}

// NewTestClient creates a client with custom base URL for testing.
func NewTestClient(baseURL string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL: baseURL,
	}
}

func (c *Client) GetWorld(worldName string) ([]OnlinePlayer, error) {
	u := fmt.Sprintf("%s/world/%s", c.baseURL, url.PathEscape(worldName))

	var data WorldResponse
	if err := c.getAndDecode(u, &data); err != nil {
		return nil, fmt.Errorf("fetch world: %w", err)
	}

	for i := range data.World.OnlinePlayers {
		if decoded, err := url.QueryUnescape(data.World.OnlinePlayers[i].Name); err == nil {
			data.World.OnlinePlayers[i].Name = decoded
		}
	}

	return data.World.OnlinePlayers, nil
}

func (c *Client) GetCharacter(name string) (*CharacterResponse, error) {
	// TibiaData requires single quotes to be encoded or handled specific way?
	// The original code replaced encoded single quote with literal single quote.
	// keeping it as is to avoid regression.
	safeName := strings.ReplaceAll(url.PathEscape(name), "%27", "'")
	u := fmt.Sprintf("%s/character/%s", c.baseURL, safeName)

	var data CharacterResponse
	if err := c.getAndDecode(u, &data); err != nil {
		return nil, fmt.Errorf("fetch character: %w", err)
	}

	return &data, nil
}

func (c *Client) GetGuild(name string) (*GuildResponse, error) {
	safeName := strings.ReplaceAll(url.PathEscape(name), "%27", "'")
	u := fmt.Sprintf("%s/guild/%s", c.baseURL, safeName)

	var data GuildResponse
	if err := c.getAndDecode(u, &data); err != nil {
		return nil, fmt.Errorf("fetch guild: %w", err)
	}

	return &data, nil
}

func (c *Client) getAndDecode(url string, dest interface{}) error {
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(dest); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	return nil
}

// -- Middleware --

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
		if strings.Contains(path, "/world/") {
			endpoint = "world"
		} else if strings.Contains(path, "/character/") {
			endpoint = "character"
		} else if strings.Contains(path, "/guild/") {
			endpoint = "guild"
		}
	}

	metrics.TibiaDataRequestDuration.WithLabelValues(endpoint, status).Observe(duration)
	metrics.TibiaDataRequests.WithLabelValues(endpoint, status).Inc()

	return resp, err
}
