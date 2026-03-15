package tools

import (
	"context"
	"fmt"
	"os/exec"
)

// ExecTool executes shell commands.
type ExecTool struct{}

// StatusLine returns a formatted status line for the shell command.
func (t *ExecTool) StatusLine(args map[string]any) string {
	if cmd, ok := args["command"].(string); ok {
		return fmt.Sprintf("exec> %s", cmd)
	}
	return "exec> (unknown command)"
}

// NewExecTool creates a new exec tool.
func NewExecTool() *ExecTool {
	return &ExecTool{}
}

// Name returns the tool name.
func (t *ExecTool) Name() string {
	return "exec"
}

// Execute runs a shell command and returns its output.
func (t *ExecTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	cmdStr, ok := args["command"].(string)
	if !ok {
		return "", fmt.Errorf("command argument is required")
	}

	cmd := exec.CommandContext(ctx, "sh", "-c", cmdStr)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("command failed: %w", err)
	}

	return string(output), nil
}
