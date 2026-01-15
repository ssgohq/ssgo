// Package gen provides core generation utilities for code generators.
// It includes template management, file operations, and AST-based completion.
package gen

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"text/template"
)

// Common errors.
var (
	ErrOutputDirRequired = errors.New("output directory is required")
	ErrModuleRequired    = errors.New("module name is required")
	ErrTemplateNotFound  = errors.New("template not found")
)

// StepFunc is a function that executes a generation step.
type StepFunc func(ctx context.Context, r *Runner) error

// Step represents a single generation step.
type Step struct {
	// Name is the step name for logging.
	Name string

	// Run is the step execution function.
	Run StepFunc

	// Tags allow partial generation selection.
	// If Runner.OnlyTags is set, only steps with matching tags are executed.
	// Examples: "scaffold", "logic", "handlers", "config"
	Tags []string
}

// Generator interface for all generators.
type Generator interface {
	// Name returns the generator name for logging.
	Name() string

	// Steps returns the list of generation steps.
	Steps() []Step
}

// CommonOptions contains options shared by all generators.
type CommonOptions struct {
	// OutputDir is the output directory for generated files.
	OutputDir string

	// Module is the Go module name.
	Module string

	// Verbose enables verbose logging.
	Verbose bool

	// Optional components for generated code
	WithTrace bool   // Enable OpenTelemetry tracing config
	WithDB    string // Database type: "postgres", "mysql", or ""
	WithRedis bool   // Enable Redis config
}

// Validate validates the common options.
func (o *CommonOptions) Validate() error {
	if o.OutputDir == "" {
		return ErrOutputDirRequired
	}
	if o.Module == "" {
		return ErrModuleRequired
	}
	return nil
}

// RunnerConfig configures a Runner.
type RunnerConfig struct {
	// Options contains common generation options.
	Options CommonOptions

	// TemplatesFS is the embedded filesystem containing templates.
	TemplatesFS fs.FS

	// TemplateDir is the subdirectory within TemplatesFS (e.g., "hertz", "kitex").
	TemplateDir string

	// FuncMap contains additional template functions (merged with DefaultFuncMap).
	FuncMap template.FuncMap

	// Logger for output. If nil, uses StdLogger.
	Logger Logger

	// WriteFS is the filesystem for writing files. If nil, uses OSWriteFS.
	WriteFS WriteFS

	// Hooks contains pre/post hooks. If nil, no hooks are run.
	Hooks *HookRegistry
}

// Runner executes generation steps.
type Runner struct {
	// Opt contains common options.
	Opt CommonOptions

	// Tpl is the template manager.
	Tpl *TemplateManager

	// Files is the file manager.
	Files *FileManager

	// Log is the logger.
	Log Logger

	// OnlyTags filters which steps to run.
	// If nil or empty, all steps are executed.
	// If set, only steps with at least one matching tag are executed.
	OnlyTags map[string]bool

	// Data is custom data accessible by steps.
	// Generators can store spec, options, or other data here.
	Data map[string]any

	// Hooks contains pre/post hooks.
	Hooks *HookRegistry
}

// NewRunner creates a new Runner.
func NewRunner(cfg RunnerConfig) *Runner {
	// Merge function maps
	funcMap := MergeFuncMap(DefaultFuncMap(), cfg.FuncMap)

	// Create logger
	log := cfg.Logger
	if log == nil {
		log = NewStdLogger(cfg.Options.Verbose)
	}

	// Create filesystem
	wfs := cfg.WriteFS
	if wfs == nil {
		wfs = &OSWriteFS{}
	}

	return &Runner{
		Opt:   cfg.Options,
		Tpl:   NewTemplateManager(cfg.TemplatesFS, cfg.TemplateDir, funcMap),
		Files: NewFileManagerWithFS(wfs, log),
		Log:   log,
		Data:  make(map[string]any),
		Hooks: cfg.Hooks,
	}
}

// Run executes all steps of a generator.
func (r *Runner) Run(ctx context.Context, g Generator) error {
	r.Log.Printf("Generating %s...", g.Name())
	r.Log.Printf("  Output: %s", r.Opt.OutputDir)
	r.Log.Printf("  Module: %s", r.Opt.Module)

	// Run pre-hooks
	if r.Hooks != nil {
		if err := r.Hooks.RunPreHooks(ctx, r); err != nil {
			return fmt.Errorf("pre-hook failed: %w", err)
		}
	}

	for _, step := range g.Steps() {
		if r.shouldSkip(step) {
			r.Log.Verbosef("  [skip] %s (tag not selected)", step.Name)
			continue
		}

		r.Log.Verbosef("  Generating %s...", step.Name)
		if err := step.Run(ctx, r); err != nil {
			return fmt.Errorf("step %s failed: %w", step.Name, err)
		}
	}

	// Run post-hooks
	if r.Hooks != nil {
		if err := r.Hooks.RunPostHooks(ctx, r); err != nil {
			return fmt.Errorf("post-hook failed: %w", err)
		}
	}

	r.Log.Printf("Done: %s", g.Name())
	return nil
}

// RunWithTags executes only steps with matching tags.
func (r *Runner) RunWithTags(ctx context.Context, g Generator, tags ...string) error {
	r.OnlyTags = make(map[string]bool)
	for _, tag := range tags {
		r.OnlyTags[tag] = true
	}
	return r.Run(ctx, g)
}

// shouldSkip returns true if the step should be skipped based on OnlyTags.
func (r *Runner) shouldSkip(step Step) bool {
	if len(r.OnlyTags) == 0 {
		return false
	}

	// Step must have at least one matching tag
	for _, tag := range step.Tags {
		if r.OnlyTags[tag] {
			return false
		}
	}
	return true
}

// GetData retrieves data by key, with type assertion.
func GetData[T any](r *Runner, key string) (T, bool) {
	val, ok := r.Data[key]
	if !ok {
		var zero T
		return zero, false
	}
	typed, ok := val.(T)
	return typed, ok
}

// SetData sets data by key.
func (r *Runner) SetData(key string, value any) {
	r.Data[key] = value
}
