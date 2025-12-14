package tracker

import (
	"log/slog"
	"sync"

	"death-level-tracker/internal/config"
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
