// Package sqlc provides code generation for SQLC-based database layers
package sqlc

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/ssgohq/ssgo/tool/internal/generator/common"
	"github.com/ssgohq/ssgo/tool/internal/generator/templates"
)

// InitOptions represents options for initializing SQLC directories
type InitOptions struct {
	OutputDir     string // Output directory (existing service)
	MigrationPath string // Path to external migration project or migrations directory
	SchemaName    string // Schema name for queries (e.g., lineart)
	SampleEntity  string // Optional: entity name for sample query
	DBType        string // Database type (postgres or mysql)
	Verbose       bool   // Verbose mode
}

// InitGenerator generates initial SQLC directories (query/, internal/data/db/, internal/store/)
type InitGenerator struct {
	opts    InitOptions
	funcMap template.FuncMap
}

// NewInitGenerator creates a new InitGenerator
func NewInitGenerator(opts InitOptions) *InitGenerator {
	if opts.DBType == "" {
		opts.DBType = "postgres"
	}
	g := &InitGenerator{opts: opts}
	g.funcMap = g.createFuncMap()
	return g
}

// createFuncMap creates template function map
func (g *InitGenerator) createFuncMap() template.FuncMap {
	return template.FuncMap{
		"ToSnakeCase":  common.ToSnakeCase,
		"ToCamelCase":  common.ToCamelCase,
		"ToPascalCase": common.ToPascalCase,
		"ToKebabCase":  common.ToKebabCase,
		"lower":        strings.ToLower,
		"upper":        strings.ToUpper,
	}
}

