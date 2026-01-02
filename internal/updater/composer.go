package updater

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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

// composerJSON represents the relevant parts of composer.json
type composerJSON struct {
	Require map[string]string `json:"require"`
}

// Update runs composer update and returns changed files
func (p *ComposerPlugin) Update(ctx context.Context, dir string) (bool, []string, error) {
	lockPath := filepath.Join(dir, "composer.lock")

	// Get original hash
	originalHash, err := fileHash(lockPath)
	if err != nil && !os.IsNotExist(err) {
		return false, nil, fmt.Errorf("failed to hash composer.lock: %w", err)
	}

	// Detect PHP version from composer.json
	phpBin := p.detectPHPVersion(dir)

	// Run composer update with the appropriate PHP version
	cmd := exec.CommandContext(ctx, phpBin, "/usr/bin/composer", "update",
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

// detectPHPVersion reads composer.json and determines the best PHP version to use
func (p *ComposerPlugin) detectPHPVersion(dir string) string {
	composerPath := filepath.Join(dir, "composer.json")

	data, err := os.ReadFile(composerPath)
	if err != nil {
		return "/usr/bin/php" // Default to PHP
	}

	var composer composerJSON
	if err := json.Unmarshal(data, &composer); err != nil {
		return "/usr/bin/php"
	}

	phpConstraint, ok := composer.Require["php"]
	if !ok {
		return "/usr/bin/php"
	}

	// Parse the constraint and pick the best matching version
	// Available: php82, php83, php84
	return p.selectPHPVersion(phpConstraint)
}

// selectPHPVersion selects the best PHP version based on the constraint
func (p *ComposerPlugin) selectPHPVersion(constraint string) string {
	constraint = strings.TrimSpace(constraint)
	if strings.Contains(constraint, "8.4") {
		return "/usr/bin/php84"
	}
	if strings.Contains(constraint, "8.3") {
		return "/usr/bin/php83"
	}
	if strings.Contains(constraint, "8.2") {
		return "/usr/bin/php82"
	}
	if strings.Contains(constraint, "8.5") {
		return "/usr/bin/php85"
	}

	return "/usr/bin/php"
}
