package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ListTool provides directory listing operations.
type LsTool struct {
	workspace string
}

// NewListTool creates a new list tool.
func NewLsTool(workspace string) *LsTool {
	return &LsTool{workspace: workspace}
}

// Name returns the tool name.
func (t *LsTool) Name() string {
	return "ls"
}

// StatusLine returns a formatted status line for the list operation.
func (t *LsTool) StatusLine(args map[string]any) string {
	path, _ := args["path"].(string)
	if path == "" {
		path = "."
	}
	return fmt.Sprintf("ls> %s", path)
}

// ToolSchema returns the parameter schema for the ls tool.
func (t *LsTool) ToolSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "The path to list, relative to workspace. Defaults to \".\" (current directory).",
			},
		},
		"description": "List directory contents in the workspace directory. Only relative paths are allowed.",
		// path optional, no required field
	}
}

// Execute performs directory listing based on the action.
func (t *LsTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	relPath, ok := args["path"].(string)
	if !ok {
		relPath = "."
	}

	// Check for absolute paths
	if filepath.IsAbs(relPath) {
		return "", fmt.Errorf("paths must be relative to the workspace directory. Use relative paths like 'file.md' instead of absolute paths.")
	}

	absPath, err := ValidatePath(t.workspace, relPath)
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
