// Package sqlc provides code generation for SQLC-based database layers
package sqlc

import (
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
)

// ConfigUpdater updates config.go with database configuration
type ConfigUpdater struct {
	outputDir string
	module    string
	verbose   bool
}

// NewConfigUpdater creates a new ConfigUpdater
func NewConfigUpdater(outputDir, module string, verbose bool) *ConfigUpdater {
	return &ConfigUpdater{
		outputDir: outputDir,
		module:    module,
		verbose:   verbose,
	}
}

// Update updates config.go to include DBConfig
func (u *ConfigUpdater) Update() error {
	configPath := filepath.Join(u.outputDir, "internal", "config", "config.go")

	content, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config.go: %w", err)
	}

	contentStr := string(content)

	// Check if DBConfig already exists
	if strings.Contains(contentStr, "DBConfig") {
		u.logVerbose("  DBConfig already exists in config.go\n")
		return nil
	}

	// Find Config struct
	structStart := strings.Index(contentStr, "type Config struct {")
	if structStart == -1 {
		return fmt.Errorf("config struct not found in config.go")
	}

	// Find end of struct opening
	structBodyStart := structStart + len("type Config struct {")

	// Add DB field after first newline in struct
	newlinePos := strings.Index(contentStr[structBodyStart:], "\n")
	if newlinePos == -1 {
		return fmt.Errorf("malformed Config struct")
	}
	insertPos := structBodyStart + newlinePos + 1

	dbField := "\tDB DBConfig `mapstructure:\"db\"`\n"

	// Add DBConfig struct definition before Config struct
	dbConfigDef := `// DBConfig holds database configuration
type DBConfig struct {
	Host     string ` + "`mapstructure:\"host\"`" + `
	Port     int    ` + "`mapstructure:\"port\"`" + `
	User     string ` + "`mapstructure:\"user\"`" + `
	Password string ` + "`mapstructure:\"password\"`" + `
	Database string ` + "`mapstructure:\"database\"`" + `
	SSLMode  string ` + "`mapstructure:\"sslmode\"`" + `
}

`

	// Insert DBConfig definition before Config struct
	newContent := contentStr[:structStart] + dbConfigDef + contentStr[structStart:insertPos] + dbField + contentStr[insertPos:]

	formatted, err := format.Source([]byte(newContent))
	if err != nil {
		u.logVerbose("  Warning: could not format config.go: %v\n", err)
		formatted = []byte(newContent)
	}

	if err := os.WriteFile(configPath, formatted, 0o644); err != nil { //nolint:gosec // configPath is constructed from trusted outputDir + known subpath
		return fmt.Errorf("failed to write config.go: %w", err)
	}

	u.logVerbose("  Updated config.go with DBConfig\n")
	return nil
}

// UpdateConfigYaml updates etc/config.yaml or etc/api.yaml to include db section
func (u *ConfigUpdater) UpdateConfigYaml() error {
	// Try common config file locations
	configPaths := []string{
		filepath.Join(u.outputDir, "etc", "api.yaml"),
		filepath.Join(u.outputDir, "etc", "config.yaml"),
		filepath.Join(u.outputDir, "etc", "dms.yaml"),
	}

	var configPath string
	for _, p := range configPaths {
		if _, err := os.Stat(p); err == nil {
			configPath = p
			break
		}
	}

	if configPath == "" {
		u.logVerbose("  etc/*.yaml config file not found, skipping\n")
		return nil
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		u.logVerbose("  %s not found, skipping\n", configPath)
		return nil
	}

	contentStr := string(content)

	// Check if db section already exists (not commented)
	if strings.Contains(contentStr, "\ndb:") {
		u.logVerbose("  db section already exists in config.yaml\n")
		return nil
	}

	// Add db section at the end
	dbSection := `
db:
  host: localhost
  port: 5432
  user: postgres
  password: postgres
  database: mydb
  sslmode: disable
`
	newContent := strings.TrimRight(contentStr, "\n") + "\n" + dbSection

	if err := os.WriteFile(configPath, []byte(newContent), 0o644); err != nil { //nolint:gosec // configPath is selected from a fixed set of known paths
		return fmt.Errorf("failed to write %s: %w", filepath.Base(configPath), err)
	}

	u.logVerbose("  Updated %s with db section\n", configPath)
	return nil
}

// logVerbose logs a message if verbose mode is enabled
func (u *ConfigUpdater) logVerbose(format string, args ...interface{}) {
	if u.verbose {
		fmt.Printf(format, args...)
	}
}
