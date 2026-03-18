// Code scaffolded by ssgo. Safe to edit.

package svc

import (
	"{{.Module}}/internal/config"
{{if .HasRPCClients}}	"github.com/ssgohq/goten-core/srpc"
{{range .RPCClients}}	"{{$.TypesModule}}/kitex_gen/{{.Package}}/{{.ServiceLower}}"
{{end}}{{end}})

// APIServiceContext contains all dependencies for API handlers.
// It embeds BaseServiceContext for shared dependencies.
type APIServiceContext struct {
	BaseServiceContext
	Config config.APIConfig
{{range .RPCClients}}
	// {{.Name}}Rpc calls {{.ServiceName}} RPC service
	{{.Name}}Rpc {{.ServiceLower}}.Client
{{end}}
	// Add your other API-specific dependencies here:
	// Logger  *slog.Logger
	// Tracer  trace.Tracer
}

// NewAPIServiceContext creates a new APIServiceContext with all dependencies initialized
func NewAPIServiceContext(c config.APIConfig) *APIServiceContext {
	return &APIServiceContext{
		Config: c,
{{range .RPCClients}}
		{{.Name}}Rpc: srpc.MustNewClient({{.ServiceLower}}.NewClient, &c.{{.Name}}Rpc),
{{end}}	}
}

// ServiceContext is an alias for APIServiceContext for backward compatibility.
type ServiceContext = APIServiceContext

// NewServiceContext creates a new ServiceContext (alias for NewAPIServiceContext).
func NewServiceContext(c config.APIConfig) *ServiceContext {
	return NewAPIServiceContext(c)
}
