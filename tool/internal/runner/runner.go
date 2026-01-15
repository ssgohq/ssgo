package runner

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ssgohq/ssgo/internal/util/log"
	"github.com/ssgohq/ssgo/tool/internal/tui"
)

// Runner orchestrates the entire service running infrastructure.
type Runner struct {
	// Configuration
	config   *RunnerConfig
	workDir  string
	noWatch  bool
	noBuild  bool
	noTUI    bool
	verbose  bool
	services []string

	// Components
	supervisor *Supervisor
	watcher    *Watcher
	logger     *log.Logger

	// Shutdown
	shutdownCh chan struct{}
}

// New creates a new Runner instance.
func New(opts Options) *Runner {
	return &Runner{
		config:     opts.Config,
		workDir:    opts.WorkDir,
		noWatch:    opts.NoWatch,
		noBuild:    opts.NoBuild,
		noTUI:      opts.NoTUI,
		verbose:    opts.Verbose,
		services:   opts.Services,
		logger:     log.New(),
		shutdownCh: make(chan struct{}),
	}
}

// Run starts the runner and blocks until shutdown.
func (r *Runner) Run(ctx context.Context) error {
	// Validate config
	if len(r.config.Services) == 0 {
		return ErrNoServices
	}

	// Create supervisor (tuiMode = !noTUI)
	r.supervisor = NewSupervisor(r.config, r.workDir, r.logger, r.noBuild, r.noWatch, !r.noTUI)

	// Create cancellable context
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Start services
	if err := r.supervisor.Start(runCtx, r.services); err != nil {
		return err
	}

	// Start file watcher if enabled
	if !r.noWatch {
		services := r.supervisor.Services()
		fileEvents := make(chan FileChangeEvent, 100)

		watcher, err := NewWatcher(services, r.workDir, r.config.BuildDelay, r.logger, fileEvents)
		if err != nil {
			r.logger.Warn("Failed to create file watcher: %v", err)
		} else {
			r.watcher = watcher
			if err := r.watcher.Start(runCtx); err != nil {
				r.logger.Warn("Failed to start file watcher: %v", err)
			}

			// Handle file changes
			go r.handleFileChanges(runCtx, fileEvents)
		}
	}

	// Run TUI or plain mode
	if r.noTUI {
		return r.runPlainMode(runCtx)
	}
	return r.runTUIMode(runCtx)
}

// runTUIMode runs the TUI interface.
func (r *Runner) runTUIMode(ctx context.Context) error {
	// Setup signal handling for TUI mode
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Create TUI model
	model := tui.NewModel(r.supervisor)

	// Create Bubble Tea program
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	// Handle signals in goroutine - just forward to TUI quit
	go func() {
		select {
		case <-sigCh:
			p.Send(tui.QuitMsg{})
		case <-ctx.Done():
			p.Send(tui.QuitMsg{})
		case <-r.shutdownCh:
			p.Send(tui.QuitMsg{})
		}
	}()

	// Forward supervisor events to TUI
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-r.shutdownCh:
				return
			case event := <-r.supervisor.ProcessEvents():
				// Convert to TUI message
				msg := tui.ProcessEventMsg{
					Service: event.Service,
					State:   tui.ProcessState(event.State),
					Message: event.Message,
					Error:   event.Error,
				}
				p.Send(msg)
			case logEntry := <-r.supervisor.LogEvents():
				// Forward log entry to TUI
				p.Send(tui.LogMsg{
					Service: logEntry.Service,
					Message: logEntry.Message,
					Time:    logEntry.Time,
				})
			}
		}
	}()

	// Run TUI
	if _, err := p.Run(); err != nil {
		return err
	}

	// TUI exited, stop the old signal handler and set up fresh one
	signal.Stop(sigCh)

	forceCh := make(chan os.Signal, 1)
	signal.Notify(forceCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(forceCh)

	r.logger.Info("Shutting down... (press Ctrl+C to force)")

	// Start shutdown in background
	done := make(chan struct{})
	go func() {
		r.Shutdown()
		close(done)
	}()

	// Wait for shutdown or force quit
	select {
	case <-done:
		// Normal shutdown complete
	case <-forceCh:
		r.logger.Warn("Force killing all processes...")
		r.supervisor.ForceKillAll()
	}

	return nil
}

// runPlainMode runs without TUI, just logging to stdout.
func (r *Runner) runPlainMode(ctx context.Context) error {
	// Setup signal handling for plain mode
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	log.PrintBanner()
	r.logger.Info("Running %d services (press Ctrl+C to stop)", len(r.supervisor.Services()))

	// Print service status
	for name, state := range r.supervisor.ServiceStates() {
		r.logger.Info("  %s: %s", name, state)
	}

	// Wait for shutdown signal
	select {
	case <-sigCh:
		r.logger.Info("Shutting down... (press Ctrl+C to force)")
	case <-ctx.Done():
		r.logger.Info("Context cancelled")
	case <-r.shutdownCh:
		r.logger.Info("Shutdown requested")
	}

	// Stop old handler and set up fresh one for force quit
	signal.Stop(sigCh)

	forceCh := make(chan os.Signal, 1)
	signal.Notify(forceCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(forceCh)

	// Start shutdown in background, listen for force quit
	done := make(chan struct{})
	go func() {
		r.Shutdown()
		close(done)
	}()

	// Wait for shutdown or force quit
	select {
	case <-done:
		// Normal shutdown complete
	case <-forceCh:
		r.logger.Warn("Force killing all processes...")
		r.supervisor.ForceKillAll()
	}

	return nil
}

// handleFileChanges processes file change events and triggers rebuilds.
func (r *Runner) handleFileChanges(ctx context.Context, events <-chan FileChangeEvent) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-r.shutdownCh:
			return
		case event := <-events:
			if r.supervisor.IsShutdown() {
				return
			}

			// Only log in non-TUI mode (TUI shows state via ProcessEvent)
			if r.noTUI {
				r.logger.Info("File changed in %s, restarting...", event.Service)
				if r.verbose {
					for _, f := range event.Files {
						r.logger.Info("  %s", f)
					}
				}
			}

			// Restart the service and its dependents
			if err := r.supervisor.RestartDependents(ctx, event.Service, true); err != nil {
				if r.noTUI {
					r.logger.Error("Failed to restart %s: %v", event.Service, err)
				}
			}
		}
	}
}

// Shutdown stops all services gracefully.
func (r *Runner) Shutdown() {
	select {
	case <-r.shutdownCh:
		// Already closed
	default:
		close(r.shutdownCh)
	}

	// Stop watcher
	if r.watcher != nil {
		_ = r.watcher.Stop()
	}

	// Stop all services
	if r.supervisor != nil {
		r.supervisor.StopAll()
	}
}
