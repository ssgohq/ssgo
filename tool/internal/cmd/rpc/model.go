package cmd

import (
	"fmt"
	"os"

	"github.com/ssgohq/ssgo/internal/util/log"
	"github.com/ssgohq/ssgo/tool/internal/cmdctx"
	gen "github.com/ssgohq/ssgo/tool/internal/generator/kitex"
)

// runModel generates shared models (kitex_gen) only from a .proto file
func runModel(ctx *cmdctx.Context) error {
	opts, err := resolveModelOptions(ctx)
	if err != nil {
		return err
	}

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
