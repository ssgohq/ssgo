package middleware

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"{{.Module}}/internal/svc"
)

// {{.Name}} middleware
func {{.Name}}(svcCtx *svc.ServiceContext) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		// TODO: implement middleware logic here
		// Example: authentication, logging, rate limiting, etc.

		// Continue to next handler
		c.Next(ctx)
	}
}