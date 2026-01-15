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
	switch subCmd {
	case "new":
		return runNew(ctx)
	case "gen":
		return runGen(ctx)
	case "model":
		return runModel(ctx)
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
		completions := []string{"new", "gen", "model"}
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
  gen          Generate Kitex code from .proto file
  model        Generate shared model (kitex_gen) only

Examples:
  ss rpc new user
  ss rpc gen --proto idl/user.proto --service UserService -m github.com/org/user-rpc
  ss rpc model --proto idl/user.proto -m github.com/org/common-pb -o common-pb

Flags for 'gen':
  --proto, -p <file>    Path to .proto file (required)
  --service <name>      Service name e.g. UserService (required)
  --module, -m <name>   Go module name (required)
  --dir, -o <path>      Output directory (default: .)
  --use <import>        Import path for shared types
  --gen-path <name>     Generated code path (default: kitex_gen)
  --with-trace          Enable OpenTelemetry
  --with-redis          Add Redis config

Flags for 'model':
  --proto, -p <file>    Path to .proto file (required)
  --module, -m <name>   Go module name (required)
  --dir, -o <path>      Output directory (required)
  --gen-path <name>     Generated code path (default: kitex_gen)`)
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
