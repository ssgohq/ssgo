module {{.Module}}

go {{.GoVersion}}

require (
	github.com/cloudwego/hertz v0.9.3
{{if .HasRPCClients}}	github.com/cloudwego/kitex v0.15.4
	github.com/kitex-contrib/obs-opentelemetry v0.3.0
{{end}}	github.com/hertz-contrib/obs-opentelemetry v0.1.1
	github.com/ssgohq/goten-core latest
	go.uber.org/zap v1.27.0
	gopkg.in/yaml.v3 v3.0.1
)