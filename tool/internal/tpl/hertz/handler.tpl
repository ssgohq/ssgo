package {{.Package}}

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"{{.Module}}/internal/logic/{{.Group}}"
	"{{.Module}}/internal/svc"
	"{{.Module}}/internal/types"
)

// {{.HandlerName}} {{.Doc}}
func {{.HandlerName}}(svcCtx *svc.ServiceContext) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
{{if .HasRequest}}		var req types.{{.RequestType}}
		if err := c.BindAndValidate(&req); err != nil {
			c.JSON(400, map[string]interface{}{"error": err.Error()})
			return
		}

		l := {{.LogicPackage}}.New{{.LogicName}}(ctx, svcCtx)
{{if .HasResponse}}		resp, err := l.{{.LogicMethod}}(&req)
		if err != nil {
			c.JSON(500, map[string]interface{}{"error": err.Error()})
			return
		}

		c.JSON(200, resp)
{{else}}		err := l.{{.LogicMethod}}(&req)
		if err != nil {
			c.JSON(500, map[string]interface{}{"error": err.Error()})
			return
		}

		c.JSON(200, map[string]interface{}{"message": "success"})
{{end}}{{else}}		l := {{.LogicPackage}}.New{{.LogicName}}(ctx, svcCtx)
{{if .HasResponse}}		resp, err := l.{{.LogicMethod}}()
		if err != nil {
			c.JSON(500, map[string]interface{}{"error": err.Error()})
			return
		}

		c.JSON(200, resp)
{{else}}		err := l.{{.LogicMethod}}()
		if err != nil {
			c.JSON(500, map[string]interface{}{"error": err.Error()})
			return
		}

		c.JSON(200, map[string]interface{}{"message": "success"})
{{end}}{{end}}	}
}