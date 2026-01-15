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
	apiFile := ctx.GetFlag("api")
	if apiFile == "" {
		apiFile = ctx.GetFlag("a")
	}
	if apiFile == "" {
		return fmt.Errorf("--api or -a flag is required")
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

	// 1. Parse the .api file
	apiSpec, err := ast.Parse(apiFile)
	if err != nil {
		return fmt.Errorf("failed to parse .api file: %w", err)
	}

	apiSpec, err = ast.ResolveImports(apiSpec)
	if err != nil {
		return fmt.Errorf("failed to resolve imports: %w", err)
	}

	// 2. Convert to spec
	serviceSpec, err := spec.FromAST(apiSpec)
	if err != nil {
		return fmt.Errorf("failed to convert to spec: %w", err)
	}

	// 3. Generate OpenAPI spec
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
