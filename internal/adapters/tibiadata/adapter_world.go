package tibiadata

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"death-level-tracker/internal/adapters/metrics"
	"death-level-tracker/internal/adapters/tibiadata/scraper"
	"death-level-tracker/internal/core/domain"
)

// FetchWorld gets online players from TibiaData API.
func (a *Adapter) FetchWorld(ctx context.Context, world string) ([]domain.Player, error) {
	onlinePlayers, err := a.client.GetWorld(world)
	if err != nil {
		slog.Error("Failed to fetch world players", "world", world, "error", err)
		return nil, err
	}
	slog.Info("Fetched online players", "world", world, "count", len(onlinePlayers))

	players := make([]domain.Player, len(onlinePlayers))
	for i, p := range onlinePlayers {
		players[i] = domain.Player{
			Name:     p.Name,
			Level:    p.Level,
			Vocation: p.Vocation,
			World:    world,
		}
	}

	return players, nil
}

// FetchWorldFromTibiaCom scrapes Tibia.com as a fallback/alternative source.
func (a *Adapter) FetchWorldFromTibiaCom(ctx context.Context, world string) (map[string]int, error) {
	start := time.Now()
	targetURL := fmt.Sprintf("https://www.tibia.com/community/?subtopic=worlds&world=%s", world)

	req, err := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	a.addBrowserHeaders(req)

	resp, err := a.tibiaComClient.Do(req)

	status := "error"
	if err == nil {
		status = fmt.Sprintf("%d", resp.StatusCode)
	}
	duration := time.Since(start).Seconds()

	metrics.TibiaComRequestDuration.WithLabelValues(status).Observe(duration)
	metrics.TibiaComRequests.WithLabelValues(status).Inc()

	if err != nil {
		slog.Error("Failed to fetch tibia.com world page", "world", world, "error", err)
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Error("Unexpected status from tibia.com", "world", world, "status", resp.StatusCode)
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	players, err := scraper.ParseTibiaComWorld(resp.Body)
	if err != nil {
		slog.Error("Failed to parse tibia.com HTML", "world", world, "error", err)
		return nil, fmt.Errorf("parse HTML: %w", err)
	}

	slog.Info("Fetched online players from tibia.com", "world", world, "count", len(players))
	return players, nil
}

func (a *Adapter) addBrowserHeaders(req *http.Request) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
}
