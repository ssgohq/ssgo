# ssgo

All-in-one Go development toolkit with code generation and runtime utilities.

## Features

- **Code Generation**: Generate API handlers, RPC services, and database models
- **Runtime SDK**: Production-ready components for logging, tracing, metrics, and more
- **Database Support**: Generate models and repositories for GORM, Bun, and SQLC
- **API Development**: Generate Hertz HTTP handlers from API specifications
- **RPC Services**: Generate Kitex RPC services from Proto definitions
- **Development Tools**: Hot reload, log aggregation, and process management

## Installation

### Using Homebrew (recommended)

```bash
brew install --cask ssgohq/tap/ss
```

### Upgrade

```bash
brew upgrade --cask ssgohq/tap/ss
```

### From Source

```bash
go install github.com/ssgohq/ssgo/tool/cmd/ssgo@latest
```

### Manual Download

Download the latest release from [GitHub Releases](https://github.com/ssgohq/ssgo/releases).

## Quick Start

### Check Version

```bash
ss version
```

### Generate API from Specification

```bash
# Create a new API project
ss api new myapp

# Generate handlers from .api file
ss api gen ./idl/api.api
```

### Generate RPC Service

```bash
# Generate RPC service from Proto
ss rpc gen ./idl/service.proto

# Generate a new RPC project
ss rpc new myservice
```

### Generate Database Models

```bash
# Parse database schema and generate GORM models
ss db parse --dsn "user:pass@tcp(localhost:3306)/mydb" --output ./internal/model --orm gorm

# Generate Bun models
ss db bun --dsn "postgres://user:pass@localhost:5432/mydb" --output ./internal/model

# Initialize SQLC project
ss db sqlc init
```

### Run Development Server

```bash
# Run with hot reload
ss run ./cmd/main.go
```

## Commands

| Command | Description |
|---------|-------------|
| `ss version` | Show version information |
| `ss api new <name>` | Create a new API project |
| `ss api gen <file>` | Generate API handlers from specification |
| `ss rpc new <name>` | Create a new RPC service |
| `ss rpc gen <file>` | Generate RPC service from Proto |
| `ss db parse` | Parse database schema and generate models |
| `ss db gorm` | Generate GORM models |
| `ss db bun` | Generate Bun models |
| `ss db sqlc` | Generate SQLC configuration and queries |
| `ss run <main>` | Run application with hot reload |

## SDK Components

The ssgo SDK provides production-ready components:

```go
import (
    "github.com/ssgohq/ssgo/sdk/app"
    "github.com/ssgohq/ssgo/sdk/logx"
    "github.com/ssgohq/ssgo/sdk/metric"
    "github.com/ssgohq/ssgo/sdk/trace"
)
```

### Available Packages

| Package | Description |
|---------|-------------|
| `sdk/app` | Application lifecycle management |
| `sdk/lifecycle` | Service lifecycle and health checks |
| `sdk/logx` | Structured logging with zap |
| `sdk/metric` | Prometheus metrics (counter, gauge, histogram) |
| `sdk/trace` | OpenTelemetry tracing |
| `sdk/middleware` | HTTP middleware (CORS, JWT, logging) |
| `sdk/srpc` | RPC client and server utilities |
| `sdk/stores` | Database connection helpers |

## Project Structure

```
ssgo/
├── sdk/                  # Runtime SDK packages
│   ├── app/             # Application management
│   ├── lifecycle/       # Service lifecycle
│   ├── logx/            # Logging
│   ├── metric/          # Metrics
│   ├── middleware/      # HTTP middleware
│   ├── srpc/            # RPC utilities
│   ├── stores/          # Database connections
│   └── trace/           # Tracing
├── tool/                 # CLI tool
│   ├── cmd/ssgo/        # Main CLI entry point
│   └── internal/        # CLI internal packages
├── internal/             # Shared internal packages
│   ├── ast/             # AST parsers
│   └── dbparser/        # Database schema parser
└── examples/             # Example projects
```

## Development

### Prerequisites

- Go 1.25+
- golangci-lint (for linting)
- goreleaser (for releases)

### Build

```bash
make build
```

### Test

```bash
make test
```

### Lint

```bash
make lint
```

### Install Locally

```bash
make install
```

### Clean

```bash
make clean
```

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'feat: add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## Related Projects

- [Hertz](https://github.com/cloudwego/hertz) - HTTP framework
- [Kitex](https://github.com/cloudwego/kitex) - RPC framework