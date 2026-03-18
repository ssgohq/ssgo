// Code scaffolded by ssgo. Safe to edit.

package config

import (
	"fmt"

{{- if .HasRPCClients}}
	"github.com/ssgohq/goten-core/srpc"
{{- end}}
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
// APIConfig holds HTTP API transport configuration.
// It embeds BaseConfig for shared fields (Name, Log, Trace, Metric).
type APIConfig struct {
	BaseConfig `yaml:",inline"`
	Host       string `yaml:"host,omitempty"`
	Port       int    `yaml:"port,omitempty"`
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
}

// Addr returns the server address
func (c APIConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	JWTSecret   string `yaml:"jwtSecret,omitempty"`
	JWTExpire   int64  `yaml:"jwtExpire,omitempty"`
	TokenHeader string `yaml:"tokenHeader,omitempty"`
}

// Config is an alias for APIConfig for backward compatibility with generated handlers.
type Config = APIConfig
