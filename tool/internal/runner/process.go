// Package runner provides process and service management for ssgo run command.
package runner

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/ssgohq/ssgo/internal/util/log"
)

// ProcessState represents the current state of a process.
type ProcessState string

const (
	StateIdle     ProcessState = "idle"
	StateBuilding ProcessState = "building"
	StateStarting ProcessState = "starting"
	StateRunning  ProcessState = "running"
	StateStopping ProcessState = "stopping"
	StateStopped  ProcessState = "stopped"
	StateError    ProcessState = "error"
)

// Process manages a single service process.
type Process struct {
	mu sync.RWMutex

	// Configuration
	config    ServiceConfig
	workDir   string
	killDelay time.Duration

	// State
	state   ProcessState
	cmd     *exec.Cmd
	pid     int
	exitErr error
	done    chan struct{} // closed when process exits

	// Logging
	logger  *log.ServiceLogger
	stdout  io.Writer
	stderr  io.Writer
	tuiMode bool // When true, skip direct logging (use events only)

	// Event channel for state changes
	events chan<- ProcessEvent
}

// ProcessEvent represents a process state change event.
type ProcessEvent struct {
	Service string
	State   ProcessState
	Message string
	Error   error
}

// NewProcess creates a new Process instance.
func NewProcess(cfg ServiceConfig, workDir string, killDelay time.Duration, logger *log.ServiceLogger, events chan<- ProcessEvent) *Process {
	return &Process{
		config:    cfg,
		workDir:   workDir,
		killDelay: killDelay,
		state:     StateIdle,
		logger:    logger,
		stdout:    logger,
		stderr:    logger,
		events:    events,
	}
}

// SetOutput sets custom stdout/stderr writers (used for TUI mode).
func (p *Process) SetOutput(stdout, stderr io.Writer) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stdout = stdout
	p.stderr = stderr
	p.tuiMode = true
}

// Name returns the service name.
func (p *Process) Name() string {
	return p.config.Name
}

// log logs a message only in non-TUI mode.
func (p *Process) log(level string, format string, args ...interface{}) {
	if p.tuiMode {
		return
	}
	switch level {
	case "info":
		p.logger.Info(format, args...)
	case "success":
		p.logger.Success(format, args...)
	case "warn":
		p.logger.Warn(format, args...)
	case "error":
		p.logger.Error(format, args...)
	}
}

// State returns the current process state.
func (p *Process) State() ProcessState {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.state
}

// IsRunning returns true if the process is running.
func (p *Process) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.state == StateRunning
}

// PID returns the process ID, or 0 if not running.
func (p *Process) PID() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.pid
}

// setState updates the state and sends an event.
func (p *Process) setState(state ProcessState, message string, err error) {
	p.mu.Lock()
	p.state = state
	p.mu.Unlock()

	if p.events != nil {
		p.events <- ProcessEvent{
			Service: p.config.Name,
			State:   state,
			Message: message,
			Error:   err,
		}
	}
}

// Build runs the build command if configured.
func (p *Process) Build(ctx context.Context) error {
	if p.config.Build == "" {
		return nil
	}

	p.setState(StateBuilding, "Building...", nil)
	p.log("info", "Building...")

	dir := p.resolveDir()
	cmd := exec.CommandContext(ctx, "sh", "-c", p.config.Build)
	cmd.Dir = dir
	cmd.Env = p.buildEnv()
	cmd.Stdout = p.stdout
	cmd.Stderr = p.stderr

	if err := cmd.Run(); err != nil {
		p.setState(StateError, "Build failed", err)
		p.log("error", "Build failed: %v", err)
		return fmt.Errorf("build failed: %w", err)
	}

	p.log("success", "Build complete")
	return nil
}

