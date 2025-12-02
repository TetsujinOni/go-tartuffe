# Plugin Development Guide

go-tartuffe supports two types of plugins:
- **Protocol Plugins**: Custom network protocols (e.g., gRPC, WebSocket, MongoDB wire protocol)
- **Repository Plugins**: Custom persistence backends (e.g., Redis, PostgreSQL, MongoDB)

Both can be implemented as:
- **Out-of-process**: Subprocess-based plugins compatible with mountebank (any language)
- **In-process**: Go plugin `.so` files for native performance

## Protocol Plugins

### Out-of-Process Protocol Plugins (mountebank-compatible)

Out-of-process plugins run as separate processes and communicate with go-tartuffe via JSON/HTTP.

#### Configuration (protocols.json)

Create a `protocols.json` file:

```json
{
  "grpc": {
    "createCommand": "/path/to/grpc-plugin"
  },
  "websocket": {
    "createCommand": "node /path/to/ws-plugin.js"
  }
}
```

Start go-tartuffe with:
```bash
tartuffe --protofile protocols.json
```

#### Plugin Lifecycle

1. **Startup**: go-tartuffe spawns your plugin with the imposter config as the last CLI argument
2. **Initialization**: Plugin reads config, starts listening, and outputs startup JSON to stdout
3. **Runtime**: Plugin handles requests and calls back to go-tartuffe for stub matching
4. **Shutdown**: go-tartuffe sends SIGINT; plugin shuts down gracefully

#### Startup Message

Your plugin must output a JSON startup message to stdout:

```json
{
  "port": 3000,
  "pid": 12345,
  "metadata": {
    "version": "1.0.0"
  }
}
```

#### Callback API

When your plugin receives a request, POST to go-tartuffe to find matching stubs:

```
POST http://localhost:2525/imposters/{port}/_requests
Content-Type: application/json

{
  "request": {
    "method": "GET",
    "path": "/api/users",
    "headers": {"Accept": "application/json"},
    "body": ""
  }
}
```

Response:
```json
{
  "response": {
    "statusCode": 200,
    "headers": {"Content-Type": "application/json"},
    "body": "{\"users\": []}"
  },
  "stubIndex": 0,
  "matched": true
}
```

### In-Process Protocol Plugins (Go)

For native Go plugins, implement the `protocol.ProtocolPlugin` interface:

```go
package main

import (
    "github.com/TetsujinOni/go-tartuffe/internal/models"
    "github.com/TetsujinOni/go-tartuffe/internal/plugin/protocol"
)

type MyProtocol struct{}

func (p *MyProtocol) Name() string {
    return "myprotocol"
}

func (p *MyProtocol) CreateServer(imp *models.Imposter, callback protocol.CallbackClient) (protocol.ProtocolServer, error) {
    return &MyServer{imposter: imp, callback: callback}, nil
}

func (p *MyProtocol) ValidateConfig(imp *models.Imposter) error {
    return nil
}

func (p *MyProtocol) DefaultPort() int {
    return 9000
}

// Export the plugin
var ProtocolPlugin protocol.ProtocolPlugin = &MyProtocol{}
```

Build as a Go plugin:
```bash
go build -buildmode=plugin -o myprotocol.so
```

Load with:
```bash
tartuffe --plugins /path/to/plugins/
```

## Repository Plugins

### Connection Strings

go-tartuffe uses URI-style connection strings:

```bash
# In-memory (default)
tartuffe

# Filesystem
tartuffe --impostersRepository file:///var/lib/tartuffe

# Redis (requires plugin)
tartuffe --impostersRepository redis://localhost:6379/0?prefix=mb:

# PostgreSQL (requires plugin)
tartuffe --impostersRepository postgres://user:pass@localhost/mountebank
```

### In-Process Repository Plugins (Go)

Implement the `repository.RepositoryPlugin` interface:

```go
package main

import (
    "github.com/TetsujinOni/go-tartuffe/internal/models"
    pluginrepo "github.com/TetsujinOni/go-tartuffe/internal/plugin/repository"
)

type RedisRepository struct {
    // Redis client, etc.
}

func (r *RedisRepository) Name() string {
    return "redis"
}

func (r *RedisRepository) Initialize(config pluginrepo.Config) error {
    // Connect to Redis using config.ConnectionString
    return nil
}

func (r *RedisRepository) Close() error {
    // Close Redis connection
    return nil
}

func (r *RedisRepository) HealthCheck() error {
    // PING Redis
    return nil
}

// Implement all repository.Repository methods...
func (r *RedisRepository) Add(imp *models.Imposter) error { ... }
func (r *RedisRepository) Get(port int) (*models.Imposter, error) { ... }
// ... etc.

// Export the factory
var Name string = "redis"

func RepositoryPluginFactory(config pluginrepo.Config) (pluginrepo.RepositoryPlugin, error) {
    repo := &RedisRepository{}
    if err := repo.Initialize(config); err != nil {
        return nil, err
    }
    return repo, nil
}
```

Build and load the same way as protocol plugins.

## Plugin Configuration Reference

### CLI Flags

| Flag | Description |
|------|-------------|
| `--protofile` | Path to protocols.json for out-of-process protocols |
| `--plugins` | Directory containing Go plugin .so files |
| `--impostersRepository` | Repository connection string |

### Repository Connection String Formats

| Scheme | Format | Example |
|--------|--------|---------|
| `memory` | `memory://` | Default in-memory storage |
| `file` | `file:///path` | `file:///var/lib/tartuffe` |
| `redis` | `redis://[pass@]host:port/db?prefix=...` | `redis://localhost:6379/0` |
| `postgres` | `postgres://user:pass@host:port/db` | `postgres://mb:secret@localhost/mountebank` |

## Built-in Protocols

go-tartuffe includes these built-in protocols:

- `http` - HTTP/1.1 server
- `https` - HTTPS server (TLS)
- `tcp` - Raw TCP server
- `smtp` - SMTP server

## Examples

See the `examples/plugins/` directory for complete examples:

- `examples/plugins/echo-protocol/` - Simple echo protocol (out-of-process, Go)
- `examples/plugins/grpc-protocol/` - gRPC protocol plugin
- `examples/plugins/redis-repository/` - Redis repository plugin

## Debugging Plugins

### Out-of-Process Plugins

1. Set `--loglevel debug` on go-tartuffe
2. Check plugin stdout/stderr for errors
3. Use `curl` to manually test the callback endpoint

### In-Process Plugins

1. Build with debug symbols: `go build -gcflags="all=-N -l" -buildmode=plugin`
2. Use `dlv` to debug the main process
3. Check go-tartuffe logs for plugin loading errors
