// Package sqlc provides code generation for SQLC-based database layers
package sqlc

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ssgohq/ssgo/tool/internal/generator/common"
)

// RepoGenOptions represents options for repository generation
type RepoGenOptions struct {
	OutputDir string // Output directory (project root)
	Module    string // Go module name (read from go.mod if empty)
	WithTrace bool   // Enable OpenTelemetry tracing
	Verbose   bool   // Verbose mode
}

// RepoGenerator generates wrapper repositories from SQLC queries
type RepoGenerator struct {
	opts RepoGenOptions
}

// NewRepoGenerator creates a new RepoGenerator
func NewRepoGenerator(opts RepoGenOptions) *RepoGenerator {
	return &RepoGenerator{opts: opts}
}

// GenerateFromSqlc generates repositories based on SQLC output
func (g *RepoGenerator) GenerateFromSqlc() error {
	dbDir := filepath.Join(g.opts.OutputDir, "internal", "data", "db")

	// Check if SQLC output exists
	if _, err := os.Stat(dbDir); os.IsNotExist(err) {
		return fmt.Errorf("SQLC output not found at %s. Run 'sqlc generate' first", dbDir)
	}

	// Read module from go.mod if not provided
	module := g.opts.Module
	if module == "" {
		var err error
		module, err = g.readModule()
		if err != nil {
			return err
		}
	}

	// Parse SQLC output
	parser := NewParser(dbDir, g.opts.Verbose)

	models, err := parser.ParseModels()
	if err != nil {
		return fmt.Errorf("failed to parse models: %w", err)
	}

	queries, err := parser.ParseQueries()
	if err != nil {
		return fmt.Errorf("failed to parse queries: %w", err)
	}

	if g.opts.Verbose {
		fmt.Printf("Found %d models and %d queries\n", len(models), len(queries))
	}

	// Group queries by entity (extract entity name from method name)
	entitiesMap := g.groupQueriesByEntity(queries)

	// Create repository directory
	repoDir := filepath.Join(g.opts.OutputDir, "internal", "repository")
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		return err
	}

	// Generate repositories for each entity
	for entityName, entityQueries := range entitiesMap {
		if err := g.generateRepository(module, entityName, entityQueries); err != nil {
			return fmt.Errorf("failed to generate repository for %s: %w", entityName, err)
		}
	}

	return nil
}

// groupQueriesByEntity groups queries by their associated entity
// It extracts entity name from method names like GetUserByID -> User
func (g *RepoGenerator) groupQueriesByEntity(queries []common.QueryInfo) map[string][]common.QueryInfo {
	result := make(map[string][]common.QueryInfo)

	// Common patterns: GetUser, CreateUser, ListUsers, DeleteUser, UpdateUser, CountUsers
	patterns := []struct {
		regex  *regexp.Regexp
		suffix string
	}{
		{regexp.MustCompile(`^(Get|Create|Update|Delete|Find)([A-Z][a-z]+)`), ""},
		{regexp.MustCompile(`^(List|Count)([A-Z][a-z]+)s$`), "s"},
	}

	for _, q := range queries {
		entityName := ""

		for _, p := range patterns {
			matches := p.regex.FindStringSubmatch(q.Name)
			if len(matches) >= 3 {
				entityName = matches[2]
				break
			}
		}

		// Fallback: use model name if we couldn't extract from method name
		if entityName == "" && q.ModelName != "" {
			entityName = g.extractEntityFromModel(q.ModelName)
		}

		// Skip if we still can't determine entity
		if entityName == "" {
			if g.opts.Verbose {
				fmt.Printf("  Warning: Could not determine entity for query %s\n", q.Name)
			}
			continue
		}

		result[entityName] = append(result[entityName], q)
	}

	return result
}

// extractEntityFromModel extracts entity name from SQLC model name
// e.g., LineartUser -> User, SummarifyTask -> Task
func (g *RepoGenerator) extractEntityFromModel(modelName string) string {
	// Remove common prefixes (schema names)
	prefixes := []string{"Lineart", "Summarify", "Nikki", "Onesystem"}
	for _, prefix := range prefixes {
		if strings.HasPrefix(modelName, prefix) {
			return strings.TrimPrefix(modelName, prefix)
		}
	}
	return modelName
}

