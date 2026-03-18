package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ssgohq/ssgo/tool/internal/manifest"
	"github.com/ssgohq/ssgo/tool/internal/scanner"
)

// helpers — create files inside a temp directory.

func mkfile(t *testing.T, dir, rel, content string) {
	t.Helper()
	abs := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("mkfile MkdirAll %s: %v", rel, err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatalf("mkfile WriteFile %s: %v", rel, err)
	}
}

func mkdir(t *testing.T, dir, rel string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, rel), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", rel, err)
	}
}

func fileExists(t *testing.T, dir, rel string) bool {
	t.Helper()
	_, err := os.Stat(filepath.Join(dir, rel))
	return err == nil
}

// --- Scenario 1: API-only ---

// TestIntegration_APIOnly verifies that a directory with only .api contracts
// is scanned as having API contracts and no RPC contracts.
func TestIntegration_APIOnly(t *testing.T) {
	dir := t.TempDir()
	mkfile(t, dir, "api/user.api", "// user service api")

	result, err := scanner.ScanDir(dir)
	if err != nil {
		t.Fatalf("ScanDir: %v", err)
	}

	if len(result.APIContracts()) == 0 {
		t.Error("expected at least one API contract")
	}
	if len(result.RPCContracts()) != 0 {
		t.Errorf("expected no RPC contracts, got %d", len(result.RPCContracts()))
	}
}

// TestIntegration_APIOnly_PlanHasAPIStep verifies that the plan for an API-only
// directory contains an API generation step.
func TestIntegration_APIOnly_PlanHasAPIStep(t *testing.T) {
	dir := t.TempDir()
	mkfile(t, dir, "api/order.api", "// order api")

	result, err := scanner.ScanDir(dir)
	if err != nil {
		t.Fatalf("ScanDir: %v", err)
	}

	m, err := manifest.Load(dir)
	if err != nil {
		t.Fatalf("manifest.Load: %v", err)
	}

	plan := buildPlan(result, m)
	if len(plan.Steps) == 0 {
		t.Fatal("expected at least one plan step for API-only layout")
	}

	hasAPI := false
	for _, s := range plan.Steps {
		if s.Transport == "api" {
			hasAPI = true
		}
	}
	if !hasAPI {
		t.Error("expected an api transport step in the plan")
	}
}

// --- Scenario 2: RPC-only ---

// TestIntegration_RPCOnly verifies that a directory with only .proto contracts
// yields RPC contracts but no API contracts.
func TestIntegration_RPCOnly(t *testing.T) {
	dir := t.TempDir()
	mkfile(t, dir, "proto/user.proto", "syntax = \"proto3\";")

	result, err := scanner.ScanDir(dir)
	if err != nil {
		t.Fatalf("ScanDir: %v", err)
	}

	if len(result.RPCContracts()) == 0 {
		t.Error("expected at least one RPC contract")
	}
	if len(result.APIContracts()) != 0 {
		t.Errorf("expected no API contracts, got %d", len(result.APIContracts()))
	}
}

// TestIntegration_RPCOnly_PlanHasRPCStep verifies the plan for RPC-only.
func TestIntegration_RPCOnly_PlanHasRPCStep(t *testing.T) {
	dir := t.TempDir()
	mkfile(t, dir, "proto/svc.proto", "syntax = \"proto3\";")

	result, err := scanner.ScanDir(dir)
	if err != nil {
		t.Fatalf("ScanDir: %v", err)
	}
	m, _ := manifest.Load(dir)
	plan := buildPlan(result, m)

	hasRPC := false
	for _, s := range plan.Steps {
		if s.Transport == "rpc" {
			hasRPC = true
		}
	}
	if !hasRPC {
		t.Error("expected an rpc transport step in the plan")
	}
}

// --- Scenario 3: Hybrid-capable ---

// TestIntegration_HybridCapable_StateDetected verifies that a directory with
// both .api and .proto contracts (no generated layout yet) is detected as
// hybrid-capable.
func TestIntegration_HybridCapable_StateDetected(t *testing.T) {
	dir := t.TempDir()
	mkfile(t, dir, "api/user.api", "// user")
	mkfile(t, dir, "proto/user.proto", "syntax = \"proto3\";")

	result, err := scanner.ScanDir(dir)
	if err != nil {
		t.Fatalf("ScanDir: %v", err)
	}

	if result.State != scanner.ServiceStateHybridCapable {
		t.Errorf("state: got %q want %q", result.State, scanner.ServiceStateHybridCapable)
	}
}

// --- Scenario 4: Add-API-to-RPC ---

// TestIntegration_AddAPItoRPC verifies that after an RPC layout is in place,
// adding a .api file causes the scanner to produce a conflict (API contracts
// found but no API layout) — consistent with the expected add-transport flow.
func TestIntegration_AddAPItoRPC(t *testing.T) {
	dir := t.TempDir()

	// Simulate existing RPC layout.
	mkdir(t, dir, "kitex_gen")
	mkdir(t, dir, "internal/rpc/server")

	// Add a new .api contract.
	mkfile(t, dir, "api/order.api", "// order api")

	result, err := scanner.ScanDir(dir)
	if err != nil {
		t.Fatalf("ScanDir: %v", err)
	}

	// Scanner should detect RPC-only state with an API contract conflict.
	if result.State != scanner.ServiceStateRPCOnly {
		t.Errorf("state: got %q want %q", result.State, scanner.ServiceStateRPCOnly)
	}
	if !result.HasConflicts() {
		t.Error("expected conflict: api contracts present but no api layout")
	}
	if len(result.APIContracts()) == 0 {
		t.Error("expected api contracts to be discovered")
	}
}

