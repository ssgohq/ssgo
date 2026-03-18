// Package sqlc provides code generation for SQLC-based database layers
package sqlc

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// safeWriteFile writes data to path only if path is contained within baseDir.
// This prevents path traversal attacks when baseDir is derived from external input.
func safeWriteFile(baseDir, path string, data []byte, perm os.FileMode) error {
	cleanBase := filepath.Clean(baseDir) + string(filepath.Separator)
	cleanPath := filepath.Clean(path)

	if !strings.HasPrefix(cleanPath, cleanBase) {
		return fmt.Errorf("path %q is outside base directory %q", path, baseDir)
	}

	return os.WriteFile(cleanPath, data, perm) // #nosec G703 -- path is validated against baseDir above
}
