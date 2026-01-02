package updater

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/janyksteenbeek/updati/internal/config"
	gh "github.com/janyksteenbeek/updati/internal/github"
)

// Result represents the result of an update operation
type Result struct {
	Repository   *gh.Repository
	Success      bool
	Updated      bool
	Error        error
	PRNumber     int
	PRURL        string
	Branch       string
	ChangedFiles []string
}

// Updater handles updating repositories using registered plugins
type Updater struct {
	cfg    *config.Config
	client *gh.Client
}

// New creates a new Updater
func New(cfg *config.Config, client *gh.Client) *Updater {
	return &Updater{
		cfg:    cfg,
		client: client,
	}
}

// Update updates a single repository
func (u *Updater) Update(ctx context.Context, repo *gh.Repository) *Result {
	result := &Result{
		Repository: repo,
	}

	// Create temp directory for the repo
	tmpDir, err := os.MkdirTemp("", "updati-"+repo.Name+"-")
	if err != nil {
		result.Error = fmt.Errorf("failed to create temp directory: %w", err)
		return result
	}
	defer os.RemoveAll(tmpDir)

	// Clone the repository
	if err := u.cloneRepo(ctx, repo, tmpDir); err != nil {
		result.Error = fmt.Errorf("failed to clone repository: %w", err)
		return result
	}

	// Determine target branch
	targetBranch := u.determineTargetBranch(repo)
	result.Branch = targetBranch

	// Create branch if using PR mode
	if u.cfg.CreatePR {
		if err := u.createBranch(tmpDir, targetBranch); err != nil {
			result.Error = fmt.Errorf("failed to create branch: %w", err)
			return result
		}
	}

	// Run all applicable plugins
	updated, changedFiles, err := u.runPlugins(ctx, tmpDir, repo)
	if err != nil {
		result.Error = err
		return result
	}

	result.ChangedFiles = changedFiles

	if !updated {
		result.Success = true
		result.Updated = false
		return result
	}

	if u.cfg.DryRun {
		result.Success = true
		result.Updated = true
		return result
	}

	// Commit and push changes
	if err := u.commitAndPush(ctx, tmpDir, targetBranch); err != nil {
		result.Error = fmt.Errorf("failed to commit and push: %w", err)
		return result
	}

	// Create pull request if configured
	if u.cfg.CreatePR {
		pr, err := u.client.CreatePullRequest(
			ctx,
			repo,
			u.cfg.PRTitle,
			u.cfg.PRBody,
			targetBranch,
			repo.DefaultRef,
			u.cfg.Labels,
		)
		if err != nil {
			result.Error = fmt.Errorf("failed to create pull request: %w", err)
			return result
		}
		result.PRNumber = pr.GetNumber()
		result.PRURL = pr.GetHTMLURL()
	}

	result.Success = true
	result.Updated = true
	return result
}

// runPlugins runs all applicable plugins for the repository
func (u *Updater) runPlugins(ctx context.Context, dir string, repo *gh.Repository) (bool, []string, error) {
	var anyUpdated bool
	var allChangedFiles []string

	for _, plugin := range Plugins() {
		// Check if plugin is enabled in config
		if !u.isPluginEnabled(plugin.Name()) {
			continue
		}

		// Check if plugin detects its dependency manager
		if !plugin.Detect(repo) {
			continue
		}

		// Run the plugin
		updated, changedFiles, err := plugin.Update(ctx, dir)
		if err != nil {
			return false, nil, fmt.Errorf("%s: %w", plugin.Name(), err)
		}

		if updated {
			anyUpdated = true
			allChangedFiles = append(allChangedFiles, changedFiles...)
		}
	}

	return anyUpdated, allChangedFiles, nil
}

// isPluginEnabled checks if a plugin is enabled in the config
func (u *Updater) isPluginEnabled(name string) bool {
	switch name {
	case "composer":
		return u.cfg.UpdateComposer
	case "npm":
		return u.cfg.UpdateNPM
	default:
		return true // Enable unknown plugins by default
	}
}

func (u *Updater) determineTargetBranch(repo *gh.Repository) string {
	if u.cfg.CreatePR {
		return u.cfg.PRBranch
	}
	if u.cfg.BaseBranch != "" {
		return u.cfg.BaseBranch
	}
	return repo.DefaultRef
}

func (u *Updater) cloneRepo(ctx context.Context, repo *gh.Repository, dir string) error {
	cloneURL := strings.Replace(
		repo.CloneURL,
		"https://",
		fmt.Sprintf("https://x-access-token:%s@", u.cfg.GitHubToken),
		1,
	)

	cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", "-b", repo.DefaultRef, cloneURL, dir)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone failed: %s", stderr.String())
	}

	return nil
}

func (u *Updater) createBranch(dir, branchName string) error {
	cmd := exec.Command("git", "checkout", "-B", branchName)
	cmd.Dir = dir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git checkout failed: %s", stderr.String())
	}

	return nil
}

func (u *Updater) commitAndPush(ctx context.Context, dir, branchName string) error {
	cmds := [][]string{
		{"git", "config", "user.email", "updati@github.com"},
		{"git", "config", "user.name", "Updati Bot"},
		{"git", "add", "-A"},
		{"git", "commit", "-m", u.cfg.CommitMessage},
		{"git", "push", "-f", "origin", branchName},
	}

	for _, args := range cmds {
		cmd := exec.CommandContext(ctx, args[0], args[1:]...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			if strings.Contains(stderr.String(), "nothing to commit") {
				continue
			}
			return fmt.Errorf("%s failed: %s", args[0], stderr.String())
		}
	}

	return nil
}
