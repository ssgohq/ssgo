// Package sqlc provides code generation for SQLC-based database layers
package sqlc

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Runner handles running the sqlc CLI
type Runner struct {
	workDir string
	verbose bool
}

// NewRunner creates a new Runner
func NewRunner(workDir string, verbose bool) *Runner {
	return &Runner{
		workDir: workDir,
		verbose: verbose,
	}
}

// CheckSqlcInstalled checks if sqlc CLI is installed
func (r *Runner) CheckSqlcInstalled() error {
	_, err := exec.LookPath("sqlc")
	if err != nil {
		return fmt.Errorf("sqlc CLI not found. Please install it: go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest")
	}
	return nil
}

// CheckSqlcConfig checks if sqlc.yaml exists
func (r *Runner) CheckSqlcConfig() error {
	configPath := filepath.Join(r.workDir, "sqlc.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("sqlc.yaml not found in %s. Run 'ss db sqlc init' first", r.workDir)
	}
	return nil
}

// Generate runs sqlc generate command
func (r *Runner) Generate() error {
	if err := r.CheckSqlcInstalled(); err != nil {
		return err
	}

	if err := r.CheckSqlcConfig(); err != nil {
		return err
	}

	cmd := exec.Command("sqlc", "generate")
	cmd.Dir = r.workDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if r.verbose {
		fmt.Printf("Running: sqlc generate (in %s)\n", r.workDir)
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("sqlc generate failed: %w", err)
	}

	return nil
}

// Verify runs sqlc verify command
func (r *Runner) Verify() error {
	if err := r.CheckSqlcInstalled(); err != nil {
		return err
	}

	if err := r.CheckSqlcConfig(); err != nil {
		return err
	}

	cmd := exec.Command("sqlc", "verify")
	cmd.Dir = r.workDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if r.verbose {
		fmt.Printf("Running: sqlc verify (in %s)\n", r.workDir)
	}

	return cmd.Run()
}

// Diff runs sqlc diff command
func (r *Runner) Diff() error {
	if err := r.CheckSqlcInstalled(); err != nil {
		return err
	}

	if err := r.CheckSqlcConfig(); err != nil {
		return err
	}

	cmd := exec.Command("sqlc", "diff")
	cmd.Dir = r.workDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if r.verbose {
		fmt.Printf("Running: sqlc diff (in %s)\n", r.workDir)
	}

	return cmd.Run()
}
