name: {{.ServiceName}}
host: 0.0.0.0
port: 8080

{{if .HasRPC}}
rpc:
  {{range .RPCServices}}
  {{.ServiceLower}}:
    serviceName: {{.ServiceLower}}-rpc
    discovery:
      type: consul
      consul:
        address: localhost:8500
  {{end}}
{{end}}

# Uncomment to enable tracing:
# trace:
#   name: {{.ServiceName}}
#   endpoint: localhost:4317