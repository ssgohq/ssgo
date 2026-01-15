package gorm

// GenOptions configures the GORM generator
type GenOptions struct {
	// DSN is the database connection string
	DSN string

	// SchemaName is the database schema (default: "public" for postgres)
	SchemaName string

	// Tables specifies which tables to generate (nil = all)
	// Supports glob patterns: user*, *_log
	Tables []string

	// ExcludeTables patterns to exclude
	ExcludeTables []string

	// OutputDir is the output directory for generated files
	OutputDir string

	// ModuleName is the Go module name for imports
	ModuleName string

	// ModelPackage is the package name for models (default: "model")
	ModelPackage string

	// RepoPackage is the package name for repositories (default: "repository")
	RepoPackage string

	// ModelOnly generates only models, not repositories
	ModelOnly bool

	// RepoOnly generates only repositories, not models
	RepoOnly bool

	// StrictNullable uses sql.Null* types instead of pointers for nullable
	StrictNullable bool

	// WithTrace adds OpenTelemetry tracing to repositories
	WithTrace bool

	// Verbose enables detailed logging
	Verbose bool

	// SoftDelete enables gorm soft delete (adds DeletedAt field)
	SoftDelete bool

	// WithHooks generates hook methods (BeforeCreate, etc.)
	WithHooks bool
}

// DefaultGenOptions returns sensible defaults
func DefaultGenOptions() GenOptions {
	return GenOptions{
		SchemaName:   "public",
		OutputDir:    ".",
		ModelPackage: "model",
		RepoPackage:  "repository",
		SoftDelete:   true,
	}
}
