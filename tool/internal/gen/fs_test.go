package gen

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMemWriteFS_WriteAndRead(t *testing.T) {
	fs := NewMemWriteFS()

	// Write file
	content := []byte("hello world")
	if err := fs.WriteFile("/test/file.txt", content, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Read file
	data, err := fs.ReadFile("/test/file.txt")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if string(data) != "hello world" {
		t.Errorf("ReadFile() = %q, want %q", string(data), "hello world")
	}
}

func TestMemWriteFS_Stat(t *testing.T) {
	fs := NewMemWriteFS()

	// Write file
	if err := fs.WriteFile("/test/file.txt", []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Stat file
	info, err := fs.Stat("/test/file.txt")
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}

	if info.Name() != "file.txt" {
		t.Errorf("Stat().Name() = %q, want %q", info.Name(), "file.txt")
	}

	if info.IsDir() {
		t.Error("Stat().IsDir() = true, want false")
	}

	if info.Size() != 7 {
		t.Errorf("Stat().Size() = %d, want 7", info.Size())
	}
}

func TestMemWriteFS_MkdirAll(t *testing.T) {
	fs := NewMemWriteFS()

	// Create directory
	if err := fs.MkdirAll("/a/b/c", 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	// Check directory exists
	info, err := fs.Stat("/a/b/c")
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}

	if !info.IsDir() {
		t.Error("Stat().IsDir() = false, want true")
	}

	// Check parent directories were created
	dirs := fs.GetDirs()
	expectedDirs := map[string]bool{"/a": true, "/a/b": true, "/a/b/c": true}
	for _, dir := range dirs {
		if !expectedDirs[dir] {
			// May have cleaned path, check without leading slash
			clean := filepath.Clean(dir)
			if !expectedDirs[clean] && !expectedDirs["/"+clean] {
				t.Logf("Unexpected dir: %s", dir)
			}
		}
	}
}

func TestMemWriteFS_Remove(t *testing.T) {
	fs := NewMemWriteFS()

	// Write file
	if err := fs.WriteFile("/test/file.txt", []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Remove file
	if err := fs.Remove("/test/file.txt"); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}

	// File should not exist
	_, err := fs.Stat("/test/file.txt")
	if !os.IsNotExist(err) {
		t.Errorf("Stat() after Remove() should return ErrNotExist, got %v", err)
	}

	// Remove non-existent file should error
	err = fs.Remove("/non/existent")
	if !os.IsNotExist(err) {
		t.Errorf("Remove() non-existent should return ErrNotExist, got %v", err)
	}
}

func TestMemWriteFS_GetFiles(t *testing.T) {
	fs := NewMemWriteFS()

	// Write multiple files
	files := map[string]string{
		"/a/file1.txt": "content1",
		"/b/file2.txt": "content2",
		"/c/file3.txt": "content3",
	}

	for path, content := range files {
		if err := fs.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// Get all files
	result := fs.GetFiles()
	if len(result) != len(files) {
		t.Errorf("GetFiles() returned %d files, want %d", len(result), len(files))
	}
}

func TestMemWriteFS_DataIsolation(t *testing.T) {
	fs := NewMemWriteFS()

	// Write file
	original := []byte("original content")
	if err := fs.WriteFile("/test.txt", original, 0o644); err != nil {
		t.Fatal(err)
	}

	// Modify original slice
	original[0] = 'X'

	// Read should return unmodified content
	data, err := fs.ReadFile("/test.txt")
	if err != nil {
		t.Fatal(err)
	}

	if data[0] == 'X' {
		t.Error("MemWriteFS should store a copy, not reference to original data")
	}

	// Modify returned data
	data[0] = 'Y'

	// Read again should return unmodified content
	data2, _ := fs.ReadFile("/test.txt")
	if data2[0] == 'Y' {
		t.Error("MemWriteFS should return a copy, not reference to stored data")
	}
}

func TestDryRunWriteFS_RecordsOperations(t *testing.T) {
	inner := NewMemWriteFS()
	// Pre-create a file in inner FS for reading
	inner.WriteFile("/existing.txt", []byte("existing"), 0o644)

	dryRun := NewDryRunWriteFS(inner, nil)

	// Write file (should not actually write)
	if err := dryRun.WriteFile("/new/file.txt", []byte("content"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Mkdir (should not actually create)
	if err := dryRun.MkdirAll("/new/dir", 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	// Remove (should not actually remove)
	if err := dryRun.Remove("/existing.txt"); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}

	// Check operations were recorded
	ops := dryRun.GetOps()
	if len(ops) != 3 {
		t.Errorf("GetOps() returned %d ops, want 3", len(ops))
	}

	// Verify operations
	expectedOps := []struct {
		op   string
		path string
	}{
		{"write", "/new/file.txt"},
		{"mkdir", "/new/dir"},
		{"remove", "/existing.txt"},
	}

	for i, expected := range expectedOps {
		if i >= len(ops) {
			break
		}
		if ops[i].Op != expected.op {
			t.Errorf("ops[%d].Op = %q, want %q", i, ops[i].Op, expected.op)
		}
		if ops[i].Path != expected.path {
			t.Errorf("ops[%d].Path = %q, want %q", i, ops[i].Path, expected.path)
		}
	}

	// Verify file was NOT written to inner FS
	_, err := inner.Stat("/new/file.txt")
	if err == nil {
		t.Error("DryRun should not write files to inner FS")
	}

	// Verify existing file was NOT removed
	_, err = inner.Stat("/existing.txt")
	if err != nil {
		t.Error("DryRun should not remove files from inner FS")
	}
}

func TestDryRunWriteFS_ReadPassthrough(t *testing.T) {
	inner := NewMemWriteFS()
	inner.WriteFile("/test.txt", []byte("content"), 0o644)

	dryRun := NewDryRunWriteFS(inner, nil)

	// Read should work (passthrough to inner)
	data, err := dryRun.ReadFile("/test.txt")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if string(data) != "content" {
		t.Errorf("ReadFile() = %q, want %q", string(data), "content")
	}

	// Stat should work (passthrough to inner)
	info, err := dryRun.Stat("/test.txt")
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}

	if info.Name() != "test.txt" {
		t.Errorf("Stat().Name() = %q, want %q", info.Name(), "test.txt")
	}
}

func TestOSWriteFS(t *testing.T) {
	fs := &OSWriteFS{}
	tmpDir := t.TempDir()

	// Test MkdirAll
	testDir := filepath.Join(tmpDir, "a", "b", "c")
	if err := fs.MkdirAll(testDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	// Test WriteFile
	testFile := filepath.Join(testDir, "test.txt")
	if err := fs.WriteFile(testFile, []byte("hello"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Test ReadFile
	data, err := fs.ReadFile(testFile)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != "hello" {
		t.Errorf("ReadFile() = %q, want %q", string(data), "hello")
	}

	// Test Stat
	info, err := fs.Stat(testFile)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if info.Name() != "test.txt" {
		t.Errorf("Stat().Name() = %q, want %q", info.Name(), "test.txt")
	}

	// Test Remove
	if err := fs.Remove(testFile); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	_, err = fs.Stat(testFile)
	if !os.IsNotExist(err) {
		t.Error("File should not exist after Remove()")
	}
}
