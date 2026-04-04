package tools

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// GitTool runs git commands with a whitelist of non-destructive subcommands.
type GitTool struct {
	workspaceDir string
}

// NewGitTool creates a new git tool.
func NewGitTool(workspaceDir string) *GitTool {
	return &GitTool{workspaceDir: workspaceDir}
}

// StatusLine returns a formatted status line for the git command.
func (t *GitTool) StatusLine(args map[string]any) string {
	if argv, ok := args["args"].([]any); ok {
		var strs []string
		for _, v := range argv {
			if s, ok := v.(string); ok {
				strs = append(strs, s)
			}
		}
		// Path is now required, so we expect it to be present
		if path, ok := args["path"].(string); ok {
			return fmt.Sprintf("(%s) git> %s", path, strings.Join(strs, " "))
		}
		// Fallback (should not happen if required)
		return fmt.Sprintf("git> %s", strings.Join(strs, " "))
	}
	return "git> (unknown args)"
}

// ToolSchema returns the parameter schema for the git tool.
func (t *GitTool) ToolSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"args": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "Arguments to pass to the git command (e.g., ['status', '--short']). Only non-destructive subcommands are allowed.",
			},
			"path": map[string]any{
				"type":        "string",
				"description": "The working directory for the git command, relative to workspace. This parameter is required.",
			},
		},
		"required": []string{"args", "path"},
	}
}

// Name returns the tool name.
func (t *GitTool) Name() string {
	return "git"
}

// Execute runs a git command and returns its output.
func (t *GitTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	argvAny, ok := args["args"]
	if !ok {
		return "", fmt.Errorf("args argument is required")
	}
	argvInterface, ok := argvAny.([]any)
	if !ok {
		return "", fmt.Errorf("args must be a list of strings")
	}
	var argv []string
	for _, v := range argvInterface {
		if s, ok := v.(string); ok {
			argv = append(argv, s)
		} else {
			return "", fmt.Errorf("all args must be strings")
		}
	}

	// Extract subcommand: first non-flag argument
	subcommand := ""
	for _, arg := range argv {
		if len(arg) > 0 && arg[0] != '-' {
			subcommand = arg
			break
		}
	}

	// Whitelist of non-destructive git subcommands
	whitelist := map[string]bool{
		"status":      true,
		"log":         true,
		"diff":        true,
		"show":        true,
		"branch":      true,
		"checkout":    true,
		"clone":       true,
		"fetch":       true,
		"pull":        true,
		"remote":      true,
		"config":      true,
		"ls-files":    true,
		"ls-tree":     true,
		"rev-parse":   true,
		"describe":    true,
		"blame":       true,
		"grep":        true,
		"stash":       true,
		"show-branch": true,
		"tag":         true,
		"notes":       true,
		"verify":      true,
		"whatchanged": true,
		"help":        true,
	}

	if subcommand != "" && !whitelist[subcommand] {
		return "", fmt.Errorf("git subcommand %q is not allowed", subcommand)
	}

	// Path is now required
	pathAny, ok := args["path"]
	if !ok {
		return "", fmt.Errorf("path argument is required")
	}
	pathArg, ok := pathAny.(string)
	if !ok {
		return "", fmt.Errorf("path must be a string")
	}

	// Determine working directory using ValidatePath
	absPath, err := ValidatePath(t.workspaceDir, pathArg)
	if err != nil {
		return "", err
	}

	// Prepare command: git <args>
	cmdArgs := append([]string{"git"}, argv...)
	cmd := exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)
	cmd.Dir = absPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git command failed: %w", err)
	}
	return string(output), nil
}
