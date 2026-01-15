// Code scaffolded by ssgo. Safe to edit.

package svc

import (
	"{{.Module}}/internal/config"
)

// ServiceContext holds all dependencies for the service.
// Add your dependencies here (database connections, caches, clients, etc.)
type ServiceContext struct {
	Config config.Config
	// Add your dependencies here:
	// DB     *bun.DB
	// Redis  *redis.Client
	// etc.
}

// NewServiceContext creates a new ServiceContext with the given config.
// Initialize all dependencies here.
func NewServiceContext(c config.Config) (*ServiceContext, error) {
	svc := &ServiceContext{
		Config: c,
		// Initialize dependencies:
		// DB:    initDB(c),
		// Redis: initRedis(c),
	}

	return svc, nil
}