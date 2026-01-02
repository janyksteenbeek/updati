package runner

import (
	"context"
	"fmt"

	"github.com/janyksteenbeek/updati/internal/config"
	"github.com/janyksteenbeek/updati/internal/github"
	"github.com/janyksteenbeek/updati/internal/updater"
	"github.com/janyksteenbeek/updati/internal/worker"
)

// Runner orchestrates the update process
type Runner struct {
	cfg    *config.Config
	client *github.Client
}

// New creates a new Runner
func New(cfg *config.Config) *Runner {
	client := github.NewClient(cfg.GitHubToken, cfg.Owner)
	return &Runner{
		cfg:    cfg,
		client: client,
	}
}

// Run executes the update process
func (r *Runner) Run(ctx context.Context) error {
	r.printBanner()

	// List repositories
	fmt.Println("📦 Fetching repositories...")
	repos, err := r.client.ListRepositories(ctx)
	if err != nil {
		return fmt.Errorf("failed to list repositories: %w", err)
	}

	fmt.Printf("   Found %d repositories\n", len(repos))

	// Filter repositories by pattern
	var matchedRepos []*github.Repository
	for _, repo := range repos {
		if r.cfg.MatchesRepo(repo.Name) {
			matchedRepos = append(matchedRepos, repo)
		}
	}

	fmt.Printf("   %d repositories match patterns\n", len(matchedRepos))
	fmt.Println()

	if len(matchedRepos) == 0 {
		fmt.Println("No repositories to process.")
		return nil
	}

	// Create updater and worker pool
	upd := updater.New(r.cfg, r.client)
	pool := worker.New(r.cfg.Workers, upd, r.client)

	// Process repositories
	fmt.Println("🔄 Processing repositories...")
	fmt.Println()

	result := pool.Process(ctx, matchedRepos)

	// Print summary
	r.printSummary(result)

	if result.Failed > 0 {
		return fmt.Errorf("%d repositories failed to update", result.Failed)
	}

	return nil
}

func (r *Runner) printBanner() {
	fmt.Println("🚀 Updati - Dependency Updater")
	fmt.Printf("   Owner: %s\n", r.cfg.Owner)
	fmt.Printf("   Workers: %d\n", r.cfg.Workers)
	fmt.Printf("   Dry Run: %v\n", r.cfg.DryRun)
	fmt.Printf("   Mode: %s\n", r.modeString())
	if len(r.cfg.RepoPatterns) > 0 {
		fmt.Printf("   Patterns: %v\n", r.cfg.RepoPatterns)
	}
	fmt.Println()
}

func (r *Runner) modeString() string {
	if r.cfg.DryRun {
		return "dry-run"
	}
	if r.cfg.CreatePR {
		return "pull-request"
	}
	return "direct-push"
}

func (r *Runner) printSummary(result *worker.ProcessResult) {
	fmt.Println()
	fmt.Println("📊 Summary")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("   Total repositories:  %d\n", result.Total)
	fmt.Printf("   Successful:          %d\n", result.Successful)
	fmt.Printf("   Updated:             %d\n", result.Updated)
	fmt.Printf("   Skipped:             %d\n", result.Skipped)
	fmt.Printf("   Failed:              %d\n", result.Failed)
	fmt.Println()

	// Print detailed results for updates and failures
	if result.Updated > 0 {
		fmt.Println("✅ Updated repositories:")
		for _, res := range result.Results {
			if res.Updated && res.Error == nil {
				if res.PRURL != "" {
					fmt.Printf("   - %s (PR: %s)\n", res.Repository.FullName, res.PRURL)
				} else {
					fmt.Printf("   - %s (pushed to %s)\n", res.Repository.FullName, res.Branch)
				}
			}
		}
		fmt.Println()
	}

	if result.Failed > 0 {
		fmt.Println("❌ Failed repositories:")
		for _, res := range result.Results {
			if res.Error != nil {
				fmt.Printf("   - %s: %v\n", res.Repository.FullName, res.Error)
			}
		}
		fmt.Println()
	}
}
