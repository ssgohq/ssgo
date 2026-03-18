package cmd

import (
	"path/filepath"
	"testing"

	"github.com/ssgohq/ssgo/tool/internal/cmdctx"
	gen "github.com/ssgohq/ssgo/tool/internal/generator/kitex"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func makeCtx(flags map[string]interface{}) *cmdctx.Context {
	ctx := cmdctx.New()
	if flags != nil {
		ctx.Flags = flags
	}
	return ctx
}

func protoWithServices(names ...string) *gen.Proto {
	var services []gen.Service
	for _, name := range names {
		services = append(services, gen.Service{Name: name})
	}
	return &gen.Proto{Services: services}
}

// ---------------------------------------------------------------------------
// resolveFlag
// ---------------------------------------------------------------------------

func TestResolveFlag_LongForm(t *testing.T) {
	ctx := makeCtx(map[string]interface{}{"proto": "service.proto"})
	if got := resolveFlag(ctx, "proto", "p"); got != "service.proto" {
		t.Errorf("got %q, want %q", got, "service.proto")
	}
}

func TestResolveFlag_ShortForm(t *testing.T) {
	ctx := makeCtx(map[string]interface{}{"p": "service.proto"})
	if got := resolveFlag(ctx, "proto", "p"); got != "service.proto" {
		t.Errorf("got %q, want %q", got, "service.proto")
	}
}

func TestResolveFlag_LongTakesPrecedence(t *testing.T) {
	ctx := makeCtx(map[string]interface{}{"proto": "long.proto", "p": "short.proto"})
	if got := resolveFlag(ctx, "proto", "p"); got != "long.proto" {
		t.Errorf("long form should take precedence, got %q", got)
	}
}

func TestResolveFlag_Empty(t *testing.T) {
	ctx := makeCtx(nil)
	if got := resolveFlag(ctx, "proto", "p"); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

// ---------------------------------------------------------------------------
// buildProtoIncludes
// ---------------------------------------------------------------------------

func TestBuildProtoIncludes_TwoLevels(t *testing.T) {
	protoFile := "/workspace/shared-proto/proto/service/v1/service.proto"
	includes := buildProtoIncludes(protoFile)

	wantDir := filepath.Dir(protoFile)
	wantParent := filepath.Dir(wantDir)

	if len(includes) != 2 {
		t.Fatalf("expected 2 includes, got %d: %v", len(includes), includes)
	}
	if includes[0] != wantDir {
		t.Errorf("includes[0]: got %q, want %q", includes[0], wantDir)
	}
	if includes[1] != wantParent {
		t.Errorf("includes[1]: got %q, want %q", includes[1], wantParent)
	}
}

func TestBuildProtoIncludes_RootProto(t *testing.T) {
	// filepath.Dir("service.proto") == "." → skipped
	includes := buildProtoIncludes("service.proto")
	if len(includes) != 0 {
		t.Errorf("expected 0 includes for root-level proto, got %v", includes)
	}
}

// ---------------------------------------------------------------------------
// cloneCtxWithFlag
// ---------------------------------------------------------------------------

func TestCloneCtxWithFlag_SetsValue(t *testing.T) {
	ctx := makeCtx(map[string]interface{}{"a": "1"})
	ctx.WorkingDir = "/workspace"
	ctx.Debug = true

	clone := cloneCtxWithFlag(ctx, "b", "2")
	if clone.Flags["b"] != "2" {
		t.Errorf("flag b: got %v, want %q", clone.Flags["b"], "2")
	}
	if clone.Flags["a"] != "1" {
		t.Errorf("original flag a should be preserved")
	}
	if clone.WorkingDir != "/workspace" {
		t.Errorf("WorkingDir not preserved")
	}
	if !clone.Debug {
		t.Errorf("Debug not preserved")
	}
}

func TestCloneCtxWithFlag_EmptyValueSkips(t *testing.T) {
	ctx := makeCtx(nil)
	clone := cloneCtxWithFlag(ctx, "b", "")
	if _, ok := clone.Flags["b"]; ok {
		t.Errorf("empty value should not set flag b")
	}
}

// ---------------------------------------------------------------------------
// cloneCtxWithArgs
// ---------------------------------------------------------------------------

func TestCloneCtxWithArgs_SlicesSubcommand(t *testing.T) {
	ctx := makeCtx(nil)
	ctx.Args = []string{"gen", "my-service"}

	clone := cloneCtxWithArgs(ctx, ctx.Args[1:])
	if len(clone.Args) != 1 || clone.Args[0] != "my-service" {
		t.Errorf("args: got %v, want [my-service]", clone.Args)
	}
	// Original ctx.Args should be untouched
	if len(ctx.Args) != 2 {
		t.Errorf("original args should be untouched, got %v", ctx.Args)
	}
}

// ---------------------------------------------------------------------------
// resolveService
// ---------------------------------------------------------------------------

func TestResolveService_FromFlag(t *testing.T) {
	ctx := makeCtx(map[string]interface{}{"service": "MyService"})
	proto := protoWithServices("OtherService")
	got, err := resolveService(ctx, proto)
	if err != nil {
		t.Fatal(err)
	}
	if got != "MyService" {
		t.Errorf("flag should take precedence: got %q, want MyService", got)
	}
}

func TestResolveService_AutoDetectSingle(t *testing.T) {
	ctx := makeCtx(nil)
	proto := protoWithServices("UserService")
	got, err := resolveService(ctx, proto)
	if err != nil {
		t.Fatal(err)
	}
	if got != "UserService" {
		t.Errorf("got %q, want UserService", got)
	}
}

func TestResolveService_ErrorNoServices(t *testing.T) {
	ctx := makeCtx(nil)
	proto := protoWithServices()
	_, err := resolveService(ctx, proto)
	if err == nil {
		t.Error("expected error for proto with no services")
	}
}

func TestResolveService_ErrorMultipleServices(t *testing.T) {
	ctx := makeCtx(nil)
	proto := protoWithServices("AuthService", "UserService")
	_, err := resolveService(ctx, proto)
	if err == nil {
		t.Error("expected error when multiple services and --service not specified")
	}
}

// ---------------------------------------------------------------------------
// resolveUseTypes
// ---------------------------------------------------------------------------

func TestResolveUseTypes_FromFlag(t *testing.T) {
	ctx := makeCtx(map[string]interface{}{"use": "github.com/org/shared-proto/kitex_gen/auth"})
	proto := &gen.Proto{RawGoPackage: "github.com/org/shared-proto/kitex_gen/auth;authv1"}
	got := resolveUseTypes(ctx, proto)
	if got != "github.com/org/shared-proto/kitex_gen/auth" {
		t.Errorf("flag should take precedence: got %q", got)
	}
}

func TestResolveUseTypes_DerivedFromGoPackageWithAlias(t *testing.T) {
	ctx := makeCtx(nil)
	proto := &gen.Proto{
		RawGoPackage: "github.com/org/shared-proto/kitex_gen/service/v1;servicev1",
	}
	got := resolveUseTypes(ctx, proto)
	if got != "github.com/org/shared-proto/kitex_gen/service/v1" {
		t.Errorf("got %q, want github.com/org/shared-proto/kitex_gen/service/v1", got)
	}
}

func TestResolveUseTypes_DerivedFromGoPackageNoAlias(t *testing.T) {
	ctx := makeCtx(nil)
	proto := &gen.Proto{RawGoPackage: "github.com/org/shared-proto/kitex_gen/auth"}
	got := resolveUseTypes(ctx, proto)
	if got != "github.com/org/shared-proto/kitex_gen/auth" {
		t.Errorf("got %q, want github.com/org/shared-proto/kitex_gen/auth", got)
	}
}

func TestResolveUseTypes_ShortGoPackageEmpty(t *testing.T) {
	ctx := makeCtx(nil)
	proto := &gen.Proto{RawGoPackage: "authpkg"} // no "/" — not a full import path
	got := resolveUseTypes(ctx, proto)
	if got != "" {
		t.Errorf("short go_package should yield empty use, got %q", got)
	}
}

func TestResolveUseTypes_EmptyGoPackageEmpty(t *testing.T) {
	ctx := makeCtx(nil)
	proto := &gen.Proto{}
	got := resolveUseTypes(ctx, proto)
	if got != "" {
		t.Errorf("empty go_package should yield empty use, got %q", got)
	}
}
