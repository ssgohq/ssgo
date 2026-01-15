package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ssgohq/ssgo/tool/internal/cmdctx"
	"github.com/ssgohq/ssgo/tool/internal/generator/gorm"
)

// runGorm handles the gorm subcommand
func runGorm(ctx *cmdctx.Context) error {
	if len(ctx.Args) == 0 {
		return printGormHelp()
	}

	subcmd := ctx.Args[0]
	ctx.Args = ctx.Args[1:]

	switch subcmd {
	case "gen":
		return runGormGen(ctx)
	case "help", "-h", "--help":
		return printGormHelp()
	default:
		return fmt.Errorf("unknown gorm subcommand: %s\nRun 'ss db gorm help' for usage", subcmd)
	}
}

// runGormGen handles the gorm gen command
func runGormGen(ctx *cmdctx.Context) error {
	dsn := ctx.GetFlag("dsn")
	if dsn == "" {
		dsn = os.Getenv("SS_DB_DSN")
	}

	if dsn == "" {
		return printGormGenHelp()
	}

	// Get output directory
	dir := ctx.GetFlag("dir")
	if dir == "" {
		dir = "."
	}
	outputDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("invalid directory: %w", err)
	}

	// Get module name from go.mod if not specified
	moduleName := ctx.GetFlag("module")
	if moduleName == "" {
		moduleName = detectModuleName(outputDir)
	}

	// Parse tables flag
	var tables []string
	if t := ctx.GetFlag("tables"); t != "" {
		tables = strings.Split(t, ",")
		for i := range tables {
			tables[i] = strings.TrimSpace(tables[i])
		}
	}

	// Parse exclude flag
	var exclude []string
	if e := ctx.GetFlag("exclude"); e != "" {
		exclude = strings.Split(e, ",")
		for i := range exclude {
			exclude[i] = strings.TrimSpace(exclude[i])
		}
	}

	opts := gorm.GenOptions{
		DSN:            dsn,
		SchemaName:     ctx.GetFlag("schema"),
		Tables:         tables,
		ExcludeTables:  exclude,
		OutputDir:      outputDir,
		ModuleName:     moduleName,
		ModelPackage:   ctx.GetFlag("model-pkg"),
		RepoPackage:    ctx.GetFlag("repo-pkg"),
		ModelOnly:      ctx.GetFlagBool("model-only"),
		RepoOnly:       ctx.GetFlagBool("repo-only"),
		StrictNullable: ctx.GetFlagBool("strict"),
		WithTrace:      ctx.GetFlagBool("trace"),
		SoftDelete:     !ctx.GetFlagBool("no-soft-delete"),
		WithHooks:      ctx.GetFlagBool("hooks"),
		Verbose:        ctx.GetFlagBool("verbose") || ctx.GetFlagBool("v"),
	}

	// Apply defaults
	if opts.ModelPackage == "" {
		opts.ModelPackage = "model"
	}
	if opts.RepoPackage == "" {
		opts.RepoPackage = "repository"
	}

	gen := gorm.NewGenerator(opts)
	return gen.Generate()
}

func printGormHelp() error {
	fmt.Println(`ss db gorm - Generate GORM models and repositories

Usage:
  ss db gorm <command> [flags]

Commands:
  gen     Generate models and repositories from database schema

Use "ss db gorm <command> --help" for more information about a command.`)
	return nil
}

func printGormGenHelp() error {
	fmt.Println(`ss db gorm gen - Generate GORM code from database

Usage:
  ss db gorm gen --dsn <connection_string> [options]

Options:
  --dsn <string>        Database connection string (or set SS_DB_DSN env var)
  --schema <string>     Schema name (default: public for PostgreSQL)
  --tables <string>     Comma-separated table names (default: all)
  --exclude <string>    Comma-separated tables to exclude
  --dir, -o <path>      Output directory (default: current directory)
  --module <string>     Go module name (auto-detected from go.mod)
  --model-pkg <string>  Model package name (default: model)
  --repo-pkg <string>   Repository package name (default: repository)
  --model-only          Generate only models, not repositories
  --repo-only           Generate only repositories, not models
  --strict              Use sql.Null* types instead of pointers for nullable
  --trace               Add OpenTelemetry tracing to repositories
  --no-soft-delete      Disable soft delete support (default: enabled)
  --hooks               Generate hook methods (BeforeCreate, etc.)
  -v, --verbose         Verbose output

Examples:
  # Generate from PostgreSQL
  ss db gorm gen --dsn 'postgres://user:pass@localhost:5432/mydb?sslmode=disable'

  # Generate from MySQL
  ss db gorm gen --dsn 'user:pass@tcp(localhost:3306)/mydb?parseTime=true'

  # Generate specific tables
  ss db gorm gen --dsn '...' --tables users,posts,comments

  # Generate only models
  ss db gorm gen --dsn '...' --model-only

  # With OpenTelemetry tracing
  ss db gorm gen --dsn '...' --trace

  # Without soft delete
  ss db gorm gen --dsn '...' --no-soft-delete

  # Using environment variable
  export SS_DB_DSN='postgres://user:pass@localhost:5432/mydb?sslmode=disable'
  ss db gorm gen`)
	return nil
}

// completeGorm handles completion for gorm subcommands
func completeGorm(ctx *cmdctx.Context) {
	if len(ctx.Args) == 0 {
		fmt.Println("gen")
		fmt.Println("help")
		return
	}

	subcmd := ctx.Args[0]
	switch subcmd {
	case "gen":
		flags := []string{
			"--dsn", "--schema", "--tables", "--exclude",
			"--dir", "-o", "--module",
			"--model-pkg", "--repo-pkg",
			"--model-only", "--repo-only",
			"--strict", "--trace",
			"--no-soft-delete", "--hooks",
			"-v", "--verbose",
		}
		for _, f := range flags {
			fmt.Println(f)
		}
	}
}
