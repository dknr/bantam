package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ViewTool provides file read operations.
type ViewTool struct {
	workspace string
}

// NewViewTool creates a new view tool.
func NewViewTool(workspace string) *ViewTool {
	return &ViewTool{workspace: workspace}
}

// validatePath ensures the resolved path is within the workspace directory.
func (t *ViewTool) validatePath(relPath string) (string, error) {
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
func (t *ViewTool) Name() string {
	return "view"
}

// StatusLine returns a formatted status line for the view operation.
func (t *ViewTool) StatusLine(args map[string]any) string {
	path, _ := args["path"].(string)
	return fmt.Sprintf("view> %s", path)
}

// Execute performs file read based on the action.
func (t *ViewTool) Execute(ctx context.Context, args map[string]any) (any, error) {
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

	data, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	return string(data), nil
}