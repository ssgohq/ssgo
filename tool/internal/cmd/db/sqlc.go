package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/ssgohq/ssgo/tool/internal/cmdctx"
	"github.com/ssgohq/ssgo/tool/internal/generator/sqlc"
)

// runSqlcCommand handles the sqlc subcommand
func runSqlcCommand(ctx *cmdctx.Context) error {
	if len(ctx.Args) == 0 {
		return printSqlcHelp()
	}

	subcmd := ctx.Args[0]
	ctx.Args = ctx.Args[1:] // shift args

	switch subcmd {
	case "init":
		return runSqlcInit(ctx)
	case "gen":
		return runSqlcGen(ctx)
	case "help", "-h", "--help":
		return printSqlcHelp()
	default:
		return fmt.Errorf("unknown sqlc subcommand: %s\nRun 'ss db sqlc help' for usage", subcmd)
	}
}

// runSqlcInit handles the sqlc init command
func runSqlcInit(ctx *cmdctx.Context) error {
	// Get flags from SDK context (parsed by ss-cli)
	dir := ctx.GetFlag("dir")
	if dir == "" {
		dir = "."
	}

	// Resolve output directory
	outputDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("invalid directory: %w", err)
	}

	dbType := ctx.GetFlag("db-type")
	if dbType == "" {
		dbType = "postgres"
	}

	gen := sqlc.NewInitGenerator(sqlc.InitOptions{
		OutputDir:     outputDir,
		MigrationPath: ctx.GetFlag("migrations"),
		SchemaName:    ctx.GetFlag("schema"),
		SampleEntity:  ctx.GetFlag("entity"),
		DBType:        dbType,
		Verbose:       ctx.GetFlagBool("verbose"),
	})

	return gen.Generate()
}

// runSqlcGen handles the sqlc gen command
func runSqlcGen(ctx *cmdctx.Context) error {
	// Get flags from SDK context (parsed by ss-cli)
	dir := ctx.GetFlag("dir")
	if dir == "" {
		dir = "."
	}

	// Resolve output directory
	outputDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("invalid directory: %w", err)
	}

	gen := sqlc.NewGenCommand(sqlc.GenOptions{
		OutputDir: outputDir,
		Verbose:   ctx.GetFlagBool("verbose"),
		WithTrace: ctx.GetFlagBool("trace"),
	})

	return gen.Generate()
}

// printSqlcHelp prints the help for sqlc command
func printSqlcHelp() error {
	fmt.Println(`ss db sqlc - Generate SQLC-based database layer

Usage:
  ss db sqlc <command> [flags]

Commands:
  init    Initialize SQLC in a service (creates sqlc.yaml, query/, store/)
  gen     Generate supporting code after 'sqlc generate' (repositories, config updates)

Examples:
  # Initialize SQLC in a Kitex RPC service
  ss db sqlc init --dir ./my-rpc-service --migrations ../migrations

  # Initialize with a sample query template
  ss db sqlc init --dir ./my-service --entity User --schema myapp

  # After writing SQL queries and running 'sqlc generate'
  ss db sqlc gen --dir ./my-rpc-service

  # Generate with OpenTelemetry tracing support
  ss db sqlc gen --dir ./my-service --trace

Flags:
  init:
    -d, --dir         Service directory (default: current directory)
    -m, --migrations  Path to migrations directory for SQLC schema
    -s, --schema      Schema name for SQL queries (e.g., lineart)
    -e, --entity      Sample entity name to generate query template
        --db-type     Database type: postgres or mysql (default: postgres)
    -v, --verbose     Verbose output

  gen:
    -d, --dir         Service directory (default: current directory)
    -t, --trace       Enable OpenTelemetry tracing in generated code
    -v, --verbose     Verbose output

Workflow:
  1. Create a service with 'ss rpc gen' or 'ss api gen'
  2. Run 'ss db sqlc init --dir ./my-service --migrations ../migrations'
  3. Write your SQL queries in query/*.sql
  4. Run 'sqlc generate' to generate type-safe Go code
  5. Run 'ss db sqlc gen --dir ./my-service' to generate repositories and update config

Note: SQLC CLI must be installed: go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`)
	return nil
}

// completeSqlc provides completion for sqlc subcommands
func completeSqlc(ctx *cmdctx.Context) {
	if len(ctx.Args) == 0 {
		fmt.Println("init")
		fmt.Println("gen")
		fmt.Println("help")
		return
	}

	subcmd := ctx.Args[0]
	switch subcmd {
	case "init":
		fmt.Println("-d")
		fmt.Println("--dir")
		fmt.Println("-m")
		fmt.Println("--migrations")
		fmt.Println("-s")
		fmt.Println("--schema")
		fmt.Println("-e")
		fmt.Println("--entity")
		fmt.Println("--db-type")
		fmt.Println("-v")
		fmt.Println("--verbose")
	case "gen":
		fmt.Println("-d")
		fmt.Println("--dir")
		fmt.Println("-t")
		fmt.Println("--trace")
		fmt.Println("-v")
		fmt.Println("--verbose")
	}
}
