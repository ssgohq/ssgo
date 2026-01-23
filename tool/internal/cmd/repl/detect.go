package repl

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// GrpcService represents a detected gRPC service
type GrpcService struct {
	Name        string   // service name from .ss.yaml
	Dir         string   // absolute path to service directory
	ProtoFiles  []string // detected proto files (absolute paths)
	ImportPaths []string // import paths for evans
	Address     string   // host:port from etc/config.yaml
	UseModules  []string // shared proto module paths
}

// DetectServices scans .ss.yaml and detects gRPC services
func DetectServices(workDir string) ([]GrpcService, error) {
	configPath := filepath.Join(workDir, ".ss.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var rawConfig struct {
		Run struct {
			Services []struct {
				Name string   `yaml:"name"`
				Dir  string   `yaml:"dir"`
				Use  []string `yaml:"use"`
			} `yaml:"services"`
		} `yaml:"run"`
	}

	if err := yaml.Unmarshal(data, &rawConfig); err != nil {
		return nil, err
	}

	var services []GrpcService
	for _, svc := range rawConfig.Run.Services {
		// Resolve service directory
		svcDir := svc.Dir
		if !filepath.IsAbs(svcDir) {
			svcDir = filepath.Join(workDir, svc.Dir)
		}

		// Detect proto files - first try service dir, then workspace root
		protos, importPaths := DetectProtos(svcDir, svc.Use)
		if len(protos) == 0 {
			// Fallback: try workspace root for protos
			protos, importPaths = DetectProtos(workDir, svc.Use)
		}
		if len(protos) == 0 {
			continue // Skip services without proto files
		}

		// Detect address - try service dir first
		address, _ := DetectAddress(svcDir)
		if address == "" {
			address = "localhost:8888"
		}

		services = append(services, GrpcService{
			Name:        svc.Name,
			Dir:         svcDir,
			ProtoFiles:  protos,
			ImportPaths: importPaths,
			Address:     address,
			UseModules:  svc.Use,
		})
	}

	return services, nil
}

// DetectProtos finds proto files for a service
func DetectProtos(serviceDir string, useModules []string) ([]string, []string) {
	var protos, importPaths []string
	protoDirs := []string{"idl", "proto", "protos", "api"}

	// 1. Local protos
	for _, dir := range protoDirs {
		path := filepath.Join(serviceDir, dir)
		if files := findProtoFiles(path); len(files) > 0 {
			protos = append(protos, files...)
			importPaths = append(importPaths, path)
			// Also add subdirectories as import paths
			importPaths = append(importPaths, extractImportPaths(files)...)
		}
	}

	// 2. Shared modules (from --use)
	for _, usePath := range useModules {
		absPath := resolveUsePath(serviceDir, usePath)
		for _, dir := range protoDirs {
			path := filepath.Join(absPath, dir)
			if files := findProtoFiles(path); len(files) > 0 {
				protos = append(protos, files...)
				importPaths = append(importPaths, path)
				importPaths = append(importPaths, extractImportPaths(files)...)
			}
		}
	}

	// 3. Auto-detect from go.mod replace directives (fallback)
	if len(useModules) == 0 {
		replaces := parseGoModReplaces(serviceDir)
		for _, replacePath := range replaces {
			absPath := resolveUsePath(serviceDir, replacePath)
			for _, dir := range protoDirs {
				path := filepath.Join(absPath, dir)
				if files := findProtoFiles(path); len(files) > 0 {
					protos = append(protos, files...)
					importPaths = append(importPaths, path)
					importPaths = append(importPaths, extractImportPaths(files)...)
				}
			}
		}
	}

	return unique(protos), unique(importPaths)
}

// extractImportPaths extracts import paths needed for proto files
// For file at /path/to/proto/package/file.proto with import "package/other.proto",
// we need /path/to/proto as import path
func extractImportPaths(protoFiles []string) []string {
	seen := make(map[string]bool)
	var paths []string
	for _, f := range protoFiles {
		dir := filepath.Dir(f)
		if !seen[dir] {
			seen[dir] = true
			paths = append(paths, dir)
		}
		// Also add parent directory (for imports like "package/file.proto")
		parent := filepath.Dir(dir)
		if !seen[parent] {
			seen[parent] = true
			paths = append(paths, parent)
		}
	}
	return paths
}

// DetectAddress reads address from etc/config.yaml
func DetectAddress(serviceDir string) (string, error) {
	configPaths := []string{
		filepath.Join(serviceDir, "etc", "config.yaml"),
		filepath.Join(serviceDir, "etc", "config.yml"),
		filepath.Join(serviceDir, "config.yaml"),
	}

	for _, configPath := range configPaths {
		data, err := os.ReadFile(configPath)
		if err != nil {
			continue
		}

		var config struct {
			Host string `yaml:"host"`
			Port int    `yaml:"port"`
		}
		if err := yaml.Unmarshal(data, &config); err != nil {
			continue
		}

		host := config.Host
		if host == "" || host == "0.0.0.0" {
			host = "localhost"
		}

		if config.Port > 0 {
			return host + ":" + itoa(config.Port), nil
		}
	}

	return "", nil
}

// findProtoFiles finds all .proto files in a directory (including subdirectories)
func findProtoFiles(dir string) []string {
	var protos []string

	// Direct files
	if files, _ := filepath.Glob(filepath.Join(dir, "*.proto")); len(files) > 0 {
		protos = append(protos, files...)
	}

	// Subdirectories (one level deep)
	if files, _ := filepath.Glob(filepath.Join(dir, "*", "*.proto")); len(files) > 0 {
		protos = append(protos, files...)
	}

	// Two levels deep (common pattern: proto/package/file.proto)
	if files, _ := filepath.Glob(filepath.Join(dir, "*", "*", "*.proto")); len(files) > 0 {
		protos = append(protos, files...)
	}

	return protos
}

// resolveUsePath resolves a use path relative to service directory
func resolveUsePath(serviceDir, usePath string) string {
	if filepath.IsAbs(usePath) {
		return usePath
	}
	return filepath.Join(serviceDir, usePath)
}

// parseGoModReplaces parses replace directives from go.mod
func parseGoModReplaces(serviceDir string) []string {
	goModPath := filepath.Join(serviceDir, "go.mod")
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return nil
	}

	var replaces []string
	lines := strings.Split(string(data), "\n")
	inReplace := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if line == "replace (" {
			inReplace = true
			continue
		}
		if line == ")" {
			inReplace = false
			continue
		}

		if inReplace || strings.HasPrefix(line, "replace ") {
			// Parse: module => ../path
			parts := strings.Split(line, "=>")
			if len(parts) == 2 {
				path := strings.TrimSpace(parts[1])
				// Only local paths
				if strings.HasPrefix(path, ".") || strings.HasPrefix(path, "/") {
					// Remove version suffix if present
					if idx := strings.Index(path, " "); idx != -1 {
						path = path[:idx]
					}
					replaces = append(replaces, path)
				}
			}
		}
	}

	return replaces
}

// unique returns unique strings from a slice
func unique(slice []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, s := range slice {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}

// itoa converts int to string without importing strconv
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
