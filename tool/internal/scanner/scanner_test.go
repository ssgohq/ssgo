package scanner_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ssgohq/ssgo/tool/internal/scanner"
)

// helpers

func mkfile(t *testing.T, dir, rel, content string) {
	t.Helper()
	abs := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("mkfile MkdirAll: %v", err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatalf("mkfile WriteFile: %v", err)
	}
}

func mkdir(t *testing.T, dir, rel string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, rel), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
}

// --- ScanDir tests ---

func TestScanDir_Empty(t *testing.T) {
	dir := t.TempDir()
	result, err := scanner.ScanDir(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.State != scanner.ServiceStateEmpty {
		t.Errorf("state: got %q want %q", result.State, scanner.ServiceStateEmpty)
	}
	if len(result.Contracts) != 0 {
		t.Errorf("expected no contracts, got %d", len(result.Contracts))
	}
}

func TestScanDir_APIOnly_Contracts(t *testing.T) {
	dir := t.TempDir()
	mkfile(t, dir, "api/user.api", "// user api")
	mkfile(t, dir, "api/order.api", "// order api")

	result, err := scanner.ScanDir(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.APIContracts()) != 2 {
		t.Errorf("api contracts: got %d want 2", len(result.APIContracts()))
	}
	if len(result.RPCContracts()) != 0 {
		t.Errorf("rpc contracts: got %d want 0", len(result.RPCContracts()))
	}
}

func TestScanDir_RPCOnly_Contracts(t *testing.T) {
	dir := t.TempDir()
	mkfile(t, dir, "proto/user/v1/user.proto", "syntax = \"proto3\";")

	result, err := scanner.ScanDir(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.RPCContracts()) != 1 {
		t.Errorf("rpc contracts: got %d want 1", len(result.RPCContracts()))
	}
}

func TestScanDir_Mixed_HybridCapable(t *testing.T) {
	dir := t.TempDir()
	mkfile(t, dir, "api/user.api", "// user api")
	mkfile(t, dir, "proto/user.proto", "syntax = \"proto3\";")
	mkfile(t, dir, "sql/queries.sql", "SELECT 1;")

	result, err := scanner.ScanDir(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.State != scanner.ServiceStateHybridCapable {
		t.Errorf("state: got %q want %q", result.State, scanner.ServiceStateHybridCapable)
	}
	if len(result.SQLContracts()) != 1 {
		t.Errorf("sql contracts: got %d want 1", len(result.SQLContracts()))
	}
}

func TestScanDir_SuggestedOrder_RPCBeforeAPI(t *testing.T) {
	dir := t.TempDir()
	mkfile(t, dir, "api/svc.api", "")
	mkfile(t, dir, "proto/svc.proto", "")

	result, err := scanner.ScanDir(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.SuggestedOrder) < 2 {
		t.Fatalf("expected at least 2 items in suggested order, got %v", result.SuggestedOrder)
	}
	if result.SuggestedOrder[0] != "rpc" {
		t.Errorf("first suggested: got %q want %q", result.SuggestedOrder[0], "rpc")
	}
	if result.SuggestedOrder[1] != "api" {
		t.Errorf("second suggested: got %q want %q", result.SuggestedOrder[1], "api")
	}
}

func TestScanDir_DetectsAPIOnlyLayout(t *testing.T) {
	dir := t.TempDir()
	mkdir(t, dir, "internal/handler")
	mkfile(t, dir, "proto/svc.proto", "")

	result, err := scanner.ScanDir(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.State != scanner.ServiceStateAPIOnly {
		t.Errorf("state: got %q want %q", result.State, scanner.ServiceStateAPIOnly)
	}
	if !result.HasConflicts() {
		t.Error("expected conflict: proto files found but no RPC layout")
	}
}

func TestScanDir_DetectsRPCOnlyLayout(t *testing.T) {
	dir := t.TempDir()
	mkdir(t, dir, "kitex_gen")
	mkfile(t, dir, "api/svc.api", "")

	result, err := scanner.ScanDir(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.State != scanner.ServiceStateRPCOnly {
		t.Errorf("state: got %q want %q", result.State, scanner.ServiceStateRPCOnly)
	}
	if !result.HasConflicts() {
		t.Error("expected conflict: .api files found but no API layout")
	}
}

func TestScanDir_DetectsHybridLayout(t *testing.T) {
	dir := t.TempDir()
	mkdir(t, dir, "internal/api/handler")
	mkdir(t, dir, "kitex_gen")

	result, err := scanner.ScanDir(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.State != scanner.ServiceStateHybrid {
		t.Errorf("state: got %q want %q", result.State, scanner.ServiceStateHybrid)
	}
}

func TestScanDir_SkipsHiddenAndVendor(t *testing.T) {
	dir := t.TempDir()
	mkfile(t, dir, ".git/hooks/pre-commit.proto", "")
	mkfile(t, dir, "vendor/some/lib.proto", "")
	mkfile(t, dir, "api/real.api", "")

	result, err := scanner.ScanDir(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.RPCContracts()) != 0 {
		t.Errorf("expected .git/vendor protos to be skipped, got %d", len(result.RPCContracts()))
	}
	if len(result.APIContracts()) != 1 {
		t.Errorf("api contracts: got %d want 1", len(result.APIContracts()))
	}
}

func TestScanDir_EmptyRoot(t *testing.T) {
	_, err := scanner.ScanDir("")
	if err == nil {
		t.Fatal("expected error for empty root")
	}
}
