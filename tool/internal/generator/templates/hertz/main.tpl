package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/ssgohq/goten-core/app"
	"gopkg.in/yaml.v3"

	"{{.Module}}/internal/config"
	"{{.Module}}/internal/handler"
	"{{.Module}}/internal/svc"
)

var configFile = flag.String("c", "etc/config.yaml", "config file")

func main() {
	flag.Parse()
	defer app.WithLogger("{{.ServiceName}}")()

	cfg := mustLoadConfig(*configFile)
	svcCtx := svc.NewServiceContext(cfg)

	// Create pre-configured server (tracing middleware included automatically)
	addr := cfg.Addr()
	h := app.NewHertzServer(addr, app.WithTracing(cfg.Trace.IsEnabled()))
	handler.RegisterHandlers(h, svcCtx)

	// App handles lifecycle: graceful shutdown, signal handling, tracing flush.
	// To close resources (DB, Redis, etc.) on shutdown, add:
	//   .OnCleanup(func(ctx context.Context) error { return db.Close() })
	app.New(app.Config{
		Name:          cfg.Name,
		EnableTracing: cfg.Trace.IsEnabled(),
		Trace:         cfg.Trace,
	}).AddHTTP("http", h, addr).
		MustRun(context.Background())
}

func mustLoadConfig(path string) config.Config {
	var c config.Config
	data, err := os.ReadFile(path)
	if err != nil {
		panic(fmt.Sprintf("failed to read config file: %v", err))
	}
	if err := yaml.Unmarshal(data, &c); err != nil {
		panic(fmt.Sprintf("failed to parse config file: %v", err))
	}
	return c
}