package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// EditTool provides file write operations.
type SedTool struct {
	workspace string
}

// NewEditTool creates a new edit tool.
func NewSedTool(workspace string) *SedTool {
	return &SedTool{workspace: workspace}
}

// validatePath ensures the resolved path is within the workspace directory.
func (t *SedTool) validatePath(relPath string) (string, error) {
	// Clean the path to resolve any .. or .
	cleanPath := filepath.Clean(relPath)

	// Join with workspace and get absolute path
	absPath := filepath.Join(t.workspace, cleanPath)

	// Get absolute path of workspace
	absWorkspace, err := filepath.Abs(t.workspace)
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

// Name returns the tool name.
func (t *SedTool) Name() string {
	return "sed"
}

// StatusLine returns a formatted status line for the edit operation.
func (t *SedTool) StatusLine(args map[string]any) string {
	path, _ := args["path"].(string)
	return fmt.Sprintf("sed> %s", path)
}

// Execute performs file write based on the action.
func (t *SedTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	relPath, ok := args["path"].(string)
	if !ok {
		return "", fmt.Errorf("path argument is required")
	}

	// Check for absolute paths
	if filepath.IsAbs(relPath) {
		return "", fmt.Errorf("paths must be relative to the workspace directory. Use relative paths like 'file.md' instead of absolute paths.")
	}

	absPath, err := t.validatePath(relPath)
	if err != nil {
		return "", err
	}

	content, ok := args["content"].(string)
	if !ok {
		return "", fmt.Errorf("content argument is required")
	}

	// Ensure directory exists
	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(absPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return fmt.Sprintf("Successfully wrote to %s", relPath), nil
}