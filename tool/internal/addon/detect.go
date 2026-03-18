package addon

import (
	"os"
	"path/filepath"
	"strings"
)

// DetectAddons scans dir for addon markers and returns a list of addon names
// that should be applied, in registration order for determinism.
func DetectAddons(dir string) []string {
	var detected []string
	for _, a := range RegisteredAddons {
		if a.Detect(dir) {
			detected = append(detected, a.Name)
		}
	}
	return detected
}

// detectSQLC returns true when a sqlc.yaml/sqlc.yml config or any .sql files
// are present in the directory tree.
func detectSQLC(dir string) bool {
	if fileExists(dir, "sqlc.yaml") || fileExists(dir, "sqlc.yml") {
		return true
	}
	return hasSQLFiles(dir)
}

// detectRedis returns true when a redis config reference is found.
// Heuristic: any file named redis.yaml/redis.yml or Go files referencing redis.
func detectRedis(dir string) bool {
	if fileExists(dir, "etc/redis.yaml") || fileExists(dir, "etc/redis.yml") {
		return true
	}
	return fileExists(dir, "redis.yaml") || fileExists(dir, "redis.yml")
}

// detectTracing returns true when otel/jaeger/zipkin config markers are present.
func detectTracing(dir string) bool {
	for _, name := range []string{"otel.yaml", "otel.yml", "jaeger.yaml", "jaeger.yml", "tracing.yaml", "tracing.yml"} {
		if fileExists(dir, name) || fileExists(dir, "etc/"+name) {
			return true
		}
	}
	return false
}

// hasSQLFiles returns true when any .sql file exists in the directory tree
// (excluding vendor and hidden directories).
func hasSQLFiles(dir string) bool {
	found := false
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || found {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if name == "vendor" || name == "node_modules" || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(d.Name(), ".sql") {
			found = true
		}
		return nil
	})
	return found
}

// fileExists returns true when a file at relPath exists under dir.
func fileExists(dir, relPath string) bool {
	info, err := os.Stat(filepath.Join(dir, relPath))
	return err == nil && !info.IsDir()
}
