package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// FileSystemTool provides file read/write operations.
type FileSystemTool struct {
	workspace string
}

// NewFileSystemTool creates a new filesystem tool.
func NewFileSystemTool(workspace string) *FileSystemTool {
	return &FileSystemTool{workspace: workspace}
}

// Name returns the tool name.
func (t *FileSystemTool) Name() string {
	return "filesystem"
}

// StatusLine returns a formatted status line for the filesystem operation.
func (t *FileSystemTool) StatusLine(args map[string]any) string {
	op, _ := args["operation"].(string)
	path, _ := args["path"].(string)
	switch op {
	case "read":
		return fmt.Sprintf("filesystem> read %s", path)
	case "write":
		return fmt.Sprintf("filesystem> write %s", path)
	case "list":
		return fmt.Sprintf("filesystem> list %s", path)
	default:
		return fmt.Sprintf("filesystem> %s (unknown)", path)
	}
}

// Execute performs file operations.
func (t *FileSystemTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	op, ok := args["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("operation argument required")
	}

	switch op {
	case "read":
		return t.readFile(ctx, args)
	case "write":
		return t.writeFile(ctx, args)
	case "list":
		return t.listDir(ctx, args)
	default:
		return nil, fmt.Errorf("unknown operation: %s", op)
	}
}

func (t *FileSystemTool) readFile(_ context.Context, args map[string]any) (any, error) {
	path, ok := args["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path argument required")
	}

	// Resolve path - if absolute, use as-is; if relative, join with workspace
	var fullPath string
	if filepath.IsAbs(path) {
		fullPath = path
	} else {
		fullPath = filepath.Join(t.workspace, path)
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file at %s: %w", fullPath, err)
	}

	return string(data), nil
}

func (t *FileSystemTool) writeFile(_ context.Context, args map[string]any) (any, error) {
	path, ok := args["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path argument required")
	}

	content, ok := args["content"].(string)
	if !ok {
		return nil, fmt.Errorf("content argument required")
	}

	// Resolve path - if absolute, use as-is; if relative, join with workspace
	var fullPath string
	if filepath.IsAbs(path) {
		fullPath = path
	} else {
		fullPath = filepath.Join(t.workspace, path)
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	return "File written successfully", nil
}

func (t *FileSystemTool) listDir(_ context.Context, args map[string]any) (any, error) {
	path, ok := args["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path argument required")
	}

	// Resolve path - if absolute, use as-is; if relative, join with workspace
	var fullPath string
	if filepath.IsAbs(path) {
		fullPath = path
	} else {
		fullPath = filepath.Join(t.workspace, path)
	}

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to list directory: %w", err)
	}

	var result []string
	for _, entry := range entries {
		result = append(result, entry.Name())
	}

	return result, nil
}