// generateRepository generates a repository file for an entity
func (g *RepoGenerator) generateRepository(module, entityName string, queries []common.QueryInfo) error {
	var buf bytes.Buffer

	lowerEntity := strings.ToLower(entityName)
	snakeEntity := common.ToSnakeCase(entityName)

	// Package declaration
	buf.WriteString("package repository\n\n")

	// Imports
	buf.WriteString("import (\n")
	buf.WriteString("\t\"context\"\n\n")
	buf.WriteString(fmt.Sprintf("\t\"%s/internal/data/db\"\n", module))
	buf.WriteString(fmt.Sprintf("\t\"%s/internal/store\"\n", module))
	if g.opts.WithTrace {
		buf.WriteString("\n\t\"go.opentelemetry.io/otel\"\n")
	}
	buf.WriteString(")\n\n")

	// Interface
	buf.WriteString(
		fmt.Sprintf(
			"// %sRepository defines the interface for %s data access\n",
			entityName,
			lowerEntity,
		),
	)
	buf.WriteString(fmt.Sprintf("type %sRepository interface {\n", entityName))
	for _, q := range queries {
		sig := g.buildMethodSignature(q)
		buf.WriteString(fmt.Sprintf("\t%s\n", sig))
	}
	buf.WriteString("}\n\n")

	// Struct
	buf.WriteString(fmt.Sprintf("type %sRepository struct {\n", lowerEntity))
	buf.WriteString("\tstore *store.Store\n")
	buf.WriteString("}\n\n")

	// Constructor
	buf.WriteString(
		fmt.Sprintf("// New%sRepository creates a new %s repository\n", entityName, lowerEntity),
	)
	buf.WriteString(
		fmt.Sprintf(
			"func New%sRepository(store *store.Store) %sRepository {\n",
			entityName,
			entityName,
		),
	)
	buf.WriteString(fmt.Sprintf("\treturn &%sRepository{\n", lowerEntity))
	buf.WriteString("\t\tstore: store,\n")
	buf.WriteString("\t}\n")
	buf.WriteString("}\n\n")

	// Method implementations
	for _, q := range queries {
		buf.WriteString(g.generateMethod(lowerEntity, entityName, q))
		buf.WriteString("\n")
	}

	// Format and write
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		if g.opts.Verbose {
			fmt.Printf("  Warning: could not format %s_repository.go: %v\n", snakeEntity, err)
		}
		formatted = buf.Bytes()
	}

	fileName := snakeEntity + "_repository.go"
	filePath := filepath.Join(g.opts.OutputDir, "internal", "repository", fileName)

	if g.opts.Verbose {
		fmt.Printf("  Generating %s\n", filePath)
	}

	return os.WriteFile(filePath, formatted, 0o644)
}

// buildMethodSignature builds the method signature for interface
func (g *RepoGenerator) buildMethodSignature(q common.QueryInfo) string {
	params := "ctx context.Context"
	for _, p := range q.Params {
		paramType := g.prefixDBType(p.Type)
		params += fmt.Sprintf(", %s %s", p.Name, paramType)
	}

	if q.IsExec {
		return fmt.Sprintf("%s(%s) error", q.Name, params)
	}
	returnType := g.prefixDBType(q.ReturnType)
	return fmt.Sprintf("%s(%s) (%s, error)", q.Name, params, returnType)
}

// generateMethod generates a repository method implementation
func (g *RepoGenerator) generateMethod(lowerEntity, entityName string, q common.QueryInfo) string {
	var buf bytes.Buffer

	params := "ctx context.Context"
	for _, p := range q.Params {
		paramType := g.prefixDBType(p.Type)
		params += fmt.Sprintf(", %s %s", p.Name, paramType)
	}

	var returns string
	if q.IsExec {
		returns = "error"
	} else {
		returnType := g.prefixDBType(q.ReturnType)
		returns = fmt.Sprintf("(%s, error)", returnType)
	}

	// Method signature
	buf.WriteString(
		fmt.Sprintf("func (r *%sRepository) %s(%s) %s {\n", lowerEntity, q.Name, params, returns),
	)

	// Add tracing if enabled
	if g.opts.WithTrace {
		tracerName := fmt.Sprintf("%sRepository", lowerEntity)
		buf.WriteString(
			fmt.Sprintf("\tctx, span := otel.Tracer(%q).Start(ctx, %q)\n", tracerName, q.Name),
		)
		buf.WriteString("\tdefer span.End()\n\n")
	}

	// Build query call arguments
	var callArgs []string
	callArgs = append(callArgs, "ctx")
	for _, p := range q.Params {
		callArgs = append(callArgs, p.Name)
	}
	argsStr := strings.Join(callArgs, ", ")

	// Call the SQLC query
	buf.WriteString(fmt.Sprintf("\treturn r.store.Queries().%s(%s)\n", q.Name, argsStr))

	buf.WriteString("}\n")
	return buf.String()
}

// readModule reads the module name from go.mod
func (g *RepoGenerator) readModule() (string, error) {
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

// prefixDBType adds db. prefix to types that need it
func (g *RepoGenerator) prefixDBType(typeName string) string {
	if typeName == "" {
		return typeName
	}

	// Handle slice types
	if strings.HasPrefix(typeName, "[]") {
		inner := strings.TrimPrefix(typeName, "[]")
		return "[]" + g.prefixDBType(inner)
	}

	// Handle pointer types
	if strings.HasPrefix(typeName, "*") {
		inner := strings.TrimPrefix(typeName, "*")
		return "*" + g.prefixDBType(inner)
	}

	// Skip primitive types
	primitives := map[string]bool{
		"int": true, "int8": true, "int16": true, "int32": true, "int64": true,
		"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true,
		"float32": true, "float64": true,
		"string": true, "bool": true, "byte": true, "rune": true,
		"error": true, "interface{}": true,
	}

	if primitives[typeName] {
		return typeName
	}

	// Skip types that already have a package prefix
	if strings.Contains(typeName, ".") {
		return typeName
	}

	// Add db. prefix for custom types
	return "db." + typeName
}
