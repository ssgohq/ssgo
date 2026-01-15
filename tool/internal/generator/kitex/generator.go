package gen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Options represents generator options
type Options struct {
	ProtoFile string   // Path to .proto file
	OutputDir string   // Output directory
	Module    string   // Go module name
	Service   string   // Service name (e.g., UserService)
	Includes  []string // Include paths for proto files
	Verbose   bool     // Verbose mode
	Style     string   // Naming style (default: snake_case)
	UseTypes  string   // Import path for shared types module (optional)
	GenPath   string   // Custom path for generated kitex code (optional)

	// Optional components for generated code
	WithTrace bool // Enable OpenTelemetry tracing config
	WithRedis bool // Add Redis config
}

// Generator generates Kitex RPC server code with ServiceContext pattern
type Generator struct {
	opts         Options
	serviceLower string // lowercase service name
}

// NewGenerator creates a new Generator
func NewGenerator(opts Options) *Generator {
	// Extract service name without "Service" suffix for naming
	serviceLower := strings.ToLower(opts.Service)
	serviceLower = strings.TrimSuffix(serviceLower, "service")

	return &Generator{
		opts:         opts,
		serviceLower: serviceLower,
	}
}

// Generate generates the complete Kitex server code
func (g *Generator) Generate() error {
	fmt.Printf("Generating Kitex RPC server...\n")
	fmt.Printf("  Proto file: %s\n", g.opts.ProtoFile)
	fmt.Printf("  Output:     %s\n", g.opts.OutputDir)
	fmt.Printf("  Module:     %s\n", g.opts.Module)
	fmt.Printf("  Service:    %s\n", g.opts.Service)
	fmt.Println()

	// 1. Check prerequisites
	fmt.Println("Checking prerequisites...")
	if err := g.checkPrerequisites(); err != nil {
		return err
	}

	// 2. Validate proto file exists
	if err := g.validateProtoFile(); err != nil {
		return err
	}

	// 3. Create output directory
	if err := os.MkdirAll(g.opts.OutputDir, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// 4. Run kitex to generate base code
	fmt.Println("Running kitex generator...")
	if err := g.runKitex(); err != nil {
		return fmt.Errorf("kitex generation failed: %w", err)
	}

	// 5. Generate scaffold files with ServiceContext
	fmt.Println("Generating scaffold files...")
	scaffold := NewScaffold(ScaffoldOptions{
		OutputDir:    g.opts.OutputDir,
		Module:       g.opts.Module,
		Service:      g.opts.Service,
		ServiceLower: g.serviceLower,
		Verbose:      g.opts.Verbose,
		UseTypes:     g.opts.UseTypes,
		ProtoFile:    g.opts.ProtoFile,
		// Optional components
		WithTrace: g.opts.WithTrace,
		WithRedis: g.opts.WithRedis,
	})

	if err := scaffold.Generate(); err != nil {
		return fmt.Errorf("scaffold generation failed: %w", err)
	}

	// 6. Print success message
	g.printSuccess()

	return nil
}

// checkPrerequisites checks if required tools are installed
func (g *Generator) checkPrerequisites() error {
	// Check kitex is installed
	if err := CheckKitexInstalled(); err != nil {
		return err
	}
	fmt.Println("  kitex found")

	// Check protoc is installed
	if err := CheckProtocInstalled(); err != nil {
		return err
	}
	fmt.Println("  protoc found")

	return nil
}

// validateProtoFile validates the proto file exists and is readable
func (g *Generator) validateProtoFile() error {
	info, err := os.Stat(g.opts.ProtoFile)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("proto file not found: %s", g.opts.ProtoFile)
		}
		return fmt.Errorf("failed to access proto file: %w", err)
	}

	if info.IsDir() {
		return fmt.Errorf("proto path is a directory, not a file: %s", g.opts.ProtoFile)
	}

	if !strings.HasSuffix(g.opts.ProtoFile, ".proto") {
		return fmt.Errorf("file does not have .proto extension: %s", g.opts.ProtoFile)
	}

	return nil
}

// runKitex runs the kitex command to generate base code
func (g *Generator) runKitex() error {
	// Add proto file directory to includes
	includes := g.opts.Includes
	protoDir := filepath.Dir(g.opts.ProtoFile)
	if protoDir != "" && protoDir != "." {
		includes = append(includes, protoDir)
		// Also add parent directory to support imports like "package/file.proto"
		parentDir := filepath.Dir(protoDir)
		if parentDir != "" && parentDir != "." && parentDir != protoDir {
			includes = append(includes, parentDir)
		}
	}

	wrapper := NewKitexWrapper(WrapperOptions{
		ProtoFile: g.opts.ProtoFile,
		OutputDir: g.opts.OutputDir,
		Module:    g.opts.Module,
		Service:   g.opts.Service,
		Includes:  includes,
		Verbose:   g.opts.Verbose,
		Use:       g.opts.UseTypes,
	})

	return wrapper.RunKitex()
}

// printSuccess prints the success message with generated structure
func (g *Generator) printSuccess() {
	fmt.Println()
	fmt.Println("Code generation completed successfully!")
	fmt.Println()
	fmt.Printf("Generated structure:\n")
	fmt.Printf("  %s/\n", g.opts.OutputDir)
	fmt.Printf("  |-- cmd/\n")
	fmt.Printf("  |   +-- main.go                 # Entry point\n")
	fmt.Printf("  |-- internal/\n")
	fmt.Printf("  |   |-- config/\n")
	fmt.Printf("  |   |   +-- config.go           # Config struct\n")
	fmt.Printf("  |   |-- server/\n")
	fmt.Printf("  |   |   +-- %s_server.go     # Server handler (delegates to logic)\n", g.serviceLower)
	fmt.Printf("  |   |-- logic/\n")
	fmt.Printf("  |   |   +-- *_logic.go          # Individual logic files per RPC\n")
	fmt.Printf("  |   +-- svc/\n")
	fmt.Printf("  |       +-- service_context.go  # ServiceContext\n")
	fmt.Printf("  |-- kitex_gen/                  # Generated by kitex\n")
	fmt.Printf("  |-- etc/\n")
	fmt.Printf("  |   +-- config.yaml             # Configuration file\n")
	fmt.Printf("  +-- go.mod\n")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Printf("  1. cd %s\n", g.opts.OutputDir)
	fmt.Println("  2. go mod tidy")
	fmt.Println("  3. Implement business logic in internal/logic/*_logic.go files")
	fmt.Println("  4. go run cmd/main.go")
	fmt.Println()
	fmt.Println("The generated code compiles and runs immediately with stub implementations.")
}
