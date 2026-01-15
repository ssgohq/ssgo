name: {{.ServiceLower}}
host: 0.0.0.0
port: 8888

metric:
  enabled: true
  port: 6061
  metricsPath: /metrics
  healthPath: /healthz
  readyPath: /readyz
  enableMetrics: true
  enablePprof: true

discovery:
  type: consul
  consul:
    address: localhost:8500
{{if .WithTrace}}
trace:
  name: {{.ServiceLower}}
  endpoint: localhost:4317
{{else}}
# Uncomment to enable tracing:
# trace:
#   name: {{.ServiceLower}}
#   endpoint: localhost:4317
{{end}}
{{if .WithRedis}}
redis:
  host: localhost
  port: 6379
{{end}}
# Database config will be added by 'ssgo dms bun gen'
# db:
#   host: localhost
#   port: 5432
#   user: postgres
#   password: password
#   dbname: mydb
#   sslmode: disable