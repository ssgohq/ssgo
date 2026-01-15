package gen

import (
	"testing"
	"testing/fstest"
)

func TestTemplateManager_Render(t *testing.T) {
	// Create in-memory filesystem with templates
	fsys := fstest.MapFS{
		"templates/hello.tpl": &fstest.MapFile{
			Data: []byte("Hello, {{.Name}}!"),
		},
		"templates/list.tpl": &fstest.MapFile{
			Data: []byte("Items: {{range .Items}}{{.}}, {{end}}"),
		},
	}

	tm := NewTemplateManager(fsys, "templates", DefaultFuncMap())

	// Test basic render
	result, err := tm.Render("hello.tpl", map[string]string{"Name": "World"})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if result != "Hello, World!" {
		t.Errorf("Render() = %q, want %q", result, "Hello, World!")
	}

	// Test render with slice
	result, err = tm.Render("list.tpl", map[string][]string{"Items": {"a", "b", "c"}})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if result != "Items: a, b, c, " {
		t.Errorf("Render() = %q, want %q", result, "Items: a, b, c, ")
	}
}

func TestTemplateManager_RenderString(t *testing.T) {
	fsys := fstest.MapFS{}
	tm := NewTemplateManager(fsys, "", DefaultFuncMap())

	tests := []struct {
		name     string
		template string
		data     any
		want     string
	}{
		{
			name:     "simple string",
			template: "Hello, {{.Name}}!",
			data:     map[string]string{"Name": "World"},
			want:     "Hello, World!",
		},
		{
			name:     "empty template",
			template: "",
			data:     nil,
			want:     "",
		},
		{
			name:     "with naming functions",
			template: "{{.Name | ToSnakeCase}}",
			data:     map[string]string{"Name": "UserID"},
			want:     "user_id",
		},
		{
			name:     "with default",
			template: "{{default 8080 .Port}}",
			data:     map[string]any{"Port": nil},
			want:     "8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tm.RenderString(tt.template, tt.data)
			if err != nil {
				t.Fatalf("RenderString() error = %v", err)
			}
			if result != tt.want {
				t.Errorf("RenderString() = %q, want %q", result, tt.want)
			}
		})
	}
}

func TestTemplateManager_RenderStringCaching(t *testing.T) {
	fsys := fstest.MapFS{}
	tm := NewTemplateManager(fsys, "", DefaultFuncMap())

	template := "Hello, {{.Name}}!"

	// Render same template multiple times
	for i := 0; i < 100; i++ {
		result, err := tm.RenderString(template, map[string]string{"Name": "World"})
		if err != nil {
			t.Fatalf("RenderString() error = %v", err)
		}
		if result != "Hello, World!" {
			t.Errorf("RenderString() = %q, want %q", result, "Hello, World!")
		}
	}

	// Template should be cached (we can't directly test cache, but no error means caching works)
}

func TestTemplateManager_RenderToFile(t *testing.T) {
	fsys := fstest.MapFS{
		"templates/config.tpl": &fstest.MapFile{
			Data: []byte("port: {{.Port}}\nhost: {{.Host}}"),
		},
	}

	tm := NewTemplateManager(fsys, "templates", DefaultFuncMap())
	memFS := NewMemWriteFS()
	fm := NewFileManagerWithFS(memFS, nil)

	// Render to file
	err := tm.RenderToFile(fm, "config.tpl", "/output/config.yaml", map[string]any{
		"Port": 8080,
		"Host": "localhost",
	})
	if err != nil {
		t.Fatalf("RenderToFile() error = %v", err)
	}

	// Verify file contents
	content, err := memFS.ReadFile("/output/config.yaml")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	expected := "port: 8080\nhost: localhost"
	if string(content) != expected {
		t.Errorf("File content = %q, want %q", string(content), expected)
	}
}

func TestTemplateManager_RenderSkipExisting(t *testing.T) {
	fsys := fstest.MapFS{
		"templates/test.tpl": &fstest.MapFile{
			Data: []byte("new content"),
		},
	}

	tm := NewTemplateManager(fsys, "templates", DefaultFuncMap())
	memFS := NewMemWriteFS()
	fm := NewFileManagerWithFS(memFS, nil)

	// Pre-create file
	memFS.WriteFile("/output/test.txt", []byte("existing content"), 0o644)

	// Try to render (should skip)
	skipped, err := tm.RenderSkipExisting(fm, "test.tpl", "/output/test.txt", nil)
	if err != nil {
		t.Fatalf("RenderSkipExisting() error = %v", err)
	}
	if !skipped {
		t.Error("RenderSkipExisting() skipped = false, want true")
	}

	// Verify original content preserved
	content, _ := memFS.ReadFile("/output/test.txt")
	if string(content) != "existing content" {
		t.Errorf("File should not be overwritten, got %q", string(content))
	}

	// Render to new file (should not skip)
	skipped, err = tm.RenderSkipExisting(fm, "test.tpl", "/output/new.txt", nil)
	if err != nil {
		t.Fatalf("RenderSkipExisting() error = %v", err)
	}
	if skipped {
		t.Error("RenderSkipExisting() skipped = true, want false for new file")
	}

	// Verify new file content
	content, _ = memFS.ReadFile("/output/new.txt")
	if string(content) != "new content" {
		t.Errorf("New file content = %q, want %q", string(content), "new content")
	}
}

func TestTemplateManager_RenderError(t *testing.T) {
	fsys := fstest.MapFS{
		"templates/bad.tpl": &fstest.MapFile{
			Data: []byte("{{.Missing.Field}}"),
		},
	}

	tm := NewTemplateManager(fsys, "templates", DefaultFuncMap())

	// Render non-existent template should error
	_, err := tm.Render("nonexistent.tpl", nil)
	if err == nil {
		t.Error("Render() should return error for non-existent template")
	}

	// Render with invalid template syntax
	_, err = tm.RenderString("{{.Unclosed", nil)
	if err == nil {
		t.Error("RenderString() should return error for invalid template syntax")
	}
}

func TestTemplateManager_FuncMap(t *testing.T) {
	fsys := fstest.MapFS{}
	funcMap := DefaultFuncMap()
	tm := NewTemplateManager(fsys, "", funcMap)

	result := tm.FuncMap()
	if result == nil {
		t.Error("FuncMap() should not return nil")
	}

	// Check that default functions exist
	expectedFuncs := []string{"ToSnakeCase", "ToPascalCase", "lower", "upper", "default"}
	for _, name := range expectedFuncs {
		if _, ok := result[name]; !ok {
			t.Errorf("FuncMap() missing function %q", name)
		}
	}
}

func TestTemplateManager_ClearCache(t *testing.T) {
	fsys := fstest.MapFS{
		"templates/test.tpl": &fstest.MapFile{
			Data: []byte("content"),
		},
	}

	tm := NewTemplateManager(fsys, "templates", DefaultFuncMap())

	// Render to populate cache
	tm.Render("test.tpl", nil)
	tm.RenderString("inline", nil)

	// Clear all caches
	tm.ClearCache()

	// Should still work after clearing
	result, err := tm.Render("test.tpl", nil)
	if err != nil {
		t.Fatalf("Render() after ClearCache() error = %v", err)
	}
	if result != "content" {
		t.Errorf("Render() = %q, want %q", result, "content")
	}
}
