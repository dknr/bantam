package tools

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidatePath(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "bantam_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name        string
		workspace   string
		relPath     string
		wantAbsPath string
		wantErr     bool
	}{
		{
			name:     "valid relative path",
			workspace: tmpDir,
			relPath:   "subdir/file.txt",
			wantAbsPath: filepath.Join(tmpDir, "subdir/file.txt"),
			wantErr:   false,
		},
		{
			name:     "valid relative path with dot",
			workspace: tmpDir,
			relPath:   "./subdir/file.txt",
			wantAbsPath: filepath.Join(tmpDir, "subdir/file.txt"),
			wantErr:   false,
		},
		{
			name:     "valid relative path with double dot staying inside",
			workspace: tmpDir,
			relPath:   "subdir/../other.txt",
			wantAbsPath: filepath.Join(tmpDir, "other.txt"),
			wantErr:   false,
		},
		{
			name:     "invalid path traversal outside workspace",
			workspace: tmpDir,
			relPath:   "../../outside.txt",
			wantAbsPath: "",
			wantErr:   true,
		},
		{
			name:     "absolute path not allowed",
			workspace: tmpDir,
			relPath:   "/etc/passwd",
			wantAbsPath: "",
			wantErr:   true,
		},
		{
			name:     "empty relative path",
			workspace: tmpDir,
			relPath:   "",
			wantAbsPath: tmpDir,
			wantErr:   false,
		},
		{
			name:     "relative path with extra slashes",
			workspace: tmpDir,
			relPath:   "subdir//file.txt",
			wantAbsPath: filepath.Join(tmpDir, "subdir/file.txt"),
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidatePath(tt.workspace, tt.relPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.wantAbsPath {
				t.Errorf("ValidatePath() got = %v, want %v", got, tt.wantAbsPath)
			}
		})
	}
}