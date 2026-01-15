package gen

import (
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// WriteFS abstracts file system operations for testing and dry-run support.
type WriteFS interface {
	// Stat returns file info for the given path.
	Stat(name string) (fs.FileInfo, error)

	// MkdirAll creates a directory and all parent directories.
	MkdirAll(path string, perm fs.FileMode) error

	// ReadFile reads the entire file content.
	ReadFile(name string) ([]byte, error)

	// WriteFile writes data to a file, creating it if necessary.
	WriteFile(name string, data []byte, perm fs.FileMode) error

	// Remove removes a file or empty directory.
	Remove(name string) error
}

// OSWriteFS implements WriteFS using the real OS filesystem.
type OSWriteFS struct{}

var _ WriteFS = (*OSWriteFS)(nil)

func (OSWriteFS) Stat(name string) (fs.FileInfo, error) {
	return os.Stat(name)
}

func (OSWriteFS) MkdirAll(path string, perm fs.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (OSWriteFS) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

func (OSWriteFS) WriteFile(name string, data []byte, perm fs.FileMode) error {
	return os.WriteFile(name, data, perm)
}

func (OSWriteFS) Remove(name string) error {
	return os.Remove(name)
}

// MemWriteFS implements WriteFS using in-memory storage.
// Useful for testing and dry-run mode.
type MemWriteFS struct {
	mu    sync.RWMutex
	files map[string][]byte
	dirs  map[string]bool
}

var _ WriteFS = (*MemWriteFS)(nil)

// NewMemWriteFS creates a new in-memory filesystem.
func NewMemWriteFS() *MemWriteFS {
	return &MemWriteFS{
		files: make(map[string][]byte),
		dirs:  make(map[string]bool),
	}
}

func (m *MemWriteFS) Stat(name string) (fs.FileInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	name = filepath.Clean(name)

	if m.dirs[name] {
		return &memFileInfo{name: filepath.Base(name), isDir: true}, nil
	}

	if data, ok := m.files[name]; ok {
		return &memFileInfo{name: filepath.Base(name), size: int64(len(data))}, nil
	}

	return nil, os.ErrNotExist
}

func (m *MemWriteFS) MkdirAll(path string, perm fs.FileMode) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	path = filepath.Clean(path)
	m.dirs[path] = true

	// Create parent directories
	for dir := filepath.Dir(path); dir != "." && dir != "/" && dir != path; dir = filepath.Dir(dir) {
		m.dirs[dir] = true
	}

	return nil
}

func (m *MemWriteFS) ReadFile(name string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	name = filepath.Clean(name)

	if data, ok := m.files[name]; ok {
		// Return a copy to prevent mutation
		result := make([]byte, len(data))
		copy(result, data)
		return result, nil
	}

	return nil, os.ErrNotExist
}

func (m *MemWriteFS) WriteFile(name string, data []byte, perm fs.FileMode) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	name = filepath.Clean(name)

	// Ensure parent directory exists
	dir := filepath.Dir(name)
	if dir != "." && dir != "/" {
		m.dirs[dir] = true
	}

	// Store a copy to prevent mutation
	stored := make([]byte, len(data))
	copy(stored, data)
	m.files[name] = stored

	return nil
}

func (m *MemWriteFS) Remove(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	name = filepath.Clean(name)

	if _, ok := m.files[name]; ok {
		delete(m.files, name)
		return nil
	}

	if m.dirs[name] {
		delete(m.dirs, name)
		return nil
	}

	return os.ErrNotExist
}

// GetFiles returns all files in the memory filesystem (for testing).
func (m *MemWriteFS) GetFiles() map[string][]byte {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string][]byte, len(m.files))
	for k, v := range m.files {
		data := make([]byte, len(v))
		copy(data, v)
		result[k] = data
	}
	return result
}

// GetDirs returns all directories in the memory filesystem (for testing).
func (m *MemWriteFS) GetDirs() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]string, 0, len(m.dirs))
	for k := range m.dirs {
		result = append(result, k)
	}
	return result
}

// memFileInfo implements fs.FileInfo for in-memory files.
type memFileInfo struct {
	name  string
	size  int64
	isDir bool
}

func (f *memFileInfo) Name() string       { return f.name }
func (f *memFileInfo) Size() int64        { return f.size }
func (f *memFileInfo) Mode() fs.FileMode  { return 0o644 }
func (f *memFileInfo) ModTime() time.Time { return time.Time{} }
func (f *memFileInfo) IsDir() bool        { return f.isDir }
func (f *memFileInfo) Sys() any           { return nil }

// DryRunWriteFS wraps another WriteFS and logs operations without writing.
type DryRunWriteFS struct {
	inner WriteFS
	log   Logger
	ops   []DryRunOp
	mu    sync.Mutex
}

// DryRunOp represents a dry-run operation.
type DryRunOp struct {
	Op   string // "write", "mkdir", "remove"
	Path string
	Size int
}

// NewDryRunWriteFS creates a dry-run filesystem that logs operations.
func NewDryRunWriteFS(inner WriteFS, log Logger) *DryRunWriteFS {
	if inner == nil {
		inner = &OSWriteFS{}
	}
	return &DryRunWriteFS{inner: inner, log: log}
}

func (d *DryRunWriteFS) Stat(name string) (fs.FileInfo, error) {
	return d.inner.Stat(name)
}

func (d *DryRunWriteFS) MkdirAll(path string, perm fs.FileMode) error {
	d.mu.Lock()
	d.ops = append(d.ops, DryRunOp{Op: "mkdir", Path: path})
	d.mu.Unlock()

	if d.log != nil {
		d.log.Verbosef("[dry-run] mkdir %s", path)
	}
	return nil
}

func (d *DryRunWriteFS) ReadFile(name string) ([]byte, error) {
	return d.inner.ReadFile(name)
}

func (d *DryRunWriteFS) WriteFile(name string, data []byte, perm fs.FileMode) error {
	d.mu.Lock()
	d.ops = append(d.ops, DryRunOp{Op: "write", Path: name, Size: len(data)})
	d.mu.Unlock()

	if d.log != nil {
		d.log.Verbosef("[dry-run] write %s (%d bytes)", name, len(data))
	}
	return nil
}

func (d *DryRunWriteFS) Remove(name string) error {
	d.mu.Lock()
	d.ops = append(d.ops, DryRunOp{Op: "remove", Path: name})
	d.mu.Unlock()

	if d.log != nil {
		d.log.Verbosef("[dry-run] remove %s", name)
	}
	return nil
}

// GetOps returns all recorded operations (for testing/reporting).
func (d *DryRunWriteFS) GetOps() []DryRunOp {
	d.mu.Lock()
	defer d.mu.Unlock()

	result := make([]DryRunOp, len(d.ops))
	copy(result, d.ops)
	return result
}
