package common

import (
	"testing"
)

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"User", "user"},
		{"UserProfile", "user_profile"},
		{"HTTPServer", "http_server"},
		{"APIVersion", "api_version"},
		{"user", "user"},
		{"userName", "user_name"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ToSnakeCase(tt.input)
			if result != tt.expected {
				t.Errorf("ToSnakeCase(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToPascalCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"user", "User"},
		{"user_profile", "UserProfile"},
		{"user-profile", "UserProfile"},
		{"User", "User"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ToPascalCase(tt.input)
			if result != tt.expected {
				t.Errorf("ToPascalCase(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToCamelCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"User", "user"},
		{"user_profile", "userProfile"},
		{"UserProfile", "userProfile"},
		{"user", "user"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ToCamelCase(tt.input)
			if result != tt.expected {
				t.Errorf("ToCamelCase(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToKebabCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"User", "user"},
		{"UserProfile", "user-profile"},
		{"user_profile", "user-profile"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ToKebabCase(tt.input)
			if result != tt.expected {
				t.Errorf("ToKebabCase(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestPluralize(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"User", "Users"},
		{"Task", "Tasks"},
		{"Box", "Boxes"},
		{"Class", "Classes"},
		{"Category", "Categories"},
		{"Day", "Days"},
		{"Toy", "Toys"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := Pluralize(tt.input)
			if result != tt.expected {
				t.Errorf("Pluralize(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNewEntityInfo(t *testing.T) {
	tests := []struct {
		input      string
		name       string
		snakeName  string
		camelName  string
		pluralName string
	}{
		{"User", "User", "user", "user", "Users"},
		{"user_profile", "UserProfile", "user_profile", "userProfile", "UserProfiles"},
		{"Task", "Task", "task", "task", "Tasks"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			entity := NewEntityInfo(tt.input)
			if entity.Name != tt.name {
				t.Errorf("NewEntityInfo(%q).Name = %q, want %q", tt.input, entity.Name, tt.name)
			}
			if entity.SnakeName != tt.snakeName {
				t.Errorf("NewEntityInfo(%q).SnakeName = %q, want %q", tt.input, entity.SnakeName, tt.snakeName)
			}
			if entity.CamelName != tt.camelName {
				t.Errorf("NewEntityInfo(%q).CamelName = %q, want %q", tt.input, entity.CamelName, tt.camelName)
			}
			if entity.PluralName != tt.pluralName {
				t.Errorf("NewEntityInfo(%q).PluralName = %q, want %q", tt.input, entity.PluralName, tt.pluralName)
			}
		})
	}
}

func TestReadModuleFromGoMod(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name: "standard go.mod",
			content: `module github.com/example/myproject

go 1.21

require (
    github.com/example/dep v1.0.0
)`,
			expected: "github.com/example/myproject",
		},
		{
			name:     "simple module",
			content:  "module example.com/test",
			expected: "example.com/test",
		},
		{
			name:     "empty content",
			content:  "",
			expected: "",
		},
		{
			name:     "no module line",
			content:  "go 1.21",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ReadModuleFromGoMod(tt.content)
			if result != tt.expected {
				t.Errorf("ReadModuleFromGoMod() = %q, want %q", result, tt.expected)
			}
		})
	}
}
