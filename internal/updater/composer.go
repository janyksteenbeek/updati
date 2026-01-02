package updater

import (
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

// Update runs composer upgrade and returns changed files
func (p *ComposerPlugin) Update(ctx context.Context, dir string) (bool, []string, error) {
	lockPath := filepath.Join(dir, "composer.lock")
	jsonPath := filepath.Join(dir, "composer.json")

	// Get original hashes
	lockHash, _ := fileHash(lockPath)
	jsonHash, _ := fileHash(jsonPath)

	// Run composer upgrade with all dependencies
	cmd := exec.CommandContext(ctx, "composer", "upgrade",
		"--no-interaction",
		"--no-scripts",
		"--prefer-dist",
		"--with-all-dependencies",
		"--ignore-platform-reqs",
	)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"COMPOSER_NO_INTERACTION=1",
		"COMPOSER_NO_AUDIT=1",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, nil, fmt.Errorf("composer upgrade failed: %s", string(output))
	}

	// Check which files changed
	var changedFiles []string

	newLockHash, _ := fileHash(lockPath)
	if lockHash != newLockHash {
		changedFiles = append(changedFiles, "composer.lock")
	}

	newJsonHash, _ := fileHash(jsonPath)
	if jsonHash != newJsonHash {
		changedFiles = append(changedFiles, "composer.json")
	}

	return len(changedFiles) > 0, changedFiles, nil
}