// Start starts the service process.
func (p *Process) Start(ctx context.Context) error {
	p.mu.Lock()
	if p.state == StateRunning {
		p.mu.Unlock()
		return nil
	}
	p.mu.Unlock()

	p.setState(StateStarting, "Starting...", nil)
	p.log("info", "Starting...")

	// Determine the command to run
	cmdStr := p.config.Cmd
	if cmdStr == "" {
		cmdStr = p.config.Run
	}
	if cmdStr == "" {
		return fmt.Errorf("no command specified for service %s", p.config.Name)
	}

	dir := p.resolveDir()
	cmd := exec.CommandContext(ctx, "sh", "-c", cmdStr)
	cmd.Dir = dir
	cmd.Env = p.buildEnv()
	cmd.Stdout = p.stdout
	cmd.Stderr = p.stderr

	// Set platform-specific process attributes for clean shutdown
	setPlatformSysProcAttr(cmd)

	if err := cmd.Start(); err != nil {
		p.setState(StateError, "Failed to start", err)
		p.log("error", "Failed to start: %v", err)
		return fmt.Errorf("failed to start: %w", err)
	}

	p.mu.Lock()
	p.cmd = cmd
	p.pid = cmd.Process.Pid
	p.done = make(chan struct{})
	p.mu.Unlock()

	p.setState(StateRunning, "Running", nil)
	p.log("success", "Started (PID: %d)", p.pid)

	// Monitor process in background
	go p.monitor(cmd)

	return nil
}

// monitor watches the process and updates state on exit.
func (p *Process) monitor(cmd *exec.Cmd) {
	err := cmd.Wait()

	p.mu.Lock()
	p.exitErr = err
	p.cmd = nil
	p.pid = 0
	currentState := p.state
	done := p.done
	p.done = nil
	p.mu.Unlock()

	// Signal that process has exited
	if done != nil {
		close(done)
	}

	// Only update state if we weren't already stopping
	if currentState != StateStopping && currentState != StateStopped {
		if err != nil {
			p.setState(StateError, "Exited with error", err)
			p.log("error", "Exited: %v", err)
		} else {
			p.setState(StateStopped, "Exited normally", nil)
			p.log("info", "Exited normally")
		}
	}
}

// Stop stops the service process gracefully.
func (p *Process) Stop() error {
	p.mu.Lock()
	cmd := p.cmd
	if cmd == nil || cmd.Process == nil {
		p.state = StateStopped
		p.mu.Unlock()
		return nil
	}
	pid := p.pid
	done := p.done
	p.mu.Unlock()

	p.setState(StateStopping, "Stopping...", nil)
	p.log("info", "Stopping (PID: %d)...", pid)

	// Send SIGINT first for graceful shutdown
	if err := killProcessGroup(pid, signalInterrupt()); err != nil {
		// Try SIGTERM if SIGINT fails
		_ = killProcessGroup(pid, signalTerminate())
	}

	// Wait for monitor goroutine to signal process exit
	select {
	case <-done:
		p.setState(StateStopped, "Stopped", nil)
		p.log("info", "Stopped gracefully")
	case <-time.After(p.killDelay):
		// Force kill
		_ = killProcessGroup(pid, signalKill())
		<-done
		p.setState(StateStopped, "Killed", nil)
		p.log("warn", "Force killed after timeout")
	}

	return nil
}

// Restart stops and starts the process.
func (p *Process) Restart(ctx context.Context, rebuild bool) error {
	if err := p.Stop(); err != nil {
		return err
	}

	if rebuild && p.config.Build != "" {
		if err := p.Build(ctx); err != nil {
			return err
		}
	}

	return p.Start(ctx)
}

// resolveDir resolves the service working directory.
func (p *Process) resolveDir() string {
	if p.config.Dir == "" {
		return p.workDir
	}
	if strings.HasPrefix(p.config.Dir, "/") {
		return p.config.Dir
	}
	return p.workDir + "/" + p.config.Dir
}

// buildEnv constructs the environment variables for the process.
func (p *Process) buildEnv() []string {
	env := os.Environ()
	env = append(env, p.config.Env...)
	return env
}

// ExitError returns the last exit error, if any.
func (p *Process) ExitError() error {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.exitErr
}

// ForceKill immediately kills the process with SIGKILL.
func (p *Process) ForceKill() {
	p.mu.Lock()
	pid := p.pid
	p.mu.Unlock()

	if pid > 0 {
		_ = killProcessGroup(pid, signalKill())
		p.setState(StateStopped, "Force killed", nil)
	}
}
