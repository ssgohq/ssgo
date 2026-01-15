package logic

import (
	"context"
{{- if .WithTrace }}

	"go.opentelemetry.io/otel"
{{- end }}

	"{{.Module}}/internal/svc"
	"{{.TypesModule}}/kitex_gen/{{.ServiceLower}}"
)

// {{.MethodName}}Logic contains the business logic for {{.MethodName}}
type {{.MethodName}}Logic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// New{{.MethodName}}Logic creates a new {{.MethodName}}Logic
func New{{.MethodName}}Logic(ctx context.Context, svcCtx *svc.ServiceContext) *{{.MethodName}}Logic {
	return &{{.MethodName}}Logic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// {{.MethodName}} handles the {{.MethodName}} request.
func (l *{{.MethodName}}Logic) {{.MethodName}}(req *{{.ServiceLower}}.{{.RequestType}}) (*{{.ServiceLower}}.{{.ResponseType}}, error) {
{{- if .WithTrace }}
	ctx, span := otel.Tracer("{{.ServiceLower}}").Start(l.ctx, "{{.MethodName}}")
	defer span.End()
	_ = ctx // use ctx for downstream calls
{{- end }}

	// TODO: add your logic here
	return &{{.ServiceLower}}.{{.ResponseType}}{}, nil
}