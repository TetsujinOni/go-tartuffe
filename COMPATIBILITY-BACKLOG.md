# Compatibility Backlog

Remaining work to improve mountebank API compatibility.

## Current Status

**Progress**: 216/252 passing (85.7%) | 216/237 adjusted (91.1%) ✅ **TARGET EXCEEDED**
**Target**: 75%+ compatibility
**Last Validation**: 2026-01-19

## Remaining Failures: 36 Tests

**Breakdown**:
- Actionable: 21 tests (see below)
- Security/Architectural (Won't Fix): 14 tests
- Deliberate Design Difference: 1 test

**Test Mapping**: See [docs/MOUNTEBANK-TEST-MAPPING.md](docs/MOUNTEBANK-TEST-MAPPING.md) for complete analysis.

## Actionable Failures by Priority

### P0 - Critical (2 tests)

#### TCP endOfRequestResolver (2 tests)
- **Status**: Model exists, not implemented
- **Tests**:
  - `should allow binary requests extending beyond a single packet using endOfRequestResolver`
  - `should allow text requests extending beyond a single packet using endOfRequestResolver`
- **File**: `mbTest/api/tcp/tcpImposterTest.js`
- **Work**: Implement custom request boundary detection in TCP server
- **Files**: `internal/imposter/tcp_server.go`

### P1 - High Priority (11 tests)

#### HTTP Proxy Features (2 tests)
- **Tests**:
  - `should proxy to https` - Cross-protocol proxying (HTTP→HTTPS)
  - `should allow proxy stubs to invalid domains` - DNS error handling
- **File**: `mbTest/api/http/httpProxyStubTest.js`
- **Files**: `internal/imposter/proxy.go`

#### HTTPS Features (2 tests)
- **Tests**:
  - `should support sending key/cert pair during imposter creation` - Certificate validation
  - `should support proxying to origin server requiring mutual auth` - mTLS proxy
- **File**: `mbTest/api/https/httpsCertificateTest.js`
- **Files**: `internal/imposter/http_server.go`

#### TCP Implementation (7 tests)
- **Tests**:
  - `should provide access to all requests` - TCP request recording
  - `should split each packet into a separate request by default` - Packet splitting logic
  - `should allow proxy stubs to invalid hosts` - Error handling
  - **TCP Proxy** (4 tests):
    - `should obey endOfRequestResolver` - endOfRequestResolver in proxy mode
    - `should gracefully deal with DNS errors`
    - `should gracefully deal with non listening ports`
    - `should reject non-tcp protocols` - Protocol validation
- **Files**: `mbTest/api/tcp/*.js`
- **Work**: Implement TCP proxy forwarding, error handling, packet splitting
- **Files**: `internal/imposter/tcp_server.go`, TCP proxy handler

### P2 - Medium Priority (8 tests)

#### HTTP Behavior Composition (2 tests)
- **Tests**:
  - `should compose multiple behaviors together (old interface for backwards compatibility)`
  - `should apply multiple behaviors in sequence with repeat (new format)`
- **File**: `mbTest/api/http/httpBehaviorsTest.js`
- **Work**: Implement repeat behavior, fix behavior composition
- **Files**: `internal/imposter/behaviors.go`

#### HTTP Proxy Advanced Features (3 tests)
- **Tests**:
  - `should handle the connect method` - CONNECT method support
  - `should persist behaviors from origin server` - Behavior persistence
  - `should support adding latency to saved responses...` - addWaitBehavior
- **File**: `mbTest/api/http/httpProxyStubTest.js`
- **Work**: CONNECT tunneling, behavior persistence, latency tracking
- **Files**: `internal/imposter/proxy.go`

#### HTTP Fault Handling (2 tests)
- **Tests**: `should do nothing when undefined fault is specified` (HTTP + HTTPS)
- **Files**: `mbTest/api/http/httpFaultTest.js`, `mbTest/api/https/httpsFaultTest.js`
- **Work**: Gracefully handle unknown fault types

#### HTTP Injection (1 test)
- **Test**: `should provide access to all requests` - Requests array in injection context
- **File**: `mbTest/api/http/httpInjectionTest.js`
- **Work**: Pass requests array to JavaScript injection functions
- **Files**: `internal/imposter/inject.go`

### P3 - Low Priority (3 tests)

#### Async JavaScript (3 tests)
- **Tests**:
  - `should allow asynchronous injection` (HTTP injection)
  - `should allow asynchronous injection (old interface)` (TCP injection)
  - `should allow asynchronous injection` (TCP injection)
- **Status**: goja ES5.1 limitation - no native Promise/async support
- **Work**: May require upstream goja enhancement or workarounds
- **Priority**: Low - rarely used feature

## Won't Fix (Security/Architectural - 14 tests)

### shellTransform Behavior (8 tests)
- **Reason**: Arbitrary command execution security risk
- **Tests**: 4 HTTP + 4 HTTPS behavior tests
- **Alternative**: Use `decorate` behavior with sandboxed JavaScript
- **Reference**: `docs/SECURITY.md`

### Process Object Access (2 tests)
- **Reason**: Environment variable exposure security risk
- **Tests**: HTTP + HTTPS injection tests
- **Decision**: Security sandbox is priority over compatibility

### Private Key Return (1 test)
- **Reason**: Security hardening - keys could be logged/exposed
- **Test**: HTTPS key/cert pair creation test
- **Decision**: Never return private keys in API responses

### HTTP Proxy Replayable Export (1 test)
- **Test**: `should support retrieving replayable JSON with proxies removed for later playback`
- **Status**: Partial implementation (removeProxies works, but export format differs)
- **Priority**: Low - edge case feature

### TCP Old Proxy Syntax (1 test)
- **Test**: `should support old proxy syntax for backwards compatibility`
- **Reason**: Legacy compatibility for deprecated format
- **Decision**: Not supporting deprecated syntax

### TCP Behavior Composition (1 test)
- **Test**: `should compose multiple behaviors together`
- **Status**: Low priority TCP-specific edge case

## Deliberate Design Difference (1 test)

### Prometheus Metrics Format
- **Test**: Metrics-related test in `mbTest/api/http/httpMetricsTest.js`
- **Reason**: Industry-standard Prometheus format vs. mountebank custom JSON
- **Benefits**: Native scraping, better performance, wider tooling compatibility
- **Decision**: Modern observability standard over legacy format

## Validation Procedure

See `.claude/claude.md` for complete validation workflow.

### Quick Validation

```bash
cd /home/tetsujinoni/work/mountebank
pkill -f tartuffe 2>/dev/null || true
MB_EXECUTABLE=/home/tetsujinoni/work/go-tartuffe/bin/tartuffe-wrapper.sh npm run test:api
```

Expected: ~215 passing / 37 failing (85.3%)

## Documentation References

- **Test Mapping**: [docs/MOUNTEBANK-TEST-MAPPING.md](docs/MOUNTEBANK-TEST-MAPPING.md)
- **Controller Tests**: [docs/CONTROLLER-API-TEST-MAPPING.md](docs/CONTROLLER-API-TEST-MAPPING.md)
- **HTTP Proxy**: [docs/HTTP-PROXY-TEST-MAPPING.md](docs/HTTP-PROXY-TEST-MAPPING.md)
- **Security**: [docs/SECURITY.md](docs/SECURITY.md)
- **Workflows**: [.claude/claude.md](.claude/claude.md)

---

**Note**: Historical results and fix summaries are in `docs/` directory. This backlog contains only remaining work.
