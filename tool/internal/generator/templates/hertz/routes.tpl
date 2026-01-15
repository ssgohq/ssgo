package handler

import (
	"github.com/cloudwego/hertz/pkg/app/server"
	"{{.Module}}/internal/svc"
{{range .Imports}}	"{{.}}"
{{end}})

func RegisterHandlers(r *server.Hertz, svcCtx *svc.ServiceContext) {
{{range $group := .Groups}}
	// Group: {{$group.Name}}{{if $group.Prefix}}
	{{$group.VarName}} := r.Group("{{$group.Prefix}}"){{else}}
	{{$group.VarName}} := r{{end}}{{range $group.Middleware}}
	{{$group.VarName}}.Use(middleware.{{.}}(svcCtx)){{end}}
	{
{{range $group.Routes}}		{{$group.VarName}}.{{.Method}}("{{.Path}}", {{.Package}}.{{.Handler}}(svcCtx))
{{end}}	}
{{end}}}