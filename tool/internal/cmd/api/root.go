package cmd

import (
	"fmt"

	"github.com/ssgohq/ssgo/tool/internal/cmdctx"
)

// Execute runs the appropriate command based on arguments
func Execute(ctx *cmdctx.Context) error {
	if len(ctx.Args) == 0 {
		return printHelp()
	}

	cmd := ctx.Args[0]
	ctx.Args = ctx.Args[1:] // shift args

	switch cmd {
	case "new":
		return runNew(ctx)
	case "gen":
		return runGen(ctx)
	case "logic":
		return runLogic(ctx)
	case "doc":
		return runDoc(ctx)
	case "-h", "--help", "help":
		return printHelp()
	default:
		return fmt.Errorf("unknown command: %s", cmd)
	}
}

// Complete handles shell completion
func Complete(ctx *cmdctx.Context) {
	if len(ctx.Args) == 0 {
		fmt.Println("new")
		fmt.Println("gen")
		fmt.Println("logic")
		fmt.Println("doc")
		return
	}

	cmd := ctx.Args[0]
	switch cmd {
	case "new":
		// Complete service names (no completion needed)
	case "gen", "logic":
		completeFlags(ctx, []string{"-a", "--api", "-o", "--dir", "-m", "--module"})
	case "doc":
		completeFlags(ctx, []string{"-a", "--api", "-o", "--dir", "--format"})
	}
}

func completeFlags(_ *cmdctx.Context, flags []string) {
	for _, flag := range flags {
		fmt.Println(flag)
	}
}

func printHelp() error {
	fmt.Println(`ss-plugin-api - Generate Hertz HTTP servers from .api files

Usage:
  ss api <command> [flags]
  ss api <command>              (zero-flag mode — reads from .ss.yaml api section)

Commands:
  new     Create a new .api file template
  gen     Generate Hertz code from .api file
  logic   Generate only logic files
  doc     Generate OpenAPI documentation

.ss.yaml api section (zero-flag mode):
  api:
    apis:
      - file: api/user.api
        dir: .
        options:
          port: 8080
          with_logic: true
          format: json

Examples:
  ss api new user
  ss api gen --api api/user.api -m github.com/org/user-api
  ss api logic --api api/user.api -m github.com/org/user-api
  ss api doc --api api/user.api --format yaml

  # Zero-flag mode (reads .ss.yaml):
  ss api gen
  ss api doc
  ss api logic`)
	return nil
}
