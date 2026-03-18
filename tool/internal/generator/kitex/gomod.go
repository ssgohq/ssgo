package gen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ReadModuleFromGoMod reads the Go module name from a go.mod file in the given directory.
// Returns empty string if go.mod doesn't exist or can't be parsed.
func ReadModuleFromGoMod(dir string) (string, error) {
	gomodPath := filepath.Join(dir, "go.mod")
	data, err := os.ReadFile(gomodPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil // No go.mod, not an error
		}
		return "", fmt.Errorf("failed to read go.mod: %w", err)
	}

	// Parse module line: "module github.com/org/repo"
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			module := strings.TrimPrefix(line, "module ")
			module = strings.TrimSpace(module)
			return module, nil
		}
	}

	return "", fmt.Errorf("no module directive found in %s", gomodPath)
}

// GoPackageToImportPath extracts the Go import path from a proto go_package option.
// For full-path formats like "github.com/org/proto/kitex_gen/pkg/v1;alias",
// it returns the path part: "github.com/org/proto/kitex_gen/pkg/v1".
// For short formats like "user" or "lineartdms", it returns empty string.
func GoPackageToImportPath(rawGoPackage string) string {
	if rawGoPackage == "" {
		return ""
	}

	// Strip alias after ";"
	path := rawGoPackage
	if idx := strings.Index(path, ";"); idx > 0 {
		path = path[:idx]
	}

	// Only return if it's a full import path (contains "/")
	if !strings.Contains(path, "/") {
		return ""
	}

	return path
}
