package cmd

import (
	"fmt"
	"strconv"

	ast "github.com/ssgohq/ssgo/internal/ast/api"
	"github.com/ssgohq/ssgo/internal/util/gomod"
	"github.com/ssgohq/ssgo/internal/util/log"
	"github.com/ssgohq/ssgo/tool/internal/cmdctx"
	"github.com/ssgohq/ssgo/tool/internal/generator/hertz"
	spec "github.com/ssgohq/ssgo/tool/internal/spec/api"
)

func runGen(ctx *cmdctx.Context) error {
	// Check for help flag
	if ctx.GetFlagBool("help") || ctx.GetFlagBool("h") {
		return printGenHelp()
	}

	apiFile := ctx.GetFlag("api")
	if apiFile == "" {
		apiFile = ctx.GetFlag("a")
	}
	if apiFile == "" {
		return printGenHelp()
	}

	outputDir := ctx.GetFlag("dir")
	if outputDir == "" {
		outputDir = ctx.GetFlag("o")
	}
	if outputDir == "" {
		outputDir = "."
	}

	module := ctx.GetFlag("module")
	if module == "" {
		module = ctx.GetFlag("m")
	}
	if module == "" {
		module = gomod.ReadModule(outputDir)
		if module == "" {
			return fmt.Errorf("--module or -m flag is required")
		}
	}

	withLogic := true
	if ctx.HasFlag("with-logic") {
		withLogic = ctx.GetFlagBool("with-logic")
	}

	port := 8080
	if ctx.HasFlag("port") {
		portStr := ctx.GetFlag("port")
		if portStr != "" {
			if p, err := strconv.Atoi(portStr); err == nil && p > 0 {
				port = p
			}
		}
	}

	log.Info("Generating API server from API definition...")
	log.Info("  API file:   %s", apiFile)
	log.Info("  Output:     %s", outputDir)
	log.Info("  Module:     %s", module)
	log.Info("  With Logic: %v", withLogic)

	// 1. Parse the .api file
	apiSpec, err := ast.Parse(apiFile)
	if err != nil {
		return fmt.Errorf("failed to parse .api file: %w", err)
	}

	// Resolve imports if any
	apiSpec, err = ast.ResolveImports(apiSpec)
	if err != nil {
		return fmt.Errorf("failed to resolve imports: %w", err)
	}

	// 2. Convert to spec
	serviceSpec, err := spec.FromAST(apiSpec)
	if err != nil {
		return fmt.Errorf("failed to convert to spec: %w", err)
	}

	// 3. Generate code
	generator := hertz.NewAPIGenerator(serviceSpec, hertz.APIOptions{
		Options: hertz.Options{
			Output: outputDir,
			Module: module,
		},
		WithLogic: withLogic,
		Port:      port,
	})

	if err := generator.Generate(); err != nil {
		return fmt.Errorf("failed to generate code: %w", err)
	}

	log.Success("API generation completed!")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Printf("  1. cd %s\n", outputDir)
	fmt.Printf("  2. go mod tidy\n")
	fmt.Printf("  3. Implement business logic in internal/logic/\n")
	fmt.Printf("  4. go run cmd/main.go\n")

	return nil
}

func printGenHelp() error {
	fmt.Println(`ss api gen - Generate Hertz code from .api file

Usage:
	 ss api gen --api <file> [options]

Options:
	 -a, --api <file>      Path to .api file (required)
	 -o, --dir <dir>       Output directory (default: .)
	 -m, --module <name>   Go module name (auto-detected from go.mod)
	     --port <number>   Server port (default: 8080)
	     --with-logic      Generate logic files (default: true)
	 -h, --help            Show this help

Examples:
	 ss api gen --api api/user.api
	 ss api gen --api api/user.api -m github.com/org/user-api
	 ss api gen --api api/user.api -o ./output --port 9000`)
	return nil
}
