package bun

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ssgohq/ssgo/internal/dbparser"
)

// Generator generates Bun models and repositories from database schema
type Generator struct {
	opts   GenOptions
	parser dbparser.Parser
	schema *dbparser.Schema
}

// NewGenerator creates a new Bun generator
func NewGenerator(opts GenOptions) *Generator {
	return &Generator{opts: opts}
}

// Generate generates Bun models and repositories
func (g *Generator) Generate() error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create parser based on DSN
	parser, err := dbparser.NewParser(g.opts.DSN)
	if err != nil {
		return fmt.Errorf("failed to create parser: %w", err)
	}
	g.parser = parser

	g.logVerbose("Database type: %s\n", parser.DatabaseType())

	// Connect to database
	g.logVerbose("Connecting to database...\n")
	if err := parser.Connect(ctx, g.opts.DSN); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer parser.Close()
	g.logVerbose("Connected!\n")

	// Set default schema based on database type
	schemaName := g.opts.SchemaName
	if schemaName == "" {
		if parser.DatabaseType() == dbparser.DatabaseTypePostgres {
			schemaName = "public"
		}
		// MySQL: empty schema means use the database from DSN
	}

	// Parse schema
	parseOpts := dbparser.ParseOptions{
		ExcludeTables: g.opts.ExcludeTables,
		Verbose:       g.opts.Verbose,
	}

	g.logVerbose("Parsing schema '%s'...\n", schemaName)
	tables, err := parser.ParseTables(ctx, schemaName, g.opts.Tables, parseOpts)
	if err != nil {
		return fmt.Errorf("failed to parse tables: %w", err)
	}

	enums, err := parser.GetEnums(ctx, schemaName)
	if err != nil {
		g.logVerbose("Warning: failed to get enums: %v\n", err)
	}

	g.schema = &dbparser.Schema{
		Name:     schemaName,
		Tables:   tables,
		Enums:    enums,
		ParsedAt: time.Now(),
	}

	g.logVerbose("Found %d tables, %d enums\n", len(tables), len(enums))

	// Create type mapper
	mapper := dbparser.NewTypeMapper(parser.DatabaseType())
	mapperOpts := dbparser.MapperOptions{
		NullableAsPointer: !g.opts.StrictNullable,
		JSONAsRawMessage:  true,
	}

	// Map SQL types to Go types
	for _, table := range tables {
		dbparser.MapColumnsToGo(table, mapper, mapperOpts)
	}

	// Generate models
	if !g.opts.RepoOnly {
		if err := g.generateModels(); err != nil {
			return fmt.Errorf("failed to generate models: %w", err)
		}
	}

	// Generate repositories
	if !g.opts.ModelOnly {
		if err := g.generateRepositories(); err != nil {
			return fmt.Errorf("failed to generate repositories: %w", err)
		}
	}

	return nil
}

// generateModels generates Bun model files
func (g *Generator) generateModels() error {
	modelDir := filepath.Join(g.opts.OutputDir, g.opts.ModelPackage)
	if err := os.MkdirAll(modelDir, 0o755); err != nil {
		return fmt.Errorf("failed to create model directory: %w", err)
	}

	// Generate enum types first
	if len(g.schema.Enums) > 0 {
		if err := g.generateEnums(modelDir); err != nil {
			return err
		}
	}

	// Generate model for each table
	for _, table := range g.schema.Tables {
		if err := g.generateModel(modelDir, table); err != nil {
			return fmt.Errorf("failed to generate model for %s: %w", table.Name, err)
		}
	}

	return nil
}

// generateRepositories generates Bun repository files
func (g *Generator) generateRepositories() error {
	repoDir := filepath.Join(g.opts.OutputDir, g.opts.RepoPackage)
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		return fmt.Errorf("failed to create repository directory: %w", err)
	}

	// Generate base repository
	if err := g.generateBaseRepository(repoDir); err != nil {
		return err
	}

	// Generate repository for each table
	for _, table := range g.schema.Tables {
		if err := g.generateRepository(repoDir, table); err != nil {
			return fmt.Errorf("failed to generate repository for %s: %w", table.Name, err)
		}
	}

	return nil
}

func (g *Generator) logVerbose(format string, args ...interface{}) {
	if g.opts.Verbose {
		fmt.Fprintf(os.Stderr, format, args...)
	}
}