// --- Scenario 5: Add-RPC-to-API ---

// TestIntegration_AddRPCtoAPI verifies that after an API layout is in place,
// adding a .proto file produces a conflict (RPC contracts but no RPC layout).
func TestIntegration_AddRPCtoAPI(t *testing.T) {
	dir := t.TempDir()

	// Simulate existing API layout.
	mkdir(t, dir, "internal/api/handler")
	mkdir(t, dir, "internal/api/logic")

	// Add a new .proto contract.
	mkfile(t, dir, "proto/order.proto", "syntax = \"proto3\";")

	result, err := scanner.ScanDir(dir)
	if err != nil {
		t.Fatalf("ScanDir: %v", err)
	}

	// Scanner should detect API-only state with an RPC contract conflict.
	if result.State != scanner.ServiceStateAPIOnly {
		t.Errorf("state: got %q want %q", result.State, scanner.ServiceStateAPIOnly)
	}
	if !result.HasConflicts() {
		t.Error("expected conflict: proto contracts present but no rpc layout")
	}
	if len(result.RPCContracts()) == 0 {
		t.Error("expected rpc contracts to be discovered")
	}
}

// --- Scenario 6: Dual-single mode (manifest) ---

// TestIntegration_DualSingle_ManifestMode verifies that a manifest with
// BinaryModeSingle and both transports enabled reports IsHybrid=true and
// EffectiveBinaryMode=single.
func TestIntegration_DualSingle_ManifestMode(t *testing.T) {
	dir := t.TempDir()

	// Write a .ss.yaml manifest with single binary mode.
	m := &manifest.ServiceManifest{
		Binary: manifest.BinaryConfig{Mode: manifest.BinaryModeSingle},
		Transports: manifest.TransportsConfig{
			API: manifest.APITransportConfig{Enabled: true, APIs: []string{"api/svc.api"}},
			RPC: manifest.RPCTransportConfig{Enabled: true, Protos: []string{"proto/svc.proto"}},
		},
	}
	if err := manifest.Save(dir, m); err != nil {
		t.Fatalf("manifest.Save: %v", err)
	}

	loaded, err := manifest.Load(dir)
	if err != nil {
		t.Fatalf("manifest.Load: %v", err)
	}

	if !loaded.IsHybrid() {
		t.Error("expected IsHybrid=true")
	}
	if loaded.EffectiveBinaryMode() != manifest.BinaryModeSingle {
		t.Errorf("effective mode: got %q want %q", loaded.EffectiveBinaryMode(), manifest.BinaryModeSingle)
	}
}

// --- Scenario 7: Hybrid layout already generated ---

// TestIntegration_HybridLayout_DetectedAsHybrid verifies that a directory with
// both API and RPC generated layouts is detected as ServiceStateHybrid.
func TestIntegration_HybridLayout_DetectedAsHybrid(t *testing.T) {
	dir := t.TempDir()
	mkdir(t, dir, "internal/api/handler")
	mkdir(t, dir, "kitex_gen")

	result, err := scanner.ScanDir(dir)
	if err != nil {
		t.Fatalf("ScanDir: %v", err)
	}

	if result.State != scanner.ServiceStateHybrid {
		t.Errorf("state: got %q want %q", result.State, scanner.ServiceStateHybrid)
	}
}

// --- Scenario 8: Dual-split, both plans populated ---

// TestIntegration_DualSplit_PlanBothSteps verifies that a hybrid-capable dir
// with both .api and .proto contracts produces a plan with both api and rpc steps.
func TestIntegration_DualSplit_PlanBothSteps(t *testing.T) {
	dir := t.TempDir()
	mkfile(t, dir, "api/svc.api", "// svc api")
	mkfile(t, dir, "proto/svc.proto", "syntax = \"proto3\";")

	result, err := scanner.ScanDir(dir)
	if err != nil {
		t.Fatalf("ScanDir: %v", err)
	}
	m, _ := manifest.Load(dir)
	plan := buildPlan(result, m)

	transports := map[string]bool{}
	for _, s := range plan.Steps {
		transports[s.Transport] = true
	}

	if !transports["api"] {
		t.Error("expected api step in dual-split plan")
	}
	if !transports["rpc"] {
		t.Error("expected rpc step in dual-split plan")
	}
}

// TestIntegration_FileExistenceHelpers demonstrates the fileExists helper.
// (Primarily ensures the helper compiles and works for other tests.)
func TestIntegration_FileExistenceHelper(t *testing.T) {
	dir := t.TempDir()
	mkfile(t, dir, "hello.txt", "world")

	if !fileExists(t, dir, "hello.txt") {
		t.Error("expected hello.txt to exist")
	}
	if fileExists(t, dir, "does-not-exist.txt") {
		t.Error("expected does-not-exist.txt to not exist")
	}
}
