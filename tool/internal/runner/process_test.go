package runner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ssgohq/ssgo/internal/util/log"
)

func TestProcessState(t *testing.T) {
	events := make(chan ProcessEvent, 10)
	logger := log.New().ForService("test", "cyan")

	cfg := ServiceConfig{
		Name: "test-svc",
		Cmd:  "echo hello",
	}

	p := NewProcess(cfg, t.TempDir(), 5*time.Second, logger, events)

	// Initial state should be idle
	if p.State() != StateIdle {
		t.Errorf("Initial state = %v, want %v", p.State(), StateIdle)
	}

	// Should not be running
	if p.IsRunning() {
		t.Error("should not be running initially")
	}

	// PID should be 0
	if p.PID() != 0 {
		t.Errorf("PID = %d, want 0", p.PID())
	}
}

func TestProcessName(t *testing.T) {
	logger := log.New().ForService("test", "cyan")
	cfg := ServiceConfig{
		Name: "my-service",
		Cmd:  "echo test",
	}

	p := NewProcess(cfg, t.TempDir(), 5*time.Second, logger, nil)

	if p.Name() != "my-service" {
		t.Errorf("Name() = %q, want %q", p.Name(), "my-service")
	}
}

func TestProcessSetState(t *testing.T) {
	events := make(chan ProcessEvent, 10)
	logger := log.New().ForService("test", "cyan")

	cfg := ServiceConfig{
		Name: "test-svc",
		Cmd:  "echo hello",
	}

	p := NewProcess(cfg, t.TempDir(), 5*time.Second, logger, events)

	// Set state
	p.setState(StateRunning, "Started", nil)

	// Verify state changed
	if p.State() != StateRunning {
		t.Errorf("State = %v, want %v", p.State(), StateRunning)
	}

	// Verify event was sent
	select {
	case event := <-events:
		if event.Service != "test-svc" {
			t.Errorf("event.Service = %q, want %q", event.Service, "test-svc")
		}
		if event.State != StateRunning {
			t.Errorf("event.State = %v, want %v", event.State, StateRunning)
		}
		if event.Message != "Started" {
			t.Errorf("event.Message = %q, want %q", event.Message, "Started")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("expected event not received")
	}
}

func TestProcessResolveDir(t *testing.T) {
	logger := log.New().ForService("test", "cyan")
	workDir := "/home/user/project"

	tests := []struct {
		name     string
		dir      string
		expected string
	}{
		{
			name:     "empty dir uses workDir",
			dir:      "",
			expected: "/home/user/project",
		},
		{
			name:     "absolute path",
			dir:      "/opt/service",
			expected: "/opt/service",
		},
		{
			name:     "relative path",
			dir:      "./api",
			expected: "/home/user/project/./api",
		},
		{
			name:     "relative without dot",
			dir:      "services/api",
			expected: "/home/user/project/services/api",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := ServiceConfig{
				Name: "test",
				Dir:  tt.dir,
				Cmd:  "echo test",
			}

			p := NewProcess(cfg, workDir, 5*time.Second, logger, nil)
			got := p.resolveDir()
			if got != tt.expected {
				t.Errorf("resolveDir() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestProcessBuildEnv(t *testing.T) {
	logger := log.New().ForService("test", "cyan")

	cfg := ServiceConfig{
		Name: "test",
		Cmd:  "echo test",
		Env:  []string{"CUSTOM_VAR=value", "ANOTHER=123"},
	}

	p := NewProcess(cfg, t.TempDir(), 5*time.Second, logger, nil)
	env := p.buildEnv()

	// Should include system env
	if len(env) <= 2 {
		t.Error("env should include system environment")
	}

	// Should include custom env
	hasCustomVar := false
	hasAnother := false
	for _, e := range env {
		if e == "CUSTOM_VAR=value" {
			hasCustomVar = true
		}
		if e == "ANOTHER=123" {
			hasAnother = true
		}
	}

	if !hasCustomVar {
		t.Error("env should include CUSTOM_VAR")
	}
	if !hasAnother {
		t.Error("env should include ANOTHER")
	}
}

func TestProcessStartAndStop(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	events := make(chan ProcessEvent, 100)
	logger := log.New().ForService("test", "cyan")
	tmpDir := t.TempDir()

	cfg := ServiceConfig{
		Name: "sleeper",
		Cmd:  "sleep 10",
	}

	p := NewProcess(cfg, tmpDir, 2*time.Second, logger, events)

	ctx := context.Background()

	// Start the process
	err := p.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Wait for running state
	time.Sleep(100 * time.Millisecond)

	if !p.IsRunning() {
		t.Error("process should be running")
	}

	if p.PID() == 0 {
		t.Error("PID should not be 0")
	}

	// Stop the process
	err = p.Stop()
	if err != nil {
		t.Errorf("Stop failed: %v", err)
	}

	// Should be stopped
	if p.IsRunning() {
		t.Error("process should not be running after stop")
	}

	if p.PID() != 0 {
		t.Errorf("PID should be 0 after stop, got %d", p.PID())
	}
}

func TestProcessBuild(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	events := make(chan ProcessEvent, 10)
	logger := log.New().ForService("test", "cyan")
	tmpDir := t.TempDir()

	// Create a simple build script
	buildFile := filepath.Join(tmpDir, "built.txt")
	cfg := ServiceConfig{
		Name:  "builder",
		Build: "echo built > built.txt",
		Cmd:   "echo run",
		Dir:   tmpDir,
	}

	p := NewProcess(cfg, tmpDir, 2*time.Second, logger, events)

	ctx := context.Background()
	err := p.Build(ctx)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Verify build ran
	if _, err := os.Stat(buildFile); os.IsNotExist(err) {
		t.Error("build should have created built.txt")
	}
}

func TestProcessNoBuildCommand(t *testing.T) {
	logger := log.New().ForService("test", "cyan")

	cfg := ServiceConfig{
		Name: "no-build",
		Cmd:  "echo run",
	}

	p := NewProcess(cfg, t.TempDir(), 2*time.Second, logger, nil)

	ctx := context.Background()
	err := p.Build(ctx)

	// Should succeed (no-op)
	if err != nil {
		t.Errorf("Build with no command should succeed, got: %v", err)
	}
}

func TestProcessSetOutput(t *testing.T) {
	logger := log.New().ForService("test", "cyan")

	cfg := ServiceConfig{
		Name: "test",
		Cmd:  "echo test",
	}

	p := NewProcess(cfg, t.TempDir(), 2*time.Second, logger, nil)

	// Initially tuiMode should be false
	if p.tuiMode {
		t.Error("tuiMode should be false initially")
	}

	// Set custom output
	logs := make(chan LogEntry, 10)
	capture := NewLogCapture("test", logs)
	p.SetOutput(capture, capture)

	// Now tuiMode should be true
	if !p.tuiMode {
		t.Error("tuiMode should be true after SetOutput")
	}
}

func TestProcessStateConstants(t *testing.T) {
	// Verify state constants are distinct
	states := []ProcessState{
		StateIdle,
		StateBuilding,
		StateStarting,
		StateRunning,
		StateStopping,
		StateStopped,
		StateError,
	}

	seen := make(map[ProcessState]bool)
	for _, s := range states {
		if seen[s] {
			t.Errorf("duplicate state: %v", s)
		}
		seen[s] = true
	}
}

func TestProcessExitError(t *testing.T) {
	logger := log.New().ForService("test", "cyan")

	cfg := ServiceConfig{
		Name: "test",
		Cmd:  "echo test",
	}

	p := NewProcess(cfg, t.TempDir(), 2*time.Second, logger, nil)

	// Initially no exit error
	if p.ExitError() != nil {
		t.Error("ExitError should be nil initially")
	}
}
