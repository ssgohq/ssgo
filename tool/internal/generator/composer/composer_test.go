package composer_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ssgohq/ssgo/tool/internal/generator/composer"
)

// --- Options.Validate() ---

func TestValidate_MissingOutputDir(t *testing.T) {
	opts := composer.Options{
		Module:  "github.com/test/svc",
		WithAPI: true,
	}
	if err := opts.Validate(); err == nil {
		t.Fatal("expected error for missing OutputDir")
	}
}

func TestValidate_MissingModule(t *testing.T) {
	opts := composer.Options{
		OutputDir: "/tmp/svc",
		WithAPI:   true,
	}
	if err := opts.Validate(); err == nil {
		t.Fatal("expected error for missing Module")
	}
}

func TestValidate_NoTransports(t *testing.T) {
	opts := composer.Options{
		OutputDir: "/tmp/svc",
		Module:    "github.com/test/svc",
		WithAPI:   false,
		WithRPC:   false,
	}
	if err := opts.Validate(); err == nil {
		t.Fatal("expected error when no transports are enabled")
	}
}

func TestValidate_Valid_APIOnly(t *testing.T) {
	opts := composer.Options{
		OutputDir: "/tmp/svc",
		Module:    "github.com/test/svc",
		WithAPI:   true,
	}
	if err := opts.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_Valid_RPCOnly(t *testing.T) {
	opts := composer.Options{
		OutputDir: "/tmp/svc",
		Module:    "github.com/test/svc",
		WithRPC:   true,
	}
	if err := opts.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_Valid_Both(t *testing.T) {
	opts := composer.Options{
		OutputDir: "/tmp/svc",
		Module:    "github.com/test/svc",
		WithAPI:   true,
		WithRPC:   true,
	}
	if err := opts.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- Generate() ---

func TestGenerate_CreatesServerMainGo(t *testing.T) {
	dir := t.TempDir()
	opts := composer.Options{
		OutputDir:   dir,
		Module:      "github.com/test/my-service",
		ServiceName: "MyService",
		WithAPI:     true,
		WithRPC:     true,
	}

	gen, err := composer.New(opts)
	if err != nil {
		t.Fatalf("New(): %v", err)
	}

	if err := gen.Generate(); err != nil {
		t.Fatalf("Generate(): %v", err)
	}

	mainPath := filepath.Join(dir, "cmd", "server", "main.go")
	if _, err := os.Stat(mainPath); os.IsNotExist(err) {
		t.Fatalf("expected cmd/server/main.go to be created")
	}
}

func TestGenerate_ServerMainContainsModule(t *testing.T) {
	dir := t.TempDir()
	opts := composer.Options{
		OutputDir: dir,
		Module:    "github.com/acme/hello-svc",
		WithRPC:   true,
	}

	gen, err := composer.New(opts)
	if err != nil {
		t.Fatalf("New(): %v", err)
	}
	if err := gen.Generate(); err != nil {
		t.Fatalf("Generate(): %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "cmd", "server", "main.go"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(data), "github.com/acme/hello-svc") {
		t.Error("expected module path in cmd/server/main.go")
	}
}

func TestGenerate_APIOnly_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	opts := composer.Options{
		OutputDir: dir,
		Module:    "github.com/test/api-svc",
		WithAPI:   true,
		WithRPC:   false,
	}

	gen, err := composer.New(opts)
	if err != nil {
		t.Fatalf("New(): %v", err)
	}
	if err := gen.Generate(); err != nil {
		t.Fatalf("Generate(): %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "cmd", "server", "main.go")); os.IsNotExist(err) {
		t.Fatal("cmd/server/main.go not created for API-only single binary")
	}
}

func TestGenerate_Idempotent(t *testing.T) {
	dir := t.TempDir()
	opts := composer.Options{
		OutputDir: dir,
		Module:    "github.com/test/svc",
		WithAPI:   true,
		WithRPC:   true,
	}

	gen, err := composer.New(opts)
	if err != nil {
		t.Fatalf("New(): %v", err)
	}

	if err := gen.Generate(); err != nil {
		t.Fatalf("first Generate(): %v", err)
	}
	if err := gen.Generate(); err != nil {
		t.Fatalf("second Generate(): %v", err)
	}
}

func TestGenerate_CreatesSharedBaseFiles(t *testing.T) {
	dir := t.TempDir()
	opts := composer.Options{
		OutputDir: dir,
		Module:    "github.com/test/base-svc",
		WithAPI:   true,
	}

	gen, err := composer.New(opts)
	if err != nil {
		t.Fatalf("New(): %v", err)
	}
	if err := gen.Generate(); err != nil {
		t.Fatalf("Generate(): %v", err)
	}

	// Shared scaffold files must exist.
	for _, rel := range []string{"internal/config/base.go", "internal/svc/base.go"} {
		if _, err := os.Stat(filepath.Join(dir, rel)); os.IsNotExist(err) {
			t.Errorf("expected shared file to be created: %s", rel)
		}
	}
}
