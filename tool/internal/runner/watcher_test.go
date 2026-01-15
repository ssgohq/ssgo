package runner

import (
	"testing"
	"time"

	"github.com/ssgohq/ssgo/internal/util/log"
)

func TestMatchesInclude(t *testing.T) {
	w := &Watcher{}

	tests := []struct {
		name     string
		path     string
		patterns []string
		expected bool
	}{
		{
			name:     "empty patterns matches all",
			path:     "main.go",
			patterns: []string{},
			expected: true,
		},
		{
			name:     "exact match",
			path:     "main.go",
			patterns: []string{"main.go"},
			expected: true,
		},
		{
			name:     "glob pattern",
			path:     "handler.go",
			patterns: []string{"**/*.go"},
			expected: true,
		},
		{
			name:     "nested file matches",
			path:     "internal/handler/user.go",
			patterns: []string{"**/*.go"},
			expected: true,
		},
		{
			name:     "no match",
			path:     "main.py",
			patterns: []string{"**/*.go"},
			expected: false,
		},
		{
			name:     "multiple patterns - one matches",
			path:     "config.yaml",
			patterns: []string{"**/*.go", "**/*.yaml"},
			expected: true,
		},
		{
			name:     "multiple patterns - none match",
			path:     "README.md",
			patterns: []string{"**/*.go", "**/*.yaml"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := w.matchesInclude(tt.path, tt.patterns)
			if got != tt.expected {
				t.Errorf("matchesInclude(%q, %v) = %v, want %v", tt.path, tt.patterns, got, tt.expected)
			}
		})
	}
}

func TestMatchesExclude(t *testing.T) {
	w := &Watcher{}

	tests := []struct {
		name     string
		path     string
		patterns []string
		expected bool
	}{
		{
			name:     "empty patterns excludes nothing",
			path:     "main.go",
			patterns: []string{},
			expected: false,
		},
		{
			name:     "vendor directory",
			path:     "vendor/lib/file.go",
			patterns: []string{"**/vendor/**"},
			expected: true,
		},
		{
			name:     "test files",
			path:     "handler_test.go",
			patterns: []string{"**/*_test.go"},
			expected: true,
		},
		{
			name:     "testdata directory",
			path:     "testdata/fixtures/data.json",
			patterns: []string{"**/testdata/**"},
			expected: true,
		},
		{
			name:     "normal file not excluded",
			path:     "main.go",
			patterns: []string{"**/*_test.go", "**/vendor/**"},
			expected: false,
		},
		{
			name:     "multiple patterns - one matches",
			path:     "user_test.go",
			patterns: []string{"**/vendor/**", "**/*_test.go"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := w.matchesExclude(tt.path, tt.patterns)
			if got != tt.expected {
				t.Errorf("matchesExclude(%q, %v) = %v, want %v", tt.path, tt.patterns, got, tt.expected)
			}
		})
	}
}

func TestNewWatcher(t *testing.T) {
	tmpDir := t.TempDir()
	events := make(chan FileChangeEvent, 100)
	logger := log.New()

	services := []ServiceConfig{
		{
			Name: "api",
			Dir:  "./api",
			Watch: WatchConfig{
				Include: []string{"**/*.go"},
				Exclude: []string{"**/*_test.go"},
			},
		},
		{
			Name: "worker",
			Dir:  "./worker",
		},
	}

	w, err := NewWatcher(services, tmpDir, 500*time.Millisecond, logger, events)
	if err != nil {
		t.Fatalf("NewWatcher failed: %v", err)
	}
	defer func() { _ = w.Stop() }()

	// Verify service directory mapping
	if len(w.serviceDirs) != 2 {
		t.Errorf("expected 2 service dirs, got %d", len(w.serviceDirs))
	}

	// Check api service dir
	if _, ok := w.serviceDirs["api"]; !ok {
		t.Error("expected api service dir to be mapped")
	}

	// Check worker service dir
	if _, ok := w.serviceDirs["worker"]; !ok {
		t.Error("expected worker service dir to be mapped")
	}
}

func TestWatcherAddPendingDebounce(t *testing.T) {
	tmpDir := t.TempDir()
	events := make(chan FileChangeEvent, 100)
	logger := log.New()

	services := []ServiceConfig{
		{Name: "api", Dir: tmpDir},
	}

	w, err := NewWatcher(services, tmpDir, 100*time.Millisecond, logger, events)
	if err != nil {
		t.Fatalf("NewWatcher failed: %v", err)
	}
	defer func() { _ = w.Stop() }()

	// Add multiple pending changes rapidly
	w.addPending("api", "/path/file1.go")
	w.addPending("api", "/path/file2.go")
	w.addPending("api", "/path/file3.go")

	// Wait for debounce
	time.Sleep(200 * time.Millisecond)

	// Should receive single batched event
	select {
	case event := <-events:
		if event.Service != "api" {
			t.Errorf("Service = %q, want %q", event.Service, "api")
		}
		if len(event.Files) != 3 {
			t.Errorf("expected 3 files in batch, got %d", len(event.Files))
		}
	case <-time.After(500 * time.Millisecond):
		t.Error("expected file change event not received")
	}
}

func TestWatcherFlushPending(t *testing.T) {
	tmpDir := t.TempDir()
	events := make(chan FileChangeEvent, 100)
	logger := log.New()

	services := []ServiceConfig{
		{Name: "svc", Dir: tmpDir},
	}

	w, err := NewWatcher(services, tmpDir, 1*time.Hour, logger, events)
	if err != nil {
		t.Fatalf("NewWatcher failed: %v", err)
	}
	defer func() { _ = w.Stop() }()

	// Add pending without waiting for debounce timer
	w.mu.Lock()
	w.pending["svc"] = map[string]struct{}{
		"/path/a.go": {},
		"/path/b.go": {},
	}
	w.mu.Unlock()

	// Manually flush
	w.flushPending("svc")

	// Should receive event immediately
	select {
	case event := <-events:
		if len(event.Files) != 2 {
			t.Errorf("expected 2 files, got %d", len(event.Files))
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("expected event after flush")
	}

	// Pending should be cleared
	w.mu.Lock()
	if w.pending["svc"] != nil {
		t.Error("pending should be nil after flush")
	}
	w.mu.Unlock()
}

func TestWatcherStop(t *testing.T) {
	tmpDir := t.TempDir()
	events := make(chan FileChangeEvent, 100)
	logger := log.New()

	services := []ServiceConfig{
		{Name: "api", Dir: tmpDir},
	}

	w, err := NewWatcher(services, tmpDir, 100*time.Millisecond, logger, events)
	if err != nil {
		t.Fatalf("NewWatcher failed: %v", err)
	}

	// Add pending with timer
	w.addPending("api", "/path/file.go")

	// Stop should cancel timers
	err = w.Stop()
	if err != nil {
		t.Errorf("Stop failed: %v", err)
	}

	// Should not receive any events after stop
	time.Sleep(200 * time.Millisecond)
	select {
	case event := <-events:
		t.Errorf("should not receive events after stop, got: %v", event)
	default:
		// Expected - no events
	}
}
