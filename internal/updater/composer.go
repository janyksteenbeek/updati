package updater

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	gh "github.com/janyksteenbeek/updati/internal/github"
)

// ComposerPlugin handles Composer dependency updates
type ComposerPlugin struct{}

// Name returns the plugin name
func (p *ComposerPlugin) Name() string {
	return "composer"
}

// Detect checks if the repository has a composer.json
func (p *ComposerPlugin) Detect(repo *gh.Repository) bool {
	return repo.HasComposer
}

// Update runs composer update and returns changed files
func (p *ComposerPlugin) Update(ctx context.Context, dir string) (bool, []string, error) {
	lockPath := filepath.Join(dir, "composer.lock")

	// Get original hash
	originalHash, err := fileHash(lockPath)
	if err != nil && !os.IsNotExist(err) {
		return false, nil, fmt.Errorf("failed to hash composer.lock: %w", err)
	}

	// Run composer update
	cmd := exec.CommandContext(ctx, "composer", "update",
		"--no-interaction",
		"--no-scripts",
		"--no-plugins",
		"--prefer-dist",
		"--ignore-platform-reqs",
	)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "COMPOSER_NO_INTERACTION=1")

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return false, nil, fmt.Errorf("composer update failed: %s", stderr.String())
	}

	// Check if file changed
	newHash, err := fileHash(lockPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil, nil
		}
		return false, nil, fmt.Errorf("failed to hash composer.lock after update: %w", err)
	}

	if originalHash != newHash {
		return true, []string{"composer.lock"}, nil
	}

	return false, nil, nil
}
