package gen

import (
	"errors"
	"fmt"
)

// StepError provides rich context for debugging generation failures.
type StepError struct {
	Generator string // Generator name (e.g., "asynq-scaffold")
	Step      string // Step name (e.g., "job_handler")
	Path      string // Output file path
	Mode      string // Update mode: skip/cover/append
	Template  string // Template name or "inline"
	Cause     error  // Underlying error
}

// Error implements the error interface.
func (e *StepError) Error() string {
	if e.Path != "" {
		return fmt.Sprintf("[%s] step '%s' failed for %s (mode=%s, tpl=%s): %v",
			e.Generator, e.Step, e.Path, e.Mode, e.Template, e.Cause)
	}
	return fmt.Sprintf("[%s] step '%s' failed (tpl=%s): %v",
		e.Generator, e.Step, e.Template, e.Cause)
}

// Unwrap returns the underlying error.
func (e *StepError) Unwrap() error {
	return e.Cause
}

// WrapStepError creates a StepError with full context.
// Returns nil if err is nil.
func WrapStepError(gen, step, path, mode, tpl string, err error) error {
	if err == nil {
		return nil
	}
	return &StepError{
		Generator: gen,
		Step:      step,
		Path:      path,
		Mode:      mode,
		Template:  tpl,
		Cause:     err,
	}
}

// NewStepError creates a new StepError with the given message.
func NewStepError(gen, step, path, mode, tpl, message string) error {
	return &StepError{
		Generator: gen,
		Step:      step,
		Path:      path,
		Mode:      mode,
		Template:  tpl,
		Cause:     errors.New(message),
	}
}

// StepErrorBuilder helps build StepError with fluent API.
type StepErrorBuilder struct {
	err StepError
}

// NewStepErrorBuilder creates a new StepErrorBuilder.
func NewStepErrorBuilder() *StepErrorBuilder {
	return &StepErrorBuilder{}
}

// Generator sets the generator name.
func (b *StepErrorBuilder) Generator(name string) *StepErrorBuilder {
	b.err.Generator = name
	return b
}

// Step sets the step name.
func (b *StepErrorBuilder) Step(name string) *StepErrorBuilder {
	b.err.Step = name
	return b
}

// Path sets the output path.
func (b *StepErrorBuilder) Path(path string) *StepErrorBuilder {
	b.err.Path = path
	return b
}

// Mode sets the update mode.
func (b *StepErrorBuilder) Mode(mode string) *StepErrorBuilder {
	b.err.Mode = mode
	return b
}

// Template sets the template name.
func (b *StepErrorBuilder) Template(tpl string) *StepErrorBuilder {
	b.err.Template = tpl
	return b
}

// Wrap wraps an existing error. Returns nil if err is nil.
func (b *StepErrorBuilder) Wrap(err error) error {
	if err == nil {
		return nil
	}
	b.err.Cause = err
	return &b.err
}

// Errorf creates an error with a formatted message.
func (b *StepErrorBuilder) Errorf(format string, args ...any) error {
	b.err.Cause = fmt.Errorf(format, args...)
	return &b.err
}

// IsStepError checks if an error is a StepError.
func IsStepError(err error) bool {
	var stepErr *StepError
	return errors.As(err, &stepErr)
}

// GetStepError extracts StepError from an error chain.
func GetStepError(err error) *StepError {
	var stepErr *StepError
	if errors.As(err, &stepErr) {
		return stepErr
	}
	return nil
}
