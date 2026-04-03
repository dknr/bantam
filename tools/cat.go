package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// ViewTool provides file read operations.
type CatTool struct {
	workspace string
}

// NewViewTool creates a new view tool.
func NewCatTool(workspace string) *CatTool {
	return &CatTool{workspace: workspace}
}

// Name returns the tool name.
func (t *CatTool) Name() string {
	return "cat"
}

// StatusLine returns a formatted status line for the view operation.
func (t *CatTool) StatusLine(args map[string]any) string {
	path, _ := args["path"].(string)
	return fmt.Sprintf("cat> %s", path)
}

// ToolSchema returns the parameter schema for the cat tool.
func (t *CatTool) ToolSchema() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Path relative to workspace. Use relative paths only (e.g., file.md, subdir/file.txt). Never use absolute paths.",
			},
		},
		"description": "Read file contents from the workspace directory. Only relative paths are allowed.",
		"required":    []string{"path"},
	}
}

// Execute performs file read based on the action.
func (t *CatTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	relPath, ok := args["path"].(string)
	if !ok {
		return "", fmt.Errorf("path argument is required")
	}

	// Check for absolute paths
	if filepath.IsAbs(relPath) {
		return "", fmt.Errorf("paths must be relative to the workspace directory. Use relative paths like 'file.md' instead of absolute paths.")
	}

	absPath, err := ValidatePath(t.workspace, relPath)
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	return string(data), nil
}