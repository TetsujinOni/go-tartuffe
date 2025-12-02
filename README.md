# go-tartuffe

A high-performance Go implementation of [mountebank](https://www.mbtest.org), the open source tool for creating test doubles over the wire.

This project is an AI-assisted port exploring full-repo replatforming while addressing scaling challenges inherent in Node.js-hosted tooling.

## Features

- **Protocol Support**: HTTP, HTTPS, TCP, SMTP, gRPC
- **Mountebank API Compatible**: Drop-in replacement for mountebank 2.9.3
- **Predicates**: equals, deepEquals, contains, startsWith, endsWith, matches, exists, not, or, and, inject
- **Responses**: is, proxy, inject, fault
- **Behaviors**: wait, repeat, copy, lookup, decorate, shellTransform
- **Persistence**: In-memory, filesystem, or custom repository plugins
- **Extensible**: Go plugin system for custom protocols and repositories

## Installation

### From Source

```bash
git clone https://github.com/TetsujinOni/go-tartuffe.git
cd go-tartuffe
go build -o bin/tartuffe ./cmd/tartuffe
```

### Docker

```bash
docker pull ghcr.io/tetsujinoni/go-tartuffe:latest
docker run -p 2525:2525 ghcr.io/tetsujinoni/go-tartuffe:latest
```

Or build locally:

```bash
docker build -t tartuffe .
docker run -p 2525:2525 tartuffe
```

## Quick Start

Start the server:

```bash
./bin/tartuffe --port 2525
```

Create an imposter:

```bash
curl -X POST http://localhost:2525/imposters -H "Content-Type: application/json" -d '{
  "port": 4545,
  "protocol": "http",
  "stubs": [{
    "responses": [{"is": {"statusCode": 200, "body": "Hello, World!"}}],
    "predicates": [{"equals": {"path": "/hello"}}]
  }]
}'
```

Test it:

```bash
curl http://localhost:4545/hello
# Hello, World!
```

## CLI Options

| Flag | Default | Description |
|------|---------|-------------|
| `--port` | 2525 | Port for the mountebank server |
| `--host` | "" | Hostname to bind to |
| `--allowInjection` | false | Allow JavaScript injection |
| `--localOnly` | false | Only accept requests from localhost |
| `--configfile` | "" | Load imposters from file (supports EJS templates) |
| `--datadir` | "" | Directory for imposter persistence |
| `--loglevel` | info | Log level (debug, info, warn, error) |
| `--logfile` | mb.log | Log file path |
| `--nologfile` | false | Disable file logging |
| `--debug` | false | Include stub match info in responses |
| `--origin` | "" | CORS allowed origin |
| `--apikey` | "" | API key for authentication |
| `--protofile` | "" | Custom protocols configuration |
| `--plugins` | "" | Directory containing Go plugins |

### Subcommands

```bash
# Save current imposters to file
./bin/tartuffe save --savefile imposters.json

# Switch proxies to replay mode
./bin/tartuffe replay

# Stop a running instance
./bin/tartuffe stop --pidfile mb.pid
```

## Docker Usage

```bash
# Basic usage
docker run -p 2525:2525 tartuffe

# With injection enabled
docker run -p 2525:2525 tartuffe --allowInjection

# With persistent data
docker run -p 2525:2525 -v $(pwd)/data:/app/data tartuffe --datadir /app/data

# With config file
docker run -p 2525:2525 -v $(pwd)/imposters.json:/app/imposters.json tartuffe --configfile /app/imposters.json
```

## Development

```bash
# Run tests
make test

# Build binary
make build

# Run linter
make lint

# Format code
make fmt
```

## Documentation

- [Plugin Development](docs/plugins.md) - Creating custom protocol and repository plugins
- [Mountebank Documentation](https://www.mbtest.org/docs/gettingStarted) - API reference and concepts

## Compatibility

This project aims for API compatibility with mountebank 2.9.3. Some advanced features may have implementation differences.

## License

MIT License - see [LICENSE](LICENSE) for details.

## Acknowledgments

This project is a port of [mountebank](https://www.mbtest.org) by Brandon Byars. See the original project for comprehensive documentation on service virtualization concepts.
