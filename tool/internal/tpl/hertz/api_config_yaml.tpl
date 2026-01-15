name: {{.ServiceName}}
host: 0.0.0.0
port: {{.Port}}

log:
  level: info
  format: json

metric:
  enabled: true
  port: 6060
  path: /metrics
  healthPath: /healthz
  readyPath: /readyz
{{range .RPCClients}}
{{.Name}}Rpc:
  serviceName: {{.ServiceName}}
  discovery:
    type: consul
    consul:
      address: localhost:8500
{{end}}
{{if .WithTrace}}
trace:
  name: {{.ServiceName}}
  endpoint: localhost:4317
{{else}}
# Uncomment to enable tracing:
# trace:
#   name: {{.ServiceName}}
#   endpoint: localhost:4317
{{end}}
{{if .WithDB}}
db:
{{- if eq .WithDB "postgres"}}
  host: localhost
  port: 5432
  user: postgres
  password: postgres
  database: mydb
  sslmode: disable
{{- else}}
  host: localhost
  port: 3306
  user: root
  password: password
  database: mydb
  sslmode: ""
{{- end}}
{{end}}
{{if .WithRedis}}
redis:
  host: localhost
  port: 6379
{{end}}
# Uncomment to enable authentication:
# auth:
#   jwtSecret: your-secret-key-change-in-production
#   jwtExpire: 86400
#   tokenHeader: Authorization