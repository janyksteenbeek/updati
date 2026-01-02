package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/janyksteenbeek/updati/internal/config"
	"github.com/janyksteenbeek/updati/internal/runner"
	"github.com/urfave/cli/v2"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	app := &cli.App{
		Name:    "updati",
		Usage:   "Automatically update Laravel projects across multiple repositories",
		Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
		Authors: []*cli.Author{
			{Name: "Jany Steenbeek", Email: "jany@janyksteenbeek.nl"},
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "Path to config file",
				EnvVars: []string{"UPDATI_CONFIG"},
			},
			&cli.StringFlag{
				Name:    "owner",
				Aliases: []string{"o"},
				Usage:   "GitHub owner (user or organization)",
				EnvVars: []string{"UPDATI_OWNER", "INPUT_OWNER"},
			},
			&cli.StringFlag{
				Name:    "token",
				Aliases: []string{"t"},
				Usage:   "GitHub personal access token",
				EnvVars: []string{"GITHUB_TOKEN", "INPUT_GITHUB_TOKEN"},
			},
			&cli.StringSliceFlag{
				Name:    "pattern",
				Aliases: []string{"p"},
				Usage:   "Regex pattern to match repository names (can be specified multiple times)",
				EnvVars: []string{"UPDATI_REPO_PATTERNS", "INPUT_REPO_PATTERNS"},
			},
			&cli.IntFlag{
				Name:    "workers",
				Aliases: []string{"w"},
				Usage:   "Number of concurrent workers",
				Value:   5,
				EnvVars: []string{"UPDATI_WORKERS", "INPUT_WORKERS"},
			},
			&cli.BoolFlag{
				Name:    "dry-run",
				Aliases: []string{"n"},
				Usage:   "Perform a dry run without making changes",
				EnvVars: []string{"UPDATI_DRY_RUN", "INPUT_DRY_RUN"},
			},
			&cli.BoolFlag{
				Name:    "push",
				Usage:   "Push directly to base branch instead of creating PR",
				EnvVars: []string{"UPDATI_PUSH"},
			},
			&cli.StringFlag{
				Name:    "base-branch",
				Aliases: []string{"b"},
				Usage:   "Base branch to update or create PRs against",
				Value:   "main",
				EnvVars: []string{"UPDATI_BASE_BRANCH", "INPUT_BASE_BRANCH"},
			},
		},
		Action: run,
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(c *cli.Context) error {
	// Set up context with cancellation
	ctx, cancel := context.WithCancel(c.Context)
	defer cancel()

	// Handle signals
	go handleSignals(cancel)

	// Load configuration
	cfg, err := loadConfig(c)
	if err != nil {
		return err
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Run the updater
	r := runner.New(cfg)
	return r.Run(ctx)
}

func loadConfig(c *cli.Context) (*config.Config, error) {
	var cfg *config.Config
	var err error

	// Load from config file if specified
	if configFile := c.String("config"); configFile != "" {
		cfg, err = config.Load(configFile)
		if err != nil {
			return nil, err
		}
	} else {
		cfg, err = config.LoadFromEnv()
		if err != nil {
			return nil, err
		}
	}

	// Apply CLI flag overrides
	if token := c.String("token"); token != "" {
		cfg.GitHubToken = token
	}
	if owner := c.String("owner"); owner != "" {
		cfg.Owner = owner
	}
	if patterns := c.StringSlice("pattern"); len(patterns) > 0 {
		cfg.RepoPatterns = patterns
		if err := cfg.CompilePatterns(); err != nil {
			return nil, err
		}
	}
	if c.IsSet("workers") {
		cfg.Workers = c.Int("workers")
	}
	if c.IsSet("base-branch") {
		cfg.BaseBranch = c.String("base-branch")
	}
	if c.Bool("dry-run") {
		cfg.DryRun = true
	}
	if c.Bool("push") {
		cfg.CreatePR = false
	}

	return cfg, nil
}

func handleSignals(cancel context.CancelFunc) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	fmt.Println("\nReceived interrupt signal, shutting down...")
	cancel()
}
