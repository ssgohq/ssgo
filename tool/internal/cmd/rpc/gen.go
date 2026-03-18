package cmd

import (
	"fmt"
	"strings"

	"github.com/ssgohq/ssgo/internal/util/log"
	"github.com/ssgohq/ssgo/tool/internal/cmdctx"
	gen "github.com/ssgohq/ssgo/tool/internal/generator/kitex"
)

// runGen generates a full Kitex RPC server from a .proto file
func runGen(ctx *cmdctx.Context) error {
	if ctx.GetFlagBool("help") || ctx.GetFlagBool("h") {
		return printGenHelp()
	}

	opts, err := resolveGenOptions(ctx)
	if err != nil {
		return err
	}
	if opts == nil {
		return printGenHelp()
	}

	// Validate service name convention
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
		Verbose:   ctx.Debug,
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

Options:
	 -p, --proto <file>     Path to .proto file (required)
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

Examples:
	 # Minimal (auto-detect service, module, and --use):
	 ss rpc gen -p proto/auth.proto -o my-auth-svc

	 # Explicit (override auto-detection):
	 ss rpc gen -p proto/auth.proto --service AuthService -m github.com/org/auth-rpc

	 # With shared types module:
	 ss rpc gen -p shared-proto/proto/user.proto -o user-svc --with-trace`)
	return nil
}
