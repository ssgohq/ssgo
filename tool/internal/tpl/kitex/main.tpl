package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/cloudwego/kitex/server"
	"github.com/kitex-contrib/monitor-prometheus"
	"github.com/ssgohq/goten-core/app"
	"github.com/ssgohq/goten-core/metric"
	"github.com/ssgohq/goten-core/srpc"
	"gopkg.in/yaml.v3"

	"{{.Module}}/internal/config"
	svrImpl "{{.Module}}/internal/server"
	"{{.Module}}/internal/svc"
	"{{.TypesModule}}/kitex_gen/{{.ServiceLower}}/{{.ServiceLower}}service"
)

var configFile = flag.String("c", "etc/config.yaml", "config file")

func main() {
	flag.Parse()
	defer app.WithLogger("{{.ServiceLower}}")()

	cfg := mustLoadConfig(*configFile)
	svcCtx, err := svc.NewServiceContext(cfg)
	if err != nil {
		panic(fmt.Sprintf("failed to create service context: %v", err))
	}

	metric.StartAgent(cfg.Metric)

	opts := srpc.NewServerBuilder(&cfg.ServerConfig).Build()
	// Use DisableServer option since metric.StartAgent already provides promhttp endpoint
	opts = append(opts, server.WithTracer(prometheus.NewServerTracer(
		cfg.Metric.Addr(),
		cfg.Metric.MetricsPath,
		prometheus.WithDisableServer(true),
	)))

	svr := {{.ServiceLower}}service.NewServer(
		svrImpl.New{{.Service}}Server(svcCtx),
		opts...,
	)

	app.New(app.Config{
		Name:          cfg.Name,
		EnableTracing: cfg.Trace.IsEnabled(),
		Trace:         cfg.Trace,
	}).AddRPC("rpc", svr).
		OnStart(app.HookMarkReady, func(ctx context.Context) error {
			metric.SetReady(true)
			return nil
		}).
		OnStop(app.HookMarkNotReady, func(ctx context.Context) error {
			metric.SetReady(false)
			return nil
		}).
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
	c.ServerConfig.SetDefaults()
	c.Metric.SetDefaults()
	return c
}