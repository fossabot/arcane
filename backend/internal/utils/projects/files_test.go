package projects

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/getarcaneapp/arcane/backend/internal/common"
)

func TestWriteIncludeFilePermissions(t *testing.T) {
	// Save original perms
	origFilePerm := common.FilePerm
	origDirPerm := common.DirPerm
	defer func() {
		common.FilePerm = origFilePerm
		common.DirPerm = origDirPerm
	}()

	projectDir := t.TempDir()
	includePath := filepath.Join("includes", "config.yaml")
	content := "services: {}\n"

	t.Run("Uses custom permissions", func(t *testing.T) {
		common.FilePerm = 0600
		common.DirPerm = 0700

		if err := WriteIncludeFile(projectDir, includePath, content, ExternalPathsConfig{}); err != nil {
			t.Fatalf("WriteIncludeFile() returned error: %v", err)
		}

		targetPath := filepath.Join(projectDir, includePath)
		info, err := os.Stat(targetPath)
		if err != nil {
			t.Fatalf("failed to stat include file: %v", err)
		}

		// On Linux/macOS, we can check permissions. On Windows, it's more limited.
		if runtime.GOOS != "windows" {
			if info.Mode().Perm() != 0600 {
				t.Errorf("unexpected file permissions: got %o, want %o", info.Mode().Perm(), 0600)
			}

			dirInfo, err := os.Stat(filepath.Dir(targetPath))
			if err != nil {
				t.Fatalf("failed to stat include directory: %v", err)
			}
			if dirInfo.Mode().Perm() != 0700 {
				t.Errorf("unexpected directory permissions: got %o, want %o", dirInfo.Mode().Perm(), 0700)
			}
		}
	})
}

func TestWriteIncludeFileCreatesSafeDirectory(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	includePath := filepath.Join("includes", "config.yaml")
	content := "services: {}\n"

	if err := WriteIncludeFile(projectDir, includePath, content, ExternalPathsConfig{}); err != nil {
		t.Fatalf("WriteIncludeFile() returned error: %v", err)
	}

	targetPath := filepath.Join(projectDir, includePath)
	data, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("failed to read include file: %v", err)
	}

	if string(data) != content {
		t.Fatalf("unexpected file content: got %q, want %q", string(data), content)
	}
}

func TestWriteIncludeFileRejectsSymlinkEscape(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires elevated privileges on Windows")
	}
	t.Parallel()

	projectDir := t.TempDir()
	outsideDir := t.TempDir()

	linkPath := filepath.Join(projectDir, "link")
	if err := os.Symlink(outsideDir, linkPath); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	includePath := filepath.Join("link", "escape.yaml")
	err := WriteIncludeFile(projectDir, includePath, "malicious: true\n", ExternalPathsConfig{})
	if err == nil {
		t.Fatalf("WriteIncludeFile() succeeded but expected rejection for symlink escape")
	}
}

func TestValidateFilePathWithinProject(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()

	tests := []struct {
		name      string
		filePath  string
		wantError bool
	}{
		{"relative path within project", "subdir/file.txt", false},
		{"nested path within project", "a/b/c/file.txt", false},
		{"path traversal attempt", "../outside.txt", true},
		{"absolute path outside project", "/tmp/outside.txt", true},
		{"empty path", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateFilePath(projectDir, tt.filePath, ExternalPathsConfig{}, PathValidationOptions{})
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateFilePath() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestValidateFilePathWithAllowedExternalPaths(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	allowedDir := t.TempDir()

	cfg := ExternalPathsConfig{
		AllowedPaths: []string{allowedDir},
	}

	tests := []struct {
		name      string
		filePath  string
		wantError bool
	}{
		{"path within allowed directory", filepath.Join(allowedDir, "file.txt"), false},
		{"path within project", "subdir/file.txt", false},
		{"path outside both", "/tmp/notallowed/file.txt", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateFilePath(projectDir, tt.filePath, cfg, PathValidationOptions{})
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateFilePath() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestValidateFilePathReservedNames(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()

	tests := []struct {
		name               string
		filePath           string
		checkReservedNames bool
		wantError          bool
	}{
		{"compose.yaml at root with check", "compose.yaml", true, true},
		{"compose.yaml at root without check", "compose.yaml", false, false},
		{"compose.yaml in subdir with check", "subdir/compose.yaml", true, false},
		{".env at root with check", ".env", true, true},
		{".arcane at root with check", ".arcane", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := PathValidationOptions{CheckReservedNames: tt.checkReservedNames}
			_, err := ValidateFilePath(projectDir, tt.filePath, ExternalPathsConfig{}, opts)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateFilePath() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestWriteCustomFileValidation(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()

	// Writing to a path outside project should fail without allowed paths
	err := WriteCustomFile(projectDir, "/tmp/outside.txt", "content", ExternalPathsConfig{})
	if err == nil {
		t.Error("WriteCustomFile() should reject path outside project")
	}

	// Writing to project directory should work
	err = WriteCustomFile(projectDir, "subdir/file.txt", "content", ExternalPathsConfig{})
	if err != nil {
		t.Errorf("WriteCustomFile() failed for valid path: %v", err)
	}
}

func TestIncludeAndCustomFilesShareValidation(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	allowedDir := t.TempDir()
	cfg := ExternalPathsConfig{AllowedPaths: []string{allowedDir}}

	// Both include and custom files should allow writing to allowed external paths
	externalFile := filepath.Join(allowedDir, "shared.yaml")

	err := WriteIncludeFile(projectDir, externalFile, "services: {}\n", cfg)
	if err != nil {
		t.Errorf("WriteIncludeFile() should allow writing to allowed external path: %v", err)
	}

	err = WriteCustomFile(projectDir, externalFile, "updated content", cfg)
	if err != nil {
		t.Errorf("WriteCustomFile() should allow writing to allowed external path: %v", err)
	}

	// Both should reject paths outside project and allowed paths
	outsideFile := "/tmp/not-allowed/file.yaml"

	err = WriteIncludeFile(projectDir, outsideFile, "content", cfg)
	if err == nil {
		t.Error("WriteIncludeFile() should reject path outside project and allowed paths")
	}

	err = WriteCustomFile(projectDir, outsideFile, "content", cfg)
	if err == nil {
		t.Error("WriteCustomFile() should reject path outside project and allowed paths")
	}
}
