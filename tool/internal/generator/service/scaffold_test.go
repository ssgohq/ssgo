package service

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSharedScaffold_Generate(t *testing.T) {
	dir := t.TempDir()
	opts := ScaffoldOptions{
		OutputDir: dir,
		Module:    "github.com/test/my-service",
		Verbose:   false,
	}

	s := NewSharedScaffold(opts)
	if err := s.Generate(); err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	// All expected files must be created.
	expected := []string{
		"internal/config/base.go",
		"internal/svc/base.go",
		"go.mod",
	}
	for _, rel := range expected {
		path := filepath.Join(dir, rel)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file not created: %s", rel)
		}
	}
}

func TestSharedScaffold_SkipExisting(t *testing.T) {
	dir := t.TempDir()

	// Pre-create a file that should be skipped.
	configDir := filepath.Join(dir, "internal/config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	baseConfigPath := filepath.Join(configDir, "base.go")
	sentinel := []byte("// user modified content\npackage config\n")
	if err := os.WriteFile(baseConfigPath, sentinel, 0o644); err != nil {
		t.Fatal(err)
	}

	opts := ScaffoldOptions{OutputDir: dir, Module: "github.com/test/svc"}
	s := NewSharedScaffold(opts)
	if err := s.Generate(); err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	// Pre-existing file must not be overwritten.
	got, err := os.ReadFile(baseConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(sentinel) {
		t.Errorf("skip-if-exists violated: file was overwritten")
	}
}

func TestSharedScaffold_Idempotent(t *testing.T) {
	dir := t.TempDir()
	opts := ScaffoldOptions{OutputDir: dir, Module: "github.com/test/svc"}
	s := NewSharedScaffold(opts)

	// Run twice — should not fail.
	if err := s.Generate(); err != nil {
		t.Fatalf("first Generate() error: %v", err)
	}
	if err := s.Generate(); err != nil {
		t.Fatalf("second Generate() error: %v", err)
	}
}

func TestSharedScaffold_Validate(t *testing.T) {
	cases := []struct {
		name    string
		opts    ScaffoldOptions
		wantErr bool
	}{
		{"valid", ScaffoldOptions{OutputDir: "/tmp", Module: "github.com/x/y"}, false},
		{"missing output", ScaffoldOptions{Module: "github.com/x/y"}, true},
		{"missing module", ScaffoldOptions{OutputDir: "/tmp"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.opts.Validate()
			if tc.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestOwnership_NoConflict(t *testing.T) {
	// API-owned files must not be writable by RPC generator and vice versa.
	apiFiles := []string{"cmd/api/main.go", "internal/config/api.go", "internal/svc/api.go", "etc/api.yaml"}
	rpcFiles := []string{"cmd/rpc/main.go", "internal/config/rpc.go", "internal/svc/rpc.go", "etc/rpc.yaml"}

	for _, f := range apiFiles {
		if CanWrite(f, OwnerRPC) {
			t.Errorf("RPC should not be able to write API-owned file: %s", f)
		}
		if !CanWrite(f, OwnerAPI) {
			t.Errorf("API should be able to write API-owned file: %s", f)
		}
	}

	for _, f := range rpcFiles {
		if CanWrite(f, OwnerAPI) {
			t.Errorf("API should not be able to write RPC-owned file: %s", f)
		}
		if !CanWrite(f, OwnerRPC) {
			t.Errorf("RPC should be able to write RPC-owned file: %s", f)
		}
	}
}

func TestOwnership_SharedFiles(t *testing.T) {
	sharedFiles := []string{"go.mod", ".gitignore", "internal/config/base.go", "internal/svc/base.go"}
	for _, f := range sharedFiles {
		if !CanWrite(f, OwnerShared) {
			t.Errorf("Shared owner should be able to write: %s", f)
		}
		if !CanWrite(f, OwnerAPI) {
			t.Errorf("API should be able to write shared file: %s", f)
		}
		if !CanWrite(f, OwnerRPC) {
			t.Errorf("RPC should be able to write shared file: %s", f)
		}
	}
}
