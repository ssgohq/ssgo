// Package gen provides code generation for Kitex RPC server from .proto files
package gen

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// KitexWrapper wraps the kitex command to generate base code
type KitexWrapper struct {
	protoFile string   // Path to .proto file
	outputDir string   // Output directory
	module    string   // Go module name
	service   string   // Service name
	includes  []string // Include paths for proto files
	verbose   bool     // Verbose mode
	use       string   // Import path for shared types module (kitex -use flag)
	genPath   string   // Custom path for generated code (kitex -gen-path flag)
}

// WrapperOptions represents options for KitexWrapper
type WrapperOptions struct {
	ProtoFile string
	OutputDir string
	Module    string
	Service   string
	Includes  []string
	Verbose   bool
	Use       string // Import path for shared types module
	GenPath   string // Custom path for generated code
}

// NewKitexWrapper creates a new KitexWrapper
func NewKitexWrapper(opts WrapperOptions) *KitexWrapper {
	return &KitexWrapper{
		protoFile: opts.ProtoFile,
		outputDir: opts.OutputDir,
		module:    opts.Module,
		service:   opts.Service,
		includes:  opts.Includes,
		verbose:   opts.Verbose,
		use:       opts.Use,
		genPath:   opts.GenPath,
	}
}

// CheckKitexInstalled verifies kitex is available in PATH
func CheckKitexInstalled() error {
	_, err := exec.LookPath("kitex")
	if err != nil {
		return fmt.Errorf("kitex command not found in PATH. Please install it first:\n" +
			"  go install github.com/cloudwego/kitex/tool/cmd/kitex@latest")
	}
	return nil
}

// CheckProtocInstalled verifies protoc is available in PATH
func CheckProtocInstalled() error {
	_, err := exec.LookPath("protoc")
	if err != nil {
		return fmt.Errorf("protoc command not found in PATH. Please install it first:\n" +
			"  https://grpc.io/docs/protoc-installation/")
	}
	return nil
}

// RunKitex calls kitex command to generate base code
func (w *KitexWrapper) RunKitex() error {
	// Ensure output directory exists
	if err := os.MkdirAll(w.outputDir, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Build kitex command arguments
	args := []string{
		"-module", w.module,
	}

	// Add service name if specified
	if w.service != "" {
		args = append(args, "-service", w.service)
	}

	// Add -use flag for shared types module
	if w.use != "" {
		args = append(args, "-use", w.use)
	}

	// Add -gen-path flag for custom generated code path
	if w.genPath != "" {
		args = append(args, "-gen-path", w.genPath)
	}

	// Add include paths
	for _, inc := range w.includes {
		args = append(args, "-I", inc)
	}

	// Add verbose flag
	if w.verbose {
		args = append(args, "-v")
	}

	// Add proto file (must be last)
	protoPath, err := filepath.Abs(w.protoFile)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for proto file: %w", err)
	}
	args = append(args, protoPath)

	// Create command
	cmd := exec.Command("kitex", args...)
	cmd.Dir = w.outputDir

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if w.verbose {
		fmt.Printf("Running: kitex %s\n", strings.Join(args, " "))
		fmt.Printf("Working directory: %s\n", w.outputDir)
	}

	// Run command
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("kitex command failed: %w\nstdout: %s\nstderr: %s",
			err, stdout.String(), stderr.String())
	}

	if w.verbose {
		if stdout.Len() > 0 {
			fmt.Printf("kitex stdout:\n%s\n", stdout.String())
		}
		if stderr.Len() > 0 {
			fmt.Printf("kitex stderr:\n%s\n", stderr.String())
		}
	}

	// Clean up kitex's auto-generated handler.go and main.go files
	// We generate our own versions in the scaffold
	w.cleanupKitexScaffold()

	return nil
}

// cleanupKitexScaffold removes kitex's auto-generated scaffold files
// that we replace with our own templates
func (w *KitexWrapper) cleanupKitexScaffold() {
	// Files that kitex generates when -service flag is used
	filesToRemove := []string{
		filepath.Join(w.outputDir, "handler.go"),
		filepath.Join(w.outputDir, "main.go"),
		filepath.Join(w.outputDir, "go.mod"), // Remove kitex's go.mod, we generate our own with dependencies
	}

	for _, f := range filesToRemove {
		if _, err := os.Stat(f); err == nil {
			os.Remove(f)
			if w.verbose {
				fmt.Printf("Cleaned up kitex scaffold: %s\n", filepath.Base(f))
			}
		}
	}
}

// GetKitexGenPath returns the path to kitex_gen directory
func (w *KitexWrapper) GetKitexGenPath() string {
	genPath := "kitex_gen"
	if w.genPath != "" {
		genPath = w.genPath
	}
	return filepath.Join(w.outputDir, genPath)
}

// GetProtoDir returns the directory containing the proto file
func (w *KitexWrapper) GetProtoDir() string {
	return filepath.Dir(w.protoFile)
}
