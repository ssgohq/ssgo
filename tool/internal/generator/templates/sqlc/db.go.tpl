package store

import (
	"context"
	"fmt"
{{if .WithTrace}}
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
{{else}}
	"github.com/jackc/pgx/v5/pgxpool"
{{end}}

	"{{.Module}}/internal/config"
)

{{if .WithTrace}}
// pgxTracer implements pgx.QueryTracer for OpenTelemetry
type pgxTracer struct {
	tracer trace.Tracer
}

// newPgxTracer creates a new pgx tracer
func newPgxTracer() *pgxTracer {
	return &pgxTracer{
		tracer: otel.Tracer("pgx"),
	}
}

// TraceQueryStart is called at the beginning of Query, QueryRow, and Exec calls
func (t *pgxTracer) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	ctx, _ = t.tracer.Start(ctx, "pgx.query",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("db.system", "postgresql"),
			attribute.String("db.statement", data.SQL),
		),
	)
	return ctx
}

// TraceQueryEnd is called at the end of Query, QueryRow, and Exec calls
func (t *pgxTracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	span := trace.SpanFromContext(ctx)
	if data.Err != nil {
		span.RecordError(data.Err)
		span.SetStatus(codes.Error, data.Err.Error())
	}
	span.SetAttributes(attribute.String("db.command_tag", data.CommandTag.String()))
	span.End()
}

// NewPgxPool creates a new pgxpool.Pool from config with tracing
func NewPgxPool(cfg *config.DBConfig) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database, cfg.SSLMode,
	)

	poolCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pool config: %w", err)
	}

	// Add OpenTelemetry tracing
	poolCfg.ConnConfig.Tracer = newPgxTracer()

	pool, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}

	// Verify connection
	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return pool, nil
}
{{else}}
// NewPgxPool creates a new pgxpool.Pool from config
func NewPgxPool(cfg *config.DBConfig) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database, cfg.SSLMode,
	)

	poolCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pool config: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}

	// Verify connection
	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return pool, nil
}
{{end}}