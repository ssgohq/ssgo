package sqlc

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ssgohq/ssgo/tool/internal/generator/common"
)

func TestParser_ParseModels(t *testing.T) {
	// Create a temporary directory with test files
	tmpDir := t.TempDir()

	// Create a sample models.go file
	modelsContent := `package db

import (
	"time"
)

type User struct {
	ID        int64     ` + "`json:\"id\"`" + `
	Name      string    ` + "`json:\"name\"`" + `
	Email     string    ` + "`json:\"email\"`" + `
	CreatedAt time.Time ` + "`json:\"created_at\"`" + `
}

type Task struct {
	ID          int64   ` + "`json:\"id\"`" + `
	Title       string  ` + "`json:\"title\"`" + `
	Description *string ` + "`json:\"description\"`" + `
	UserID      int64   ` + "`json:\"user_id\"`" + `
}
`
	err := os.WriteFile(filepath.Join(tmpDir, "models.go"), []byte(modelsContent), 0o644)
	if err != nil {
		t.Fatalf("failed to write test models.go: %v", err)
	}

	parser := NewParser(tmpDir, false)
	models, err := parser.ParseModels()
	if err != nil {
		t.Fatalf("ParseModels() error = %v", err)
	}

	if len(models) != 2 {
		t.Errorf("ParseModels() returned %d models, want 2", len(models))
	}

	// Check User model
	var userModel *ModelInfo
	for _, m := range models {
		if m.Name == "User" {
			userModel = &m
			break
		}
	}

	if userModel == nil {
		t.Fatal("User model not found")
	}

	if len(userModel.Fields) != 4 {
		t.Errorf("User model has %d fields, want 4", len(userModel.Fields))
	}
}

func TestParser_ParseQueries(t *testing.T) {
	// Create a temporary directory with test files
	tmpDir := t.TempDir()

	// Create a sample querier.go file
	querierContent := `package db

import (
	"context"
)

type Querier interface {
	GetUserByID(ctx context.Context, id int64) (User, error)
	ListUsers(ctx context.Context, limit int32, offset int32) ([]User, error)
	CreateUser(ctx context.Context, arg CreateUserParams) (User, error)
	DeleteUser(ctx context.Context, id int64) error
	CountUsers(ctx context.Context) (int64, error)
}
`
	err := os.WriteFile(filepath.Join(tmpDir, "querier.go"), []byte(querierContent), 0o644)
	if err != nil {
		t.Fatalf("failed to write test querier.go: %v", err)
	}

	parser := NewParser(tmpDir, false)
	queries, err := parser.ParseQueries()
	if err != nil {
		t.Fatalf("ParseQueries() error = %v", err)
	}

	if len(queries) != 5 {
		t.Errorf("ParseQueries() returned %d queries, want 5", len(queries))
	}

	// Check GetUserByID query
	var getUserQuery *common.QueryInfo
	for i, q := range queries {
		if q.Name == "GetUserByID" {
			getUserQuery = &queries[i]
			break
		}
	}

	if getUserQuery == nil {
		t.Fatal("GetUserByID query not found")
	}

	if getUserQuery.ReturnType != "User" {
		t.Errorf("GetUserByID.ReturnType = %q, want %q", getUserQuery.ReturnType, "User")
	}

	if getUserQuery.IsMany {
		t.Error("GetUserByID.IsMany = true, want false")
	}

	if len(getUserQuery.Params) != 1 {
		t.Errorf("GetUserByID has %d params, want 1", len(getUserQuery.Params))
	}

	// Check ListUsers query
	var listUsersQuery *common.QueryInfo
	for i, q := range queries {
		if q.Name == "ListUsers" {
			listUsersQuery = &queries[i]
			break
		}
	}

	if listUsersQuery == nil {
		t.Fatal("ListUsers query not found")
	}

	if !listUsersQuery.IsMany {
		t.Error("ListUsers.IsMany = false, want true")
	}

	// Check DeleteUser query (exec)
	var deleteUserQuery *common.QueryInfo
	for i, q := range queries {
		if q.Name == "DeleteUser" {
			deleteUserQuery = &queries[i]
			break
		}
	}

	if deleteUserQuery == nil {
		t.Fatal("DeleteUser query not found")
	}

	if !deleteUserQuery.IsExec {
		t.Error("DeleteUser.IsExec = false, want true")
	}
}

func TestParser_ModelsNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	parser := NewParser(tmpDir, false)
	_, err := parser.ParseModels()
	if err == nil {
		t.Error("ParseModels() expected error for missing models.go")
	}
}

func TestParser_QuerierNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	parser := NewParser(tmpDir, false)
	_, err := parser.ParseQueries()
	if err == nil {
		t.Error("ParseQueries() expected error for missing querier.go")
	}
}

func TestParser_GetModelByName(t *testing.T) {
	models := []ModelInfo{
		{Name: "User"},
		{Name: "Task"},
	}

	parser := NewParser("", false)

	// Test finding existing model
	found := parser.GetModelByName(models, "User")
	if found == nil {
		t.Error("GetModelByName(\"User\") returned nil, want *ModelInfo")
	}
	if found != nil && found.Name != "User" {
		t.Errorf("GetModelByName(\"User\").Name = %q, want \"User\"", found.Name)
	}

	// Test finding non-existing model
	notFound := parser.GetModelByName(models, "NonExistent")
	if notFound != nil {
		t.Error("GetModelByName(\"NonExistent\") returned non-nil, want nil")
	}
}
