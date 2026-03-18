// Package cmd provides command handlers for the rpc commands.
package cmd

import (
	"fmt"

	"github.com/ssgohq/ssgo/internal/util/log"
	"github.com/ssgohq/ssgo/tool/internal/cmdctx"
)

// Execute handles the main command dispatch
func Execute(ctx *cmdctx.Context) error {
	if len(ctx.Args) == 0 {
		return printHelp()
	}

	subCmd := ctx.Args[0]
	// Create a sub-context with the subcommand name removed so that
	// subcommands see only their own positional arguments (e.g., service name).
	subCtx := cloneCtxWithArgs(ctx, ctx.Args[1:])

	switch subCmd {
	case "new":
		return runNew(subCtx)
	case "gen":
		return runGen(subCtx)
	case "model":
		return runModel(subCtx)
	case "sync":
		return runSync(subCtx)
	case "help", "-h", "--help":
		return printHelp()
	default:
		return fmt.Errorf("unknown rpc subcommand: %s", subCmd)
	}
}

// Complete handles completion requests
func Complete(ctx *cmdctx.Context) {
	args := ctx.Args
	toComplete := ctx.GetCompletionToComplete()

	// If no args, suggest subcommands
	if len(args) == 0 || (len(args) == 1 && toComplete != "") {
		completions := []string{"new", "gen", "model", "sync"}
		filtered := log.FilterCompletions(completions, toComplete)
		log.PrintCompletions(filtered)
		return
	}

	// Handle subcommand completions
	subCmd := args[0]
	switch subCmd {
	case "gen":
		completeGen(ctx, toComplete)
	case "model":
		completeModel(ctx, toComplete)
	case "sync":
		completeSync(ctx, toComplete)
	case "new":
		// No specific completions for new
	}
}

func printHelp() error {
	fmt.Println(`Generate Kitex RPC server from .proto files

Usage:
  ss rpc <command> [flags]

Commands:
  new <name>   Create a new .proto file template
  gen          Generate Kitex code from .proto file (or from .ss.yaml rpc section)
  model        Generate shared model (kitex_gen) only (or from .ss.yaml rpc section)
  sync         Generate model + service in one shot (model → tidy → gen → tidy)

Examples:
  ss rpc new user
  ss rpc gen --proto idl/user.proto --service UserService -m github.com/org/user-rpc
  ss rpc model --proto idl/user.proto -m github.com/org/common-pb -o common-pb

  # Zero-flag mode (reads .ss.yaml rpc section):
  ss rpc model
  ss rpc gen 1s-auth-svc
  ss rpc sync

Flags for 'gen':
  --proto, -p <file>    Path to .proto file (auto from .ss.yaml when omitted)
  --service <name>      Service name (auto-detected from proto)
  --module, -m <name>   Go module name (auto from go.mod)
  --dir, -o <path>      Output directory (default: .)
  --use <import>        Import path for shared types (auto from go_package)
  --gen-path <name>     Generated code path (default: kitex_gen)
  --with-trace          Enable OpenTelemetry
  --with-redis          Add Redis config

Flags for 'model':
  --proto, -p <file>    Path to .proto file (auto from .ss.yaml when omitted)
  --module, -m <name>   Go module name (auto from go.mod)
  --dir, -o <path>      Output directory
  --gen-path <name>     Generated code path (default: kitex_gen)

Flags for 'sync':
  -p, --proto <file>    Path to .proto file (auto from .ss.yaml when omitted)
  --model-dir <dir>     Output dir for shared model generation
  --dir, -o <path>      Service output directory
  --with-trace          Enable OpenTelemetry
  --with-redis          Add Redis config`)
	return nil
}

func completeGen(ctx *cmdctx.Context, toComplete string) {
	flags := []string{
		"--proto", "-p",
		"--service",
		"--module", "-m",
		"--dir", "-o",
		"--use",
		"--gen-path",
		"--with-trace",
		"--with-redis",
	}
	filtered := log.FilterCompletions(flags, toComplete)
	log.PrintCompletions(filtered)
}

func completeModel(ctx *cmdctx.Context, toComplete string) {
	flags := []string{
		"--proto", "-p",
		"--module", "-m",
		"--dir", "-o",
		"--gen-path",
	}
	filtered := log.FilterCompletions(flags, toComplete)
	log.PrintCompletions(filtered)
}

func completeSync(ctx *cmdctx.Context, toComplete string) {
	flags := []string{
		"--proto", "-p",
		"--model-dir",
		"--module", "-m",
		"--dir", "-o",
		"--service",
		"--use",
		"--gen-path",
		"--with-trace",
		"--with-redis",
	}
	filtered := log.FilterCompletions(flags, toComplete)
	log.PrintCompletions(filtered)
}
