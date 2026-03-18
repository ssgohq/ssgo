package cmd

import (
	"fmt"
	"path/filepath"
	"strconv"

	ast "github.com/ssgohq/ssgo/internal/ast/api"
	"github.com/ssgohq/ssgo/internal/util/gomod"
	"github.com/ssgohq/ssgo/internal/util/log"
	"github.com/ssgohq/ssgo/tool/internal/cmdctx"
	"github.com/ssgohq/ssgo/tool/internal/generator/hertz"
	spec "github.com/ssgohq/ssgo/tool/internal/spec/api"
)

func runGen(ctx *cmdctx.Context) error {
	if ctx.GetFlagBool("help") || ctx.GetFlagBool("h") {
		return printGenHelp()
	}

	// Explicit CLI invocation (--api flag provided) — single-file mode
	if ctx.GetFlag("api") != "" || ctx.GetFlag("a") != "" {
		return runGenSingle(ctx)
	}

	// Fall back to .ss.yaml zero-flag batch mode
	cfg, err := LoadApiConfig(ctx.WorkingDir)
	if err != nil {
		return fmt.Errorf("failed to load .ss.yaml: %w", err)
	}
	if cfg.IsEmpty() {
		return printGenHelp()
	}

	// Optional positional arg: filter by api file basename
	serviceFilter := ""
	if len(ctx.Args) > 0 {
		serviceFilter = ctx.Args[0]
	}

	return runGenFromConfig(ctx, cfg, serviceFilter)
}

// runGenSingle generates a single API server from explicit CLI flags.
func runGenSingle(ctx *cmdctx.Context) error {
	apiFile := ctx.GetFlag("api")
	if apiFile == "" {
		apiFile = ctx.GetFlag("a")
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

	return executeGen(apiFile, outputDir, module, port, withLogic)
}

// runGenFromConfig generates API servers for services declared in .ss.yaml.
func runGenFromConfig(ctx *cmdctx.Context, cfg *ApiConfig, serviceFilter string) error {
	for _, svc := range cfg.Apis {
		// Apply optional filter by api file basename
		if serviceFilter != "" {
			base := filepath.Base(svc.File)
			if base != serviceFilter && svc.File != serviceFilter {
				continue
			}
		}

		resolved, err := ResolveApiConfig(ctx, svc)
		if err != nil {
			return fmt.Errorf("api %s: %w", svc.File, err)
		}

		log.Info("--- Generating API: %s ---", svc.File)
		if err := executeGen(
			resolved.ApiFile,
			resolved.OutputDir,
			resolved.Module,
			resolved.Port,
			resolved.WithLogic,
		); err != nil {
			return fmt.Errorf("api %s: %w", svc.File, err)
		}
	}
	return nil
}

// executeGen runs the parse→spec→generate flow for one API service.
func executeGen(apiFile, outputDir, module string, port int, withLogic bool) error {
	log.Info("Generating API server from API definition...")
	log.Info("  API file:   %s", apiFile)
	log.Info("  Output:     %s", outputDir)
	log.Info("  Module:     %s", module)
	log.Info("  With Logic: %v", withLogic)

	apiSpec, err := ast.Parse(apiFile)
	if err != nil {
		return fmt.Errorf("failed to parse .api file: %w", err)
	}

	apiSpec, err = ast.ResolveImports(apiSpec)
	if err != nil {
		return fmt.Errorf("failed to resolve imports: %w", err)
	}

	serviceSpec, err := spec.FromAST(apiSpec)
	if err != nil {
		return fmt.Errorf("failed to convert to spec: %w", err)
	}

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
  ss api gen [file-basename]    (zero-flag mode — reads from .ss.yaml api section)

Options:
  -a, --api <file>      Path to .api file (required when not using .ss.yaml)
  -o, --dir <dir>       Output directory (default: .)
  -m, --module <name>   Go module name (auto-detected from go.mod)
      --port <number>   Server port (default: 8080)
      --with-logic      Generate logic files (default: true)
  -h, --help            Show this help

.ss.yaml api section (zero-flag mode):
  api:
    apis:
      - file: api/user.api
        dir: .
        options:
          port: 8080
          with_logic: true
          format: json

Examples:
  ss api gen --api api/user.api
  ss api gen --api api/user.api -m github.com/org/user-api
  ss api gen --api api/user.api -o ./output --port 9000

  # Zero-flag (reads .ss.yaml):
  ss api gen

  # Zero-flag (specific api file from .ss.yaml):
  ss api gen user.api`)
	return nil
}
