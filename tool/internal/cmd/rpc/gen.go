package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ssgohq/ssgo/internal/util/log"
	"github.com/ssgohq/ssgo/tool/internal/cmdctx"
	gen "github.com/ssgohq/ssgo/tool/internal/generator/kitex"
)

// runGen generates a full Kitex RPC server from a .proto file
func runGen(ctx *cmdctx.Context) error {
	// Check for help flag
	if ctx.GetFlagBool("help") || ctx.GetFlagBool("h") {
		return printGenHelp()
	}

	protoFile := ctx.GetFlag("proto")
	if protoFile == "" {
		protoFile = ctx.GetFlag("p")
	}
	if protoFile == "" {
		return printGenHelp()
	}

	service := ctx.GetFlag("service")
	if service == "" {
		return fmt.Errorf("--service flag is required")
	}

	module := ctx.GetFlag("module")
	if module == "" {
		module = ctx.GetFlag("m")
	}
	if module == "" {
		return fmt.Errorf("--module or -m flag is required")
	}

	outputDir := ctx.GetFlag("dir")
	if outputDir == "" {
		outputDir = ctx.GetFlag("o")
	}
	if outputDir == "" {
		outputDir = "."
	}

	useTypes := ctx.GetFlag("use")
	genPath := ctx.GetFlag("gen-path")
	withTrace := ctx.GetFlagBool("with-trace")
	withRedis := ctx.GetFlagBool("with-redis")
	verbose := ctx.Debug

	// Convert to absolute paths
	absOutputDir, err := filepath.Abs(outputDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	absProtoFile, err := filepath.Abs(protoFile)
	if err != nil {
		return fmt.Errorf("failed to get absolute proto path: %w", err)
	}

	// Validate service name
	if !strings.HasSuffix(service, "Service") {
		log.Warning("Service name '%s' doesn't end with 'Service'. This is a convention in Kitex.", service)
	}

	log.Info("Generating RPC server from Proto definition...")
	log.Info("  Proto file: %s", absProtoFile)
	log.Info("  Output:     %s", absOutputDir)
	log.Info("  Module:     %s", module)
	log.Info("  Service:    %s", service)

	// Create generator
	g := gen.NewGenerator(gen.Options{
		ProtoFile: absProtoFile,
		OutputDir: absOutputDir,
		Module:    module,
		Service:   service,
		Verbose:   verbose,
		UseTypes:  useTypes,
		GenPath:   genPath,
		WithTrace: withTrace,
		WithRedis: withRedis,
	})

	// Run generation
	if err := g.Generate(); err != nil {
		return fmt.Errorf("generation failed: %w", err)
	}

	log.Success("RPC generation completed!")
	return nil
}

func printGenHelp() error {
	fmt.Println(`ss rpc gen - Generate Kitex RPC server from .proto file

Usage:
	 ss rpc gen --proto <file> --service <name> -m <module> [options]

Options:
	 -p, --proto <file>     Path to .proto file (required)
	     --service <name>   Service name in proto file (required)
	 -m, --module <name>    Go module name (required)
	 -o, --dir <dir>        Output directory (default: .)
	     --use <module>     Use external types module (kitex_gen)
	     --gen-path <path>  Path for generated kitex_gen
	     --with-trace       Enable OpenTelemetry tracing
	     --with-redis       Enable Redis integration
	 -h, --help             Show this help

Examples:
	 ss rpc gen --proto idl/user.proto --service UserService -m github.com/org/user-rpc
	 ss rpc gen --proto idl/user.proto --service UserService -m github.com/org/user-rpc --use github.com/org/common-pb
	 ss rpc gen --proto idl/user.proto --service UserService -m github.com/org/user-rpc --with-trace`)
	return nil
}
