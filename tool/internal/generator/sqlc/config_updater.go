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

	if err := safeWriteFile(u.outputDir, configPath, formatted, 0o644); err != nil {
		return fmt.Errorf("failed to write config.go: %w", err)
	}

	u.logVerbose("  Updated config.go with DBConfig\n")
	return nil
}

// UpdateConfigYaml updates all found config yamls (api.yaml, rpc.yaml, etc.) to include db section.
// It searches all candidate paths and updates every file found, supporting hybrid services.
func (u *ConfigUpdater) UpdateConfigYaml() error {
	// Candidate config file locations — all are checked and updated if found
	candidatePaths := []string{
		filepath.Join(u.outputDir, "etc", "api.yaml"),
		filepath.Join(u.outputDir, "etc", "rpc.yaml"),
		filepath.Join(u.outputDir, "etc", "config.yaml"),
		filepath.Join(u.outputDir, "etc", "dms.yaml"),
	}

	var found []string
	for _, p := range candidatePaths {
		if _, err := os.Stat(p); err == nil {
			found = append(found, p)
		}
	}

	if len(found) == 0 {
		u.logVerbose("  etc/*.yaml config file not found, skipping\n")
		return nil
	}

	for _, configPath := range found {
		if err := u.updateSingleConfigYaml(configPath); err != nil {
			return err
		}
	}

	return nil
}

// updateSingleConfigYaml updates one config yaml file to include db section if missing.
func (u *ConfigUpdater) updateSingleConfigYaml(configPath string) error {
	content, err := os.ReadFile(configPath)
	if err != nil {
		u.logVerbose("  %s not readable, skipping\n", configPath)
		return nil
	}

	contentStr := string(content)

	// Check if db section already exists (not commented)
	if strings.Contains(contentStr, "\ndb:") {
		u.logVerbose("  db section already exists in %s\n", filepath.Base(configPath))
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

	if err := safeWriteFile(u.outputDir, configPath, []byte(newContent), 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", filepath.Base(configPath), err)
	}

	u.logVerbose("  Updated %s with db section\n", filepath.Base(configPath))
	return nil
}

// logVerbose logs a message if verbose mode is enabled
func (u *ConfigUpdater) logVerbose(format string, args ...interface{}) {
	if u.verbose {
		fmt.Printf(format, args...)
	}
}
