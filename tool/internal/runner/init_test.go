package runner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverServices(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()

	// Create service1 with go.mod and main.go
	svc1Dir := filepath.Join(tmpDir, "service1")
	_ = os.MkdirAll(svc1Dir, 0o755)
	_ = os.WriteFile(filepath.Join(svc1Dir, "go.mod"), []byte("module github.com/test/service1\n\ngo 1.21\n"), 0o644)
	_ = os.WriteFile(filepath.Join(svc1Dir, "main.go"), []byte("package main\nfunc main() {}\n"), 0o644)

	// Create service2 with cmd/server structure
	svc2Dir := filepath.Join(tmpDir, "service2")
	svc2CmdDir := filepath.Join(svc2Dir, "cmd", "server")
	_ = os.MkdirAll(svc2CmdDir, 0o755)
	_ = os.WriteFile(filepath.Join(svc2Dir, "go.mod"), []byte("module github.com/test/service2\n\ngo 1.21\n"), 0o644)
	_ = os.WriteFile(filepath.Join(svc2CmdDir, "main.go"), []byte("package main\nfunc main() {}\n"), 0o644)

	// Create library (no main.go) - should be skipped
	libDir := filepath.Join(tmpDir, "lib")
	_ = os.MkdirAll(libDir, 0o755)
	_ = os.WriteFile(filepath.Join(libDir, "go.mod"), []byte("module github.com/test/lib\n\ngo 1.21\n"), 0o644)
	_ = os.WriteFile(filepath.Join(libDir, "lib.go"), []byte("package lib\n"), 0o644)

	// Run discovery
	services, err := discoverServices(tmpDir)
	if err != nil {
		t.Fatalf("discoverServices failed: %v", err)
	}

	// Should find 2 services (service1 and service2), not lib
	if len(services) != 2 {
		t.Errorf("expected 2 services, got %d", len(services))
		for _, svc := range services {
			t.Logf("  found: %s at %s", svc.Name, svc.Dir)
		}
	}

	// Verify service names
	names := make(map[string]bool)
	for _, svc := range services {
		names[svc.Name] = true
	}
	if !names["service1"] {
		t.Error("expected to find service1")
	}
	if !names["service2"] {
		t.Error("expected to find service2")
	}
	if names["lib"] {
		t.Error("should not find lib (no main.go)")
	}
}

func TestDiscoverServicesSkipsVendor(t *testing.T) {
	tmpDir := t.TempDir()

	// Create main service in a subdirectory (not root)
	svcDir := filepath.Join(tmpDir, "myservice")
	_ = os.MkdirAll(svcDir, 0o755)
	_ = os.WriteFile(filepath.Join(svcDir, "go.mod"), []byte("module github.com/test/myservice\n\ngo 1.21\n"), 0o644)
	_ = os.WriteFile(filepath.Join(svcDir, "main.go"), []byte("package main\nfunc main() {}\n"), 0o644)

	// Create vendor directory with go.mod (should be skipped)
	vendorDir := filepath.Join(tmpDir, "vendor", "some-dep")
	_ = os.MkdirAll(vendorDir, 0o755)
	_ = os.WriteFile(filepath.Join(vendorDir, "go.mod"), []byte("module github.com/some/dep\n"), 0o644)
	_ = os.WriteFile(filepath.Join(vendorDir, "main.go"), []byte("package main\nfunc main() {}\n"), 0o644)

	services, err := discoverServices(tmpDir)
	if err != nil {
		t.Fatalf("discoverServices failed: %v", err)
	}

	// Should only find the myservice, not vendor
	if len(services) != 1 {
		t.Errorf("expected 1 service, got %d", len(services))
	}
	if len(services) > 0 && services[0].Name == "some-dep" {
		t.Error("should not discover services in vendor directory")
	}
	if len(services) > 0 && services[0].Name != "myservice" {
		t.Errorf("expected service name 'myservice', got %q", services[0].Name)
	}
}

