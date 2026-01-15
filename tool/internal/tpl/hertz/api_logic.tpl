package {{.Package}}

import (
	"context"
	"fmt"

	"github.com/ssgohq/goten-core/logx"
	"{{.Module}}/internal/pkg/httputil"
	"{{.Module}}/internal/svc"
	"{{.Module}}/internal/types"
{{if .HasRPCClients}}	// Import RPC types
	// "MODULE/kitex_gen/PACKAGE"
{{end}})

type {{.LogicName}} struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func New{{.LogicName}}(ctx context.Context, svcCtx *svc.ServiceContext) *{{.LogicName}} {
	return &{{.LogicName}}{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

{{if .HasRequest}}{{if .HasResponse}}func (l *{{.LogicName}}) {{.Method}}(req *types.{{.RequestType}}) (*types.{{.ResponseType}}, error) {
	// TODO: Implement business logic here
	//
	// Pattern for calling RPC service:
	// 1. Transform API request to RPC request
	// 2. Call RPC service via client from l.svcCtx
	// 3. Handle errors with httputil.NewAppError
	// 4. Transform RPC response to API response
	//
{{if .HasRPCClients}}	// Example using RPC client:
	// rpcReq := &package.SomeRequest{
	//     // Map fields from req to rpcReq
	// }
	//
	// rpcResp, err := l.svcCtx.SomeClient.Client().SomeMethod(l.ctx, rpcReq)
	// if err != nil {
	//     logx.Errorw("Failed to call RPC",
	//         "error", err,
	//     )
	//     return nil, httputil.NewAppError(
	//         httputil.CodeRPCError,
	//         fmt.Sprintf("RPC call failed: %v", err),
	//         err,
	//     )
	// }
	//
	// return &types.{{.ResponseType}}{
	//     // Map fields from rpcResp to response
	// }, nil
{{end}}
	// Suppress unused import warnings
	_ = fmt.Sprintf
	_ = logx.Info
	_ = httputil.CodeInternalError

	return &types.{{.ResponseType}}{}, nil
}
{{else}}func (l *{{.LogicName}}) {{.Method}}(req *types.{{.RequestType}}) error {
	// TODO: Implement business logic here
{{if .HasRPCClients}}	// Example using RPC client - see above pattern
{{end}}
	// Suppress unused import warnings
	_ = fmt.Sprintf
	_ = logx.Info
	_ = httputil.CodeInternalError

	return nil
}
{{end}}{{else}}{{if .HasResponse}}func (l *{{.LogicName}}) {{.Method}}() (*types.{{.ResponseType}}, error) {
	// TODO: Implement business logic here
{{if .HasRPCClients}}	// Example using RPC client - see above pattern
{{end}}
	// Suppress unused import warnings
	_ = fmt.Sprintf
	_ = logx.Info
	_ = httputil.CodeInternalError

	return &types.{{.ResponseType}}{}, nil
}
{{else}}func (l *{{.LogicName}}) {{.Method}}() error {
	// TODO: Implement business logic here
	return nil
}
{{end}}{{end}}