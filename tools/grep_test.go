package tools

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestGrepTool(t *testing.T) {
	// Create a temporary directory for testing
	dir, err := os.MkdirTemp("", "bantam_grep_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	// Create a test file
	testFile := filepath.Join(dir, "test.txt")
	content := `hello world
this is a test
grep tool test
another line
world hello
`
	err = os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Create a subdirectory and a file there
	subdir := filepath.Join(dir, "subdir")
	err = os.MkdirAll(subdir, 0755)
	if err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}
	subFile := filepath.Join(subdir, "sub.txt")
	subContent := `subdirectory content
with hello world
and more
`
	err = os.WriteFile(subFile, []byte(subContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write sub file: %v", err)
	}

	// Create the grep tool
	tool := NewGrepTool(dir)

	// Test literal search for "hello"
	args := map[string]any{
		"pattern":      "hello",
		"literal_text": true,
	}
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Grep tool failed: %v", err)
	}
	expected := `test.txt:1:hello world
test.txt:5:world hello
subdir/sub.txt:2:with hello world
`
	// Compare sorted lines
	if !compareSortedLines(t, result.(string), expected) {
		t.Errorf("Expected lines (sorted):\n%s\nGot lines (sorted):\n%s", sortLines(expected), sortLines(result.(string)))
	}

	// Test regex search for "hello.*world"
	args = map[string]any{
		"pattern":      "hello.*world",
		"literal_text": false,
	}
	result, err = tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Grep tool regex failed: %v", err)
	}
	expectedRegex := `test.txt:1:hello world
subdir/sub.txt:2:with hello world
`
	if !compareSortedLines(t, result.(string), expectedRegex) {
		t.Errorf("Expected regex lines (sorted):\n%s\nGot lines (sorted):\n%s", sortLines(expectedRegex), sortLines(result.(string)))
	}

	// Test path argument
	args = map[string]any{
		"pattern":      "subdirectory",
		"literal_text": true,
		"path":         "subdir",
	}
	result, err = tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Grep tool with path failed: %v", err)
	}
	expectedPath := `subdir/sub.txt:1:subdirectory content
`
	if !compareSortedLines(t, result.(string), expectedPath) {
		t.Errorf("Expected path lines (sorted):\n%s\nGot lines (sorted):\n%s", sortLines(expectedPath), sortLines(result.(string)))
	}

	// Test no matches
	args = map[string]any{
		"pattern":      "nomatch",
		"literal_text": true,
	}
	result, err = tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Grep tool nomatch failed: %v", err)
	}
	if result.(string) != "No matches found." {
		t.Errorf("Expected 'No matches found.', got: %v", result)
	}
}

// sortLines returns the lines of a string sorted lexicographically.
func sortLines(s string) string {
	lines := strings.Split(s, "\n")
	// Remove empty lines that might be from trailing newline
	var nonEmpty []string
	for _, line := range lines {
		if line != "" {
			nonEmpty = append(nonEmpty, line)
		}
	}
	sort.Strings(nonEmpty)
	return strings.Join(nonEmpty, "\n")
}

// compareSortedLines compares two strings by their sorted lines.
func compareSortedLines(t *testing.T, got, want string) bool {
	if sortLines(got) == sortLines(want) {
		return true
	}
	t.Logf("Expected sorted lines:\n%s", sortLines(want))
	t.Logf("Got sorted lines:\n%s", sortLines(got))
	return false
}
