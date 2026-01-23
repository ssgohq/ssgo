# ssgo

All-in-one Go development toolkit with code generation and runtime utilities.

## Features

- **Code Generation**: Generate API handlers, RPC services, and database models
- **Runtime SDK**: Production-ready components for logging, tracing, metrics, and more
- **Database Support**: Generate models and repositories for GORM, Bun, and SQLC
- **API Development**: Generate Hertz HTTP handlers from API specifications
- **RPC Services**: Generate Kitex RPC services from Proto definitions
- **Development Tools**: Hot reload, TUI log aggregation, and process management
- **gRPC REPL**: Interactive gRPC client with auto proto detection (Evans)

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
go install github.com/ssgohq/ssgo/tool/cmd/ss@latest
```

### Manual Download

Download the latest release from [GitHub Releases](https://github.com/ssgohq/ssgo/releases).

## Quick Start

### Initialize Project

```bash
# Scan project and generate .ss.yaml config
ss init
```

### Run Development Server

```bash
# Run all services with hot reload and TUI
ss dev

# Run specific services
ss dev user-service order-service

# Run without TUI
ss dev --no-tui
```

### Generate API from Specification

```bash
# Create a new API specification
ss api new user

# Generate Hertz handlers from .api file
ss api gen --api api/user.api -m github.com/myorg/user-api
```

### Generate RPC Service

```bash
# Create a new proto file
ss rpc new user

# Generate Kitex RPC service
ss rpc gen --proto idl/user.proto --service UserService -m github.com/myorg/user-rpc

# Generate shared proto models only
ss rpc model --proto idl/user.proto -m github.com/myorg/common-pb -o common-pb
```

### Generate Database Models

```bash
# Generate Bun models from database
ss db bun gen --dsn "postgres://user:pass@localhost:5432/mydb"

# Generate GORM models
ss db gorm gen --dsn "postgres://user:pass@localhost:5432/mydb"

# Initialize SQLC project
ss db sqlc init --dir ./my-service --migrations ../migrations
```

### Interactive gRPC REPL

```bash
# Interactive: select service from .ss.yaml, enter Evans REPL
ss repl

# Enter REPL for specific service
ss repl user-service

# CLI mode: list services
ss repl user-service cli list

# CLI mode: call a method
ss repl user-service cli call UserService.GetUser
```

## Commands

| Command | Description |
|---------|-------------|
| `ss version` | Show version information |
| `ss init` | Initialize project config (.ss.yaml) |
| `ss dev [services...]` | Run services with hot reload |
| `ss api new <name>` | Create a new API specification |
| `ss api gen` | Generate API handlers from .api file |
| `ss api doc` | Generate OpenAPI documentation |
| `ss rpc new <name>` | Create a new proto file |
| `ss rpc gen` | Generate RPC service from proto |
| `ss rpc model` | Generate shared proto models |
| `ss db bun gen` | Generate Bun models and repositories |
| `ss db gorm gen` | Generate GORM models and repositories |
| `ss db sqlc init` | Initialize SQLC configuration |
| `ss db sqlc gen` | Generate SQLC code |
| `ss repl [service]` | Interactive gRPC REPL (Evans) |

## Configuration

### .ss.yaml

```yaml
run:
  build_delay: 500ms
  kill_delay: 5s
  services:
    - name: user-service
      dir: ./services/user
      cmd: go run ./cmd/main.go -f etc/config.yaml
      color: cyan
      use:
        - ../common-pb    # shared proto modules
      watch:
        include:
          - "**/*.go"
          - "**/*.yaml"
        exclude:
          - "**/vendor/**"
          - "**/*_test.go"

    - name: order-service
      dir: ./services/order
      cmd: go run ./cmd/main.go -f etc/config.yaml
      color: green
      depends_on:
        - user-service
```