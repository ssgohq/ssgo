// Code scaffolded by ssgo. Safe to edit.

package svc

import (
	"{{.Module}}/internal/config"
{{if .HasRPCClients}}	"github.com/ssgohq/goten-core/srpc"
{{range .RPCClients}}	"{{$.TypesModule}}/kitex_gen/{{.Package}}/{{.ServiceLower}}"
{{end}}{{end}})

// ServiceContext contains all dependencies for API handlers
type ServiceContext struct {
	Config config.Config
{{range .RPCClients}}
	// {{.Name}}Rpc calls {{.ServiceName}} RPC service
	{{.Name}}Rpc {{.ServiceLower}}.Client
{{end}}
	// Add your other dependencies here:
	// Logger  *slog.Logger
	// Tracer  trace.Tracer
}

// NewServiceContext creates a new ServiceContext with all dependencies initialized
func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config: c,
{{range .RPCClients}}
		{{.Name}}Rpc: srpc.MustNewClient({{.ServiceLower}}.NewClient, &c.{{.Name}}Rpc),
{{end}}	}
}