// Generate creates SQLC-specific directories and configuration
func (g *InitGenerator) Generate() error {
	fmt.Printf("Initializing SQLC in project...\n")
	fmt.Printf("  Output:     %s\n", g.opts.OutputDir)
	if g.opts.MigrationPath != "" {
		fmt.Printf("  Migrations: %s\n", g.opts.MigrationPath)
	}
	if g.opts.SchemaName != "" {
		fmt.Printf("  Schema:     %s\n", g.opts.SchemaName)
	}
	if g.opts.SampleEntity != "" {
		fmt.Printf("  Sample:     %s\n", g.opts.SampleEntity)
	}
	fmt.Println()

	// Check if go.mod exists (project must exist)
	gomodPath := filepath.Join(g.opts.OutputDir, "go.mod")
	if _, err := os.Stat(gomodPath); os.IsNotExist(err) {
		return fmt.Errorf(
			"go.mod not found in %s. Create a service first before adding SQLC support",
			g.opts.OutputDir,
		)
	}

	// Create SQLC-specific directories
	dirs := []string{
		filepath.Join(g.opts.OutputDir, "query"),
		filepath.Join(g.opts.OutputDir, "internal", "data", "db"),
		filepath.Join(g.opts.OutputDir, "internal", "store"),
		filepath.Join(g.opts.OutputDir, "internal", "repository"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
		if g.opts.Verbose {
			fmt.Printf("  Created %s\n", dir)
		}
	}

	// Generate sqlc.yaml
	if err := g.generateSqlcYaml(); err != nil {
		return fmt.Errorf("failed to generate sqlc.yaml: %w", err)
	}

	// Generate store.go
	if err := g.generateStore(); err != nil {
		return fmt.Errorf("failed to generate store.go: %w", err)
	}

	// Generate sample query if entity name is provided
	if g.opts.SampleEntity != "" {
		if err := g.generateSampleQuery(); err != nil {
			return fmt.Errorf("failed to generate sample query: %w", err)
		}
	}

	g.printSuccess()
	return nil
}

// generateSqlcYaml generates sqlc.yaml configuration
func (g *InitGenerator) generateSqlcYaml() error {
	engine := "postgresql"
	sqlPackage := "pgx/v5"
	if g.opts.DBType == "mysql" {
		engine = "mysql"
		sqlPackage = "database/sql"
	}

	// Determine schema path
	schemaPath := g.opts.MigrationPath
	if schemaPath == "" {
		// Create schema directory if no migrations path provided
		schemaPath = "schema"
		schemaDir := filepath.Join(g.opts.OutputDir, "schema")
		if err := os.MkdirAll(schemaDir, 0o755); err != nil {
			return fmt.Errorf("failed to create schema directory: %w", err)
		}

		// Generate sample schema file
		if err := g.generateSampleSchema(schemaDir); err != nil {
			return fmt.Errorf("failed to generate sample schema: %w", err)
		}
	}

	data := struct {
		Engine     string
		SQLPackage string
		SchemaPath string
	}{
		Engine:     engine,
		SQLPackage: sqlPackage,
		SchemaPath: schemaPath,
	}

	content, err := g.executeTemplate("sqlc/sqlc.yaml.tpl", data)
	if err != nil {
		return err
	}

	path := filepath.Join(g.opts.OutputDir, "sqlc.yaml")

	// Don't overwrite existing
	if _, err := os.Stat(path); err == nil {
		if g.opts.Verbose {
			fmt.Printf("  %s already exists, skipping...\n", path)
		}
		return nil
	}

	if g.opts.Verbose {
		fmt.Printf("  Creating %s\n", path)
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

// generateSampleSchema generates a sample schema file
func (g *InitGenerator) generateSampleSchema(schemaDir string) error {
	// Determine table name from entity or use default
	entityName := g.opts.SampleEntity
	if entityName == "" {
		entityName = "User"
	}
	entity := common.NewEntityInfo(entityName)

	tableName := entity.SnakeName + "s"
	schemaPrefix := ""
	if g.opts.SchemaName != "" {
		schemaPrefix = g.opts.SchemaName + "."
	}

	var content string
	if g.opts.DBType == "mysql" {
		content = fmt.Sprintf(`-- Schema for %s
-- Generated by ss db sqlc init
-- Edit this file to define your database schema

CREATE TABLE IF NOT EXISTS %s%s (
		  id INT AUTO_INCREMENT PRIMARY KEY,
		  name VARCHAR(255) NOT NULL,
		  email VARCHAR(255) UNIQUE NOT NULL,
		  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- Add more tables below as needed
`, entity.Name, schemaPrefix, tableName)
	} else {
		content = fmt.Sprintf(`-- Schema for %s
-- Generated by ss db sqlc init
-- Edit this file to define your database schema

%sCREATE TABLE IF NOT EXISTS %s%s (
		  id SERIAL PRIMARY KEY,
		  name VARCHAR(255) NOT NULL,
		  email VARCHAR(255) UNIQUE NOT NULL,
		  created_at TIMESTAMPTZ DEFAULT NOW(),
		  updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Add more tables below as needed
`, entity.Name, g.createSchemaSQL(), schemaPrefix, tableName)
	}

	path := filepath.Join(schemaDir, "001_init.sql")

	// Don't overwrite existing
	if _, err := os.Stat(path); err == nil {
		if g.opts.Verbose {
			fmt.Printf("  %s already exists, skipping...\n", path)
		}
		return nil
	}

	if g.opts.Verbose {
		fmt.Printf("  Creating %s\n", path)
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

// createSchemaSQL generates CREATE SCHEMA SQL if schema name is provided
func (g *InitGenerator) createSchemaSQL() string {
	if g.opts.SchemaName == "" {
		return ""
	}
	return fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s;\n\n", g.opts.SchemaName)
}

// generateStore generates internal/store/store.go
func (g *InitGenerator) generateStore() error {
	// Read module from go.mod
	module, err := g.readModule()
	if err != nil {
		return err
	}

	data := struct {
		Module string
	}{
		Module: module,
	}

	content, err := g.executeTemplate("sqlc/store.go.tpl", data)
	if err != nil {
		return err
	}

	path := filepath.Join(g.opts.OutputDir, "internal", "store", "store.go")

	// Don't overwrite existing
	if _, err := os.Stat(path); err == nil {
		if g.opts.Verbose {
			fmt.Printf("  %s already exists, skipping...\n", path)
		}
		return nil
	}

	if g.opts.Verbose {
		fmt.Printf("  Creating %s\n", path)
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

// generateSampleQuery generates a sample query file for the given entity
func (g *InitGenerator) generateSampleQuery() error {
	entity := common.NewEntityInfo(g.opts.SampleEntity)
	isPostgres := g.opts.DBType == "postgres"

	tableName := entity.SnakeName + "s"

	data := struct {
		Entity     common.EntityInfo
		TableName  string
		IsPostgres bool
		SchemaName string
	}{
		Entity:     entity,
		TableName:  tableName,
		IsPostgres: isPostgres,
		SchemaName: g.opts.SchemaName,
	}

	content, err := g.executeTemplate("sqlc/sample_query.sql.tpl", data)
	if err != nil {
		return err
	}

	path := filepath.Join(g.opts.OutputDir, "query", entity.SnakeName+".sql")

	// Don't overwrite existing
	if _, err := os.Stat(path); err == nil {
		if g.opts.Verbose {
			fmt.Printf("  %s already exists, skipping...\n", path)
		}
		return nil
	}

	if g.opts.Verbose {
		fmt.Printf("  Creating %s\n", path)
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

// executeTemplate executes a template and returns the result
func (g *InitGenerator) executeTemplate(tplPath string, data interface{}) (string, error) {
	content, err := templates.SQLCTemplates.ReadFile(tplPath)
	if err != nil {
		return "", fmt.Errorf("failed to read template %s: %w", tplPath, err)
	}

	tpl, err := template.New(filepath.Base(tplPath)).Funcs(g.funcMap).Parse(string(content))
	if err != nil {
		return "", fmt.Errorf("failed to parse template %s: %w", tplPath, err)
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", tplPath, err)
	}

	return buf.String(), nil
}

// readModule reads the module name from go.mod
func (g *InitGenerator) readModule() (string, error) {
	gomodPath := filepath.Join(g.opts.OutputDir, "go.mod")
	data, err := os.ReadFile(gomodPath)
	if err != nil {
		return "", fmt.Errorf("go.mod not found: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimPrefix(line, "module "), nil
		}
	}

	return "", fmt.Errorf("could not parse module from go.mod")
}

// printSuccess prints the success message
func (g *InitGenerator) printSuccess() {
	fmt.Println()
	fmt.Println("✓ SQLC initialized successfully!")
	fmt.Println()
	fmt.Printf("Created structure:\n")
	fmt.Printf("  %s/\n", g.opts.OutputDir)
	fmt.Printf("  ├── sqlc.yaml           # SQLC configuration\n")
	fmt.Printf("  ├── query/              # SQL queries\n")
	if g.opts.SampleEntity != "" {
		entity := common.NewEntityInfo(g.opts.SampleEntity)
		fmt.Printf("  │   └── %s.sql\n", entity.SnakeName)
	}
	if g.opts.MigrationPath == "" {
		fmt.Printf("  ├── schema/             # Database schema\n")
		fmt.Printf("  │   └── 001_init.sql\n")
	}
	fmt.Printf("  └── internal/\n")
	fmt.Printf("      ├── data/db/        # Generated by 'sqlc generate'\n")
	fmt.Printf("      ├── repository/     # Generated by 'ss db sqlc gen'\n")
	fmt.Printf("      └── store/\n")
	fmt.Printf("          └── store.go   # Store with transaction support\n")
	fmt.Println()
	fmt.Println("Next steps:")
	if g.opts.MigrationPath == "" {
		fmt.Println("  1. Edit schema/001_init.sql with your table definitions")
		fmt.Println("  2. Add your SQL queries to query/*.sql")
		fmt.Println("  3. Run: sqlc generate")
		fmt.Println("  4. Run: ss db sqlc gen --dir " + g.opts.OutputDir)
	} else {
		fmt.Println("  1. Add your SQL queries to query/*.sql")
		fmt.Println("  2. Run: sqlc generate")
		fmt.Println("  3. Run: ss db sqlc gen --dir " + g.opts.OutputDir)
		fmt.Println()
		fmt.Println("Schema referenced from: " + g.opts.MigrationPath)
	}
}
