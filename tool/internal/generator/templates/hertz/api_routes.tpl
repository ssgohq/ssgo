// Code scaffolded by ssgo. Safe to edit.
// Add your custom routes and middleware configuration here.

package handler

import (
	"github.com/cloudwego/hertz/pkg/app/server"
	"{{.Module}}/internal/svc"
)

// RegisterHandlers registers all API routes with the Hertz server.
// This is the main entry point for route registration.
func RegisterHandlers(r *server.Hertz, svcCtx *svc.ServiceContext) {
	// Register generated handlers from .api spec
	registerGeneratedHandlers(r, svcCtx)

	// ============================================================
	// Add your custom routes or global middleware below
	// ============================================================
	// Example global middleware:
	// r.Use(middleware.YourGlobalMiddleware())
	//
	// Example custom route:
	// r.GET("/health", func(ctx context.Context, c *app.RequestContext) {
	//     c.JSON(200, map[string]string{"status": "ok"})
	// })
}
