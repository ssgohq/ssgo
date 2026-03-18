package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ssgohq/ssgo/internal/util/log"
	"github.com/ssgohq/ssgo/tool/internal/cmdctx"
	gen "github.com/ssgohq/ssgo/tool/internal/generator/kitex"
)

// runGen generates a full Kitex RPC server from a .proto file.
//
// When no --proto flag is provided, runGen falls back to the rpc: section of
// .ss.yaml and generates all services (or the service matched by an optional
// positional argument).
func runGen(ctx *cmdctx.Context) error {
	if ctx.GetFlagBool("help") || ctx.GetFlagBool("h") {
		return printGenHelp()
	}

	// Explicit CLI invocation (--proto provided)
	if ctx.GetFlag("proto") != "" || ctx.GetFlag("p") != "" {
		return runGenSingle(ctx)
	}

	// Fall back to .ss.yaml batch mode
	cfg, err := LoadRpcConfig(ctx.WorkingDir)
	if err != nil {
		return fmt.Errorf("failed to load .ss.yaml: %w", err)
	}
	if cfg.IsEmpty() {
		return printGenHelp()
	}

	// Optional positional arg: filter by service dir name
	serviceFilter := ""
	if len(ctx.Args) > 0 {
		serviceFilter = ctx.Args[0]
	}

	return runGenFromConfig(ctx, cfg, serviceFilter)
}

// runGenSingle generates a single RPC server from explicit CLI flags.
func runGenSingle(ctx *cmdctx.Context) error {
	opts, err := resolveGenOptions(ctx)
	if err != nil {
		return err
	}
	if opts == nil {
		return printGenHelp()
	}
	return executeGen(opts)
}

// runGenFromConfig generates RPC servers for services declared in .ss.yaml.
func runGenFromConfig(ctx *cmdctx.Context, cfg *RpcConfig, serviceFilter string) error {
	pm := cfg.ProtoModuleConfig()

	for _, svc := range cfg.Services {
		// Apply optional service filter (match by dir basename or full dir)
		if serviceFilter != "" {
			base := filepath.Base(svc.Dir)
			if base != serviceFilter && svc.Dir != serviceFilter {
				continue
			}
		}

		for _, protoRel := range svc.Protos {
			// Proto path: proto_module.dir / proto_rel
			protoPath := filepath.Join(pm.Dir, protoRel)
			if !filepath.IsAbs(protoPath) {
				protoPath = filepath.Join(ctx.WorkingDir, protoPath)
			}

			// Derive genOptions using the same auto-detection logic
			opts, err := buildGenOptionsFromConfig(ctx, pm, svc, protoPath)
			if err != nil {
				return fmt.Errorf("service %s: %w", svc.Dir, err)
			}

			log.Info("--- Generating service: %s ---", svc.Dir)
			if err := executeGen(opts); err != nil {
				return fmt.Errorf("service %s: %w", svc.Dir, err)
			}
		}
	}
	return nil
}

// buildGenOptionsFromConfig constructs genOptions from .ss.yaml + auto-detection.
// CLI flags override .ss.yaml values where both are provided.
func buildGenOptionsFromConfig(
	ctx *cmdctx.Context,
	pm ProtoModuleConfig,
	svc ServiceConfig,
	absProtoFile string,
) (*genOptions, error) {
	absProtoFile, err := filepath.Abs(absProtoFile)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute proto path: %w", err)
	}

	proto, err := gen.ParseProto(absProtoFile)
	if err != nil {
		return nil, fmt.Errorf("failed to parse proto file: %w", err)
	}

	outputDir := svc.Dir
	if !filepath.IsAbs(outputDir) {
		outputDir = filepath.Join(ctx.WorkingDir, outputDir)
	}
	absOutputDir, err := filepath.Abs(outputDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute output path: %w", err)
	}

	// Service: CLI flag > auto-detect from proto
	service := resolveFlag(ctx, "service", "s")
	if service == "" {
		service, err = resolveService(ctx, proto)
		if err != nil {
			return nil, err
		}
	}

	// Module: CLI flag > go.mod in service dir
	module := resolveFlag(ctx, "module", "m")
	if module == "" {
		module, err = resolveModule(ctx, absOutputDir)
		if err != nil {
			return nil, err
		}
	}

	// UseTypes: CLI flag > auto-derive from proto go_package
	useTypes := ctx.GetFlag("use")
	if useTypes == "" {
		useTypes = resolveUseTypes(ctx, proto)
	}

	// GenPath: CLI flag > proto_module.gen_path
	genPath := ctx.GetFlag("gen-path")
	if genPath == "" {
		genPath = pm.EffectiveGenPath()
		if genPath == "kitex_gen" {
			genPath = "" // let generator use its own default
		}
	}

	// Boolean options: CLI flags override .ss.yaml
	withTrace := ctx.GetFlagBool("with-trace") || svc.Options.WithTrace
	withRedis := ctx.GetFlagBool("with-redis") || svc.Options.WithRedis

	return &genOptions{
		absProtoFile: absProtoFile,
		absOutputDir: absOutputDir,
		module:       module,
		service:      service,
		useTypes:     useTypes,
		genPath:      genPath,
		withTrace:    withTrace,
		withRedis:    withRedis,
	}, nil
}

