package gen

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileManager_WriteFile(t *testing.T) {
	memFS := NewMemWriteFS()
	fm := NewFileManagerWithFS(memFS, nil)

	// Write file (should auto-create directories)
	err := fm.WriteFile("/a/b/c/test.txt", []byte("content"))
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Verify file exists
	data, err := memFS.ReadFile("/a/b/c/test.txt")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != "content" {
		t.Errorf("ReadFile() = %q, want %q", string(data), "content")
	}
}

func TestFileManager_WriteFileSkipExisting(t *testing.T) {
	memFS := NewMemWriteFS()
	fm := NewFileManagerWithFS(memFS, nil)

	// Pre-create file
	memFS.WriteFile("/test.txt", []byte("original"), 0o644)

	// Try to write (should skip)
	skipped, err := fm.WriteFileSkipExisting("/test.txt", []byte("new"))
	if err != nil {
		t.Fatalf("WriteFileSkipExisting() error = %v", err)
	}
	if !skipped {
		t.Error("WriteFileSkipExisting() skipped = false, want true")
	}

	// Verify original content
	data, _ := memFS.ReadFile("/test.txt")
	if string(data) != "original" {
		t.Errorf("File should not be overwritten, got %q", string(data))
	}

	// Write new file (should not skip)
	skipped, err = fm.WriteFileSkipExisting("/new.txt", []byte("new content"))
	if err != nil {
		t.Fatalf("WriteFileSkipExisting() error = %v", err)
	}
	if skipped {
		t.Error("WriteFileSkipExisting() skipped = true, want false for new file")
	}
}

func TestFileManager_ReadFile(t *testing.T) {
	memFS := NewMemWriteFS()
	fm := NewFileManagerWithFS(memFS, nil)

	// Write file
	memFS.WriteFile("/test.txt", []byte("hello"), 0o644)

	// Read file
	data, err := fm.ReadFile("/test.txt")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != "hello" {
		t.Errorf("ReadFile() = %q, want %q", string(data), "hello")
	}

	// Read non-existent file
	_, err = fm.ReadFile("/nonexistent.txt")
	if err == nil {
		t.Error("ReadFile() should return error for non-existent file")
	}
}

func TestFileManager_Exists(t *testing.T) {
	memFS := NewMemWriteFS()
	fm := NewFileManagerWithFS(memFS, nil)

	// Create file and directory
	memFS.WriteFile("/file.txt", []byte("content"), 0o644)
	memFS.MkdirAll("/dir", 0o755)

	if !fm.Exists("/file.txt") {
		t.Error("Exists() = false, want true for existing file")
	}

	if !fm.Exists("/dir") {
		t.Error("Exists() = false, want true for existing directory")
	}

	if fm.Exists("/nonexistent") {
		t.Error("Exists() = true, want false for non-existent path")
	}
}

func TestFileManager_IsDir(t *testing.T) {
	memFS := NewMemWriteFS()
	fm := NewFileManagerWithFS(memFS, nil)

	memFS.WriteFile("/file.txt", []byte("content"), 0o644)
	memFS.MkdirAll("/dir", 0o755)

	if fm.IsDir("/file.txt") {
		t.Error("IsDir() = true, want false for file")
	}

	if !fm.IsDir("/dir") {
		t.Error("IsDir() = false, want true for directory")
	}

	if fm.IsDir("/nonexistent") {
		t.Error("IsDir() = true, want false for non-existent path")
	}
}

func TestFileManager_IsFile(t *testing.T) {
	memFS := NewMemWriteFS()
	fm := NewFileManagerWithFS(memFS, nil)

	memFS.WriteFile("/file.txt", []byte("content"), 0o644)
	memFS.MkdirAll("/dir", 0o755)

	if !fm.IsFile("/file.txt") {
		t.Error("IsFile() = false, want true for file")
	}

	if fm.IsFile("/dir") {
		t.Error("IsFile() = true, want false for directory")
	}

	if fm.IsFile("/nonexistent") {
		t.Error("IsFile() = true, want false for non-existent path")
	}
}

func TestFileManager_MkdirAll(t *testing.T) {
	memFS := NewMemWriteFS()
	fm := NewFileManagerWithFS(memFS, nil)

	err := fm.MkdirAll("/a/b/c")
	if err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	if !fm.IsDir("/a/b/c") {
		t.Error("Directory should exist after MkdirAll()")
	}
}

func TestFileManager_CreateDirs(t *testing.T) {
	memFS := NewMemWriteFS()
	fm := NewFileManagerWithFS(memFS, nil)

	dirs := []string{"/dir1", "/dir2/nested", "/dir3"}
	err := fm.CreateDirs(dirs)
	if err != nil {
		t.Fatalf("CreateDirs() error = %v", err)
	}

	for _, dir := range dirs {
		if !fm.IsDir(dir) {
			t.Errorf("Directory %q should exist", dir)
		}
	}
}

func TestFileManager_Remove(t *testing.T) {
	memFS := NewMemWriteFS()
	fm := NewFileManagerWithFS(memFS, nil)

	// Create and remove file
	memFS.WriteFile("/test.txt", []byte("content"), 0o644)

	err := fm.Remove("/test.txt")
	if err != nil {
		t.Fatalf("Remove() error = %v", err)
	}

	if fm.Exists("/test.txt") {
		t.Error("File should not exist after Remove()")
	}
}

func TestFileManager_FS(t *testing.T) {
	memFS := NewMemWriteFS()
	fm := NewFileManagerWithFS(memFS, nil)

	if fm.FS() != memFS {
		t.Error("FS() should return the underlying WriteFS")
	}
}

func TestFileManager_DefaultFS(t *testing.T) {
	// Test NewFileManager creates OSWriteFS by default
	fm := NewFileManager(nil)

	_, ok := fm.FS().(*OSWriteFS)
	if !ok {
		t.Error("NewFileManager() should use OSWriteFS by default")
	}
}

func TestFileManager_RemoveAll(t *testing.T) {
	// Test with real filesystem
	tmpDir := t.TempDir()
	fm := NewFileManager(nil)

	// Create nested structure
	nestedDir := filepath.Join(tmpDir, "a", "b", "c")
	if err := os.MkdirAll(nestedDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nestedDir, "file.txt"), []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	// RemoveAll
	targetDir := filepath.Join(tmpDir, "a")
	err := fm.RemoveAll(targetDir)
	if err != nil {
		t.Fatalf("RemoveAll() error = %v", err)
	}

	if fm.Exists(targetDir) {
		t.Error("Directory should not exist after RemoveAll()")
	}
}
