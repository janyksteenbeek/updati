package config

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config holds the application configuration
type Config struct {
	// GitHub authentication
	GitHubToken string `yaml:"github_token"`

	// Repository matching
	RepoPatterns []string `yaml:"repo_patterns"` // Regex patterns for matching repos
	Owner        string   `yaml:"owner"`         // GitHub owner (user or org)

	// Concurrency settings
	Workers int `yaml:"workers"` // Number of concurrent workers

	// Update settings
	UpdateComposer bool     `yaml:"update_composer"` // Update composer dependencies
	UpdateNPM      bool     `yaml:"update_npm"`      // Update npm dependencies
	CreatePR       bool     `yaml:"create_pr"`       // Create pull request instead of direct push
	BaseBranch     string   `yaml:"base_branch"`     // Branch to base updates on
	PRBranch       string   `yaml:"pr_branch"`       // Branch name for PRs
	CommitMessage  string   `yaml:"commit_message"`  // Custom commit message
	PRTitle        string   `yaml:"pr_title"`        // Custom PR title
	PRBody         string   `yaml:"pr_body"`         // Custom PR body
	DryRun         bool     `yaml:"dry_run"`         // Don't actually make changes
	Labels         []string `yaml:"labels"`          // Labels to add to PRs

	// Compiled patterns (not from config file)
	compiledPatterns []*regexp.Regexp
}

// DefaultConfig returns a config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Workers:        5,
		UpdateComposer: true,
		UpdateNPM:      true,
		CreatePR:       true,
		BaseBranch:     "main",
		PRBranch:       "updati/dependencies",
		CommitMessage:  "chore(deps): update dependencies",
		PRTitle:        "⬆️ Update dependencies",
		PRBody:         "This PR was automatically created by [Updati](https://github.com/janyksteenbeek/updati) to update project dependencies.",
		Labels:         []string{"dependencies", "automated"},
	}
}

// Load loads configuration from a YAML file
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Override with environment variables
	cfg.applyEnvOverrides()

	// Compile regex patterns
	if err := cfg.CompilePatterns(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// LoadFromEnv loads configuration entirely from environment variables
func LoadFromEnv() (*Config, error) {
	cfg := DefaultConfig()
	cfg.applyEnvOverrides()

	if err := cfg.CompilePatterns(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// applyEnvOverrides applies environment variable overrides to config
func (c *Config) applyEnvOverrides() {
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		c.GitHubToken = token
	}
	if token := os.Getenv("INPUT_GITHUB_TOKEN"); token != "" {
		c.GitHubToken = token
	}

	if owner := os.Getenv("UPDATI_OWNER"); owner != "" {
		c.Owner = owner
	}
	if owner := os.Getenv("INPUT_OWNER"); owner != "" {
		c.Owner = owner
	}

	if patterns := os.Getenv("UPDATI_REPO_PATTERNS"); patterns != "" {
		c.RepoPatterns = parsePatterns(patterns)
	}
	if patterns := os.Getenv("INPUT_REPO_PATTERNS"); patterns != "" {
		c.RepoPatterns = parsePatterns(patterns)
	}

	if workers := os.Getenv("UPDATI_WORKERS"); workers != "" {
		if w, err := strconv.Atoi(workers); err == nil && w > 0 {
			c.Workers = w
		}
	}
	if workers := os.Getenv("INPUT_WORKERS"); workers != "" {
		if w, err := strconv.Atoi(workers); err == nil && w > 0 {
			c.Workers = w
		}
	}

	if branch := os.Getenv("UPDATI_BASE_BRANCH"); branch != "" {
		c.BaseBranch = branch
	}
	if branch := os.Getenv("INPUT_BASE_BRANCH"); branch != "" {
		c.BaseBranch = branch
	}

	if dryRun := os.Getenv("UPDATI_DRY_RUN"); dryRun == "true" {
		c.DryRun = true
	}
	if dryRun := os.Getenv("INPUT_DRY_RUN"); dryRun == "true" {
		c.DryRun = true
	}

	if createPR := os.Getenv("UPDATI_CREATE_PR"); createPR != "" {
		c.CreatePR = createPR == "true"
	}
	if createPR := os.Getenv("INPUT_CREATE_PR"); createPR != "" {
		c.CreatePR = createPR == "true"
	}
}

// CompilePatterns compiles regex patterns for repository matching
func (c *Config) CompilePatterns() error {
	c.compiledPatterns = make([]*regexp.Regexp, 0, len(c.RepoPatterns))

	for _, pattern := range c.RepoPatterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}

		re, err := regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf("invalid regex pattern %q: %w", pattern, err)
		}
		c.compiledPatterns = append(c.compiledPatterns, re)
	}

	return nil
}

// MatchesRepo checks if a repository name matches any of the configured patterns
func (c *Config) MatchesRepo(repoName string) bool {
	// If no patterns configured, match all
	if len(c.compiledPatterns) == 0 {
		return true
	}

	for _, re := range c.compiledPatterns {
		if re.MatchString(repoName) {
			return true
		}
	}

	return false
}

// parsePatterns parses patterns from a string (supports newlines and commas)
func parsePatterns(input string) []string {
	var patterns []string

	// Split by newlines first, then by commas
	lines := strings.Split(input, "\n")
	for _, line := range lines {
		// Also support comma separation within lines
		parts := strings.Split(line, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part != "" {
				patterns = append(patterns, part)
			}
		}
	}

	return patterns
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.GitHubToken == "" {
		return fmt.Errorf("github_token is required")
	}

	if c.Owner == "" {
		return fmt.Errorf("owner is required")
	}

	if c.Workers < 1 {
		return fmt.Errorf("workers must be at least 1")
	}

	if c.Workers > 20 {
		return fmt.Errorf("workers cannot exceed 20 (GitHub rate limits)")
	}

	return nil
}

