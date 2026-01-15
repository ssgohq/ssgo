package cmd

import (
	"fmt"

	ast "github.com/ssgohq/ssgo/internal/ast/api"
	"github.com/ssgohq/ssgo/internal/util/gomod"
	"github.com/ssgohq/ssgo/internal/util/log"
	"github.com/ssgohq/ssgo/tool/internal/cmdctx"
	"github.com/ssgohq/ssgo/tool/internal/generator/hertz"
	spec "github.com/ssgohq/ssgo/tool/internal/spec/api"
)

func runLogic(ctx *cmdctx.Context) error {
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

	log.Info("Generating logic files from API definition...")
	log.Info("  API file: %s", apiFile)
	log.Info("  Output:   %s", outputDir)
	log.Info("  Module:   %s", module)

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

	// 3. Generate only logic files
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
