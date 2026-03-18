// Package service implements the `ss service` command group.
// Subcommands: inspect, plan, gen, sync.
package service

import (
	"fmt"
	"os"

	"github.com/ssgohq/ssgo/tool/internal/cmdctx"
)

// Options holds parsed flags for all service subcommands.
type Options struct {
	// Dir is the target service directory (positional arg or cwd).
	Dir string
	// JSON outputs machine-readable JSON.
	JSON bool
	// DryRun prints what would be done without modifying files.
	DryRun bool
	// Verbose enables verbose output.
	Verbose bool
	// Quiet suppresses non-essential output.
	Quiet bool
	// NonInteractive disables all prompts; fails fast when required inputs are missing.
	// Set automatically in CI environments or when --non-interactive is passed.
	NonInteractive bool
}

// resolveDir returns Dir if set, otherwise the current working directory.
func (o Options) resolveDir() string {
	if o.Dir != "" {
		return o.Dir
	}
	dir, _ := os.Getwd()
	return dir
}

// Execute is the entry point called from the cobra serviceCmd in root.go.
// It dispatches to the appropriate subcommand handler based on ctx.Args.
func Execute(ctx *cmdctx.Context) error {
	opts := Options{
		JSON:           ctx.GetFlagBool("json"),
		DryRun:         ctx.GetFlagBool("dry-run") || ctx.GetFlagBool("plan"),
		Verbose:        ctx.GetFlagBool("verbose") || ctx.GetFlagBool("v"),
		Quiet:          ctx.GetFlagBool("quiet") || ctx.GetFlagBool("q"),
		NonInteractive: ctx.GetFlagBool("non-interactive") || isCI(),
	}

	if len(ctx.Args) == 0 {
		return printUsage()
	}

	subcmd := ctx.Args[0]
	// Remaining positional args after subcommand name.
	rest := ctx.Args[1:]
	if len(rest) > 0 {
		opts.Dir = rest[0]
	}

	switch subcmd {
	case "inspect":
		return runInspect(opts)
	case "plan":
		opts.DryRun = true
		return runPlan(opts)
	case "gen", "generate":
		return runGen(opts)
	case "sync":
		return runSync(opts)
	default:
		return fmt.Errorf("unknown service subcommand %q; available: inspect, plan, gen, sync", subcmd)
	}
}

func printUsage() error {
	fmt.Fprint(os.Stdout, `Usage: ss service <subcommand> [dir] [flags]

Subcommands:
  inspect   Show detected contracts and current service state
  plan      Print generation plan (dry-run, deterministic)
  gen       Apply generation plan to the service directory
  sync      Regenerate from manifest with ownership-aware tidy

Flags:
  --json              Output machine-readable JSON
  --dry-run           Print what would be done without modifying files
  --plan              Alias for --dry-run
  --verbose           Enable verbose output
  --quiet             Suppress non-essential output
  --non-interactive   Disable prompts; fail fast on missing inputs (auto-set in CI)

Examples:
  ss service inspect ./my-service
  ss service plan    ./my-service --json
  ss service gen     ./my-service --dry-run
  ss service gen     ./my-service --non-interactive
  ss service sync    ./my-service
`)
	return nil
}

// isCI returns true when common CI environment variables are set.
func isCI() bool {
	return os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != "" || os.Getenv("BUILDKITE") != ""
}
