package updater

import (
	"context"

	gh "github.com/janyksteenbeek/updati/internal/github"
)

// Plugin defines the interface for dependency updaters
type Plugin interface {
	// Name returns the plugin name (e.g., "composer", "npm")
	Name() string

	// Detect checks if the repository uses this dependency manager
	Detect(repo *gh.Repository) bool

	// Update runs the update command and returns true if files changed
	Update(ctx context.Context, dir string) (updated bool, changedFiles []string, err error)
}

// registry holds all registered plugins
var registry []Plugin

// Register adds a plugin to the registry
func Register(p Plugin) {
	registry = append(registry, p)
}

// Plugins returns all registered plugins
func Plugins() []Plugin {
	return registry
}

// init registers the default plugins
func init() {
	Register(&ComposerPlugin{})
	Register(&NPMPlugin{})
}

