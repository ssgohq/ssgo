package cmd

import (
	"fmt"
	"path/filepath"

	ast "github.com/ssgohq/ssgo/internal/ast/api"
	"github.com/ssgohq/ssgo/internal/util/log"
	"github.com/ssgohq/ssgo/tool/internal/cmdctx"
	"github.com/ssgohq/ssgo/tool/internal/generator/openapi"
	spec "github.com/ssgohq/ssgo/tool/internal/spec/api"
)

func runDoc(ctx *cmdctx.Context) error {
	if ctx.GetFlagBool("help") || ctx.GetFlagBool("h") {
		return printDocHelp()
	}

	// Explicit CLI invocation (--api flag provided) — single-file mode
	if ctx.GetFlag("api") != "" || ctx.GetFlag("a") != "" {
		return runDocSingle(ctx)
	}

	// Fall back to .ss.yaml zero-flag batch mode
	cfg, err := LoadApiConfig(ctx.WorkingDir)
	if err != nil {
		return fmt.Errorf("failed to load .ss.yaml: %w", err)
	}
	if cfg.IsEmpty() {
		return printDocHelp()
	}

	// Optional positional arg: filter by api file basename
	serviceFilter := ""
	if len(ctx.Args) > 0 {
		serviceFilter = ctx.Args[0]
	}

	return runDocFromConfig(ctx, cfg, serviceFilter)
}

// runDocSingle generates OpenAPI docs from explicit CLI flags.
func runDocSingle(ctx *cmdctx.Context) error {
	apiFile := ctx.GetFlag("api")
	if apiFile == "" {
		apiFile = ctx.GetFlag("a")
	}

	outputDir := ctx.GetFlag("dir")
	if outputDir == "" {
		outputDir = ctx.GetFlag("o")
	}
	if outputDir == "" {
		outputDir = "docs"
	}

	format := ctx.GetFlag("format")
	if format == "" {
		format = "json"
	}

	return executeDoc(apiFile, outputDir, format)
}

// runDocFromConfig generates OpenAPI docs for services declared in .ss.yaml.
func runDocFromConfig(ctx *cmdctx.Context, cfg *ApiConfig, serviceFilter string) error {
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

		// Use "docs" as output dir for documentation by default
		outputDir := resolved.OutputDir
		// If the user did not pass a dir override, keep old zero-config
		// behavior of defaulting API docs to <workingDir>/docs.
		if svc.Dir == "" && ctx.GetFlag("dir") == "" && ctx.GetFlag("o") == "" {
			outputDir = filepath.Join(ctx.WorkingDir, "docs")
		}

		log.Info("--- Generating doc: %s ---", svc.File)
		if err := executeDoc(resolved.ApiFile, outputDir, resolved.Format); err != nil {
			return fmt.Errorf("api %s: %w", svc.File, err)
		}
	}
	return nil
}

// executeDoc runs the parse→spec→generate flow for OpenAPI documentation.
func executeDoc(apiFile, outputDir, format string) error {
	validFormats := map[string]bool{"json": true, "yaml": true, "yml": true}
	if !validFormats[format] {
		return fmt.Errorf("invalid format '%s', must be one of: json, yaml", format)
	}
	if format == "yml" {
		format = "yaml"
	}

	log.Info("Generating OpenAPI documentation...")
	log.Info("  API file: %s", apiFile)
	log.Info("  Output:   %s", outputDir)
	log.Info("  Format:   %s", format)

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

	gen := openapi.NewGenerator(apiSpec, serviceSpec, outputDir, format)
	if err := gen.Generate(); err != nil {
		return fmt.Errorf("failed to generate OpenAPI spec: %w", err)
	}

	outputFile := filepath.Join(outputDir, "openapi."+format)
	log.Success("Generated OpenAPI specification: %s", outputFile)
	fmt.Println()
	fmt.Println("OpenAPI spec includes:")
	fmt.Println("  • API paths and HTTP methods")
	fmt.Println("  • Request body schemas")
	fmt.Println("  • Response schemas")
	fmt.Println("  • Query/Path parameters")
	fmt.Println("  • Header parameters")
	fmt.Println("  • Security definitions (JWT)")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  • Import into Swagger UI")
	fmt.Println("  • Import into Postman")
	fmt.Println("  • Generate client SDKs")
	fmt.Println("  • API testing tools")

	return nil
}

func printDocHelp() error {
	fmt.Println(`ss api doc - Generate OpenAPI documentation from .api file

Usage:
  ss api doc --api <file> [options]
  ss api doc [file-basename]    (zero-flag mode — reads from .ss.yaml api section)

Options:
  -a, --api <file>      Path to .api file (required when not using .ss.yaml)
  -o, --dir <dir>       Output directory (default: docs)
      --format <fmt>    Output format: json|yaml (default: json)
  -h, --help            Show this help

.ss.yaml api section (zero-flag mode):
  api:
    apis:
      - file: api/user.api
        dir: .
        options:
          format: yaml

Examples:
  ss api doc --api api/user.api
  ss api doc --api api/user.api --format yaml

  # Zero-flag (reads .ss.yaml):
  ss api doc

  # Zero-flag (specific api file from .ss.yaml):
  ss api doc user.api`)
	return nil
}
