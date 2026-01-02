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

	// Clone with full history for pushing (shallow clones can cause issues)
	cmd := exec.CommandContext(ctx, "git", "clone", "-b", repo.DefaultRef, cloneURL, dir)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone failed: %s", string(output))
	}

	return nil
}

func (u *Updater) createBranch(dir, branchName string) error {
	cmd := exec.Command("git", "checkout", "-B", branchName)
	cmd.Dir = dir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git checkout failed: %s", string(output))
	}

	return nil
}

func (u *Updater) commitAndPush(ctx context.Context, dir, branchName string) error {
	// Configure git user
	if err := u.runGit(ctx, dir, "config", "user.email", "updati@github.com"); err != nil {
		return err
	}
	if err := u.runGit(ctx, dir, "config", "user.name", "Updati Bot"); err != nil {
		return err
	}

	// Stage all changes
	if err := u.runGit(ctx, dir, "add", "-A"); err != nil {
		return err
	}

	// Check if there are changes to commit
	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
	cmd.Dir = dir
	output, _ := cmd.Output()
	if len(strings.TrimSpace(string(output))) == 0 {
		return nil // Nothing to commit
	}

	// Commit
	if err := u.runGit(ctx, dir, "commit", "-m", u.cfg.CommitMessage); err != nil {
		if strings.Contains(err.Error(), "nothing to commit") {
			return nil
		}
		return err
	}

	// Push
	if err := u.runGit(ctx, dir, "push", "-f", "origin", branchName); err != nil {
		return err
	}

	return nil
}

func (u *Updater) runGit(ctx context.Context, dir string, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git %s failed: %s", args[0], string(output))
	}

	return nil
}
