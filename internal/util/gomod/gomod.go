// Package gomod provides utilities for working with Go modules.
package gomod

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

// GoModInfo contains information from a go.mod file.
type GoModInfo struct {
	Module    string // Module path
	GoVersion string // Go version (e.g., "1.21")
}

// ReadModule reads the module name from a go.mod file in the given directory.
// Returns empty string if go.mod doesn't exist or can't be parsed.
func ReadModule(dir string) string {
	info, err := Parse(filepath.Join(dir, "go.mod"))
	if err != nil {
		return ""
	}
	return info.Module
}

// Parse parses a go.mod file and returns its information.
func Parse(goModPath string) (*GoModInfo, error) {
	file, err := os.Open(goModPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	info := &GoModInfo{}
	scanner := bufio.NewScanner(file)

	moduleRegex := regexp.MustCompile(`^module\s+(.+)$`)
	goVersionRegex := regexp.MustCompile(`^go\s+(\d+\.\d+)`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if matches := moduleRegex.FindStringSubmatch(line); len(matches) > 1 {
			info.Module = strings.TrimSpace(matches[1])
		}

		if matches := goVersionRegex.FindStringSubmatch(line); len(matches) > 1 {
			info.GoVersion = matches[1]
		}

		// Stop early if we have both
		if info.Module != "" && info.GoVersion != "" {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return info, nil
}

// Exists checks if a go.mod file exists in the given directory.
func Exists(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "go.mod"))
	return err == nil
}

// Init initializes a new Go module in the given directory.
func Init(dir, module string) error {
	cmd := exec.Command("go", "mod", "init", module)
	cmd.Dir = dir
	return cmd.Run()
}

// Tidy runs go mod tidy in the given directory.
func Tidy(dir string) error {
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = dir
	return cmd.Run()
}

// GetGoVersion returns the current Go version.
// Returns format like "1.21" or "1.22".
func GetGoVersion() string {
	// First try runtime version
	version := runtime.Version()

	re := regexp.MustCompile(`go(\d+\.\d+)`)
	matches := re.FindStringSubmatch(version)
	if len(matches) > 1 {
		return matches[1]
	}

	// Fallback: try go version command
	cmd := exec.Command("go", "version")
	out, err := cmd.Output()
	if err == nil {
		matches = re.FindStringSubmatch(string(out))
		if len(matches) > 1 {
			return matches[1]
		}
	}

	// Default fallback
	return "1.21"
}

// FindModuleRoot finds the root directory containing go.mod.
// Searches from startDir up to the filesystem root.
// Returns empty string if no go.mod is found.
func FindModuleRoot(startDir string) string {
	dir := startDir
	for {
		if Exists(dir) {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			return ""
		}
		dir = parent
	}
}

// GetModuleFromDir finds and reads the module name from the go.mod in dir or its parents.
func GetModuleFromDir(dir string) string {
	root := FindModuleRoot(dir)
	if root == "" {
		return ""
	}
	return ReadModule(root)
}
