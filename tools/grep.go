package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// GrepTool searches for patterns in files.
type GrepTool struct {
	workspace string
}

// NewGrepTool creates a new grep tool.
func NewGrepTool(workspace string) *GrepTool {
	return &GrepTool{workspace: workspace}
}

// validatePath ensures the resolved path is within the workspace directory.
// This is the same validation used by the FileTool.
func (t *GrepTool) validatePath(relPath string) (string, error) {
	// Clean the path to resolve any .. or .
	cleanPath := filepath.Clean(relPath)

	// Join with workspace and get absolute path
	absPath := filepath.Join(t.workspace, cleanPath)

	// Get absolute path of workspace
	absWorkspace, err := filepath.Abs(t.workspace)
	if err != nil {
		return "", fmt.Errorf("invalid workspace: %w", err)
	}

	// Get absolute path of the target
	absTarget, err := filepath.Abs(absPath)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	// Check if target is within workspace using filepath.Rel
	rel, err := filepath.Rel(absWorkspace, absTarget)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	// If the relative path starts with .., it's outside the workspace
	if strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("path traversal not allowed: %s", relPath)
	}

	return absPath, nil
}

// Name returns the tool name.
func (t *GrepTool) Name() string {
	return "grep"
}

// StatusLine returns a formatted status line for the grep operation.
func (t *GrepTool) StatusLine(args map[string]any) string {
	pattern, _ := args["pattern"].(string)
	path, _ := args["path"].(string)
	if path == "" {
		path = "."
	}
	return fmt.Sprintf("grep> pattern=%s path=%s", pattern, path)
}

// Execute performs the grep operation.
func (t *GrepTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	pattern, ok := args["pattern"].(string)
	if !ok {
		return "", fmt.Errorf("pattern argument is required")
	}

	// Optional path, default to "."
	path, _ := args["path"].(string)
	if path == "" {
		path = "."
	}

	// Optional literal_text, default to false
	literalText, _ := args["literal_text"].(bool)

	// Validate the path
	absPath, err := t.validatePath(path)
	if err != nil {
		return "", err
	}

	// Check if it's a file or directory
	fileInfo, err := os.Stat(absPath)
	if err != nil {
		return "", fmt.Errorf("failed to stat path: %w", err)
	}

	type matchResult struct {
		file   string
		line   int
		content string
	}
	var matches []matchResult

	if fileInfo.IsDir() {
		// Walk the directory
		err := filepath.Walk(absPath, func(filePath string, info os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if info.IsDir() {
				return nil
			}
			// Skip hidden files and directories
			if strings.HasPrefix(info.Name(), ".") {
				return nil
			}
			// Search the file
			fileMatches, err := t.searchFile(filePath, pattern, literalText)
			if err != nil {
				return err
			}
			// Parse the matches (they are formatted as "file:line:content")
			for _, m := range fileMatches {
				// We expect the format from searchFile to be "relative_file:line:content"
				// But note: the searchFile returns matches with relative file path already?
				// Actually, in searchFile we format with relPath, which is relative to workspace.
				// So we can split by the first two colons.
				parts := strings.SplitN(m, ":", 3)
				if len(parts) != 3 {
					continue // skip malformed
				}
				file := parts[0]
				lineNum := 0
				fmt.Sscanf(parts[1], "%d", &lineNum)
				content := parts[2]
				matches = append(matches, matchResult{file: file, line: lineNum, content: content})
			}
			return nil
		})
		if err != nil {
			return "", fmt.Errorf("error walking directory: %w", err)
		}
	} else {
		// Search the single file
		fileMatches, err := t.searchFile(absPath, pattern, literalText)
		if err != nil {
			return "", err
		}
		for _, m := range fileMatches {
			parts := strings.SplitN(m, ":", 3)
			if len(parts) != 3 {
				continue
			}
			file := parts[0]
			lineNum := 0
			fmt.Sscanf(parts[1], "%d", &lineNum)
			content := parts[2]
			matches = append(matches, matchResult{file: file, line: lineNum, content: content})
		}
	}

	if len(matches) == 0 {
		return "No matches found.", nil
	}

	// Sort matches by file path and line number
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].file != matches[j].file {
			return matches[i].file < matches[j].file
		}
		return matches[i].line < matches[j].line
	})

	// Format the results
	var resultLines []string
	for _, m := range matches {
		resultLines = append(resultLines, fmt.Sprintf("%s:%d:%s", m.file, m.line, m.content))
	}
	return strings.Join(resultLines, "\n"), nil
}

// searchFile searches a single file for the pattern.
func (t *GrepTool) searchFile(filePath string, pattern string, literalText bool) ([]string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	lines := strings.Split(string(data), "\n")
	var matches []string

	var re *regexp.Regexp
	if !literalText {
		// Compile the regex pattern
		re, err = regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid regex pattern: %w", err)
		}
	}

	for i, line := range lines {
		var match bool
		if literalText {
			match = strings.Contains(line, pattern)
		} else {
			match = re.MatchString(line)
		}
		if match {
			// Format: file:line_number:line_content
			// We need to make the file path relative to the workspace for output
			relPath, err := filepath.Rel(t.workspace, filePath)
			if err != nil {
				relPath = filePath
			}
			matches = append(matches, fmt.Sprintf("%s:%d:%s", relPath, i+1, line))
		}
	}

	return matches, nil
}