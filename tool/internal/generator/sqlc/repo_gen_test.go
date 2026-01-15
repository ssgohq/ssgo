package sqlc

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ssgohq/ssgo/tool/internal/generator/common"
)

func TestRepoGenerator_GroupQueriesByEntity(t *testing.T) {
	gen := &RepoGenerator{
		opts: RepoGenOptions{
			Verbose: false,
		},
	}

	queries := []common.QueryInfo{
		{Name: "GetUserByID", ModelName: "User"},
		{Name: "ListUsers", ModelName: "User"},
		{Name: "CreateUser", ModelName: "User"},
		{Name: "GetTaskByID", ModelName: "Task"},
		{Name: "ListTasks", ModelName: "Task"},
	}

	grouped := gen.groupQueriesByEntity(queries)

	if len(grouped) != 2 {
		t.Errorf("groupQueriesByEntity() returned %d entities, want 2", len(grouped))
	}

	userQueries, ok := grouped["User"]
	if !ok {
		t.Fatal("User entity not found")
	}
	if len(userQueries) != 3 {
		t.Errorf("User has %d queries, want 3", len(userQueries))
	}

	taskQueries, ok := grouped["Task"]
	if !ok {
		t.Fatal("Task entity not found")
	}
	if len(taskQueries) != 2 {
		t.Errorf("Task has %d queries, want 2", len(taskQueries))
	}
}

func TestRepoGenerator_ExtractEntityFromModel(t *testing.T) {
	gen := &RepoGenerator{}

	tests := []struct {
		input    string
		expected string
	}{
		{"User", "User"},
		{"LineartUser", "User"},
		{"SummarifyTask", "Task"},
		{"NikkiProject", "Project"},
		{"OnesystemAccount", "Account"},
		{"RegularModel", "RegularModel"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := gen.extractEntityFromModel(tt.input)
			if result != tt.expected {
				t.Errorf("extractEntityFromModel(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestRepoGenerator_PrefixDBType(t *testing.T) {
	gen := &RepoGenerator{}

	tests := []struct {
		input    string
		expected string
	}{
		{"User", "db.User"},
		{"*User", "*db.User"},
		{"[]User", "[]db.User"},
		{"[]*User", "[]*db.User"},
		{"string", "string"},
		{"int64", "int64"},
		{"bool", "bool"},
		{"error", "error"},
		{"time.Time", "time.Time"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := gen.prefixDBType(tt.input)
			if result != tt.expected {
				t.Errorf("prefixDBType(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestRepoGenerator_BuildMethodSignature(t *testing.T) {
	gen := &RepoGenerator{}

	tests := []struct {
		query    common.QueryInfo
		expected string
	}{
		{
			query: common.QueryInfo{
				Name:       "GetUserByID",
				ReturnType: "User",
				Params:     []common.Param{{Name: "id", Type: "int64"}},
			},
			expected: "GetUserByID(ctx context.Context, id int64) (db.User, error)",
		},
		{
			query: common.QueryInfo{
				Name:       "ListUsers",
				ReturnType: "[]User",
				Params:     []common.Param{{Name: "limit", Type: "int32"}, {Name: "offset", Type: "int32"}},
				IsMany:     true,
			},
			expected: "ListUsers(ctx context.Context, limit int32, offset int32) ([]db.User, error)",
		},
		{
			query: common.QueryInfo{
				Name:   "DeleteUser",
				IsExec: true,
				Params: []common.Param{{Name: "id", Type: "int64"}},
			},
			expected: "DeleteUser(ctx context.Context, id int64) error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.query.Name, func(t *testing.T) {
			result := gen.buildMethodSignature(tt.query)
			if result != tt.expected {
				t.Errorf("buildMethodSignature() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestRepoGenerator_GenerateFromSqlc_RequiresSqlcOutput(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	gomodContent := `module github.com/example/testservice

go 1.21
`
	os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(gomodContent), 0o644)

	gen := NewRepoGenerator(RepoGenOptions{
		OutputDir: tmpDir,
		Verbose:   false,
	})

	err := gen.GenerateFromSqlc()
	if err == nil {
		t.Error("GenerateFromSqlc() should fail when SQLC output is missing")
	}
	if !strings.Contains(err.Error(), "SQLC output not found") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRepoGenerator_GenerateMethod(t *testing.T) {
	gen := &RepoGenerator{
		opts: RepoGenOptions{
			WithTrace: false,
		},
	}

	query := common.QueryInfo{
		Name:       "GetUserByID",
		ReturnType: "User",
		Params:     []common.Param{{Name: "id", Type: "int64"}},
	}

	method := gen.generateMethod("user", "User", query)

	if !strings.Contains(method, "func (r *userRepository) GetUserByID") {
		t.Error("generated method does not have correct signature")
	}
	if !strings.Contains(method, "r.store.Queries().GetUserByID(ctx, id)") {
		t.Error("generated method does not call store query correctly")
	}
}

func TestRepoGenerator_GenerateMethodWithTracing(t *testing.T) {
	gen := &RepoGenerator{
		opts: RepoGenOptions{
			WithTrace: true,
		},
	}

	query := common.QueryInfo{
		Name:       "GetUserByID",
		ReturnType: "User",
		Params:     []common.Param{{Name: "id", Type: "int64"}},
	}

	method := gen.generateMethod("user", "User", query)

	if !strings.Contains(method, "otel.Tracer") {
		t.Error("generated method does not include OpenTelemetry tracing")
	}
	if !strings.Contains(method, "defer span.End()") {
		t.Error("generated method does not defer span.End()")
	}
}