// executeGen validates and runs the generator for a resolved genOptions.
func executeGen(opts *genOptions) error {
	if !strings.HasSuffix(opts.service, "Service") {
		log.Warning("Service name '%s' doesn't end with 'Service'. This is a convention in Kitex.", opts.service)
	}

	log.Info("Generating RPC server from Proto definition...")
	log.Info("  Proto file: %s", opts.absProtoFile)
	log.Info("  Output:     %s", opts.absOutputDir)
	log.Info("  Module:     %s", opts.module)
	log.Info("  Service:    %s", opts.service)
	if opts.useTypes != "" {
		log.Info("  Use types:  %s", opts.useTypes)
	}

	g := gen.NewGenerator(gen.Options{
		ProtoFile: opts.absProtoFile,
		OutputDir: opts.absOutputDir,
		Module:    opts.module,
		Service:   opts.service,
		Verbose:   false,
		UseTypes:  opts.useTypes,
		GenPath:   opts.genPath,
		WithTrace: opts.withTrace,
		WithRedis: opts.withRedis,
	})

	if err := g.Generate(); err != nil {
		return fmt.Errorf("generation failed: %w", err)
	}

	log.Success("RPC generation completed!")
	return nil
}

func printGenHelp() error {
	fmt.Println(`ss rpc gen - Generate Kitex RPC server from .proto file

Usage:
	 ss rpc gen -p <file> [options]
	 ss rpc gen [service-name]   (reads from .ss.yaml rpc section)

Options:
	 -p, --proto <file>     Path to .proto file (required when not using .ss.yaml)
	     --service <name>   Service name (auto-detected from proto if single service)
	 -m, --module <name>    Go module name (auto-detected from go.mod in output dir)
	 -o, --dir <dir>        Output directory (default: .)
	     --use <module>     Use external types module (auto-derived from proto go_package)
	     --gen-path <path>  Path for generated kitex_gen
	     --with-trace       Enable OpenTelemetry tracing
	     --with-redis       Enable Redis integration
	 -h, --help             Show this help

Auto-detection:
	 --service  Extracted from proto file (if exactly one service defined)
	 -m         Read from go.mod in the output directory (if exists)
	 --use      Derived from proto's go_package option (if full import path)

.ss.yaml rpc section (zero-flag mode):
	 rpc:
	   proto_module:
	     dir: ght-prutal
	     gen_path: kitex_gen
	   services:
	     - dir: 1s-auth-svc
	       protos:
	         - proto/1sauth/v1/auth.proto
	       options:
	         with_trace: true

Examples:
	 # Minimal (auto-detect service, module, and --use):
	 ss rpc gen -p proto/auth.proto -o my-auth-svc

	 # Explicit (override auto-detection):
	 ss rpc gen -p proto/auth.proto --service AuthService -m github.com/org/auth-rpc

	 # Zero-flag (reads .ss.yaml):
	 ss rpc gen

	 # Zero-flag (specific service from .ss.yaml):
	 ss rpc gen 1s-auth-svc`)
	return nil
}
