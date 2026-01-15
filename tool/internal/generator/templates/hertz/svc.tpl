package svc

import (
	"{{.Module}}/internal/config"
)

type ServiceContext struct {
	Config config.Config
	// Add your dependencies here:
	// DB     *gorm.DB
	// Redis  *redis.Client
	// etc.
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config: c,
	}
}