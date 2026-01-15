package gen

import (
	"context"
	"os/exec"
	"path/filepath"
)

// Hook is a function called before/after generation steps.
type Hook func(ctx context.Context, r *Runner) error

// HookRegistry manages pre/post hooks for generation.
type HookRegistry struct {
	preHooks  []Hook
	postHooks []Hook
}

// NewHookRegistry creates a new HookRegistry.
func NewHookRegistry() *HookRegistry {
	return &HookRegistry{}
}

// AddPreHook adds a hook to run before all steps.
func (h *HookRegistry) AddPreHook(hook Hook) *HookRegistry {
	h.preHooks = append(h.preHooks, hook)
	return h
}

// AddPostHook adds a hook to run after all steps.
func (h *HookRegistry) AddPostHook(hook Hook) *HookRegistry {
	h.postHooks = append(h.postHooks, hook)
	return h
}

// RunPreHooks executes all pre-hooks.
func (h *HookRegistry) RunPreHooks(ctx context.Context, r *Runner) error {
	for _, hook := range h.preHooks {
		if err := hook(ctx, r); err != nil {
			return err
		}
	}
	return nil
}

// RunPostHooks executes all post-hooks.
func (h *HookRegistry) RunPostHooks(ctx context.Context, r *Runner) error {
	for _, hook := range h.postHooks {
		if err := hook(ctx, r); err != nil {
			return err
		}
	}
	return nil
}

// PreHooks returns the list of pre-hooks.
func (h *HookRegistry) PreHooks() []Hook {
	return h.preHooks
}

// PostHooks returns the list of post-hooks.
func (h *HookRegistry) PostHooks() []Hook {
	return h.postHooks
}

// Common hooks

// GoFmtHook runs go fmt on generated .go files.
var GoFmtHook Hook = func(ctx context.Context, r *Runner) error {
	outputDir := r.Opt.OutputDir
	if outputDir == "" {
		return nil
	}

	cmd := exec.CommandContext(ctx, "go", "fmt", "./...")
	cmd.Dir = outputDir
	// Ignore errors - formatting is best-effort
	_ = cmd.Run()
	return nil
}

// GoImportsHook runs goimports on generated .go files (if available).
var GoImportsHook Hook = func(ctx context.Context, r *Runner) error {
	outputDir := r.Opt.OutputDir
	if outputDir == "" {
		return nil
	}

	// Check if goimports is available
	if _, err := exec.LookPath("goimports"); err != nil {
		return nil // goimports not installed, skip
	}

	cmd := exec.CommandContext(ctx, "goimports", "-w", ".")
	cmd.Dir = outputDir
	// Ignore errors - formatting is best-effort
	_ = cmd.Run()
	return nil
}

// GoModTidyHook runs go mod tidy after generation.
var GoModTidyHook Hook = func(ctx context.Context, r *Runner) error {
	outputDir := r.Opt.OutputDir
	if outputDir == "" {
		return nil
	}

	// Check if go.mod exists
	goModPath := filepath.Join(outputDir, "go.mod")
	if !r.Files.Exists(goModPath) {
		return nil
	}

	cmd := exec.CommandContext(ctx, "go", "mod", "tidy")
	cmd.Dir = outputDir
	// Ignore errors - tidy is best-effort
	_ = cmd.Run()
	return nil
}

// GoModInitHook initializes go.mod if it doesn't exist.
func GoModInitHook(module string) Hook {
	return func(ctx context.Context, r *Runner) error {
		outputDir := r.Opt.OutputDir
		if outputDir == "" {
			return nil
		}

		// Check if go.mod exists
		goModPath := filepath.Join(outputDir, "go.mod")
		if r.Files.Exists(goModPath) {
			return nil
		}

		mod := module
		if mod == "" {
			mod = r.Opt.Module
		}
		if mod == "" {
			return nil
		}

		cmd := exec.CommandContext(ctx, "go", "mod", "init", mod)
		cmd.Dir = outputDir
		return cmd.Run()
	}
}

// LogHook creates a hook that logs a message.
func LogHook(message string) Hook {
	return func(ctx context.Context, r *Runner) error {
		r.Log.Printf(message)
		return nil
	}
}

// EnsureDirsHook creates a hook that ensures directories exist.
func EnsureDirsHook(dirs ...string) Hook {
	return func(ctx context.Context, r *Runner) error {
		for _, dir := range dirs {
			absDir := dir
			if !filepath.IsAbs(dir) {
				absDir = filepath.Join(r.Opt.OutputDir, dir)
			}
			if err := r.Files.MkdirAll(absDir); err != nil {
				return err
			}
		}
		return nil
	}
}

// DefaultPreHooks returns commonly used pre-hooks.
func DefaultPreHooks() []Hook {
	return []Hook{
		LogHook("Starting generation..."),
	}
}

// DefaultPostHooks returns commonly used post-hooks.
func DefaultPostHooks() []Hook {
	return []Hook{
		GoFmtHook,
		LogHook("Generation completed."),
	}
}