func TestHasMainGo(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(dir string)
		expected bool
	}{
		{
			name: "main.go at root",
			setup: func(dir string) {
				_ = os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0o644)
			},
			expected: true,
		},
		{
			name: "cmd/main.go",
			setup: func(dir string) {
				cmdDir := filepath.Join(dir, "cmd")
				_ = os.MkdirAll(cmdDir, 0o755)
				_ = os.WriteFile(filepath.Join(cmdDir, "main.go"), []byte("package main"), 0o644)
			},
			expected: true,
		},
		{
			name: "cmd/server/main.go",
			setup: func(dir string) {
				serverDir := filepath.Join(dir, "cmd", "server")
				_ = os.MkdirAll(serverDir, 0o755)
				_ = os.WriteFile(filepath.Join(serverDir, "main.go"), []byte("package main"), 0o644)
			},
			expected: true,
		},
		{
			name: "no main.go (library)",
			setup: func(dir string) {
				_ = os.WriteFile(filepath.Join(dir, "lib.go"), []byte("package lib"), 0o644)
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tt.setup(tmpDir)

			got := hasMainGo(tmpDir)
			if got != tt.expected {
				t.Errorf("hasMainGo() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDetectRunCommand(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(dir string)
		expected string
	}{
		{
			name: "main.go at root",
			setup: func(dir string) {
				_ = os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0o644)
			},
			expected: "go run .",
		},
		{
			name: "cmd/server directory",
			setup: func(dir string) {
				serverDir := filepath.Join(dir, "cmd", "server")
				_ = os.MkdirAll(serverDir, 0o755)
				_ = os.WriteFile(filepath.Join(serverDir, "main.go"), []byte("package main"), 0o644)
			},
			expected: "go run ./cmd/server",
		},
		{
			name: "cmd/main.go",
			setup: func(dir string) {
				cmdDir := filepath.Join(dir, "cmd")
				_ = os.MkdirAll(cmdDir, 0o755)
				_ = os.WriteFile(filepath.Join(cmdDir, "main.go"), []byte("package main"), 0o644)
			},
			expected: "go run ./cmd",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tt.setup(tmpDir)

			got := detectRunCommand(tmpDir)
			if got != tt.expected {
				t.Errorf("detectRunCommand() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestBuildRunConfig(t *testing.T) {
	services := []ServiceDiscovery{
		{Name: "api", Dir: "./api", Module: "github.com/test/api"},
		{Name: "worker", Dir: "./worker", Module: "github.com/test/worker"},
	}

	config := buildRunConfig(services)

	// Check build_delay
	if delay, ok := config["build_delay"].(string); !ok || delay != "500ms" {
		t.Errorf("expected build_delay=500ms, got %v", config["build_delay"])
	}

	// Check services
	serviceConfigs, ok := config["services"].([]map[string]interface{})
	if !ok {
		t.Fatalf("services should be []map[string]interface{}")
	}

	if len(serviceConfigs) != 2 {
		t.Errorf("expected 2 service configs, got %d", len(serviceConfigs))
	}

	// Verify first service
	if serviceConfigs[0]["name"] != "api" {
		t.Errorf("expected first service name=api, got %v", serviceConfigs[0]["name"])
	}
	if serviceConfigs[0]["color"] != "cyan" {
		t.Errorf("expected first service color=cyan, got %v", serviceConfigs[0]["color"])
	}
}

func TestUpdateRunSection(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".ss.yaml")

	// Create initial config with other sections
	initial := `name: test-project
version: "1.0"

plugins:
  - name: some-plugin
`
	_ = os.WriteFile(configPath, []byte(initial), 0o644)

	// Update with run config
	runConfig := map[string]interface{}{
		"build_delay": "500ms",
		"services": []map[string]interface{}{
			{"name": "api", "cmd": "go run ."},
		},
	}

	err := updateRunSection(configPath, runConfig)
	if err != nil {
		t.Fatalf("updateRunSection failed: %v", err)
	}

	// Read result
	data, _ := os.ReadFile(configPath)
	content := string(data)

	// Should contain original sections
	if !contains(content, "name: test-project") {
		t.Error("should preserve name section")
	}
	if !contains(content, "plugins:") {
		t.Error("should preserve plugins section")
	}

	// Should contain run section
	if !contains(content, "run:") {
		t.Error("should add run section")
	}
	if !contains(content, "build_delay:") {
		t.Error("should contain build_delay")
	}
}

func TestCreateNewConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".ss.yaml")

	runConfig := map[string]interface{}{
		"build_delay": "500ms",
		"services": []map[string]interface{}{
			{"name": "api", "cmd": "go run ."},
		},
	}

	err := createNewConfig(configPath, runConfig)
	if err != nil {
		t.Fatalf("createNewConfig failed: %v", err)
	}

	// Verify file was created
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read created config: %v", err)
	}

	content := string(data)
	if !contains(content, "run:") {
		t.Error("config should contain run section")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
