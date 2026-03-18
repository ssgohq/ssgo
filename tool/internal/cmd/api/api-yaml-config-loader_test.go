package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ssgohq/ssgo/tool/internal/cmdctx"
)

func boolPtr(b bool) *bool { return &b }

// --- LoadApiConfig tests ---

func TestLoadApiConfig_MissingFile(t *testing.T) {
	dir := t.TempDir()
	cfg, err := LoadApiConfig(dir)
	if err != nil {
		t.Fatalf("expected no error for missing .ss.yaml, got %v", err)
	}
	if !cfg.IsEmpty() {
		t.Errorf("expected empty config for missing file, got %+v", cfg)
	}
}

func TestLoadApiConfig_FullConfig(t *testing.T) {
	dir := t.TempDir()
	yml := `
api:
  apis:
    - file: api/user.api
      dir: user-svc
      options:
        port: 9090
        with_logic: false
        format: yaml
    - file: api/order.api
      dir: order-svc
      options:
        port: 8081
        with_logic: true
        format: json
`
	if err := os.WriteFile(filepath.Join(dir, ".ss.yaml"), []byte(yml), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadApiConfig(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.IsEmpty() {
		t.Fatal("expected non-empty config")
	}

	if len(cfg.Apis) != 2 {
		t.Fatalf("Apis count: got %d, want 2", len(cfg.Apis))
	}

	svc0 := cfg.Apis[0]
	if svc0.File != "api/user.api" {
		t.Errorf("Apis[0].File: got %q, want %q", svc0.File, "api/user.api")
	}
	if svc0.Dir != "user-svc" {
		t.Errorf("Apis[0].Dir: got %q, want %q", svc0.Dir, "user-svc")
	}
	if svc0.Options.Port != 9090 {
		t.Errorf("Apis[0].Options.Port: got %d, want 9090", svc0.Options.Port)
	}
	if svc0.Options.WithLogic == nil || *svc0.Options.WithLogic != false {
		t.Errorf("Apis[0].Options.WithLogic: expected false")
	}
	if svc0.Options.Format != "yaml" {
		t.Errorf("Apis[0].Options.Format: got %q, want %q", svc0.Options.Format, "yaml")
	}

	svc1 := cfg.Apis[1]
	if svc1.File != "api/order.api" {
		t.Errorf("Apis[1].File: got %q, want %q", svc1.File, "api/order.api")
	}
}

func TestLoadApiConfig_NoApiSection(t *testing.T) {
	dir := t.TempDir()
	yml := `
run:
  build_delay: 500ms
  services:
    - name: my-svc
      cmd: go run ./cmd
`
	if err := os.WriteFile(filepath.Join(dir, ".ss.yaml"), []byte(yml), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadApiConfig(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.IsEmpty() {
		t.Errorf("expected empty api config when api: section is absent, got %+v", cfg)
	}
}

func TestLoadApiConfig_InvalidYaml(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".ss.yaml"), []byte("invalid: [yaml: {"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadApiConfig(dir)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

// --- ApiConfig.IsEmpty tests ---

func TestApiConfig_IsEmpty(t *testing.T) {
	tests := []struct {
		name  string
		cfg   ApiConfig
		empty bool
	}{
		{"zero value", ApiConfig{}, true},
		{"one api entry", ApiConfig{Apis: []ApiServiceConfig{{File: "api/user.api"}}}, false},
		{"multiple entries", ApiConfig{Apis: []ApiServiceConfig{{File: "a.api"}, {File: "b.api"}}}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.cfg.IsEmpty(); got != tc.empty {
				t.Errorf("IsEmpty() = %v, want %v", got, tc.empty)
			}
		})
	}
}

// --- ResolveApiConfig tests ---

func TestResolveApiConfig_Defaults(t *testing.T) {
	// Create a temp dir with a go.mod so module auto-detect works
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/org/test\n\ngo 1.21\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx := &cmdctx.Context{
		WorkingDir: dir,
		Flags:      map[string]any{},
		Args:       []string{},
	}

	svc := ApiServiceConfig{
		File: "api/user.api",
		// Dir, Options all zero — should use defaults
	}

	resolved, err := ResolveApiConfig(ctx, svc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resolved.Port != 8080 {
		t.Errorf("Port default: got %d, want 8080", resolved.Port)
	}
	if !resolved.WithLogic {
		t.Errorf("WithLogic default: expected true")
	}
	if resolved.Format != "json" {
		t.Errorf("Format default: got %q, want %q", resolved.Format, "json")
	}
	if resolved.Module != "github.com/org/test" {
		t.Errorf("Module auto-detect: got %q, want %q", resolved.Module, "github.com/org/test")
	}
}

func TestResolveApiConfig_FlagOverride(t *testing.T) {
	dir := t.TempDir()

	ctx := &cmdctx.Context{
		WorkingDir: dir,
		Flags: map[string]any{
			"module":     "github.com/org/override",
			"m":          "github.com/org/override",
			"with-logic": false,
			"format":     "yaml",
		},
		Args: []string{},
	}

	svc := ApiServiceConfig{
		File: "api/user.api",
		Dir:  ".",
		Options: ApiServiceOptions{
			Port:      9090,
			WithLogic: boolPtr(true), // config says true, flag says false
			Format:    "json",        // config says json, flag says yaml
		},
	}

	resolved, err := ResolveApiConfig(ctx, svc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resolved.Module != "github.com/org/override" {
		t.Errorf("Module flag override: got %q, want %q", resolved.Module, "github.com/org/override")
	}
	if resolved.WithLogic {
		t.Errorf("WithLogic flag override: expected false (flag wins over config true)")
	}
	if resolved.Format != "yaml" {
		t.Errorf("Format flag override: got %q, want %q", resolved.Format, "yaml")
	}
	if resolved.Port != 9090 {
		t.Errorf("Port from config: got %d, want 9090", resolved.Port)
	}
}

func TestResolveApiConfig_MissingFile(t *testing.T) {
	dir := t.TempDir()
	ctx := &cmdctx.Context{
		WorkingDir: dir,
		Flags:      map[string]any{},
		Args:       []string{},
	}

	svc := ApiServiceConfig{File: ""} // missing file field

	_, err := ResolveApiConfig(ctx, svc)
	if err == nil {
		t.Error("expected error when File is empty")
	}
}
