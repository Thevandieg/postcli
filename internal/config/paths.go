package config

import (
	"os"
	"path/filepath"
)

const appDirName = "postcli"

// Dir returns the application config directory (XDG_CONFIG_HOME/postcli or ~/.config/postcli).
func Dir() string {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return filepath.Join(".", appDirName)
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, appDirName)
}

// DBPath returns the SQLite database path.
func DBPath() string {
	return filepath.Join(Dir(), "queue.db")
}

// TokenPath returns the OAuth token JSON path.
func TokenPath() string {
	return filepath.Join(Dir(), "oauth.json")
}

// EnvPath returns the persisted env vars file path.
func EnvPath() string {
	return filepath.Join(Dir(), "env")
}

// ThemePath returns the active theme name (one line, e.g. "violet").
func ThemePath() string {
	return filepath.Join(Dir(), "theme")
}

// EnsureDir creates the config directory if needed.
func EnsureDir() error {
	return os.MkdirAll(Dir(), 0o700)
}
