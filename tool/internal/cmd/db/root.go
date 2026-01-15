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
	case "sqlc":
		return runSqlcCommand(ctx)
	case "bun":
		return runBun(ctx)
	case "gorm":
		return runGorm(ctx)
	case "parse":
		return runParse(ctx)
	case "help", "-h", "--help":
		return printHelp()
	default:
		return fmt.Errorf("unknown command: %s\nRun 'ss db help' for usage", cmd)
	}
}

// Complete handles shell completion
func Complete(ctx *cmdctx.Context) {
	if len(ctx.Args) == 0 {
		fmt.Println("sqlc")
		fmt.Println("bun")
		fmt.Println("gorm")
		fmt.Println("parse")
		fmt.Println("help")
		return
	}

	cmd := ctx.Args[0]
	ctx.Args = ctx.Args[1:] // shift args for subcommand completion

	switch cmd {
	case "sqlc":
		completeSqlc(ctx)
	case "bun":
		completeBun(ctx)
	case "gorm":
		completeGorm(ctx)
	case "parse":
		completeParse(ctx)
	}
}

func printHelp() error {
	fmt.Println(`ss-plugin-db - Generate database layer code for Go projects

Usage:
  ss db <command> [flags]

Commands:
  sqlc    Generate type-safe Go code from SQL queries using SQLC
  bun     Generate models and repositories using uptrace/bun ORM
  gorm    Generate models and repositories using GORM ORM
  parse   Parse database schema (for testing/debugging)

SQLC Commands:
  ss db sqlc init   Initialize SQLC in a service (creates sqlc.yaml, query/, store/)
  ss db sqlc gen    Generate supporting code after 'sqlc generate'

Bun Commands:
  ss db bun gen     Generate models and repositories from database schema

GORM Commands:
  ss db gorm gen    Generate models and repositories from database schema

Parse Command:
  ss db parse --dsn <connection_string>   Parse and display database schema

Examples:
  # Initialize SQLC in a service
  ss db sqlc init --dir ./my-rpc-service --migrations ../migrations

  # Generate repositories and update config after sqlc generate
  ss db sqlc gen --dir ./my-rpc-service

  # Generate Bun models from PostgreSQL
  ss db bun gen --dsn 'postgres://user:pass@localhost:5432/mydb?sslmode=disable'

  # Generate GORM models from PostgreSQL
  ss db gorm gen --dsn 'postgres://user:pass@localhost:5432/mydb?sslmode=disable'

  # Parse database schema
  ss db parse --dsn 'postgres://user:pass@localhost:5432/mydb?sslmode=disable'

Use "ss db <command> help" for more information about a command.`)
	return nil
}
