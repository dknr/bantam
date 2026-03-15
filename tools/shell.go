package tools

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// ShellTool executes shell commands.
type ShellTool struct{}

// NewShellTool creates a new shell tool.
func NewShellTool() *ShellTool {
	return &ShellTool{}
}

// Name returns the tool name.
func (t *ShellTool) Name() string {
	return "shell"

}

// Execute runs a shell command.
func (t *ShellTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	cmdStr, ok := args["command"].(string)
	if !ok {
		return nil, fmt.Errorf("command argument required")
	}

	cmd := exec.CommandContext(ctx, "sh", "-c", cmdStr)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("command failed: %w, output: %s", err, string(output))
	}

	return strings.TrimSpace(string(output)), nil
}
