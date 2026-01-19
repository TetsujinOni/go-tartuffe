# Compatibility Backlog

Remaining work to improve mountebank API compatibility.

## Current Status

**Progress**: 219/252 passing (86.9%) | 219/239 adjusted (91.6%) ✅ **TARGET EXCEEDED**
**Target**: 75%+ compatibility
**Last Validation**: 2026-01-19

## Remaining Failures: 33 Tests

**Breakdown**:
- Actionable: 20 tests (see below)
- Security/Architectural (Won't Fix): 9 tests
- goja Async Limitation: 4 tests

**Test Mapping**: See [docs/MOUNTEBANK-TEST-MAPPING.md](docs/MOUNTEBANK-TEST-MAPPING.md) for complete analysis.

## Recently Completed

### TCP endOfRequestResolver (2 tests) - DONE ✅
- **Implemented**: 2026-01-19
- **Tests now passing**:
  - `should allow binary requests extending beyond a single packet using endOfRequestResolver`
  - `should allow text requests extending beyond a single packet using endOfRequestResolver`
- **Implementation**:
  - `internal/imposter/inject.go`: `ExecuteEndOfRequestResolver()` with binary/text mode support
  - `internal/imposter/tcp_server.go`: `readRequest()` buffers multiple packets until resolver returns true
  - `internal/models/imposter.go`: JSON serialization fixed to use "requests" field consistently

## Actionable Failures by Priority

### P1 - High Priority (13 tests)

#### HTTP Proxy Features (5 tests)
- **Tests**:
  - `should proxy to https` - Cross-protocol proxying (HTTP→HTTPS)
  - `should allow proxy stubs to invalid domains` - DNS error handling
  - `should handle the connect method` - CONNECT method support
  - `should persist behaviors from origin server` - Behavior persistence
  - `should support adding latency to saved responses...` - addWaitBehavior
- **File**: `mbTest/api/http/httpProxyStubTest.js`
- **Files**: `internal/imposter/proxy.go`

#### HTTPS Features (2 tests)
- **Tests**:
  - `should support sending key/cert pair during imposter creation` - Certificate validation
  - `should support proxying to origin server requiring mutual auth` - mTLS proxy
- **File**: `mbTest/api/https/httpsCertificateTest.js`
- **Files**: `internal/imposter/http_server.go`

#### TCP Proxy/Features (6 tests)
- **Tests**:
  - `should obey endOfRequestResolver` - endOfRequestResolver in proxy mode
  - `should gracefully deal with DNS errors`
  - `should gracefully deal with non listening ports`
  - `should reject non-tcp protocols` - Protocol validation
  - `should allow proxy stubs to invalid hosts` - Error handling
  - `should split each packet into a separate request by default` - Packet splitting
- **Files**: `mbTest/api/tcp/*.js`
- **Files**: `internal/imposter/tcp_server.go`

### P2 - Medium Priority (7 tests)

#### HTTP/HTTPS Behavior Composition (4 tests)
- **Tests**:
  - `should compose multiple behaviors together (old interface for backwards compatibility)` (HTTP + HTTPS)
  - `should apply multiple behaviors in sequence with repeat (new format)` (HTTP + HTTPS)
- **File**: `mbTest/api/http/httpBehaviorsTest.js`, `mbTest/api/https/httpsBehaviorsTest.js`
- **Work**: Fix behavior composition with shellTransform fallback, implement repeat behavior
- **Files**: `internal/imposter/behaviors.go`

#### HTTP/HTTPS Fault Handling (2 tests)
- **Tests**: `should do nothing when undefined fault is specified` (HTTP + HTTPS)
- **Files**: `mbTest/api/http/httpFaultTest.js`, `mbTest/api/https/httpsFaultTest.js`
- **Work**: Gracefully handle unknown fault types (return normal response instead of error)

#### SMTP Request Recording (1 test)
- **Test**: `should provide access to all requests`
- **File**: `mbTest/api/smtp/smtpImposterTest.js`
- **Work**: Verify SMTP request recording and JSON format

## Won't Fix (Security/Architectural - 9 tests)

### shellTransform Behavior (4 tests)
- **Reason**: Arbitrary command execution security risk
- **Tests**: 2 HTTP + 2 HTTPS shell transform tests
- **Alternative**: Use `decorate` behavior with sandboxed JavaScript
- **Reference**: `docs/SECURITY.md`

### Process Object Access (2 tests)
- **Reason**: Environment variable exposure security risk
- **Tests**: HTTP + HTTPS injection tests accessing `process.env`
- **Decision**: Security sandbox is priority over compatibility

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

## goja Async Limitation (4 tests)

### Async JavaScript Injection
- **Tests**:
  - `should allow asynchronous injection` (HTTP)
  - `should allow asynchronous injection` (HTTPS)
  - `should allow asynchronous injection (old interface)` (TCP)
  - `should allow asynchronous injection` (TCP)
- **Status**: goja ES5.1 limitation - no native Promise/async support
- **Work**: May require upstream goja enhancement or workarounds
- **Priority**: Low - rarely used feature

## Validation Procedure

See `.claude/claude.md` for complete validation workflow.

### Quick Validation

```bash
cd /home/tetsujinoni/work/mountebank
pkill -f tartuffe 2>/dev/null || true
MB_EXECUTABLE=/home/tetsujinoni/work/go-tartuffe/bin/tartuffe-wrapper.sh npm run test:api 2>&1 | tee /tmp/tartuffe-validation.log
grep -E "passing|failing" /tmp/tartuffe-validation.log | tail -3
```

Expected: 219 passing / 33 failing (86.9%)

## Documentation References

- **Test Mapping**: [docs/MOUNTEBANK-TEST-MAPPING.md](docs/MOUNTEBANK-TEST-MAPPING.md)
- **Controller Tests**: [docs/CONTROLLER-API-TEST-MAPPING.md](docs/CONTROLLER-API-TEST-MAPPING.md)
- **HTTP Proxy**: [docs/HTTP-PROXY-TEST-MAPPING.md](docs/HTTP-PROXY-TEST-MAPPING.md)
- **Security**: [docs/SECURITY.md](docs/SECURITY.md)
- **Workflows**: [.claude/claude.md](.claude/claude.md)

---

**Note**: Historical results and fix summaries are in `docs/` directory. This backlog contains only remaining work.
