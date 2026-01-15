package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ssgohq/ssgo/tool/internal/cmdctx"
	"github.com/ssgohq/ssgo/tool/internal/generator/bun"
)

// runBun handles the bun subcommand
func runBun(ctx *cmdctx.Context) error {
	if len(ctx.Args) == 0 {
		return printBunHelp()
	}

	subcmd := ctx.Args[0]
	ctx.Args = ctx.Args[1:]

	switch subcmd {
	case "gen":
		return runBunGen(ctx)
	case "help", "-h", "--help":
		return printBunHelp()
	default:
		return fmt.Errorf("unknown bun subcommand: %s\nRun 'ss db bun help' for usage", subcmd)
	}
}

// runBunGen handles the bun gen command
func runBunGen(ctx *cmdctx.Context) error {
	dsn := ctx.GetFlag("dsn")
	if dsn == "" {
		dsn = os.Getenv("SS_DB_DSN")
	}

	if dsn == "" {
		return printBunGenHelp()
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

	opts := bun.GenOptions{
		DSN:            dsn,
		SchemaName:     ctx.GetFlag("schema"), // Empty = auto-detect based on DB type
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
		Verbose:        ctx.GetFlagBool("verbose") || ctx.GetFlagBool("v"),
	}

	// Apply defaults (schema default is handled in generator based on DB type)
	if opts.ModelPackage == "" {
		opts.ModelPackage = "model"
	}
	if opts.RepoPackage == "" {
		opts.RepoPackage = "repository"
	}

	gen := bun.NewGenerator(opts)
	return gen.Generate()
}

// detectModuleName tries to detect the Go module name from go.mod
func detectModuleName(dir string) string {
	goModPath := filepath.Join(dir, "go.mod")
	content, err := os.ReadFile(goModPath)
	if err != nil {
		return ""
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimPrefix(line, "module ")
		}
	}

	return ""
}

func printBunHelp() error {
	fmt.Println(`ss db bun - Generate Bun ORM models and repositories

Usage:
  ss db bun <command> [flags]

Commands:
  gen     Generate models and repositories from database schema

Use "ss db bun <command> --help" for more information about a command.`)
	return nil
}

func printBunGenHelp() error {
	fmt.Println(`ss db bun gen - Generate Bun ORM code from database

Usage:
  ss db bun gen --dsn <connection_string> [options]

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
  -v, --verbose         Verbose output

Examples:
  # Generate from PostgreSQL
  ss db bun gen --dsn 'postgres://user:pass@localhost:5432/mydb?sslmode=disable'

  # Generate from MySQL
  ss db bun gen --dsn 'user:pass@tcp(localhost:3306)/mydb?parseTime=true'

  # Generate specific tables
  ss db bun gen --dsn '...' --tables users,posts,comments

  # Generate only models
  ss db bun gen --dsn '...' --model-only

  # With OpenTelemetry tracing
  ss db bun gen --dsn '...' --trace

  # Using environment variable
  export SS_DB_DSN='postgres://user:pass@localhost:5432/mydb?sslmode=disable'
  ss db bun gen`)
	return nil
}

// completeBun handles completion for bun subcommands
func completeBun(ctx *cmdctx.Context) {
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
			"-v", "--verbose",
		}
		for _, f := range flags {
			fmt.Println(f)
		}
	}
}
