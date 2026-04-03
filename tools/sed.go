package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// EditTool provides file write operations.
type SedTool struct {
	workspace string
}

// NewEditTool creates a new edit tool.
func NewSedTool(workspace string) *SedTool {
	return &SedTool{workspace: workspace}
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

// ToolSchema returns the parameter schema for the sed tool.
func (t *SedTool) ToolSchema() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Path relative to workspace. Use relative paths only (e.g., file.md, subdir/file.txt). Never use absolute paths.",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "The content to write to the file.",
			},
		},
		"description": "Write content to a file in the workspace directory. Only relative paths are allowed.",
		"required":    []string{"path", "content"},
	}
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

	absPath, err := ValidatePath(t.workspace, relPath)
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