package sqlc

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitGenerator_Generate(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()

	// Create go.mod file (required for init)
	gomodContent := `module github.com/example/testservice

go 1.21
`
	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(gomodContent), 0o644)
	if err != nil {
		t.Fatalf("failed to create go.mod: %v", err)
	}

	// Run the init generator
	gen := NewInitGenerator(InitOptions{
		OutputDir:     tmpDir,
		MigrationPath: "../migrations",
		SchemaName:    "testschema",
		SampleEntity:  "User",
		DBType:        "postgres",
		Verbose:       false,
	})

	err = gen.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Verify sqlc.yaml was created
	sqlcYamlPath := filepath.Join(tmpDir, "sqlc.yaml")
	if _, err := os.Stat(sqlcYamlPath); os.IsNotExist(err) {
		t.Error("sqlc.yaml was not created")
	} else {
		content, _ := os.ReadFile(sqlcYamlPath)
		if !strings.Contains(string(content), "postgresql") {
			t.Error("sqlc.yaml does not contain postgresql engine")
		}
		if !strings.Contains(string(content), "../migrations") {
			t.Error("sqlc.yaml does not contain migration path")
		}
	}

	// Verify query directory was created
	queryDir := filepath.Join(tmpDir, "query")
	if _, err := os.Stat(queryDir); os.IsNotExist(err) {
		t.Error("query directory was not created")
	}

	// Verify sample query was created
	sampleQueryPath := filepath.Join(tmpDir, "query", "user.sql")
	if _, err := os.Stat(sampleQueryPath); os.IsNotExist(err) {
		t.Error("sample query file was not created")
	} else {
		content, _ := os.ReadFile(sampleQueryPath)
		if !strings.Contains(string(content), "GetUserByID") {
			t.Error("sample query does not contain GetUserByID")
		}
		if !strings.Contains(string(content), "testschema.users") {
			t.Error("sample query does not contain schema-qualified table name")
		}
	}

	// Verify store directory was created
	storeDir := filepath.Join(tmpDir, "internal", "store")
	if _, err := os.Stat(storeDir); os.IsNotExist(err) {
		t.Error("store directory was not created")
	}

	// Verify store.go was created
	storeGoPath := filepath.Join(tmpDir, "internal", "store", "store.go")
	if _, err := os.Stat(storeGoPath); os.IsNotExist(err) {
		t.Error("store.go was not created")
	} else {
		content, _ := os.ReadFile(storeGoPath)
		if !strings.Contains(string(content), "ExecTx") {
			t.Error("store.go does not contain ExecTx function")
		}
	}
}

func TestInitGenerator_RequiresGoMod(t *testing.T) {
	// Create a temporary directory without go.mod
	tmpDir := t.TempDir()

	gen := NewInitGenerator(InitOptions{
		OutputDir: tmpDir,
		Verbose:   false,
	})

	err := gen.Generate()
	if err == nil {
		t.Error("Generate() should fail without go.mod")
	}
	if !strings.Contains(err.Error(), "go.mod not found") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestInitGenerator_MySQLSupport(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()

	// Create go.mod file
	gomodContent := `module github.com/example/testservice

go 1.21
`
	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(gomodContent), 0o644)
	if err != nil {
		t.Fatalf("failed to create go.mod: %v", err)
	}

	// Run the init generator with MySQL
	gen := NewInitGenerator(InitOptions{
		OutputDir:    tmpDir,
		SampleEntity: "User",
		DBType:       "mysql",
		Verbose:      false,
	})

	err = gen.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Verify sqlc.yaml contains MySQL config
	sqlcYamlPath := filepath.Join(tmpDir, "sqlc.yaml")
	content, _ := os.ReadFile(sqlcYamlPath)
	if !strings.Contains(string(content), "mysql") {
		t.Error("sqlc.yaml does not contain mysql engine")
	}

	// Verify sample query uses MySQL placeholder syntax
	sampleQueryPath := filepath.Join(tmpDir, "query", "user.sql")
	queryContent, _ := os.ReadFile(sampleQueryPath)
	if !strings.Contains(string(queryContent), "WHERE id = ?") {
		t.Error("MySQL query should use ? placeholder")
	}
}

func TestInitGenerator_DoesNotOverwrite(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()

	// Create go.mod file
	gomodContent := `module github.com/example/testservice

go 1.21
`
	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(gomodContent), 0o644)
	if err != nil {
		t.Fatalf("failed to create go.mod: %v", err)
	}

	// Create an existing sqlc.yaml
	existingContent := "# existing content\n"
	os.MkdirAll(filepath.Dir(filepath.Join(tmpDir, "sqlc.yaml")), 0o755)
	os.WriteFile(filepath.Join(tmpDir, "sqlc.yaml"), []byte(existingContent), 0o644)

	// Run the init generator
	gen := NewInitGenerator(InitOptions{
		OutputDir: tmpDir,
		Verbose:   false,
	})

	err = gen.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Verify sqlc.yaml was not overwritten
	content, _ := os.ReadFile(filepath.Join(tmpDir, "sqlc.yaml"))
	if string(content) != existingContent {
		t.Error("sqlc.yaml was overwritten when it should not have been")
	}
}
