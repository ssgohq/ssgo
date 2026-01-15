package {{.Package}}

import (
	"context"

	"{{.Module}}/internal/svc"
	"{{.Module}}/internal/types"
)

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
	// TODO: implement business logic here

	return &types.{{.ResponseType}}{}, nil
}
{{else}}func (l *{{.LogicName}}) {{.Method}}(req *types.{{.RequestType}}) error {
	// TODO: implement business logic here

	return nil
}
{{end}}{{else}}{{if .HasResponse}}func (l *{{.LogicName}}) {{.Method}}() (*types.{{.ResponseType}}, error) {
	// TODO: implement business logic here

	return &types.{{.ResponseType}}{}, nil
}
{{else}}func (l *{{.LogicName}}) {{.Method}}() error {
	// TODO: implement business logic here

	return nil
}
{{end}}{{end}}