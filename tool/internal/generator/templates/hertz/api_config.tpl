// Code scaffolded by ssgo. Safe to edit.

package config

import (
	"fmt"

	"github.com/ssgohq/goten-core/logx"
	"github.com/ssgohq/goten-core/metric"
{{- if .HasRPCClients}}
	"github.com/ssgohq/goten-core/srpc"
{{- end}}
	"github.com/ssgohq/goten-core/trace"
{{- if .WithRedis}}
	"github.com/ssgohq/goten-core/stores/redis"
{{- end}}
)
{{if .WithDB}}
// DBConfig holds database configuration
type DBConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
	SSLMode  string `yaml:"sslmode"`
}

// IsEnabled returns true if database configuration is provided
func (c DBConfig) IsEnabled() bool {
	return c.Host != "" && c.Database != ""
}
{{end}}
type Config struct {
	Name   string              `yaml:"name"`
	Host   string              `yaml:"host,omitempty"`
	Port   int                 `yaml:"port,omitempty"`
	Log    logx.Config         `yaml:"log,omitempty"`
	Trace  trace.Config        `yaml:"trace,omitempty"`
	Metric metric.Config `yaml:"metric,omitempty"`
{{range .RPCClients}}
	// {{.Name}}Rpc is the RPC client configuration for {{.ServiceName}} service
	{{.Name}}Rpc srpc.ClientConfig `yaml:"{{.Name}}Rpc"`
{{end}}
{{- if .WithDB}}
	DB DBConfig `yaml:"db,omitempty"`
{{end}}
{{- if .WithRedis}}
	Redis redis.Config `yaml:"redis,omitempty"`
{{end}}
	// Auth configuration (optional)
	Auth AuthConfig `yaml:"auth,omitempty"`
{{- if not .WithDB}}{{if not .WithRedis}}{{if not .HasRPCClients}}
	// Add your config here:
	// DB       DBConfig        `yaml:"db,omitempty"`
	// Redis    redis.Config    `yaml:"redis,omitempty"`
{{end}}{{end}}{{end}}
}

// Addr returns the server address
func (c Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// IsTraceEnabled returns true if tracing is enabled
func (c Config) IsTraceEnabled() bool {
	return c.Trace.IsEnabled()
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	JWTSecret   string `yaml:"jwtSecret,omitempty"`
	JWTExpire   int64  `yaml:"jwtExpire,omitempty"`
	TokenHeader string `yaml:"tokenHeader,omitempty"`
}