package cmd

import (
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/ssgohq/ssgo/internal/util/log"
	"github.com/ssgohq/ssgo/tool/internal/cmdctx"
)

// runSync implements `ss api sync`: gen → gofmt → go mod tidy.
//
// Usage:
//
//	ss api sync                     # all services from .ss.yaml
//	ss api sync <file-basename>     # specific service from .ss.yaml
//	ss api sync --api <file> [opts] # explicit flags mode
func runSync(ctx *cmdctx.Context) error {
	if ctx.GetFlagBool("help") || ctx.GetFlagBool("h") {
		return printApiSyncHelp()
	}

	// Explicit flags mode: --api or -a flag provided
	if ctx.GetFlag("api") != "" || ctx.GetFlag("a") != "" {
		return runApiSyncSingle(ctx)
	}

	// .ss.yaml batch mode
	cfg, err := LoadApiConfig(ctx.WorkingDir)
	if err != nil {
		return fmt.Errorf("failed to load .ss.yaml: %w", err)
	}
	if cfg.IsEmpty() {
		return printApiSyncHelp()
	}

	serviceFilter := ""
	if len(ctx.Args) > 0 {
		serviceFilter = ctx.Args[0]
	}

	return runApiSyncFromConfig(ctx, cfg, serviceFilter)
}

// runApiSyncSingle runs gen → tidy for a single API via CLI flags.
func runApiSyncSingle(ctx *cmdctx.Context) error {
	outputDir := ctx.GetFlag("dir")
	if outputDir == "" {
		outputDir = ctx.GetFlag("o")
	}
	if outputDir == "" {
		outputDir = "."
	}

	if err := runGenSingle(ctx); err != nil {
		return fmt.Errorf("gen step failed: %w", err)
	}

	if err := goModTidy(outputDir); err != nil {
		return fmt.Errorf("go mod tidy failed: %w", err)
	}

	log.Success("sync completed!")
	return nil
}

// runApiSyncFromConfig runs gen → tidy for each API service in .ss.yaml.
func runApiSyncFromConfig(ctx *cmdctx.Context, cfg *ApiConfig, serviceFilter string) error {
	log.Info("=== Step 1: API generation ===")
	if err := runGenFromConfig(ctx, cfg, serviceFilter); err != nil {
		return fmt.Errorf("gen step failed: %w", err)
	}

	log.Info("=== Step 2: go mod tidy for services ===")
	for _, svc := range cfg.Apis {
		if serviceFilter != "" {
			base := filepath.Base(svc.File)
			if base != serviceFilter && svc.File != serviceFilter {
				continue
			}
		}

		outputDir := svc.Dir
		if outputDir == "" {
			outputDir = "."
		}
		if !filepath.IsAbs(outputDir) {
			outputDir = filepath.Join(ctx.WorkingDir, outputDir)
		}

		if err := goModTidy(outputDir); err != nil {
			return fmt.Errorf("go mod tidy for %s failed: %w", svc.File, err)
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

func printApiSyncHelp() error {
	fmt.Println(`ss api sync - Generate API server and run go mod tidy in one command

Flow: gen → go mod tidy

Usage:
  ss api sync [file-basename]                    (reads .ss.yaml api section)
  ss api sync --api <file> -o <dir> [options]    (explicit flags mode)

Options (explicit mode):
  -a, --api <file>      Path to .api file (required)
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

Examples:
  # Sync all API services (from .ss.yaml):
  ss api sync

  # Sync specific API service:
  ss api sync user.api

  # Explicit (without .ss.yaml):
  ss api sync --api api/user.api -o ./output`)
	return nil
}
