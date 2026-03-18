package cmd

import (
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/ssgohq/ssgo/internal/util/log"
	"github.com/ssgohq/ssgo/tool/internal/cmdctx"
)

// runSync implements `ss rpc sync`: model → go mod tidy → gen → go mod tidy.
//
// Usage:
//
//	ss rpc sync                   # all services from .ss.yaml
//	ss rpc sync <service-name>    # specific service from .ss.yaml
//	ss rpc sync -p <proto> --model-dir <dir> -o <svc-dir> [options]  # explicit
func runSync(ctx *cmdctx.Context) error {
	if ctx.GetFlagBool("help") || ctx.GetFlagBool("h") {
		return printSyncHelp()
	}

	// Explicit flags mode: --proto and --model-dir are required
	if ctx.GetFlag("proto") != "" || ctx.GetFlag("p") != "" {
		return runSyncSingle(ctx)
	}

	// .ss.yaml batch mode
	cfg, err := LoadRpcConfig(ctx.WorkingDir)
	if err != nil {
		return fmt.Errorf("failed to load .ss.yaml: %w", err)
	}
	if cfg.IsEmpty() {
		return printSyncHelp()
	}

	serviceFilter := ""
	if len(ctx.Args) > 0 {
		serviceFilter = ctx.Args[0]
	}

	return runSyncFromConfig(ctx, cfg, serviceFilter)
}

// runSyncSingle runs model → tidy → gen → tidy for a single proto via CLI flags.
func runSyncSingle(ctx *cmdctx.Context) error {
	// Step 1: model generation
	modelCtx := cloneCtxWithFlag(ctx, "dir", ctx.GetFlag("model-dir"))
	if err := runModelSingle(modelCtx); err != nil {
		return fmt.Errorf("model step failed: %w", err)
	}

	// Step 2: go mod tidy in model dir
	modelDir := ctx.GetFlag("model-dir")
	if modelDir != "" {
		if err := goModTidy(modelDir); err != nil {
			return fmt.Errorf("go mod tidy (model) failed: %w", err)
		}
	}

	// Step 3: gen
	if err := runGenSingle(ctx); err != nil {
		return fmt.Errorf("gen step failed: %w", err)
	}

	// Step 4: go mod tidy in service dir
	serviceDir := resolveFlag(ctx, "dir", "o")
	if serviceDir == "" {
		serviceDir = "."
	}
	if err := goModTidy(serviceDir); err != nil {
		return fmt.Errorf("go mod tidy (service) failed: %w", err)
	}

	log.Success("sync completed!")
	return nil
}

// runSyncFromConfig runs model → tidy → gen → tidy for each service in .ss.yaml.
func runSyncFromConfig(ctx *cmdctx.Context, cfg *RpcConfig, serviceFilter string) error {
	pm := cfg.ProtoModuleConfig()

	// Step 1: model generation for all unique protos
	log.Info("=== Step 1: model generation ===")
	if err := runModelFromConfig(ctx, cfg); err != nil {
		return fmt.Errorf("model step failed: %w", err)
	}

	// Step 2: go mod tidy in proto_module dir
	if pm.Dir != "" {
		modelDir := pm.Dir
		if !filepath.IsAbs(modelDir) {
			modelDir = filepath.Join(ctx.WorkingDir, modelDir)
		}
		log.Info("=== Step 2: go mod tidy in %s ===", modelDir)
		if err := goModTidy(modelDir); err != nil {
			return fmt.Errorf("go mod tidy (model) failed: %w", err)
		}
	}

	// Step 3: gen for each service
	log.Info("=== Step 3: service generation ===")
	if err := runGenFromConfig(ctx, cfg, serviceFilter); err != nil {
		return fmt.Errorf("gen step failed: %w", err)
	}

	// Step 4: go mod tidy for each service dir
	log.Info("=== Step 4: go mod tidy for services ===")
	for _, svc := range cfg.Services {
		if serviceFilter != "" {
			base := filepath.Base(svc.Dir)
			if base != serviceFilter && svc.Dir != serviceFilter {
				continue
			}
		}
		svcDir := svc.Dir
		if !filepath.IsAbs(svcDir) {
			svcDir = filepath.Join(ctx.WorkingDir, svcDir)
		}
		if err := goModTidy(svcDir); err != nil {
			return fmt.Errorf("go mod tidy for %s failed: %w", svc.Dir, err)
		}
	}

	log.Success("sync completed!")
	return nil
}

// goModTidy runs `go mod tidy` in the given directory.
func goModTidy(dir string) error {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("failed to resolve directory: %w", err)
	}
	log.Info("Running go mod tidy in %s", absDir)
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = absDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go mod tidy: %w\n%s", err, string(out))
	}
	return nil
}

// cloneCtxWithFlag returns a shallow copy of ctx with one flag overridden.
func cloneCtxWithFlag(ctx *cmdctx.Context, key, value string) *cmdctx.Context {
	flags := make(map[string]interface{}, len(ctx.Flags))
	for k, v := range ctx.Flags {
		flags[k] = v
	}
	if value != "" {
		flags[key] = value
	}
	return &cmdctx.Context{
		Args:       ctx.Args,
		Flags:      flags,
		WorkingDir: ctx.WorkingDir,
		Debug:      ctx.Debug,
	}
}

func printSyncHelp() error {
	fmt.Println(`ss rpc sync - Generate shared models and RPC services in one command

Flow: model → go mod tidy → gen → go mod tidy

Usage:
	 ss rpc sync [service-name]                       (reads .ss.yaml rpc section)
	 ss rpc sync -p <proto> --model-dir <dir> -o <svc-dir> [options]

Options (explicit mode):
	 -p, --proto <file>     Path to .proto file (required)
	     --model-dir <dir>  Output dir for shared model (kitex_gen)
	 -o, --dir <dir>        Service output directory
	 -m, --module <name>    Go module name
	     --service <name>   Service name (auto-detected from proto)
	     --use <module>     External types import path (auto-derived)
	     --gen-path <path>  Custom kitex_gen path
	     --with-trace       Enable OpenTelemetry tracing
	     --with-redis       Enable Redis integration
	 -h, --help             Show this help

.ss.yaml rpc section (zero-flag mode):
	 rpc:
	   proto_module:
	     dir: shared-proto
	     gen_path: kitex_gen
	   services:
	     - dir: auth-svc
	       protos:
	         - proto/auth/v1/auth.proto
	       options:
	         with_trace: true

Examples:
	 # Sync all services (from .ss.yaml):
	 ss rpc sync

	 # Sync specific service:
	 ss rpc sync auth-svc

	 # Explicit (without .ss.yaml):
	 ss rpc sync -p proto/auth.proto --model-dir shared-proto -o auth-svc --with-trace`)
	return nil
}
