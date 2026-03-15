// Package paths provides a single source of truth for all Bantam file paths.
package paths

import (
	"os"
	"path/filepath"
)

// Bantam paths - calculated once at startup
var (
	BaseDir      string // ~/.bantam
	WorkspaceDir string // ~/.bantam/workspace
	SessionsDir  string // ~/.bantam/sessions
	LogsDir      string // ~/.bantam/logs
	ConfigPath   string // ~/.bantam/config.yaml
)

// Init calculates all paths based on baseDir and workspace parameters.
// If empty, defaults are used (baseDir = ~/.bantam, workspace = ~/.bantam/workspace).
func Init(basedir, ws string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	if basedir == "" {
		basedir = filepath.Join(home, ".bantam")
	}

	if ws == "" {
		ws = filepath.Join(basedir, "workspace")
	}

	BaseDir = basedir
	WorkspaceDir = ws
	SessionsDir = filepath.Join(basedir, "sessions")
	LogsDir = filepath.Join(basedir, "logs")
	ConfigPath = filepath.Join(basedir, "config.yaml")

	return nil
}
