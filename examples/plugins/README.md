# Example Plugins

This directory contains example plugins for go-tartuffe.

## Echo Protocol Plugin

A simple TCP echo protocol that demonstrates out-of-process plugin development.

### Build

```bash
cd echo-protocol
go build -o echo-plugin
```

### Configure

Create a `protocols.json` file:

```json
{
  "echo": {
    "createCommand": "/path/to/echo-plugin"
  }
}
```

### Run

```bash
# Start go-tartuffe with the plugin
tartuffe --protofile protocols.json

# Create an echo imposter
curl -X POST http://localhost:2525/imposters -d '{
  "protocol": "echo",
  "port": 3000,
  "stubs": [{
    "responses": [{
      "is": { "data": "Hello from echo!\n" }
    }]
  }]
}'

# Test it
echo "test" | nc localhost 3000
```

## Creating Your Own Plugin

See the [Plugin Development Guide](/docs/plugins.md) for complete documentation.

### Out-of-Process Protocol Plugin Checklist

1. Parse imposter config from last CLI argument
2. Start listening on the configured port
3. Output startup JSON to stdout: `{"port": 3000, "pid": 12345}`
4. For each request, POST to callback URL for stub matching
5. Handle SIGINT/SIGTERM for graceful shutdown

### In-Process Go Plugin Checklist

1. Implement `protocol.ProtocolPlugin` interface
2. Export `var ProtocolPlugin protocol.ProtocolPlugin = &YourPlugin{}`
3. Build with `go build -buildmode=plugin -o yourplugin.so`
4. Place in plugins directory and start with `--plugins /path/to/plugins/`
