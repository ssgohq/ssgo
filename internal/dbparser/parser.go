package dbparser

import (
	"context"
	"fmt"
	"strings"
)

// DatabaseType represents supported database types
type DatabaseType string

const (
	DatabaseTypePostgres DatabaseType = "postgres"
	DatabaseTypeMySQL    DatabaseType = "mysql"
)

// Parser defines the interface for database introspection
type Parser interface {
	// Connect establishes a connection to the database
	Connect(ctx context.Context, dsn string) error

	// Close closes the database connection
	Close() error

	// ParseSchema parses the entire schema
	// schemaName: "public" for PostgreSQL, database name for MySQL
	ParseSchema(ctx context.Context, schemaName string, opts ParseOptions) (*Schema, error)

	// ParseTables parses specific tables
	// If tableNames is nil or empty, parses all tables
	ParseTables(ctx context.Context, schemaName string, tableNames []string, opts ParseOptions) ([]*Table, error)

	// ListTables returns all table names in the schema
	ListTables(ctx context.Context, schemaName string) ([]string, error)

	// GetEnums returns all enum types (PostgreSQL only, returns nil for MySQL)
	GetEnums(ctx context.Context, schemaName string) ([]*Enum, error)

	// DatabaseType returns the database type
	DatabaseType() DatabaseType
}

// ParseOptions configures parsing behavior
type ParseOptions struct {
	// IncludeViews includes database views (default: false)
	IncludeViews bool

	// ExcludeTables patterns to exclude (glob patterns supported: *_backup, temp_*)
	ExcludeTables []string

	// IncludeSystemTables includes system tables like schema_migrations (default: false)
	IncludeSystemTables bool

	// Verbose enables detailed logging
	Verbose bool
}

// DefaultParseOptions returns default parse options
func DefaultParseOptions() ParseOptions {
	return ParseOptions{
		IncludeViews:        false,
		ExcludeTables:       nil,
		IncludeSystemTables: false,
		Verbose:             false,
	}
}

// DefaultSystemTables returns common system tables to exclude
func DefaultSystemTables() []string {
	return []string{
		"schema_migrations",
		"goose_db_version",
		"flyway_schema_history",
		"ar_internal_metadata",
		"spatial_ref_sys",
		"__diesel_schema_migrations",
		"_prisma_migrations",
		"knex_migrations",
		"knex_migrations_lock",
		"typeorm_metadata",
	}
}

// NewParser creates a parser based on DSN
// DSN format:
//   - PostgreSQL: postgres://user:pass@host:5432/dbname?sslmode=disable
//   - MySQL: user:pass@tcp(host:3306)/dbname?parseTime=true
func NewParser(dsn string) (Parser, error) {
	dbType := DetectDatabaseType(dsn)

	switch dbType {
	case DatabaseTypePostgres:
		return newPostgresParser(), nil
	case DatabaseTypeMySQL:
		return newMySQLParser(), nil
	default:
		return nil, fmt.Errorf("unsupported database type for DSN: %s", dsn)
	}
}

// DetectDatabaseType detects database type from DSN
func DetectDatabaseType(dsn string) DatabaseType {
	lowerDSN := strings.ToLower(dsn)

	// PostgreSQL patterns
	if strings.HasPrefix(lowerDSN, "postgres://") ||
		strings.HasPrefix(lowerDSN, "postgresql://") {
		return DatabaseTypePostgres
	}

	// PostgreSQL key=value format
	if strings.Contains(lowerDSN, "host=") && strings.Contains(lowerDSN, "sslmode=") {
		return DatabaseTypePostgres
	}

	// MySQL patterns
	if strings.Contains(lowerDSN, "@tcp(") ||
		strings.Contains(lowerDSN, "@unix(") {
		return DatabaseTypeMySQL
	}

	if strings.HasPrefix(lowerDSN, "mysql://") {
		return DatabaseTypeMySQL
	}

	return ""
}

// MatchesGlobPattern checks if name matches any of the glob patterns
// Supports: prefix*, *suffix, *contains*
func MatchesGlobPattern(name string, patterns []string) bool {
	for _, pattern := range patterns {
		if matchGlob(name, pattern) {
			return true
		}
	}
	return false
}

func matchGlob(name, pattern string) bool {
	// Exact match
	if name == pattern {
		return true
	}

	// No wildcard
	if !strings.Contains(pattern, "*") {
		return false
	}

	// *suffix
	if strings.HasPrefix(pattern, "*") && !strings.HasSuffix(pattern, "*") {
		suffix := strings.TrimPrefix(pattern, "*")
		return strings.HasSuffix(name, suffix)
	}

	// prefix*
	if strings.HasSuffix(pattern, "*") && !strings.HasPrefix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(name, prefix)
	}

	// *contains*
	if strings.HasPrefix(pattern, "*") && strings.HasSuffix(pattern, "*") {
		contains := strings.Trim(pattern, "*")
		return strings.Contains(name, contains)
	}

	return false
}

// FilterTables filters tables based on options
func FilterTables(tables []string, opts ParseOptions) []string {
	excludeSet := make(map[string]bool)

	// Add explicit excludes
	for _, t := range opts.ExcludeTables {
		if !strings.Contains(t, "*") {
			excludeSet[t] = true
		}
	}

	// Add system tables if not included
	if !opts.IncludeSystemTables {
		for _, t := range DefaultSystemTables() {
			excludeSet[t] = true
		}
	}

	var filtered []string
	for _, t := range tables {
		// Skip if in exclude set
		if excludeSet[t] {
			continue
		}

		// Skip if matches glob pattern
		if MatchesGlobPattern(t, opts.ExcludeTables) {
			continue
		}

		filtered = append(filtered, t)
	}

	return filtered
}

// Package-level parser constructors (implemented in postgres/ and mysql/ packages)
var (
	newPostgresParser func() Parser
	newMySQLParser    func() Parser
)

// RegisterPostgresParser registers the PostgreSQL parser constructor
func RegisterPostgresParser(constructor func() Parser) {
	newPostgresParser = constructor
}

// RegisterMySQLParser registers the MySQL parser constructor
func RegisterMySQLParser(constructor func() Parser) {
	newMySQLParser = constructor
}
