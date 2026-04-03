package tools

import (
	"context"
	"bytes"
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

// ToolSchema returns the parameter schema for the grep tool.
func (t *GrepTool) ToolSchema() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{
			"pattern": map[string]any{
				"type":        "string",
				"description": "The pattern to search for (regex unless literal_text is true)",
			},
			"path": map[string]any{
				"type":        "string",
				"description": "The path to search, relative to workspace. Defaults to '.' (current directory).",
			},
			"literal_text": map[string]any{
				"type":        "boolean",
				"description": "If true, treat the pattern as a literal string rather than a regular expression. Defaults to false.",
			},
		},
		"required": []string{"pattern"},
	}
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
	absPath, err := ValidatePath(t.workspace, path)
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

	var result string
		if len(matches) == 0 {
			result = "No matches found."
		} else {
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
			result = strings.Join(resultLines, "\n")
		}

		// Apply 8kB limit
		const maxOutputSize = 8192
		if len([]byte(result)) > maxOutputSize {
			resultBytes := []byte(result)
			truncated := resultBytes[:maxOutputSize]
			lastNewline := bytes.LastIndex(truncated, []byte{'\n'})
			if lastNewline == -1 {
				result = string(truncated) + "\n[Output truncated to 8kB]"
			} else {
				result = string(truncated[:lastNewline]) + "\n[Output truncated to 8kB]"
			}
		}

		return result, nil
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