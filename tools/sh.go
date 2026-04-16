package tools

import (
	"context"
	"fmt"
	"os/exec"
)

// ShTool provides shell command execution.
type ShTool struct {
	workspace string
}

// NewShTool creates a new shell tool.
func NewShTool(workspace string) *ShTool {
	return &ShTool{workspace: workspace}
}

// Name returns the tool name.
func (t *ShTool) Name() string {
	return "sh"
}

// StatusLine returns a formatted status line for the shell operation.
func (t *ShTool) StatusLine(args map[string]any) string {
	input, _ := args["input"].(string)
	if len(input) > 50 {
		return fmt.Sprintf("sh> %s...", input[:50])
	}
	return fmt.Sprintf("sh> %s", input)
}

// ToolSchema returns the parameter schema for the sh tool.
func (t *ShTool) ToolSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"input": map[string]any{
				"type":        "string",
				"description": "The shell command to execute",
			},
		},
		"required": []string{"input"},
	}
}

// Execute runs the shell command and returns its combined output.
func (t *ShTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	inputRaw, ok := args["input"]
	if !ok {
		return nil, fmt.Errorf("missing required parameter: input")
	}
	input, ok := inputRaw.(string)
	if !ok {
		return nil, fmt.Errorf("parameter 'input' must be a string")
	}
	if input == "" {
		// empty command, return empty string
		return "", nil
	}

	// Create command
	cmd := exec.CommandContext(ctx, "sh", "-c", input)
	if t.workspace != "" {
		cmd.Dir = t.workspace
	}
	// Capture combined output
	output, err := cmd.CombinedOutput()
	if err != nil {
		// If the command ran but exited with non-zero, we still want to return output
		// CombinedOutput already includes stderr; we can return output and err?
		// For consistency with other tools, we return output as string and ignore err?
		// Let's return output as string and nil error, similar to how shell would just output.
		// However, if command failed to start (e.g., sh not found), we should return error.
		// We'll check if err is ExitError; if so, we still return output.
		// We'll use a type assertion.
		if _, ok := err.(*exec.ExitError); ok {
			// Command ran but exited with non-zero
			return string(output), nil
		}
		// Failed to start
		return nil, fmt.Errorf("failed to execute shell command: %w", err)
	}
	return string(output), nil
}
