package tools

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ValidatePath ensures the resolved path is within the workspace directory.
func ValidatePath(workspace, relPath string) (string, error) {
	// Reject absolute paths; they must be relative to the workspace.
	if filepath.IsAbs(relPath) {
		return "", fmt.Errorf("paths must be relative to the workspace directory")
	}

	// Clean the path to resolve any .. or .
	cleanPath := filepath.Clean(relPath)

	// Join with workspace and get absolute path
	absPath := filepath.Join(workspace, cleanPath)

	// Get absolute path of workspace
	absWorkspace, err := filepath.Abs(workspace)
	if err != nil {
		return "", fmt.Errorf("invalid workspace: %w", err)
	}

	// Get absolute path of the target
	absTarget, err := filepath.Abs(absPath)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	// Check if target is within workspace using filepath.Rel
	rel, err := filepath.Rel(absWorkspace, absTarget)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	// If the relative path starts with .., it's outside the workspace
	if strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("path traversal not allowed: %s", relPath)
	}

	return absPath, nil
}