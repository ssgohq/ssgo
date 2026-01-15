package middleware

{{- $hasLogging := false -}}
{{- $hasCors := false -}}
{{- $hasJwt := false -}}
{{- $hasCustom := false -}}
{{- range .Middleware -}}
{{- if eq . "Logging" }}{{ $hasLogging = true }}{{- end -}}
{{- if eq . "Cors" }}{{ $hasCors = true }}{{- end -}}
{{- if eq . "Jwt" }}{{ $hasJwt = true }}{{- end -}}
{{- if and (ne . "Logging") (ne . "Cors") (ne . "Jwt") }}{{ $hasCustom = true }}{{- end -}}
{{- end }}
{{- $hasAny := or $hasLogging $hasCors $hasJwt $hasCustom }}
{{if $hasAny}}
import (
{{- if $hasCustom }}
	"context"
{{- end }}

	"github.com/cloudwego/hertz/pkg/app"
{{- if or $hasLogging $hasCors $hasJwt }}
	coremw "github.com/ssgohq/goten-core/middleware"
{{- end }}
	"{{.Module}}/internal/svc"
)
{{if $hasLogging}}
// Logging middleware with request ID and structured logging
func Logging(svcCtx *svc.ServiceContext) app.HandlerFunc {
	return coremw.AccessLog()
}
{{end}}
{{- if $hasCors}}
// Cors middleware for cross-origin requests
func Cors(svcCtx *svc.ServiceContext) app.HandlerFunc {
	return coremw.CORS(coremw.CORSConfig{})
}
{{end}}
{{- if $hasJwt}}
// Jwt middleware for JWT authentication
func Jwt(svcCtx *svc.ServiceContext) app.HandlerFunc {
	return coremw.JWT(coremw.JWTConfig{
		Secret: svcCtx.Config.Auth.JWTSecret,
	})
}
{{end}}
{{- range .Middleware}}{{if and (ne . "Logging") (ne . "Cors") (ne . "Jwt")}}
// {{.}} is a custom middleware - implement your logic here
func {{.}}(svcCtx *svc.ServiceContext) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		// TODO: Implement {{.}} middleware
		c.Next(ctx)
	}
}
{{end}}{{end}}
{{- else}}
// Add custom middleware here.
// Example:
//
// import (
//     "context"
//     "github.com/cloudwego/hertz/pkg/app"
//     "{{.Module}}/internal/svc"
// )
//
// func Auth(svcCtx *svc.ServiceContext) app.HandlerFunc {
//     return func(ctx context.Context, c *app.RequestContext) {
//         // TODO: Implement auth logic
//         c.Next(ctx)
//     }
// }
{{end}}
