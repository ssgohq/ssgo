package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ssgohq/ssgo/internal/util/log"
	"github.com/ssgohq/ssgo/tool/internal/cmdctx"
	gen "github.com/ssgohq/ssgo/tool/internal/generator/kitex"
)

// runModel generates shared models (kitex_gen) only from a .proto file
func runModel(ctx *cmdctx.Context) error {
	protoFile := ctx.GetFlag("proto")
	if protoFile == "" {
		protoFile = ctx.GetFlag("p")
	}
	if protoFile == "" {
		return fmt.Errorf("--proto or -p flag is required")
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
		return fmt.Errorf("--dir or -o flag is required for model generation")
	}

	genPath := ctx.GetFlag("gen-path")
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

	log.Info("Generating shared model (kitex_gen)...")
	log.Info("  Proto file: %s", absProtoFile)
	log.Info("  Output:     %s", absOutputDir)
	log.Info("  Module:     %s", module)

	// Check prerequisites
	if err := gen.CheckKitexInstalled(); err != nil {
		return err
	}
	if err := gen.CheckProtocInstalled(); err != nil {
		return err
	}

	// Create output directory
	if err := os.MkdirAll(absOutputDir, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Run kitex without -service flag to generate only models
	protoDir := filepath.Dir(absProtoFile)
	includes := []string{}
	if protoDir != "" && protoDir != "." {
		includes = append(includes, protoDir)
		parentDir := filepath.Dir(protoDir)
		if parentDir != "" && parentDir != "." && parentDir != protoDir {
			includes = append(includes, parentDir)
		}
	}

	wrapper := gen.NewKitexWrapper(gen.WrapperOptions{
		ProtoFile: absProtoFile,
		OutputDir: absOutputDir,
		Module:    module,
		Service:   "", // Empty service = generate only models
		Includes:  includes,
		Verbose:   verbose,
		GenPath:   genPath,
	})

	if err := wrapper.RunKitex(); err != nil {
		return fmt.Errorf("kitex generation failed: %w", err)
	}

	genPathName := "kitex_gen"
	if genPath != "" {
		genPathName = genPath
	}

	log.Success("Shared model generation completed!")
	fmt.Println()
	fmt.Printf("Generated structure:\n")
	fmt.Printf("  %s/\n", absOutputDir)
	fmt.Printf("  |-- %s/              # Generated types and interfaces\n", genPathName)
	fmt.Printf("  |   +-- <package>/       # Package from proto\n")
	fmt.Printf("  |       |-- *.pb.go      # Proto message types\n")
	fmt.Printf("  |       +-- <service>/   # Service client/server interfaces\n")
	fmt.Printf("  +-- go.mod\n")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Printf("  1. cd %s && go mod tidy\n", absOutputDir)
	fmt.Println("  2. Use this model in RPC server:")
	fmt.Printf("     ss rpc gen --proto %s --service <ServiceName> \\\n", protoFile)
	fmt.Printf("       -m <rpc_module> --use %s/%s/<package>\n", module, genPathName)

	return nil
}
