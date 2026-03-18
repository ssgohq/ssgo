package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/ssgohq/ssgo/internal/util/gomod"
)

// ServiceDiscovery holds discovered service information.
type ServiceDiscovery struct {
	Name   string
	Dir    string
	Module string // Go module name from go.mod
}

// InitConfig generates run configuration by scanning for go.mod files.
func InitConfig(workDir string) error {
	// Discover services
	services, err := discoverServices(workDir)
	if err != nil {
		return fmt.Errorf("failed to discover services: %w", err)
	}

	if len(services) == 0 {
		return fmt.Errorf("no Go services (with main.go) found in %s", workDir)
	}

	// Build run config
	runConfig := buildRunConfig(services)

	// Check for existing .ss.yaml
	// Validate configPath is within workDir to prevent path traversal
	configPath, err := safeJoin(workDir, ".ss.yaml")
	if err != nil {
		return err
	}

	if _, err := os.Stat(configPath); err == nil {
		// File exists - append/update run section only
		if err := updateRunSection(configPath, runConfig); err != nil {
			return err
		}
	} else {
		// Create new file
		if err := createNewConfig(configPath, runConfig); err != nil {
			return err
		}
	}

	fmt.Printf("Generated run config with %d service(s) in .ss.yaml\n", len(services))
	for _, svc := range services {
		if svc.Module != "" {
			fmt.Printf("  - %s (%s) [%s]\n", svc.Name, svc.Dir, svc.Module)
		} else {
			fmt.Printf("  - %s (%s)\n", svc.Name, svc.Dir)
		}
	}

	return nil
}

// updateRunSection updates only the run: section in existing .ss.yaml
func updateRunSection(configPath string, runConfig map[string]interface{}) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read .ss.yaml: %w", err)
	}

	content := string(data)

	// Generate run section YAML
	runYAML, err := yaml.Marshal(map[string]interface{}{"run": runConfig})
	if err != nil {
		return fmt.Errorf("failed to marshal run config: %w", err)
	}

	// Check if run: section already exists
	if strings.Contains(content, "\nrun:") || strings.HasPrefix(content, "run:") {
		// Replace existing run section
		// Find where run: starts and where it ends (next top-level key or EOF)
		lines := strings.Split(content, "\n")
		var newLines []string
		inRunSection := false
		runInserted := false

		for _, line := range lines {
			// Check if this is a top-level key (no leading whitespace, ends with :)
			isTopLevel := len(line) > 0 && line[0] != ' ' && line[0] != '\t' && strings.Contains(line, ":")

			if isTopLevel && strings.HasPrefix(line, "run:") {
				// Start of run section - skip it
				inRunSection = true
				continue
			}

			if inRunSection {
				if isTopLevel {
					// End of run section - insert new run config before this line
					inRunSection = false
					if !runInserted {
						newLines = append(newLines, strings.TrimSuffix(string(runYAML), "\n"))
						runInserted = true
					}
					newLines = append(newLines, line)
				}
				// Skip lines in old run section
				continue
			}

			newLines = append(newLines, line)
		}

		// If run section was at the end, append new config
		if !runInserted {
			newLines = append(newLines, strings.TrimSuffix(string(runYAML), "\n"))
		}

		content = strings.Join(newLines, "\n")
	} else {
		// Append run section at the end
		content = strings.TrimSuffix(content, "\n") + "\n\n" + string(runYAML)
	}

	// #nosec G703 -- configPath validated by safeJoin in InitConfig before being passed here
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("failed to write .ss.yaml: %w", err)
	}

	return nil
}

// createNewConfig creates a new .ss.yaml with run config
func createNewConfig(configPath string, runConfig map[string]interface{}) error {
	config := map[string]interface{}{
		"run": runConfig,
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write .ss.yaml: %w", err)
	}

	return nil
}

