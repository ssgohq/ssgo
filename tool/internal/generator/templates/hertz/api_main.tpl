package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/ssgohq/goten-core/app"
	"github.com/ssgohq/goten-core/lifecycle"
	"github.com/ssgohq/goten-core/logx"
	"github.com/ssgohq/goten-core/metric"
	"gopkg.in/yaml.v3"

	"{{.Module}}/internal/config"
	"{{.Module}}/internal/api/handler"
	"{{.Module}}/internal/svc"
)

var configFile = flag.String("c", "etc/api.yaml", "config file")

func main() {
	flag.Parse()

	cfg := mustLoadConfig(*configFile)

	// Initialize logger
	logx.Init(cfg.Log)

	svcCtx := svc.NewServiceContext(cfg)

	// Create Hertz server
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	h := app.NewHertzServer(addr, app.WithTracing(cfg.IsTraceEnabled()))

	handler.RegisterHandlers(h, svcCtx)

	// Create and run application
	application := app.New(app.Config{
		Name:          cfg.Name,
		EnableTracing: cfg.IsTraceEnabled(),
		Trace:         cfg.Trace,
	})

	// Add HTTP server
	application.AddService(lifecycle.NewHertzAdapter("http", h))

	// Add metrics server if configured
	if cfg.Metric.Enabled {
		metricServer := metric.NewServer(cfg.Metric)
		application.AddService(lifecycle.NewFuncService("metrics",
			func(ctx context.Context) error {
				metricServer.Start()
				return nil
			},
			metricServer.Stop,
		))
	}

	application.MustRun(context.Background())
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
