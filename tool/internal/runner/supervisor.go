package runner

import (
	"context"
	"fmt"
	"sync"

	"github.com/ssgohq/ssgo/internal/util/log"
)

// Supervisor manages multiple service processes.
type Supervisor struct {
	mu sync.RWMutex

	// Configuration
	config  *RunnerConfig
	workDir string
	logger  *log.Logger

	// Processes
	processes map[string]*Process

	// Ordered service list (topologically sorted)
	orderedServices []ServiceConfig

	// Event channels
	processEvents chan ProcessEvent
	fileEvents    chan FileChangeEvent
	logEvents     chan LogEntry

	// Log captures (for TUI mode)
	logCaptures map[string]*LogCapture

	// Flags
	noBuild  bool
	noWatch  bool
	tuiMode  bool
	shutdown bool
}

// NewSupervisor creates a new Supervisor instance.
func NewSupervisor(
	config *RunnerConfig,
	workDir string,
	logger *log.Logger,
	noBuild, noWatch, tuiMode bool,
) *Supervisor {
	return &Supervisor{
		config:        config,
		workDir:       workDir,
		logger:        logger,
		processes:     make(map[string]*Process),
		processEvents: make(chan ProcessEvent, 100),
		fileEvents:    make(chan FileChangeEvent, 100),
		logEvents:     make(chan LogEntry, 1000),
		logCaptures:   make(map[string]*LogCapture),
		noBuild:       noBuild,
		noWatch:       noWatch,
		tuiMode:       tuiMode,
	}
}

// LogEvents returns the channel for log entries (used in TUI mode).
func (s *Supervisor) LogEvents() <-chan LogEntry {
	return s.logEvents
}

// ProcessEvents returns the channel for process events.
func (s *Supervisor) ProcessEvents() <-chan ProcessEvent {
	return s.processEvents
}

// FileEvents returns the channel for file change events.
func (s *Supervisor) FileEvents() <-chan FileChangeEvent {
	return s.fileEvents
}

// GetProcess returns a process by name.
func (s *Supervisor) GetProcess(name string) *Process {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.processes[name]
}

// Services returns the ordered list of services.
func (s *Supervisor) Services() []ServiceConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.orderedServices
}

// Start starts all services in topological order.
func (s *Supervisor) Start(ctx context.Context, serviceNames []string) error {
	s.mu.Lock()

	// Filter services if specific names provided
	services := s.config.FilterServices(serviceNames)
	if len(services) == 0 {
		s.mu.Unlock()
		return fmt.Errorf("no services to run")
	}

	// Topological sort based on dependencies
	sorted, err := s.config.TopologicalSort(services)
	if err != nil {
		s.mu.Unlock()
		return fmt.Errorf("failed to sort services: %w", err)
	}
	s.orderedServices = sorted

	// Create processes for each service
	for _, svc := range sorted {
		serviceLogger := s.logger.ForService(svc.Name, svc.Color)
		proc := NewProcess(svc, s.workDir, s.config.KillDelay, serviceLogger, s.processEvents)

		// In TUI mode, redirect output to log capture
		if s.tuiMode {
			capture := NewLogCapture(svc.Name, s.logEvents)
			s.logCaptures[svc.Name] = capture
			proc.SetOutput(capture, capture)
		}

		s.processes[svc.Name] = proc
	}

	s.mu.Unlock()

	// Build and start services in order
	for _, svc := range sorted {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		proc := s.GetProcess(svc.Name)
		if proc == nil {
			continue
		}

		// Build if not skipped
		if !s.noBuild {
			if err := proc.Build(ctx); err != nil {
				// Continue with other services even if build fails
				s.logger.Warn("Build failed for %s, skipping: %v", svc.Name, err)
				continue
			}
		}

		// Start the process
		if err := proc.Start(ctx); err != nil {
			s.logger.Warn("Failed to start %s: %v", svc.Name, err)
		}
	}

	return nil
}

// RestartService restarts a specific service.
func (s *Supervisor) RestartService(ctx context.Context, name string, rebuild bool) error {
	proc := s.GetProcess(name)
	if proc == nil {
		return fmt.Errorf("service %s not found", name)
	}

	return proc.Restart(ctx, rebuild && !s.noBuild)
}

// RestartDependents restarts a service and all services that depend on it.
func (s *Supervisor) RestartDependents(ctx context.Context, name string, rebuild bool) error {
	// First restart the changed service
	if err := s.RestartService(ctx, name, rebuild); err != nil {
		return err
	}

	// Find and restart dependents
	s.mu.RLock()
	orderedServices := s.orderedServices
	s.mu.RUnlock()

	for _, svc := range orderedServices {
		for _, dep := range svc.DependsOn {
			if dep == name {
				// This service depends on the changed service, restart it
				if err := s.RestartService(ctx, svc.Name, false); err != nil {
					s.logger.Warn("Failed to restart dependent %s: %v", svc.Name, err)
				}
				break
			}
		}
	}

	return nil
}

// StopAll stops all services in reverse order.
func (s *Supervisor) StopAll() {
	s.mu.Lock()
	s.shutdown = true
	orderedServices := s.orderedServices
	s.mu.Unlock()

	// Stop in reverse order (dependents first)
	for i := len(orderedServices) - 1; i >= 0; i-- {
		proc := s.GetProcess(orderedServices[i].Name)
		if proc != nil {
			_ = proc.Stop()
		}
	}
}

// ForceKillAll immediately kills all processes with SIGKILL.
func (s *Supervisor) ForceKillAll() {
	s.mu.Lock()
	s.shutdown = true
	orderedServices := s.orderedServices
	s.mu.Unlock()

	for _, svc := range orderedServices {
		proc := s.GetProcess(svc.Name)
		if proc != nil {
			proc.ForceKill()
		}
	}
}

// IsShutdown returns true if the supervisor is shutting down.
func (s *Supervisor) IsShutdown() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.shutdown
}

// ServiceNames returns the names of all managed services.
func (s *Supervisor) ServiceNames() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	names := make([]string, 0, len(s.orderedServices))
	for _, svc := range s.orderedServices {
		names = append(names, svc.Name)
	}
	return names
}

// ServiceStates returns the current state of all services as strings.
func (s *Supervisor) ServiceStates() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	states := make(map[string]string)
	for name, proc := range s.processes {
		states[name] = string(proc.State())
	}
	return states
}
