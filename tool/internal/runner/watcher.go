package runner

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/fsnotify/fsnotify"
	"github.com/ssgohq/ssgo/internal/util/log"
)

// FileChangeEvent represents a file change event.
type FileChangeEvent struct {
	Service string
	Files   []string
}

// Watcher watches files for changes and triggers rebuilds.
type Watcher struct {
	mu sync.Mutex

	// Configuration
	services   []ServiceConfig
	workDir    string
	buildDelay time.Duration

	// fsnotify watcher
	watcher *fsnotify.Watcher

	// Service to directory mapping
	serviceDirs map[string]string

	// Debounce timers per service
	timers map[string]*time.Timer

	// Pending changes per service (for batching)
	pending map[string]map[string]struct{}

	// Output channel
	events chan<- FileChangeEvent

	// Logger
	logger *log.Logger
}

// NewWatcher creates a new file watcher.
func NewWatcher(services []ServiceConfig, workDir string, buildDelay time.Duration, logger *log.Logger, events chan<- FileChangeEvent) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	w := &Watcher{
		services:    services,
		workDir:     workDir,
		buildDelay:  buildDelay,
		watcher:     fsWatcher,
		serviceDirs: make(map[string]string),
		timers:      make(map[string]*time.Timer),
		pending:     make(map[string]map[string]struct{}),
		events:      events,
		logger:      logger,
	}

	// Build service directory mapping
	for _, svc := range services {
		dir := svc.Dir
		if !filepath.IsAbs(dir) {
			dir = filepath.Join(workDir, dir)
		}
		w.serviceDirs[svc.Name] = dir
	}

	return w, nil
}

// Start begins watching all service directories.
func (w *Watcher) Start(ctx context.Context) error {
	// Add directories to watch
	for _, svc := range w.services {
		dir := w.serviceDirs[svc.Name]
		if err := w.addDirRecursive(dir, svc); err != nil {
			w.logger.Warn("Failed to watch %s: %v", dir, err)
		}
	}

	// Start the event loop
	go w.loop(ctx)

	return nil
}

// addDirRecursive adds a directory and its subdirectories to the watcher.
func (w *Watcher) addDirRecursive(root string, svc ServiceConfig) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip inaccessible paths
		}

		if !info.IsDir() {
			return nil
		}

		// Skip common non-source directories
		base := filepath.Base(path)
		if base == ".git" || base == "node_modules" || base == "vendor" || base == ".idea" || base == ".vscode" {
			return filepath.SkipDir
		}

		// Check if directory should be excluded
		relPath, _ := filepath.Rel(root, path)
		for _, pattern := range svc.Watch.Exclude {
			if match, _ := doublestar.Match(pattern, relPath); match {
				return filepath.SkipDir
			}
			if match, _ := doublestar.Match(pattern, relPath+"/"); match {
				return filepath.SkipDir
			}
		}

		return w.watcher.Add(path)
	})
}

// loop processes file system events.
func (w *Watcher) loop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return

		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			w.handleEvent(event)

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			w.logger.Warn("Watcher error: %v", err)
		}
	}
}

// handleEvent processes a single file system event.
func (w *Watcher) handleEvent(event fsnotify.Event) {
	// Ignore chmod events
	if event.Op&fsnotify.Chmod == fsnotify.Chmod {
		return
	}

	path := event.Name

	// Find which service this file belongs to
	for _, svc := range w.services {
		dir := w.serviceDirs[svc.Name]
		if !strings.HasPrefix(path, dir) {
			continue
		}

		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			continue
		}

		// Check if file matches include patterns
		if !w.matchesInclude(relPath, svc.Watch.Include) {
			continue
		}

		// Check if file matches exclude patterns
		if w.matchesExclude(relPath, svc.Watch.Exclude) {
			continue
		}

		// Add to pending changes and schedule debounced event
		w.addPending(svc.Name, path)
	}

	// Handle new directories
	if event.Op&fsnotify.Create == fsnotify.Create {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			// Find the service and add the new directory
			for _, svc := range w.services {
				dir := w.serviceDirs[svc.Name]
				if strings.HasPrefix(path, dir) {
					_ = w.watcher.Add(path)
					break
				}
			}
		}
	}
}

// matchesInclude checks if a path matches any include pattern.
func (w *Watcher) matchesInclude(path string, patterns []string) bool {
	if len(patterns) == 0 {
		return true // No patterns means match all
	}

	for _, pattern := range patterns {
		if match, _ := doublestar.Match(pattern, path); match {
			return true
		}
	}
	return false
}

// matchesExclude checks if a path matches any exclude pattern.
func (w *Watcher) matchesExclude(path string, patterns []string) bool {
	for _, pattern := range patterns {
		if match, _ := doublestar.Match(pattern, path); match {
			return true
		}
	}
	return false
}

// addPending adds a file to the pending changes and schedules a debounced event.
func (w *Watcher) addPending(service, path string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Initialize pending set if needed
	if w.pending[service] == nil {
		w.pending[service] = make(map[string]struct{})
	}
	w.pending[service][path] = struct{}{}

	// Cancel existing timer
	if timer := w.timers[service]; timer != nil {
		timer.Stop()
	}

	// Schedule new debounced event
	w.timers[service] = time.AfterFunc(w.buildDelay, func() {
		w.flushPending(service)
	})
}

// flushPending sends pending changes as an event.
func (w *Watcher) flushPending(service string) {
	w.mu.Lock()
	pending := w.pending[service]
	w.pending[service] = nil
	w.timers[service] = nil
	w.mu.Unlock()

	if len(pending) == 0 {
		return
	}

	files := make([]string, 0, len(pending))
	for f := range pending {
		files = append(files, f)
	}

	if w.events != nil {
		w.events <- FileChangeEvent{
			Service: service,
			Files:   files,
		}
	}
}

// Stop stops the watcher.
func (w *Watcher) Stop() error {
	w.mu.Lock()
	for _, timer := range w.timers {
		if timer != nil {
			timer.Stop()
		}
	}
	w.mu.Unlock()

	return w.watcher.Close()
}
