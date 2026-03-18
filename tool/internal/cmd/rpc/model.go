package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ssgohq/ssgo/internal/util/log"
	"github.com/ssgohq/ssgo/tool/internal/cmdctx"
	gen "github.com/ssgohq/ssgo/tool/internal/generator/kitex"
)

// runModel generates shared models (kitex_gen) only from a .proto file.
//
// When no --proto flag is provided, runModel falls back to the rpc: section of
// .ss.yaml and generates models for all protos in proto_module.
func runModel(ctx *cmdctx.Context) error {
	// Explicit CLI invocation (--proto provided)
	if ctx.GetFlag("proto") != "" || ctx.GetFlag("p") != "" {
		return runModelSingle(ctx)
	}

	// Fall back to .ss.yaml batch mode
	cfg, err := LoadRpcConfig(ctx.WorkingDir)
	if err != nil {
		return fmt.Errorf("failed to load .ss.yaml: %w", err)
	}
	if cfg.IsEmpty() {
		return fmt.Errorf("--proto or -p flag is required (or configure rpc.proto_module in .ss.yaml)")
	}

	return runModelFromConfig(ctx, cfg)
}

// runModelSingle generates shared models from explicit CLI flags.
func runModelSingle(ctx *cmdctx.Context) error {
	opts, err := resolveModelOptions(ctx)
	if err != nil {
		return err
	}
	return executeModel(opts)
}

// runModelFromConfig generates shared models for all protos in the proto_module
// declared in .ss.yaml.
func runModelFromConfig(ctx *cmdctx.Context, cfg *RpcConfig) error {
	pm := cfg.ProtoModuleConfig()
	if pm.Dir == "" {
		return fmt.Errorf("rpc.proto_module.dir is required in .ss.yaml")
	}

	// Collect all proto files referenced across services
	seen := map[string]bool{}
	var protoFiles []string
	for _, svc := range cfg.Services {
		for _, protoRel := range svc.Protos {
			if seen[protoRel] {
				continue
			}
			seen[protoRel] = true
			protoFiles = append(protoFiles, protoRel)
		}
	}

	if len(protoFiles) == 0 {
		return fmt.Errorf("no protos listed under rpc.services in .ss.yaml")
	}

	for _, protoRel := range protoFiles {
		protoPath := filepath.Join(ctx.WorkingDir, pm.Dir, protoRel)
		outputDir := filepath.Join(ctx.WorkingDir, pm.Dir)

		opts, err := buildModelOptionsFromConfig(ctx, pm, protoPath, outputDir)
		if err != nil {
			return fmt.Errorf("proto %s: %w", protoRel, err)
		}

		log.Info("--- Generating model: %s ---", protoRel)
		if err := executeModel(opts); err != nil {
			return fmt.Errorf("proto %s: %w", protoRel, err)
		}
	}
	return nil
}

// buildModelOptionsFromConfig constructs modelOptions from .ss.yaml + auto-detection.
func buildModelOptionsFromConfig(
	ctx *cmdctx.Context,
	pm ProtoModuleConfig,
	protoPath string,
	outputDir string,
) (*modelOptions, error) {
	absProtoFile, err := filepath.Abs(protoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute proto path: %w", err)
	}
	absOutputDir, err := filepath.Abs(outputDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute output path: %w", err)
	}

	// Module: CLI flag > go.mod in output dir
	module := resolveFlag(ctx, "module", "m")
	if module == "" {
		module, err = resolveModule(ctx, absOutputDir)
		if err != nil {
			return nil, err
		}
	}

	// GenPath: CLI flag > proto_module.gen_path
	genPath := ctx.GetFlag("gen-path")
	if genPath == "" && pm.GenPath != "" {
		genPath = pm.GenPath
	}

	return &modelOptions{
		absProtoFile: absProtoFile,
		absOutputDir: absOutputDir,
		module:       module,
		genPath:      genPath,
		verbose:      ctx.Debug,
	}, nil
}

// executeModel validates and runs kitex model generation for resolved modelOptions.
func executeModel(opts *modelOptions) error {
	log.Info("Generating shared model (kitex_gen)...")
	log.Info("  Proto file: %s", opts.absProtoFile)
	log.Info("  Output:     %s", opts.absOutputDir)
	log.Info("  Module:     %s", opts.module)

	if err := gen.CheckKitexInstalled(); err != nil {
		return err
	}
	if err := gen.CheckProtocInstalled(); err != nil {
		return err
	}

	if err := os.MkdirAll(opts.absOutputDir, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	wrapper := gen.NewKitexWrapper(gen.WrapperOptions{
		ProtoFile: opts.absProtoFile,
		OutputDir: opts.absOutputDir,
		Module:    opts.module,
		Service:   "", // Empty service = generate only models
		Includes:  buildProtoIncludes(opts.absProtoFile),
		Verbose:   opts.verbose,
		GenPath:   opts.genPath,
	})

	if err := wrapper.RunKitex(); err != nil {
		return fmt.Errorf("kitex generation failed: %w", err)
	}

	genPathName := "kitex_gen"
	if opts.genPath != "" {
		genPathName = opts.genPath
	}

	log.Success("Shared model generation completed!")
	fmt.Println()
	fmt.Printf("Generated structure:\n")
	fmt.Printf("  %s/\n", opts.absOutputDir)
	fmt.Printf("  |-- %s/              # Generated types and interfaces\n", genPathName)
	fmt.Printf("  |   +-- <package>/       # Package from proto\n")
	fmt.Printf("  |       |-- *.pb.go      # Proto message types\n")
	fmt.Printf("  |       +-- <service>/   # Service client/server interfaces\n")
	fmt.Printf("  +-- go.mod\n")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Printf("  1. cd %s && go mod tidy\n", opts.absOutputDir)
	fmt.Println("  2. Use this model in RPC server:")
	fmt.Printf("     ss rpc gen -p %s -o <service-dir>\n", opts.absProtoFile)
	fmt.Println()
	fmt.Println("  Note: --service and --use are auto-detected from proto file's go_package.")
	fmt.Println("        -m is auto-detected from <service-dir>/go.mod if present.")

	return nil
}