// discoverServices finds all go.mod files and returns service info.
func discoverServices(workDir string) ([]ServiceDiscovery, error) {
	var services []ServiceDiscovery

	err := filepath.Walk(workDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip inaccessible paths
		}

		// Skip hidden directories and common non-source directories
		if info.IsDir() {
			base := info.Name()
			if strings.HasPrefix(base, ".") || base == "vendor" || base == "node_modules" || base == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}

		// Look for go.mod files
		if info.Name() == "go.mod" {
			dir := filepath.Dir(path)
			relDir, _ := filepath.Rel(workDir, dir)

			// Skip root go.mod (monorepo root)
			if relDir == "." {
				// Check if there are subdirectories with go.mod
				// If not, this is the only service
				hasSubServices := false
				_ = filepath.Walk(workDir, func(subPath string, subInfo os.FileInfo, _ error) error {
					if subInfo != nil && subInfo.Name() == "go.mod" && subPath != path {
						hasSubServices = true
						return filepath.SkipAll
					}
					return nil
				})

				if hasSubServices {
					return nil // Skip root, will pick up sub-services
				}
			}

			// Skip if no main.go found (library/proto package)
			if !hasMainGo(dir) {
				return nil
			}

			// Determine service name from directory
			name := filepath.Base(dir)
			if relDir == "." {
				name = filepath.Base(workDir)
			}

			// Read module name from go.mod
			moduleName := gomod.ReadModule(dir)

			services = append(services, ServiceDiscovery{
				Name:   name,
				Dir:    "./" + relDir,
				Module: moduleName,
			})
		}

		return nil
	})

	return services, err
}

// hasMainGo checks if directory has a main.go file (directly or in cmd/).
func hasMainGo(dir string) bool {
	// Check for main.go at root
	if _, err := os.Stat(filepath.Join(dir, "main.go")); err == nil {
		return true
	}

	// Check for cmd/ directory with main.go
	cmdDir := filepath.Join(dir, "cmd")
	if info, err := os.Stat(cmdDir); err == nil && info.IsDir() {
		// Check cmd/main.go
		if _, err := os.Stat(filepath.Join(cmdDir, "main.go")); err == nil {
			return true
		}

		// Check cmd/*/main.go (e.g., cmd/server/main.go)
		entries, err := os.ReadDir(cmdDir)
		if err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					if _, err := os.Stat(filepath.Join(cmdDir, entry.Name(), "main.go")); err == nil {
						return true
					}
				}
			}
		}
	}

	return false
}

// safeJoin constructs a path from baseDir and elem, and validates that the result
// is within baseDir to prevent path traversal attacks.
func safeJoin(baseDir, elem string) (string, error) {
	cleanBase := filepath.Clean(baseDir)
	joined := filepath.Join(cleanBase, elem)
	cleanJoined := filepath.Clean(joined)
	if !strings.HasPrefix(cleanJoined, cleanBase+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q escapes base directory %q", elem, baseDir)
	}
	return cleanJoined, nil
}

// buildRunConfig creates the run configuration map.
func buildRunConfig(services []ServiceDiscovery) map[string]interface{} {
	colors := []string{"cyan", "green", "yellow", "blue", "magenta", "red"}

	serviceConfigs := make([]map[string]interface{}, 0, len(services))
	for i, svc := range services {
		svcConfig := map[string]interface{}{
			"name":  svc.Name,
			"dir":   svc.Dir,
			"cmd":   detectRunCommand(svc.Dir),
			"color": colors[i%len(colors)],
			"watch": map[string]interface{}{
				"include": []string{"**/*.go", "**/*.yaml"},
				"exclude": []string{"**/vendor/**", "**/*_test.go", "**/testdata/**"},
			},
		}
		serviceConfigs = append(serviceConfigs, svcConfig)
	}

	return map[string]interface{}{
		"build_delay": "500ms",
		"kill_delay":  "5s",
		"services":    serviceConfigs,
	}
}

// detectRunCommand tries to detect the appropriate run command.
func detectRunCommand(dir string) string {
	// Check for common patterns
	cmdDir := filepath.Join(dir, "cmd")
	if info, err := os.Stat(cmdDir); err == nil && info.IsDir() {
		// Look for main.go in cmd/
		entries, _ := os.ReadDir(cmdDir)
		for _, entry := range entries {
			if entry.IsDir() {
				// cmd/server/, cmd/api/, etc.
				return fmt.Sprintf("go run ./cmd/%s", entry.Name())
			}
		}
		// cmd/main.go
		if _, err := os.Stat(filepath.Join(cmdDir, "main.go")); err == nil {
			return "go run ./cmd"
		}
	}

	// Check for main.go at root
	if _, err := os.Stat(filepath.Join(dir, "main.go")); err == nil {
		return "go run ."
	}

	// Default
	return "go run ./cmd/main.go"
}
