package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ssgohq/ssgo/internal/dbparser"
	_ "github.com/ssgohq/ssgo/internal/dbparser/mysql"
	_ "github.com/ssgohq/ssgo/internal/dbparser/postgres"
	"github.com/ssgohq/ssgo/tool/internal/cmdctx"
)

// runParse handles the parse subcommand for testing dbparser
func runParse(ctx *cmdctx.Context) error {
	// Flags are passed via SS_FLAG_* env vars by ss-cli (cobra parsing)
	dsn := ctx.GetFlag("dsn")
	schemaName := ctx.GetFlag("schema")
	tables := ctx.GetFlag("tables")
	outputJSON := ctx.GetFlagBool("json")
	verbose := ctx.GetFlagBool("verbose") || ctx.GetFlagBool("v")

	if dsn == "" {
		// Try environment variable
		dsn = os.Getenv("SS_DB_DSN")
	}

	if dsn == "" {
		return printParseHelp()
	}

	return executeParse(dsn, schemaName, tables, outputJSON, verbose)
}

func printParseHelp() error {
	fmt.Println(`ss db parse - Parse database schema (for testing dbparser)

Usage:
  ss db parse --dsn <connection_string> [options]

Options:
  --dsn <string>      Database connection string (or set SS_DB_DSN env var)
  --schema <string>   Schema name (default: public for PostgreSQL)
  --tables <string>   Comma-separated table names (default: all)
  --json              Output as JSON
  -v                  Verbose output

Examples:
  # PostgreSQL
  ss db parse --dsn 'postgres://user:pass@localhost:5432/mydb?sslmode=disable'

  # MySQL
  ss db parse --dsn 'user:pass@tcp(localhost:3306)/mydb?parseTime=true'

  # Parse specific tables
  ss db parse --dsn '...' --tables users,posts,comments

  # Output as JSON
  ss db parse --dsn '...' --json

  # Using environment variable
  export SS_DB_DSN='postgres://user:pass@localhost:5432/mydb?sslmode=disable'
  ss db parse`)
	return nil
}

func executeParse(dsn, schemaName, tables string, outputJSON, verbose bool) error {
	bgCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create parser
	parser, err := dbparser.NewParser(dsn)
	if err != nil {
		return fmt.Errorf("failed to create parser: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Database type: %s\n", parser.DatabaseType())

	// Connect
	fmt.Fprintln(os.Stderr, "Connecting to database...")
	if err := parser.Connect(bgCtx, dsn); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer parser.Close()
	fmt.Fprintln(os.Stderr, "Connected!")

	// Set default schema
	if schemaName == "" {
		if parser.DatabaseType() == dbparser.DatabaseTypePostgres {
			schemaName = "public"
		}
	}

	// Parse options
	opts := dbparser.ParseOptions{
		Verbose: verbose,
	}

	// Parse table names
	var tableNames []string
	if tables != "" {
		tableNames = strings.Split(tables, ",")
		for i := range tableNames {
			tableNames[i] = strings.TrimSpace(tableNames[i])
		}
	}

	// Parse schema
	fmt.Fprintf(os.Stderr, "Parsing schema '%s'...\n\n", schemaName)
	parsedTables, err := parser.ParseTables(bgCtx, schemaName, tableNames, opts)
	if err != nil {
		return fmt.Errorf("failed to parse tables: %w", err)
	}

	// Get enums (PostgreSQL only, ignore error for MySQL)
	enums, err := parser.GetEnums(bgCtx, schemaName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to get enums: %v\n", err)
	}

	// Create type mapper
	mapper := dbparser.NewTypeMapper(parser.DatabaseType())
	mapperOpts := dbparser.DefaultMapperOptions()

	// Map types
	for _, table := range parsedTables {
		dbparser.MapColumnsToGo(table, mapper, mapperOpts)
	}

	// Output
	if outputJSON {
		return outputParseJSON(parsedTables, enums)
	}
	outputParseText(parsedTables, enums)

	return nil
}

func outputParseJSON(tables []*dbparser.Table, enums []*dbparser.Enum) error {
	output := struct {
		Tables []*dbparser.Table `json:"tables"`
		Enums  []*dbparser.Enum  `json:"enums,omitempty"`
	}{
		Tables: tables,
		Enums:  enums,
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(output)
}

func outputParseText(tables []*dbparser.Table, enums []*dbparser.Enum) {
	// Print enums
	if len(enums) > 0 {
		fmt.Println("=== ENUMS ===")
		for _, enum := range enums {
			fmt.Printf("  %s: [%s]\n", enum.Name, strings.Join(enum.Values, ", "))
		}
		fmt.Println()
	}

	// Print tables
	fmt.Printf("=== TABLES (%d) ===\n\n", len(tables))

	for _, table := range tables {
		printParseTable(table)
	}
}

func printParseTable(table *dbparser.Table) {
	fmt.Printf("📋 Table: %s\n", table.Name)
	if table.Comment != "" {
		fmt.Printf("   Comment: %s\n", table.Comment)
	}

	// Primary Key
	if table.PrimaryKey != nil {
		fmt.Printf("   🔑 Primary Key: %s (%s)\n", table.PrimaryKey.Name, strings.Join(table.PrimaryKey.Columns, ", "))
	}

	// Columns
	fmt.Println("   📊 Columns:")
	for _, col := range table.Columns {
		nullable := " NOT NULL"
		if col.IsNullable {
			nullable = " NULL"
		}

		pk := ""
		if col.IsPrimaryKey {
			pk = " [PK]"
		}

		autoIncr := ""
		if col.IsAutoIncr {
			autoIncr = " [AUTO]"
		}

		unique := ""
		if col.IsUnique {
			unique = " [UNIQUE]"
		}

		defaultVal := ""
		if col.HasDefault {
			def := col.Default
			if len(def) > 30 {
				def = def[:27] + "..."
			}
			defaultVal = fmt.Sprintf(" DEFAULT %s", def)
		}

		comment := ""
		if col.Comment != "" {
			comment = fmt.Sprintf(" -- %s", col.Comment)
		}

		fmt.Printf("      %-20s %-25s -> %-20s%s%s%s%s%s%s\n",
			col.Name,
			col.ColumnType,
			col.GoType,
			nullable,
			pk,
			autoIncr,
			unique,
			defaultVal,
			comment,
		)
	}

	// Foreign Keys
	if len(table.ForeignKeys) > 0 {
		fmt.Println("   🔗 Foreign Keys:")
		for _, fk := range table.ForeignKeys {
			fmt.Printf("      %s: (%s) -> %s(%s) ON DELETE %s\n",
				fk.Name,
				strings.Join(fk.Columns, ", "),
				fk.RefTable,
				strings.Join(fk.RefColumns, ", "),
				fk.OnDelete,
			)
		}
	}

	// Indexes
	if len(table.Indexes) > 0 {
		fmt.Println("   📑 Indexes:")
		for _, idx := range table.Indexes {
			unique := ""
			if idx.IsUnique {
				unique = " [UNIQUE]"
			}
			fmt.Printf("      %s: (%s) %s%s\n",
				idx.Name,
				strings.Join(idx.Columns, ", "),
				idx.Type,
				unique,
			)
		}
	}

	fmt.Println()
}

func completeParse(_ *cmdctx.Context) {
	flags := []string{
		"--dsn",
		"--schema",
		"--tables",
		"--json",
		"-v",
	}
	for _, f := range flags {
		fmt.Println(f)
	}
}
