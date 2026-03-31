package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ListTool provides directory listing operations.
type ListTool struct {
	workspace string
}

// NewListTool creates a new list tool.
func NewListTool(workspace string) *ListTool {
	return &ListTool{workspace: workspace}
}

// validatePath ensures the resolved path is within the workspace directory.
func (t *ListTool) validatePath(relPath string) (string, error) {
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
func (t *ListTool) Name() string {
	return "list"
}

// StatusLine returns a formatted status line for the list operation.
func (t *ListTool) StatusLine(args map[string]any) string {
	path, _ := args["path"].(string)
	if path == "" {
		path = "."
	}
	return fmt.Sprintf("list> %s", path)
}

// Execute performs directory listing based on the action.
func (t *ListTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	relPath, ok := args["path"].(string)
	if !ok {
		relPath = "."
	}

	// Check for absolute paths
	if filepath.IsAbs(relPath) {
		return "", fmt.Errorf("paths must be relative to the workspace directory. Use relative paths like 'file.md' instead of absolute paths.")
	}

	absPath, err := t.validatePath(relPath)
	if err != nil {
		return "", err
	}

	entries, err := os.ReadDir(absPath)
	if err != nil {
		return "", fmt.Errorf("failed to list directory: %w", err)
	}

	var result []string
	for _, entry := range entries {
		if entry.IsDir() {
			result = append(result, fmt.Sprintf("📁 %s/", entry.Name()))
		} else {
			result = append(result, fmt.Sprintf("📄 %s", entry.Name()))
		}
	}

	return strings.Join(result, "\n"), nil
}