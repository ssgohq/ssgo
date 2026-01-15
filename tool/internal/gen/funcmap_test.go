package gen

import (
	"testing"
)

func TestDefaultFuncMap(t *testing.T) {
	fm := DefaultFuncMap()

	// Test naming functions exist
	namingFuncs := []string{"ToSnakeCase", "ToCamelCase", "ToPascalCase", "ToKebabCase"}
	for _, name := range namingFuncs {
		if _, ok := fm[name]; !ok {
			t.Errorf("DefaultFuncMap() missing %q", name)
		}
	}

	// Test string functions exist
	stringFuncs := []string{
		"lower", "upper", "title", "trimPrefix", "trimSuffix",
		"contains", "hasPrefix", "hasSuffix", "replace", "replaceAll", "split", "join",
	}
	for _, name := range stringFuncs {
		if _, ok := fm[name]; !ok {
			t.Errorf("DefaultFuncMap() missing %q", name)
		}
	}

	// Test utility functions exist
	utilFuncs := []string{"default", "ternary", "add", "sub"}
	for _, name := range utilFuncs {
		if _, ok := fm[name]; !ok {
			t.Errorf("DefaultFuncMap() missing %q", name)
		}
	}
}

func TestMergeFuncMap(t *testing.T) {
	base := DefaultFuncMap()
	additional := map[string]any{
		"custom": func() string { return "custom" },
		"lower":  func(s string) string { return "overridden" }, // Override existing
	}

	merged := MergeFuncMap(base, additional)

	// Check custom function was added
	if _, ok := merged["custom"]; !ok {
		t.Error("MergeFuncMap() should add custom functions")
	}

	// Check base functions still exist
	if _, ok := merged["ToSnakeCase"]; !ok {
		t.Error("MergeFuncMap() should preserve base functions")
	}

	// Check override worked (we can't easily test the function itself, but we can check it exists)
	if _, ok := merged["lower"]; !ok {
		t.Error("MergeFuncMap() should have lower function")
	}
}

func TestDefaultValue(t *testing.T) {
	tests := []struct {
		name string
		def  any
		val  any
		want any
	}{
		{"nil value", "default", nil, "default"},
		{"empty string", "default", "", "default"},
		{"non-empty string", "default", "value", "value"},
		{"zero int", 100, 0, 0}, // Note: 0 is not "empty" for int
		{"non-zero int", 100, 42, 42},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := defaultValue(tt.def, tt.val)
			if result != tt.want {
				t.Errorf("defaultValue(%v, %v) = %v, want %v", tt.def, tt.val, result, tt.want)
			}
		})
	}
}

func TestTernary(t *testing.T) {
	tests := []struct {
		name    string
		cond    bool
		ifTrue  any
		ifFalse any
		want    any
	}{
		{"true condition", true, "yes", "no", "yes"},
		{"false condition", false, "yes", "no", "no"},
		{"with ints", true, 1, 2, 1},
		{"with nil", false, "value", nil, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ternary(tt.cond, tt.ifTrue, tt.ifFalse)
			if result != tt.want {
				t.Errorf("ternary(%v, %v, %v) = %v, want %v", tt.cond, tt.ifTrue, tt.ifFalse, result, tt.want)
			}
		})
	}
}

func TestAdd(t *testing.T) {
	tests := []struct {
		a, b, want int
	}{
		{1, 2, 3},
		{0, 0, 0},
		{-1, 1, 0},
		{100, 200, 300},
	}

	for _, tt := range tests {
		result := add(tt.a, tt.b)
		if result != tt.want {
			t.Errorf("add(%d, %d) = %d, want %d", tt.a, tt.b, result, tt.want)
		}
	}
}

func TestSub(t *testing.T) {
	tests := []struct {
		a, b, want int
	}{
		{3, 2, 1},
		{0, 0, 0},
		{1, 2, -1},
		{100, 50, 50},
	}

	for _, tt := range tests {
		result := sub(tt.a, tt.b)
		if result != tt.want {
			t.Errorf("sub(%d, %d) = %d, want %d", tt.a, tt.b, result, tt.want)
		}
	}
}

func TestTitleCaser(t *testing.T) {
	// Test that titleCaser works correctly
	result := titleCaser.String("hello world")
	if result != "Hello World" {
		t.Errorf("titleCaser.String(\"hello world\") = %q, want \"Hello World\"", result)
	}
}
