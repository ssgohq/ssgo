package gen

import (
	"errors"
	"testing"
)

func TestStepError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  *StepError
		want string
	}{
		{
			name: "with path",
			err: &StepError{
				Generator: "my-gen",
				Step:      "step1",
				Path:      "/output/file.go",
				Mode:      "cover",
				Template:  "file.go.tpl",
				Cause:     errors.New("write failed"),
			},
			want: "[my-gen] step 'step1' failed for /output/file.go (mode=cover, tpl=file.go.tpl): write failed",
		},
		{
			name: "without path",
			err: &StepError{
				Generator: "my-gen",
				Step:      "step2",
				Template:  "other.tpl",
				Cause:     errors.New("template error"),
			},
			want: "[my-gen] step 'step2' failed (tpl=other.tpl): template error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStepError_Unwrap(t *testing.T) {
	cause := errors.New("underlying error")
	err := &StepError{
		Generator: "gen",
		Step:      "step",
		Cause:     cause,
	}

	unwrapped := err.Unwrap()
	if unwrapped == nil || unwrapped.Error() != cause.Error() {
		t.Error("Unwrap() should return the cause")
	}
}

func TestWrapStepError(t *testing.T) {
	// Test with nil error
	result := WrapStepError("gen", "step", "/path", "cover", "tpl", nil)
	if result != nil {
		t.Error("WrapStepError() with nil error should return nil")
	}

	// Test with actual error
	cause := errors.New("test error")
	result = WrapStepError("gen", "step", "/path", "cover", "tpl", cause)
	if result == nil {
		t.Fatal("WrapStepError() should return non-nil for non-nil error")
	}

	var stepErr *StepError
	if !errors.As(result, &stepErr) {
		t.Fatal("WrapStepError() should return *StepError")
	}

	if stepErr.Generator != "gen" {
		t.Errorf("Generator = %q, want %q", stepErr.Generator, "gen")
	}
	if stepErr.Step != "step" {
		t.Errorf("Step = %q, want %q", stepErr.Step, "step")
	}
	if stepErr.Path != "/path" {
		t.Errorf("Path = %q, want %q", stepErr.Path, "/path")
	}
	if stepErr.Mode != "cover" {
		t.Errorf("Mode = %q, want %q", stepErr.Mode, "cover")
	}
	if stepErr.Template != "tpl" {
		t.Errorf("Template = %q, want %q", stepErr.Template, "tpl")
	}
	if !errors.Is(stepErr.Cause, cause) {
		t.Error("Cause should be the original error")
	}
}

func TestNewStepError(t *testing.T) {
	err := NewStepError("gen", "step", "/path", "append", "tpl", "custom message")

	var stepErr *StepError
	if !errors.As(err, &stepErr) {
		t.Fatal("NewStepError() should return *StepError")
	}

	if stepErr.Cause.Error() != "custom message" {
		t.Errorf("Cause.Error() = %q, want %q", stepErr.Cause.Error(), "custom message")
	}
}

func TestStepErrorBuilder(t *testing.T) {
	cause := errors.New("cause error")

	err := NewStepErrorBuilder().
		Generator("builder-gen").
		Step("builder-step").
		Path("/builder/path").
		Mode("skip").
		Template("builder.tpl").
		Wrap(cause)

	var stepErr *StepError
	if !errors.As(err, &stepErr) {
		t.Fatal("Builder.Wrap() should return *StepError")
	}

	if stepErr.Generator != "builder-gen" {
		t.Errorf("Generator = %q, want %q", stepErr.Generator, "builder-gen")
	}
	if stepErr.Step != "builder-step" {
		t.Errorf("Step = %q, want %q", stepErr.Step, "builder-step")
	}
	if stepErr.Path != "/builder/path" {
		t.Errorf("Path = %q, want %q", stepErr.Path, "/builder/path")
	}
	if stepErr.Mode != "skip" {
		t.Errorf("Mode = %q, want %q", stepErr.Mode, "skip")
	}
	if stepErr.Template != "builder.tpl" {
		t.Errorf("Template = %q, want %q", stepErr.Template, "builder.tpl")
	}
}

func TestStepErrorBuilder_WrapNil(t *testing.T) {
	err := NewStepErrorBuilder().
		Generator("gen").
		Wrap(nil)
	if err != nil {
		t.Error("Builder.Wrap(nil) should return nil")
	}
}

func TestStepErrorBuilder_Errorf(t *testing.T) {
	err := NewStepErrorBuilder().
		Generator("gen").
		Step("step").
		Errorf("formatted %s %d", "error", 42)

	var stepErr *StepError
	if !errors.As(err, &stepErr) {
		t.Fatal("Builder.Errorf() should return *StepError")
	}

	if stepErr.Cause.Error() != "formatted error 42" {
		t.Errorf("Cause.Error() = %q, want %q", stepErr.Cause.Error(), "formatted error 42")
	}
}

func TestIsStepError(t *testing.T) {
	stepErr := &StepError{Generator: "gen", Cause: errors.New("test")}
	regularErr := errors.New("regular error")

	if !IsStepError(stepErr) {
		t.Error("IsStepError() = false for *StepError, want true")
	}

	if IsStepError(regularErr) {
		t.Error("IsStepError() = true for regular error, want false")
	}

	if IsStepError(nil) {
		t.Error("IsStepError() = true for nil, want false")
	}
}

func TestGetStepError(t *testing.T) {
	stepErr := &StepError{Generator: "gen", Cause: errors.New("test")}

	result := GetStepError(stepErr)
	if result != stepErr {
		t.Error("GetStepError() should return the StepError")
	}

	result = GetStepError(errors.New("regular"))
	if result != nil {
		t.Error("GetStepError() should return nil for regular error")
	}

	result = GetStepError(nil)
	if result != nil {
		t.Error("GetStepError() should return nil for nil error")
	}
}

func TestCommonErrors(t *testing.T) {
	// Test that common errors are defined
	if ErrOutputDirRequired == nil {
		t.Error("ErrOutputDirRequired should not be nil")
	}
	if ErrModuleRequired == nil {
		t.Error("ErrModuleRequired should not be nil")
	}
	if ErrTemplateNotFound == nil {
		t.Error("ErrTemplateNotFound should not be nil")
	}
}
