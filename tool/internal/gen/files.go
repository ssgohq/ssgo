package gen

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// FileManager handles file operations.
type FileManager struct {
	wfs WriteFS
	log Logger
}

// NewFileManager creates a new FileManager using the OS filesystem.
func NewFileManager(log Logger) *FileManager {
	return NewFileManagerWithFS(&OSWriteFS{}, log)
}

// NewFileManagerWithFS creates a FileManager with a custom WriteFS.
// Use MemWriteFS for testing or DryRunWriteFS for dry-run mode.
func NewFileManagerWithFS(wfs WriteFS, log Logger) *FileManager {
	if log == nil {
		log = NopLogger{}
	}
	if wfs == nil {
		wfs = &OSWriteFS{}
	}
	return &FileManager{wfs: wfs, log: log}
}

// FS returns the underlying WriteFS.
func (f *FileManager) FS() WriteFS {
	return f.wfs
}

// MkdirAll creates directories recursively.
func (f *FileManager) MkdirAll(dir string) error {
	return f.wfs.MkdirAll(dir, 0o755)
}

// CreateDirs creates multiple directories.
func (f *FileManager) CreateDirs(dirs []string) error {
	for _, dir := range dirs {
		if err := f.MkdirAll(dir); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	return nil
}

// WriteFile writes content to file, creating parent directories.
func (f *FileManager) WriteFile(filePath string, content []byte) error {
	dir := filepath.Dir(filePath)
	if err := f.MkdirAll(dir); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}
	return f.wfs.WriteFile(filePath, content, 0o644)
}

// WriteFileSkipExisting writes only if file doesn't exist.
// Returns skipped=true if file already exists and was not written.
func (f *FileManager) WriteFileSkipExisting(filePath string, content []byte) (skipped bool, err error) {
	_, statErr := f.wfs.Stat(filePath)
	if statErr == nil {
		return true, nil
	}
	if !errors.Is(statErr, os.ErrNotExist) && !errors.Is(statErr, fs.ErrNotExist) {
		return false, statErr
	}
	return false, f.WriteFile(filePath, content)
}

// ReadFile reads a file's content.
func (f *FileManager) ReadFile(filePath string) ([]byte, error) {
	return f.wfs.ReadFile(filePath)
}

// Exists checks if file or directory exists.
func (f *FileManager) Exists(path string) bool {
	_, err := f.wfs.Stat(path)
	return err == nil
}

// IsDir checks if path is a directory.
func (f *FileManager) IsDir(path string) bool {
	info, err := f.wfs.Stat(path)
	return err == nil && info.IsDir()
}

// IsFile checks if path is a file.
func (f *FileManager) IsFile(path string) bool {
	info, err := f.wfs.Stat(path)
	return err == nil && !info.IsDir()
}

// Remove removes a file or empty directory.
func (f *FileManager) Remove(path string) error {
	return f.wfs.Remove(path)
}

// RemoveAll removes a path and any children it contains.
// Note: Only works with OSWriteFS, MemWriteFS doesn't support recursive delete.
func (f *FileManager) RemoveAll(path string) error {
	if _, ok := f.wfs.(*OSWriteFS); ok {
		return os.RemoveAll(path)
	}
	return f.wfs.Remove(path)
}
