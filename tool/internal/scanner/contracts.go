// Package scanner discovers service contracts and detects service state
// from a directory tree.
package scanner

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// ContractKind classifies a discovered contract file.
type ContractKind string

const (
	ContractKindAPI   ContractKind = "api"   // .api files (Hertz)
	ContractKindProto ContractKind = "proto" // .proto files (Kitex)
	ContractKindSQL   ContractKind = "sql"   // .sql / schema files (SQLC)
)

// Contract represents a discovered contract file.
type Contract struct {
	// Kind is the contract type.
	Kind ContractKind
	// Path is the file path relative to the scanned root.
	Path string
	// AbsPath is the absolute file path.
	AbsPath string
}

// findContracts walks root and returns all recognised contract files.
// It follows symlinks to directories but avoids cycles via a seen-inodes map.
func findContracts(root string) ([]Contract, error) {
	var contracts []Contract
	seen := map[string]bool{}

	err := walkSafe(root, root, seen, func(absPath, relPath string, d fs.DirEntry) error {
		if d.IsDir() {
			return nil
		}
		kind, ok := contractKindForFile(d.Name())
		if !ok {
			return nil
		}
		contracts = append(contracts, Contract{
			Kind:    kind,
			Path:    relPath,
			AbsPath: absPath,
		})
		return nil
	})
	return contracts, err
}

// contractKindForFile returns the ContractKind for filename, or (_, false).
func contractKindForFile(name string) (ContractKind, bool) {
	switch {
	case strings.HasSuffix(name, ".api"):
		return ContractKindAPI, true
	case strings.HasSuffix(name, ".proto"):
		return ContractKindProto, true
	case strings.HasSuffix(name, ".sql"):
		return ContractKindSQL, true
	case name == "schema.sql" || name == "schema.prisma":
		return ContractKindSQL, true
	}
	return "", false
}

// walkSafe is a symlink-aware recursive walk. It records visited real paths to
// prevent infinite loops through circular symlinks.
func walkSafe(root, current string, seen map[string]bool, fn func(abs, rel string, d fs.DirEntry) error) error {
	real, err := filepath.EvalSymlinks(current)
	if err != nil {
		return nil // skip broken symlinks
	}
	if seen[real] {
		return nil // cycle guard
	}
	seen[real] = true

	entries, err := os.ReadDir(current)
	if err != nil {
		return nil // skip unreadable directories
	}

	for _, entry := range entries {
		absChild := filepath.Join(current, entry.Name())
		relChild, _ := filepath.Rel(root, absChild)

		if entry.IsDir() {
			if shouldSkipDir(entry.Name()) {
				continue
			}
			if err := walkSafe(root, absChild, seen, fn); err != nil {
				return err
			}
			continue
		}

		// Handle symlinks to files.
		info := entry
		if entry.Type()&fs.ModeSymlink != 0 {
			realChild, err := filepath.EvalSymlinks(absChild)
			if err != nil {
				continue // broken symlink
			}
			entries2, err2 := os.ReadDir(filepath.Dir(realChild))
			if err2 != nil {
				continue
			}
			for _, e2 := range entries2 {
				if e2.Name() == filepath.Base(realChild) {
					info = e2
					break
				}
			}
		}

		if err := fn(absChild, relChild, info); err != nil {
			return err
		}
	}
	return nil
}

// shouldSkipDir returns true for directories that should not be walked.
func shouldSkipDir(name string) bool {
	skip := map[string]bool{
		".git":         true,
		"vendor":       true,
		"node_modules": true,
		".idea":        true,
		".vscode":      true,
	}
	return skip[name] || strings.HasPrefix(name, ".")
}
