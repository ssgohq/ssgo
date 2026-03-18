package cmd

import (
	"fmt"
	"path/filepath"

	ast "github.com/ssgohq/ssgo/internal/ast/api"
	"github.com/ssgohq/ssgo/internal/util/gomod"
	"github.com/ssgohq/ssgo/internal/util/log"
	"github.com/ssgohq/ssgo/tool/internal/cmdctx"
	"github.com/ssgohq/ssgo/tool/internal/generator/hertz"
	spec "github.com/ssgohq/ssgo/tool/internal/spec/api"
)

func runLogic(ctx *cmdctx.Context) error {
	if ctx.GetFlagBool("help") || ctx.GetFlagBool("h") {
		return printLogicHelp()
	}

	// Explicit CLI invocation (--api flag provided) — single-file mode
	if ctx.GetFlag("api") != "" || ctx.GetFlag("a") != "" {
		return runLogicSingle(ctx)
	}

	// Fall back to .ss.yaml zero-flag batch mode
	cfg, err := LoadApiConfig(ctx.WorkingDir)
	if err != nil {
		return fmt.Errorf("failed to load .ss.yaml: %w", err)
	}
	if cfg.IsEmpty() {
		return printLogicHelp()
	}

	// Optional positional arg: filter by api file basename
	serviceFilter := ""
	if len(ctx.Args) > 0 {
		serviceFilter = ctx.Args[0]
	}

	return runLogicFromConfig(ctx, cfg, serviceFilter)
}

// runLogicSingle generates logic files from explicit CLI flags.
func runLogicSingle(ctx *cmdctx.Context) error {
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

	return executeLogic(apiFile, outputDir, module)
}

// runLogicFromConfig generates logic files for services declared in .ss.yaml.
func runLogicFromConfig(ctx *cmdctx.Context, cfg *ApiConfig, serviceFilter string) error {
	for _, svc := range cfg.Apis {
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

		log.Info("--- Generating logic: %s ---", svc.File)
		if err := executeLogic(resolved.ApiFile, resolved.OutputDir, resolved.Module); err != nil {
			return fmt.Errorf("api %s: %w", svc.File, err)
		}
	}
	return nil
}

// executeLogic runs the parse→spec→generate-logic-only flow for one API service.
func executeLogic(apiFile, outputDir, module string) error {
	log.Info("Generating logic files from API definition...")
	log.Info("  API file: %s", apiFile)
	log.Info("  Output:   %s", outputDir)
	log.Info("  Module:   %s", module)

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
		WithLogic: true,
	})

	if err := generator.GenerateLogicOnly(); err != nil {
		return fmt.Errorf("failed to generate logic: %w", err)
	}

	log.Success("Logic files generated successfully!")
	return nil
}

func printLogicHelp() error {
	fmt.Println(`ss api logic - Generate only logic files from .api file

Usage:
  ss api logic --api <file> [options]
  ss api logic [file-basename]    (zero-flag mode — reads from .ss.yaml api section)

Options:
  -a, --api <file>      Path to .api file (required when not using .ss.yaml)
  -o, --dir <dir>       Output directory (default: .)
  -m, --module <name>   Go module name (auto-detected from go.mod)
  -h, --help            Show this help

.ss.yaml api section (zero-flag mode):
  api:
    apis:
      - file: api/user.api
        dir: .
        options:
          with_logic: true

Examples:
  ss api logic --api api/user.api -m github.com/org/user-api

  # Zero-flag (reads .ss.yaml):
  ss api logic

  # Zero-flag (specific api file from .ss.yaml):
  ss api logic user.api`)
	return nil
}
