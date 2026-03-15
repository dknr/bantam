package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileTool provides file read/write operations.
type FileTool struct {
	workspace string
}

// NewFileTool creates a new file tool.
func NewFileTool(workspace string) *FileTool {
	return &FileTool{workspace: workspace}
}

// Name returns the tool name.
func (t *FileTool) Name() string {
	return "file"
}

// StatusLine returns a formatted status line for the file operation.
func (t *FileTool) StatusLine(args map[string]any) string {
	action, _ := args["action"].(string)
	path, _ := args["path"].(string)
	switch action {
	case "read":
		return fmt.Sprintf("file> read %s", path)
	case "write":
		return fmt.Sprintf("file> write %s", path)
	case "list":
		return fmt.Sprintf("file> list %s", path)
	default:
		return fmt.Sprintf("file> %s (unknown)", path)
	}
}

// Execute performs file operations based on the action.
func (t *FileTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	action, ok := args["action"].(string)
	if !ok {
		return "", fmt.Errorf("action argument is required")
	}

	switch action {
	case "read":
		return t.readFile(ctx, args)
	case "write":
		return t.writeFile(ctx, args)
	case "list":
		return t.listDirectory(ctx, args)
	default:
		return "", fmt.Errorf("unknown action: %s", action)
	}
}

func (t *FileTool) readFile(_ context.Context, args map[string]any) (any, error) {
	relPath, ok := args["path"].(string)
	if !ok {
		return "", fmt.Errorf("path argument is required")
	}

	absPath := filepath.Join(t.workspace, relPath)
	data, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	return string(data), nil
}

func (t *FileTool) writeFile(_ context.Context, args map[string]any) (any, error) {
	relPath, ok := args["path"].(string)
	if !ok {
		return "", fmt.Errorf("path argument is required")
	}

	content, ok := args["content"].(string)
	if !ok {
		return "", fmt.Errorf("content argument is required")
	}

	absPath := filepath.Join(t.workspace, relPath)

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

func (t *FileTool) listDirectory(_ context.Context, args map[string]any) (any, error) {
	relPath, ok := args["path"].(string)
	if !ok {
		relPath = "."
	}

	absPath := filepath.Join(t.workspace, relPath)

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
