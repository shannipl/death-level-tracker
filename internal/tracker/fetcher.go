package tracker

import (
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"death-level-tracker/internal/config"
	"death-level-tracker/internal/metrics"
	"death-level-tracker/internal/tibiadata"
)

type Fetcher struct {
	client TibiaDataClient
	config *config.Config
}

func NewFetcher(client TibiaDataClient, cfg *config.Config) *Fetcher {
	return &Fetcher{
		client: client,
		config: cfg,
	}
}

func (f *Fetcher) FetchWorld(world string) ([]tibiadata.OnlinePlayer, error) {
	players, err := f.client.GetWorld(world)
	if err != nil {
		slog.Error("Failed to fetch world players", "world", world, "error", err)
		return nil, err
	}
	slog.Info("Fetched online players", "world", world, "count", len(players))
	return players, nil
}

func (f *Fetcher) FetchWorldFromTibiaCom(world string) (map[string]int, error) {
	start := time.Now()
	url := fmt.Sprintf("https://www.tibia.com/community/?subtopic=worlds&world=%s", world)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set browser-like headers to avoid Cloudflare blocking
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		duration := time.Since(start).Seconds()
		metrics.TibiaComRequestDuration.WithLabelValues("error").Observe(duration)
		metrics.TibiaComRequests.WithLabelValues("error").Inc()
		slog.Error("Failed to fetch tibia.com world page", "world", world, "error", err)
		return nil, fmt.Errorf("failed to fetch tibia.com: %w", err)
	}
	defer resp.Body.Close()

	status := fmt.Sprintf("%d", resp.StatusCode)
	duration := time.Since(start).Seconds()
	metrics.TibiaComRequestDuration.WithLabelValues(status).Observe(duration)
	metrics.TibiaComRequests.WithLabelValues(status).Inc()

	if resp.StatusCode != http.StatusOK {
		slog.Error("Unexpected status from tibia.com", "world", world, "status", resp.StatusCode)
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	players, err := ParseTibiaComWorld(resp.Body)
	if err != nil {
		slog.Error("Failed to parse tibia.com HTML", "world", world, "error", err)
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	slog.Info("Fetched online players from tibia.com", "world", world, "count", len(players))
	return players, nil
}

func (f *Fetcher) FetchCharacterDetails(players []tibiadata.OnlinePlayer) <-chan *tibiadata.CharacterResponse {
	results := make(chan *tibiadata.CharacterResponse, len(players))
	jobs := make(chan string, len(players))
	workerCount := f.config.WorkerPoolSize

	var wg sync.WaitGroup
	for range workerCount {
		wg.Add(1)
		go f.worker(jobs, results, &wg)
	}

	go f.monitorWorkers(&wg, results)
	go f.submitJobs(players, jobs)

	return results
}

func (f *Fetcher) worker(jobs <-chan string, results chan<- *tibiadata.CharacterResponse, wg *sync.WaitGroup) {
	defer wg.Done()
	for name := range jobs {
		char, err := f.client.GetCharacter(name)
		if err != nil {
			slog.Warn("Failed to fetch character", "name", name, "error", err)
			continue
		}
		results <- char
	}
}

func (f *Fetcher) submitJobs(players []tibiadata.OnlinePlayer, jobs chan<- string) {
	defer close(jobs)
	for _, p := range players {
		if p.Level < f.config.MinLevelTrack {
			continue
		}
		jobs <- p.Name
	}
}

func (f *Fetcher) monitorWorkers(wg *sync.WaitGroup, results chan<- *tibiadata.CharacterResponse) {
	wg.Wait()
	close(results)
}
