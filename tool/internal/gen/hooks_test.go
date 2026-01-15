package gen

import (
	"context"
	"errors"
	"testing"
)

func TestHookRegistry_AddAndRun(t *testing.T) {
	registry := NewHookRegistry()

	var preRan, postRan bool

	registry.AddPreHook(func(ctx context.Context, r *Runner) error {
		preRan = true
		return nil
	})

	registry.AddPostHook(func(ctx context.Context, r *Runner) error {
		postRan = true
		return nil
	})

	// Run pre-hooks
	if err := registry.RunPreHooks(context.Background(), nil); err != nil {
		t.Fatalf("RunPreHooks() error = %v", err)
	}
	if !preRan {
		t.Error("Pre-hook should have been executed")
	}

	// Run post-hooks
	if err := registry.RunPostHooks(context.Background(), nil); err != nil {
		t.Fatalf("RunPostHooks() error = %v", err)
	}
	if !postRan {
		t.Error("Post-hook should have been executed")
	}
}

func TestHookRegistry_PreHooksOrder(t *testing.T) {
	registry := NewHookRegistry()

	var order []int

	registry.AddPreHook(func(ctx context.Context, r *Runner) error {
		order = append(order, 1)
		return nil
	})
	registry.AddPreHook(func(ctx context.Context, r *Runner) error {
		order = append(order, 2)
		return nil
	})
	registry.AddPreHook(func(ctx context.Context, r *Runner) error {
		order = append(order, 3)
		return nil
	})

	registry.RunPreHooks(context.Background(), nil)

	if len(order) != 3 {
		t.Fatalf("Expected 3 hooks to run, got %d", len(order))
	}
	if order[0] != 1 || order[1] != 2 || order[2] != 3 {
		t.Errorf("Hooks should run in order, got %v", order)
	}
}

func TestHookRegistry_ErrorStopsExecution(t *testing.T) {
	registry := NewHookRegistry()

	var ran []int
	testErr := errors.New("hook error")

	registry.AddPreHook(func(ctx context.Context, r *Runner) error {
		ran = append(ran, 1)
		return nil
	})
	registry.AddPreHook(func(ctx context.Context, r *Runner) error {
		ran = append(ran, 2)
		return testErr
	})
	registry.AddPreHook(func(ctx context.Context, r *Runner) error {
		ran = append(ran, 3) // Should not run
		return nil
	})

	err := registry.RunPreHooks(context.Background(), nil)
	if !errors.Is(err, testErr) {
		t.Errorf("RunPreHooks() error = %v, want %v", err, testErr)
	}

	if len(ran) != 2 {
		t.Errorf("Only 2 hooks should have run, got %d", len(ran))
	}
}

func TestHookRegistry_GetHooks(t *testing.T) {
	registry := NewHookRegistry()

	hook1 := func(ctx context.Context, r *Runner) error { return nil }
	hook2 := func(ctx context.Context, r *Runner) error { return nil }

	registry.AddPreHook(hook1)
	registry.AddPostHook(hook2)

	preHooks := registry.PreHooks()
	if len(preHooks) != 1 {
		t.Errorf("PreHooks() returned %d hooks, want 1", len(preHooks))
	}

	postHooks := registry.PostHooks()
	if len(postHooks) != 1 {
		t.Errorf("PostHooks() returned %d hooks, want 1", len(postHooks))
	}
}

func TestHookRegistry_Chaining(t *testing.T) {
	// Test that Add methods return *HookRegistry for chaining
	registry := NewHookRegistry().
		AddPreHook(func(ctx context.Context, r *Runner) error { return nil }).
		AddPostHook(func(ctx context.Context, r *Runner) error { return nil })

	if len(registry.PreHooks()) != 1 {
		t.Error("Chained AddPreHook should work")
	}
	if len(registry.PostHooks()) != 1 {
		t.Error("Chained AddPostHook should work")
	}
}

func TestLogHook(t *testing.T) {
	var logged bool

	// Create a runner with test logger
	memFS := NewMemWriteFS()
	runner := &Runner{
		Files: NewFileManagerWithFS(memFS, nil),
		Log: &testLogger{
			printfFn: func(format string, args ...any) {
				logged = true
			},
		},
	}

	hook := LogHook("test message")
	if err := hook(context.Background(), runner); err != nil {
		t.Fatalf("LogHook() error = %v", err)
	}

	if !logged {
		t.Error("LogHook should have logged a message")
	}
}

func TestEnsureDirsHook(t *testing.T) {
	memFS := NewMemWriteFS()
	runner := &Runner{
		Opt:   CommonOptions{OutputDir: "/output"},
		Files: NewFileManagerWithFS(memFS, nil),
	}

	hook := EnsureDirsHook("cmd", "internal/handler")
	if err := hook(context.Background(), runner); err != nil {
		t.Fatalf("EnsureDirsHook() error = %v", err)
	}

	// Check directories were created
	dirs := memFS.GetDirs()
	foundCmd := false
	foundHandler := false
	for _, dir := range dirs {
		if dir == "/output/cmd" {
			foundCmd = true
		}
		if dir == "/output/internal/handler" {
			foundHandler = true
		}
	}

	if !foundCmd {
		t.Error("EnsureDirsHook should create /output/cmd")
	}
	if !foundHandler {
		t.Error("EnsureDirsHook should create /output/internal/handler")
	}
}

func TestDefaultHooks(t *testing.T) {
	preHooks := DefaultPreHooks()
	if len(preHooks) == 0 {
		t.Error("DefaultPreHooks() should return at least one hook")
	}

	postHooks := DefaultPostHooks()
	if len(postHooks) == 0 {
		t.Error("DefaultPostHooks() should return at least one hook")
	}
}

// testLogger is a simple logger for testing
type testLogger struct {
	printfFn   func(format string, args ...any)
	verbosefFn func(format string, args ...any)
	errorfFn   func(format string, args ...any)
}

func (l *testLogger) Printf(format string, args ...any) {
	if l.printfFn != nil {
		l.printfFn(format, args...)
	}
}

func (l *testLogger) Verbosef(format string, args ...any) {
	if l.verbosefFn != nil {
		l.verbosefFn(format, args...)
	}
}

func (l *testLogger) Errorf(format string, args ...any) {
	if l.errorfFn != nil {
		l.errorfFn(format, args...)
	}
}
