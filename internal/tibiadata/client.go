package tibiadata

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const BaseURL = "https://api.tibiadata.com/v4"

type Client struct {
	httpClient *http.Client
	baseURL    string
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
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
