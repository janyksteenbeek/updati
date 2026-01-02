package worker

import (
	"context"
	"fmt"
	"sync"

	gh "github.com/janyksteenbeek/updati/internal/github"
	"github.com/janyksteenbeek/updati/internal/updater"
)

// Pool manages concurrent update workers
type Pool struct {
	workers int
	updater *updater.Updater
	client  *gh.Client
}

// New creates a new worker pool
func New(workers int, u *updater.Updater, client *gh.Client) *Pool {
	return &Pool{
		workers: workers,
		updater: u,
		client:  client,
	}
}

// ProcessResult holds the combined results of processing
type ProcessResult struct {
	Total      int
	Successful int
	Updated    int
	Failed     int
	Skipped    int
	Results    []*updater.Result
}

// Process processes all repositories concurrently
func (p *Pool) Process(ctx context.Context, repos []*gh.Repository) *ProcessResult {
	result := &ProcessResult{
		Total:   len(repos),
		Results: make([]*updater.Result, 0, len(repos)),
	}

	repoChan := make(chan *gh.Repository, len(repos))
	resultChan := make(chan *updater.Result, len(repos))

	var wg sync.WaitGroup
	for i := 0; i < p.workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			p.worker(ctx, workerID, repoChan, resultChan)
		}(i)
	}

	go func() {
		for _, repo := range repos {
			select {
			case repoChan <- repo:
			case <-ctx.Done():
				return
			}
		}
		close(repoChan)
	}()

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	for res := range resultChan {
		result.Results = append(result.Results, res)

		if res.Error != nil {
			result.Failed++
		} else if res.Updated {
			result.Updated++
			result.Successful++
		} else {
			result.Skipped++
			result.Successful++
		}
	}

	return result
}

func (p *Pool) worker(ctx context.Context, id int, repos <-chan *gh.Repository, results chan<- *updater.Result) {
	for repo := range repos {
		select {
		case <-ctx.Done():
			return
		default:
		}

		fmt.Printf("[Worker %d] Processing %s...\n", id, repo.FullName)

		// Detect what dependency managers the repo uses
		if err := p.client.DetectDependencies(ctx, repo); err != nil {
			results <- &updater.Result{
				Repository: repo,
				Error:      fmt.Errorf("failed to detect dependencies: %w", err),
			}
			continue
		}

		// Skip if no supported dependency managers found
		if !repo.HasComposer && !repo.HasNPM {
			fmt.Printf("[Worker %d] Skipping %s (no composer.json or package.json)\n", id, repo.FullName)
			results <- &updater.Result{
				Repository: repo,
				Success:    true,
				Updated:    false,
			}
			continue
		}

		// Update the repository
		result := p.updater.Update(ctx, repo)

		if result.Error != nil {
			fmt.Printf("[Worker %d] Error updating %s: %v\n", id, repo.FullName, result.Error)
		} else if result.Updated {
			if result.PRURL != "" {
				fmt.Printf("[Worker %d] Updated %s (PR: %s)\n", id, repo.FullName, result.PRURL)
			} else {
				fmt.Printf("[Worker %d] Updated %s (pushed to %s)\n", id, repo.FullName, result.Branch)
			}
		} else {
			fmt.Printf("[Worker %d] No updates needed for %s\n", id, repo.FullName)
		}

		results <- result
	}
}
