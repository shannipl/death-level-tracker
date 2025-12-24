package tibiadata

import (
	"context"
	"log/slog"
	"sync"

	"death-level-tracker/internal/core/domain"
)

// FetchCharacter gets a single character's details.
func (a *Adapter) FetchCharacter(ctx context.Context, name string) (*domain.Player, error) {
	char, err := a.client.GetCharacter(name)
	if err != nil {
		return nil, err
	}
	return a.mapCharacter(char), nil
}

// FetchCharacterDetails concurrently fetches details for a list of character names.
func (a *Adapter) FetchCharacterDetails(ctx context.Context, names []string) (chan *domain.Player, error) {
	results := make(chan *domain.Player, len(names))
	jobs := make(chan string, len(names))
	workerCount := a.config.WorkerPoolSize

	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go a.worker(ctx, jobs, results, &wg)
	}

	go func() {
		defer close(results)
		wg.Wait()
	}()

	go func() {
		defer close(jobs)
		for _, name := range names {
			jobs <- name
		}
	}()

	return results, nil
}

func (a *Adapter) worker(ctx context.Context, jobs <-chan string, results chan<- *domain.Player, wg *sync.WaitGroup) {
	defer wg.Done()
	for name := range jobs {
		select {
		case <-ctx.Done():
			return
		default:
			char, err := a.client.GetCharacter(name)
			if err != nil {
				slog.Warn("Failed to fetch character", "name", name, "error", err)
				continue
			}
			result := a.mapCharacter(char)
			if result != nil {
				results <- result
			}
		}
	}
}
