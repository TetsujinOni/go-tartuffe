# go-tartuffe Progress

A Go implementation of [mountebank](http://www.mbtest.dev/) - the first open source tool for cross-platform, multi-protocol test doubles over the wire.
Tartuffe is quite clearly not the first, and was  undertaken as both an exploration of the possibilities of Claude porting with Sonnett 4.5 support, and an opportunity to address some thread-scaling concerns in use of heavy multi-imposter mocking in integration testing. 

## Implementation Status

### Core Features

| Feature | Status | Notes |
|---------|--------|-------|
| HTTP Protocol | Implemented | Full request/response handling |
| API Endpoints | Implemented | Imposters CRUD, stubs CRUD, config, logs, /metrics |
| Request Recording | Implemented | `recordRequests` option |
| Default Responses | Implemented | `defaultResponse` configuration |
| Response Cycling | Implemented | Multiple responses with `repeat` |
| Hypermedia Links | Implemented | `_links` in responses |
| Binary Mode | Implemented | Base64 encoding/decoding for binary data (`_mode: "binary"`) |
| Host Binding | Implemented | Imposters can bind to specific hostname/IP |
| Form Parsing | Implemented | Auto-parse `application/x-www-form-urlencoded` and `multipart/form-data` |
| Prometheus Metrics | Implemented | `/metrics` endpoint for monitoring |

### Predicates

| Predicate | Status | Notes |
|-----------|--------|-------|
| equals | Implemented | Field equality matching |
| deepEquals | Implemented | Deep object equality |
| contains | Implemented | Substring matching |
| startsWith | Implemented | Prefix matching |
| endsWith | Implemented | Suffix matching |
| matches | Implemented | Regex matching |
| exists | Implemented | Field existence check |
| and | Implemented | Logical AND |
| or | Implemented | Logical OR |
| not | Implemented | Logical NOT |
| inject | Implemented | JavaScript predicate injection |

### Predicate Options

| Option | Status | Notes |
|--------|--------|-------|
| caseSensitive | Implemented | Case-sensitive matching |
| keyCaseSensitive | Implemented | Case-sensitive key matching for headers/query params |
| except | Implemented | Regex exclusion (strips matching pattern before comparison) |
| xpath | Implemented | XPath selector for XML bodies |
| jsonpath | Implemented | JSONPath selector for JSON bodies |

### Response Types

| Type | Status | Notes |
|------|--------|-------|
| is | Implemented | Static responses |
| proxy | Implemented | Proxy to real service (proxyOnce, proxyAlways, proxyTransparent modes), mTLS support |
| inject | Implemented | JavaScript response injection (using goja engine) |
| fault | Implemented | CONNECTION_RESET_BY_PEER, RANDOM_DATA_THEN_CLOSE |

### Proxy Features

| Feature | Status | Notes |
|---------|--------|-------|
| proxyOnce | Implemented | Proxy once, then replay from recorded stub |
| proxyAlways | Implemented | Always proxy, update existing stub |
| proxyTransparent | Implemented | Always proxy, no recording |
| predicateGenerators | Implemented | Generate predicates from proxied requests |
| injectHeaders | Implemented | Inject headers into proxy request |
| addWaitBehavior | Implemented | Add wait behavior based on proxy latency |
| addDecorateBehavior | Implemented | Add decorate behavior to recorded stubs |
| mTLS (cert/key) | Implemented | Client certificate for mutual TLS proxy requests |
| secureProtocol | Implemented | TLS version selection (TLSv1, TLSv1.1, TLSv1.2, TLSv1.3) |

### Behaviors

| Behavior | Status | Notes |
|----------|--------|-------|
| wait | Implemented | Response delay (number or JS function) |
| copy | Implemented | Copy request values to response (regex, jsonpath) |
| lookup | Implemented | CSV data lookup |
| decorate | Implemented | JavaScript post-processing (old and new interfaces) |
| shellTransform | Implemented | Shell command transform |
| repeat | Implemented | Response repetition |

### Configuration

| Feature | Status | Notes |
|---------|--------|-------|
| EJS Templates | Implemented | include, stringify, inject, data variables |
| Config File Loading | Implemented | `--configfile` option |
| noParse | Implemented | Raw JSON mode |
| Custom Formatters | Not Implemented | Plugin system |

### CLI Commands

| Command | Status | Notes |
|---------|--------|-------|
| start | Implemented | Start server |
| stop | Implemented | Stop server via PID file |
| save | Implemented | Save imposters to file |
| replay | Implemented | Switch proxies to replay mode |

### Security & Options

| Feature | Status | Notes |
|---------|--------|-------|
| --port | Implemented | Server port |
| --host | Implemented | Bind hostname |
| --allowInjection | Implemented | Enable JavaScript injection |
| --localOnly | Implemented | Localhost-only access |
| --ipWhitelist | Implemented | IP-based access control |
| --origin | Implemented | CORS origin |
| --apikey | Implemented | API key authentication |
| --datadir | Implemented | Filesystem persistence |
| --loglevel | Partial | Parsed but not fully used |
| --logfile | Implemented | Log file output |
| --nologfile | Implemented | Disable file logging |
| --pidfile | Implemented | PID file location |
| --debug | Implemented | Debug mode |

### Protocols

| Protocol | Status | Notes |
|----------|--------|-------|
| HTTP | Implemented | Full support |
| HTTPS | Implemented | TLS support with auto-generated or custom certs, mutual TLS |
| TCP | Implemented | Raw TCP mocking with text/binary modes, endOfRequestResolver |
| SMTP | Implemented | Email capture and recording for mock verification |
| gRPC | Implemented | Dynamic proto loading, all RPC types (unary/streaming), behaviors, reflection |

### gRPC Features

| Feature | Status | Notes |
|---------|--------|-------|
| Proto File Loading | Implemented | Runtime .proto file parsing via protocompile |
| Unary RPCs | Implemented | Request/response matching with JSON predicates |
| Server Streaming | Implemented | Use `stream` array in response for multiple messages |
| Client Streaming | Implemented | Collects all client messages, matches on first |
| Bidirectional Streaming | Implemented | Per-message matching and response |
| Request Recording | Implemented | Records service, method, message, metadata |
| gRPC Reflection | Implemented | Enable with `enableReflection: true` |
| Status Codes | Implemented | Full gRPC status code support |
| Metadata Matching | Implemented | Match on gRPC metadata (headers) |
| Behaviors | Implemented | wait, copy, decorate, lookup, shellTransform |

## Proxy Modes

| Mode | Status | Notes |
|------|--------|-------|
| proxyOnce | Implemented | Record once, then replay |
| proxyAlways | Implemented | Always proxy, record each time |
| proxyTransparent | Implemented | Proxy without recording |

## Next Steps (Priority Order)

All core protocols including gRPC (with full streaming support) are now implemented. Potential future enhancements:
1. **gRPC Proxy** - Proxy gRPC requests to real services
2. **SMTP TLS** - STARTTLS support for SMTP
3. **Custom Formatters** - Plugin system for custom formatting

## Testing

All integration tests passing. Tests cover:
- API endpoints
- HTTP imposter creation and management
- HTTPS imposter with TLS (auto-generated and custom certs, mutual TLS)
- Stub matching and responses
- Predicate evaluation (including inject predicates)
- Predicate options (except, keyCaseSensitive)
- Config file loading with EJS
- Save/replay functionality
- Proxy responses (proxyOnce, proxyAlways, proxyTransparent, predicateGenerators, injectHeaders)
- Inject responses (JavaScript execution, request access, JSON body handling)
- Fault responses (CONNECTION_RESET_BY_PEER, RANDOM_DATA_THEN_CLOSE)
- Behaviors (wait, copy, lookup, decorate, multiple behaviors combined)
- Binary mode (base64 encoding/decoding for requests and responses)
- TCP protocol (basic responses, predicates, binary mode, request recording, endOfRequestResolver)
- SMTP protocol (email capture, request recording, predicate matching)
- Selectors (JSONPath for JSON bodies, XPath for XML bodies)
- Form parsing (URL-encoded and multipart/form-data)
- Proxy mTLS (client certificates, TLS version selection)
- Prometheus metrics endpoint
- gRPC protocol (dynamic proto loading, unary/streaming RPCs, behaviors, request matching)

Run tests with:
```bash
go test ./...
```

## Compatibility

Target compatibility: mountebank v2.x API

The goal is to be a drop-in replacement for mountebank for HTTP protocol testing, with the performance benefits of a compiled Go binary.
