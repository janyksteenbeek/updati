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

// NPMPlugin handles NPM dependency updates
type NPMPlugin struct{}

// Name returns the plugin name
func (p *NPMPlugin) Name() string {
	return "npm"
}

// Detect checks if the repository has a package.json
func (p *NPMPlugin) Detect(repo *gh.Repository) bool {
	return repo.HasNPM
}

// Update runs npm update and returns changed files
func (p *NPMPlugin) Update(ctx context.Context, dir string) (bool, []string, error) {
	lockPath := filepath.Join(dir, "package-lock.json")

	// Get original hash
	originalHash, err := fileHash(lockPath)
	if err != nil && !os.IsNotExist(err) {
		return false, nil, fmt.Errorf("failed to hash package-lock.json: %w", err)
	}

	// Run npm update
	cmd := exec.CommandContext(ctx, "npm", "update", "--no-audit", "--no-fund")
	cmd.Dir = dir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return false, nil, fmt.Errorf("npm update failed: %s", stderr.String())
	}

	// Check if file changed
	newHash, err := fileHash(lockPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil, nil
		}
		return false, nil, fmt.Errorf("failed to hash package-lock.json after update: %w", err)
	}

	if originalHash != newHash {
		return true, []string{"package-lock.json"}, nil
	}

	return false, nil, nil
}

