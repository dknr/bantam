package memory

import (
	"context"
	"fmt"
	"strings"
)

// MemoryTool provides memory operations using SQLite.
type MemoryTool struct {
	db *DB
}

// NewMemoryTool creates a new memory tool with the given base directory.
func NewMemoryTool(baseDir string) (*MemoryTool, error) {
	db, err := NewDB(baseDir)
	if err != nil {
		return nil, err
	}
	return &MemoryTool{db: db}, nil
}

// Name returns the tool name.
func (t *MemoryTool) Name() string {
	return "memory"
}

// Execute executes memory operations based on the action.
func (t *MemoryTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	// Get the action from arguments
	action, ok := args["action"].(string)
	if !ok {
		return "", fmt.Errorf("action argument is required")
	}

	// Normalize action name (remove underscore prefix if present)
	action = strings.TrimPrefix(action, "memory_")
	action = strings.TrimPrefix(action, "history_")

	switch action {
	case "read":
		return t.memoryRead(ctx, args)
	case "write":
		return t.memoryWrite(ctx, args)
	case "list":
		return t.memoryList(ctx, args)
	case "search":
		return t.historySearch(ctx, args)
	case "since":
		return t.historySince(ctx, args)
	default:
		return "", fmt.Errorf("unknown action: %s", action)
	}
}

// memoryRead reads a memory entry by key.
func (t *MemoryTool) memoryRead(_ context.Context, args map[string]any) (any, error) {
	key, ok := args["key"].(string)
	if !ok {
		return "", fmt.Errorf("key argument is required")
	}

	content, exists, err := t.db.MemoryGet(key)
	if err != nil {
		return "", fmt.Errorf("failed to read memory: %w", err)
	}

	if !exists {
		return fmt.Sprintf("Memory key '%s' not found", key), nil
	}

	return content, nil
}

// memoryWrite writes a memory entry with compare-exchange.
func (t *MemoryTool) memoryWrite(_ context.Context, args map[string]any) (any, error) {
	key, ok := args["key"].(string)
	if !ok {
		return "", fmt.Errorf("key argument is required")
	}

	oldValue, _ := args["old_value"].(string)
	newValue, ok := args["new_value"].(string)
	if !ok {
		return "", fmt.Errorf("new_value argument is required")
	}

	success, err := t.db.MemoryWrite(key, oldValue, newValue)
	if err != nil {
		return "", fmt.Errorf("failed to write memory: %w", err)
	}

	if !success {
		return "Error: Memory has changed since you last read it. The compare-exchange operation failed. Please read the current value and try again.", nil
	}

	if oldValue == "" {
		return fmt.Sprintf("Successfully created memory key '%s'", key), nil
	}

	return fmt.Sprintf("Successfully updated memory key '%s'", key), nil
}

// memoryList lists all memory keys.
func (t *MemoryTool) memoryList(_ context.Context, args map[string]any) (any, error) {
	keys, err := t.db.MemoryList()
	if err != nil {
		return "", fmt.Errorf("failed to list memories: %w", err)
	}

	if len(keys) == 0 {
		return "No memories found", nil
	}

	return strings.Join(keys, "\n"), nil
}

// historySearch searches history entries by text.
func (t *MemoryTool) historySearch(_ context.Context, args map[string]any) (any, error) {
	query, ok := args["query"].(string)
	if !ok {
		return "", fmt.Errorf("query argument is required")
	}

	entries, err := t.db.HistorySearch(query)
	if err != nil {
		return "", fmt.Errorf("failed to search history: %w", err)
	}

	if len(entries) == 0 {
		return fmt.Sprintf("No history entries found matching '%s'", query), nil
	}

	return strings.Join(entries, "\n"), nil
}

// historySince returns history entries since a timestamp.
func (t *MemoryTool) historySince(_ context.Context, args map[string]any) (any, error) {
	timestamp, ok := args["timestamp"].(string)
	if !ok {
		return "", fmt.Errorf("timestamp argument is required")
	}

	entries, err := t.db.HistorySince(timestamp)
	if err != nil {
		return "", fmt.Errorf("failed to query history: %w", err)
	}

	if len(entries) == 0 {
		return fmt.Sprintf("No history entries found since %s", timestamp), nil
	}

	return strings.Join(entries, "\n"), nil
}

// Close closes the database connection.
func (t *MemoryTool) Close() error {
	return t.db.Close()
}
