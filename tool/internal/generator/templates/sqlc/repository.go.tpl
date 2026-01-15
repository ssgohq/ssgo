package repository

import (
	"context"

	"{{.Module}}/internal/data/db"
	"{{.Module}}/internal/store"
{{if .WithTrace}}
	"go.opentelemetry.io/otel"
{{end}}
)

// {{.Entity.Name}}Repository defines the interface for {{.Entity.SnakeName}} data access
type {{.Entity.Name}}Repository interface {
{{- range .Queries}}
	{{.Name}}(ctx context.Context{{range .Params}}, {{.Name}} {{.Type}}{{end}}) {{if .ReturnType}}({{.ReturnType}}, error){{else}}error{{end}}
{{- end}}
}

type {{.Entity.CamelName}}Repository struct {
	store *store.Store
}

// New{{.Entity.Name}}Repository creates a new {{.Entity.SnakeName}} repository
func New{{.Entity.Name}}Repository(store *store.Store) {{.Entity.Name}}Repository {
	return &{{.Entity.CamelName}}Repository{
		store: store,
	}
}

{{range .Queries}}
// {{.Name}} wraps the SQLC query
func (r *{{$.Entity.CamelName}}Repository) {{.Name}}(ctx context.Context{{range .Params}}, {{.Name}} {{.Type}}{{end}}) {{if .ReturnType}}({{.ReturnType}}, error){{else}}error{{end}} {
{{- if $.WithTrace}}
	ctx, span := otel.Tracer("{{$.Entity.CamelName}}Repository").Start(ctx, "{{.Name}}")
	defer span.End()

{{end}}
	return r.store.Queries().{{.Name}}(ctx{{range .Params}}, {{.Name}}{{end}})
}
{{end}}