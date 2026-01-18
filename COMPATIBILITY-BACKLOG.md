# Tartuffe Compatibility Backlog

Remaining gaps from mountebank mbTest suite validation against go-tartuffe.

## Current Status

**Mountebank Test Harness**: ✅ Working (with MB_EXECUTABLE correctly set)
**Overall Progress**: **84.1% (212/252 tests) | 89.8% adjusted (212/236)** ✅ TARGET EXCEEDED
**Last Updated**: 2026-01-18 (After predicate generator injection implementation)

**Remaining Failures**: 40 tests (24 actionable + 15 security/architectural + 1 deliberate difference)

**Recent Work**:
- ✅ **Predicate generator injection** - Implemented JavaScript injection for programmatic predicate generation in proxy mode
- ✅ **Decorate behavior header handling fix** - Fixed `response.headers = request.headers` assignment in JavaScript decorate behaviors
- ✅ **Host header handling fix** - Fixed Host header extraction and injection in proxy requests
- ✅ **JSON key ordering fix** - Fixed intermittent sub-object matching failures in JavaScript injection
- ✅ **DELETE /imposters/:id/requests endpoint** - Implemented endpoint to clear requests and proxy-generated stubs
- ✅ **JSON body storage fix (issue #656)** - Proxy now correctly stores pretty-printed JSON as objects
- ✅ HTTP Proxy implementation complete in go-tartuffe (integration tests passing)
- ✅ ProxyAlways mode, predicate generators, binary handling implemented
- ✅ HTTP Proxy tests: 26/33 passing (79%, improved from 76%)
- ✅ Query string fidelity and JSON body storage working
- ✅ Binary MIME type detection (7 types) working

## Failure Categories

### Security/Architectural Blocks - 15 tests (Won't Fix)

**ShellTransform & Behavior Composition (8 tests)** - Security risk
- Intentionally disabled for security (arbitrary command execution risk)
- Tests: `mbTest/api/http/httpBehaviorsTest.js`, `mbTest/api/https/httpsBehaviorsTest.js`
- Includes: shell transform (4 tests), behavior composition with shellTransform (4 tests)
- Workaround: Use `decorate` behavior with sandboxed JavaScript

**Process object access (2 tests)** - Security sandbox
- Exposing `process.env` and system information violates security sandbox
- Tests: HTTP/HTTPS injection tests
- Won't Fix: Security is priority over compatibility

**Async injection (4 tests)** - ES5.1 limitation
- goja ES5.1 lacks native Promise/async support
- Tests: HTTP/HTTPS/TCP async injection tests (2 HTTP/HTTPS + 2 TCP)
- Architectural limitation, not fixable

**Private key return (1 test)** - Security hardening
- Returning private keys in API responses poses significant security risk
- Test: HTTPS key/cert pair creation test
- Won't Fix: Keys could be logged/exposed inadvertently

### Deliberate Design Differences - 1 test

**Prometheus metrics format (1 test)** - Modern observability standard
- go-tartuffe uses industry-standard Prometheus metrics format instead of mountebank's custom JSON format
- Test: `mbTest/api/http/httpMetricsTest.js`
- Rationale: Prometheus is the de facto standard for modern observability, providing better integration with monitoring ecosystems (Grafana, AlertManager, etc.)
- Benefits: Native scraping support, better performance, wider tooling compatibility
- Status: Deliberate simplification toward modern norms, not a compatibility gap

### Actionable Failures - 24 tests

#### 1. HTTP/HTTPS Proxy - 6 tests ⚠️
**Status**: **Significant functionality implemented but tests failing**
**Files**: `mbTest/api/http/httpProxyStubTest.js`

**Note**: Integration tests show features working, but mountebank test failures suggest issues with:
- Test environment differences (port conflicts, timing)
- Response format expectations

**Failing tests** (6 tests):
- ❌ Proxy to HTTPS (cross-protocol)
- ❌ Invalid domain error handling
- ❌ CONNECT method (requires HTTPS tunneling)
- ❌ Persist behaviors from origin
- ❌ Add latency (addWaitBehavior)
- ❌ removeProxies export

**Fixed tests** (3 tests):
- ✅ Host header validation - Fixed by extracting Host from r.Host and handling in proxy injectHeaders
- ✅ Inject headers - Fixed by supporting map[string]string type in decorate behavior header extraction
- ✅ Predicate creation/injection - Implemented JavaScript injection for programmatic predicate generation

**Files modified**: `internal/imposter/proxy.go`, `internal/imposter/matcher.go`, `internal/imposter/selectors.go`, `internal/models/request.go`, `internal/api/handlers/imposter.go`, `internal/imposter/behaviors.go`, `internal/imposter/inject.go`, `internal/models/stub.go`

**Test files created**:
- `test/integration/http_proxy_always_test.go` (13 tests)
- `test/integration/http_proxy_edge_cases_test.go` (5 tests)

**Documentation**: See [HTTP-PROXY-TEST-MAPPING.md](docs/HTTP-PROXY-TEST-MAPPING.md) for detailed coverage analysis.

#### 2. TCP Implementation - 11 tests
**Status**: Various TCP-specific problems
**Files**: `mbTest/api/tcp/*.js`

**Failing tests** (11 tests):
- ❌ Compose multiple behaviors together
- ❌ Requests array not accessible
- ❌ Binary requests with endOfRequestResolver (2 tests)
- ❌ Old proxy syntax not parsed
- ❌ Proxy stubs to invalid hosts
- ❌ Packet splitting (default behavior)
- ❌ TCP proxy endOfRequestResolver
- ❌ DNS error handling
- ❌ Non-listening port handling
- ❌ Protocol validation (reject non-tcp)

**Files to modify**: `internal/imposter/tcp_server.go`, `internal/imposter/behaviors.go`

#### 3. Other Issues - 7 tests (Low Priority)

**Fault handling (2 tests)**:
- ❌ Undefined fault should return normal response, not close connection (HTTP + HTTPS)
- Files: `mbTest/api/http/httpFaultTest.js`, `mbTest/api/https/httpsFaultTest.js`

**Controller API (3 tests)**:
- ❌ Support returning replayable body with proxies removed (HTTP + HTTPS)
- ❌ Delete all imposters and return replayable body
- Files: `mbTest/api/http/httpControllerTest.js`, `mbTest/api/https/httpsControllerTest.js`

**SMTP (1 test)**:
- ❌ Requests array not accessible
- File: `mbTest/api/smtp/smtpImposterTest.js`

**HTTPS mTLS proxy (1 test)**:
- ❌ Proxying to origin requiring mutual auth needs fix
- File: `mbTest/api/https/httpsImposterTest.js`

## Priority Fix Order

### P0 - Critical (None)
All critical functionality is working.

### P1 - High Priority (9 tests)
1. **HTTP Proxy** - Test failures despite implemented features
   - Investigate test environment issues
   - ✅ Fixed JSON key ordering compatibility
   - Verify all proxy features against mountebank tests

### P2 - Medium Priority (11 tests)
2. **TCP Implementation** - Behaviors, proxy, endOfRequestResolver
   - TCP behavior composition
   - Requests array accessibility
   - Proxy functionality

### P3 - Low Priority (7 tests)
3. **Fault handling** - Edge cases
4. **Controller API** - Minor API completeness
5. **SMTP/HTTPS** - Edge cases

## Validation Workflow

### Prerequisites

Before running validation tests, stop any existing tartuffe processes:

```bash
# Stop all tartuffe processes
pkill -f tartuffe 2>/dev/null || true
```

### Running Mountebank Tests

**CRITICAL:** Set `MB_EXECUTABLE` environment variable to test against go-tartuffe.

```bash
cd /home/tetsujinoni/work/mountebank

# Stop any running instances first
pkill -f tartuffe 2>/dev/null || true

# API-level integration tests (primary validation)
MB_EXECUTABLE=/home/tetsujinoni/work/go-tartuffe/bin/tartuffe-wrapper.sh npm run test:api
# Expected: 206 passing, 46 failing (81.7% raw, 87.3% adjusted)

# JavaScript client tests (secondary validation)
MB_EXECUTABLE=/home/tetsujinoni/work/go-tartuffe/bin/tartuffe-wrapper.sh npm run test:js
```

**Important Notes:**
- Skip `test:cli` and `test:web` - go-tartuffe has different CLI/UI implementations
- `MB_EXECUTABLE` must point to `tartuffe-wrapper.sh` for command compatibility
- Without `MB_EXECUTABLE`, you'll test original mountebank, not go-tartuffe

### Running Go Tests

```bash
cd /home/tetsujinoni/work/go-tartuffe

# Run all tests
go test ./internal/... ./cmd/...

# Run integration tests
go test ./test/integration/... -v

# Run with coverage
go test -cover ./internal/...
```

### Full Validation Procedure

```bash
# 1. Clean state
cd /home/tetsujinoni/work/go-tartuffe
pkill -f tartuffe || true

# 2. Build latest
go build -o bin/tartuffe ./cmd/tartuffe

# 3. Run Go tests
go test ./internal/... ./cmd/...

# 4. Run mountebank validation
cd /home/tetsujinoni/work/mountebank
MB_EXECUTABLE=/home/tetsujinoni/work/go-tartuffe/bin/tartuffe-wrapper.sh npm run test:api

# 5. Clean up
pkill -f tartuffe || true
```

## References

### Documentation
- [HTTP-STUB-FIX-SUMMARY.md](docs/HTTP-STUB-FIX-SUMMARY.md) - HTTP stub fixes (2026-01-18)
- [JSON-PREDICATE-FIX-SUMMARY.md](docs/JSON-PREDICATE-FIX-SUMMARY.md) - JSON predicate fixes
- [JSON_PREDICATES_ANALYSIS.md](docs/JSON_PREDICATES_ANALYSIS.md) - JSON predicate analysis
- [COMPATIBILITY-FAILURES.md](docs/COMPATIBILITY-FAILURES.md) - Full failure analysis with line numbers
- [TEST-HARNESS-FIX.md](docs/TEST-HARNESS-FIX.md) - Mountebank test setup
- [INJECTION-COMPATIBILITY.md](docs/INJECTION-COMPATIBILITY.md) - JavaScript limitations
- [Migration Plan](/.claude/plans/curried-gathering-galaxy.md) - Phase breakdown

### Test Environment

- **MB_EXECUTABLE**: `/home/tetsujinoni/work/go-tartuffe/bin/tartuffe-wrapper.sh`
- **MB_PORT**: 2525
- **mountebank version**: 2.9.3
- **go-tartuffe branch**: feat/missing-backlog
- **Last validation**: 2026-01-18

## Achievement Summary

**Target**: 75%+ compatibility
**Current**: **84.1% raw / 89.8% adjusted** ✅ TARGET EXCEEDED

**Current Achievement:**
- ✅ **84.1% raw compatibility** (212/252 tests) - **TARGET EXCEEDED BY 9.1%!**
- ✅ **89.8% adjusted compatibility** (212/236 tests excluding security blocks)
- ✅ Core functionality including JSON predicates, HTTP stubs, gzip, and XPath all working
- ✅ All 228 go-tartuffe integration tests passing
- ✅ Security improvements over mountebank (sandboxed execution, no shellTransform)

**Remaining Work:**
- 24 actionable failures (6 HTTP Proxy, 11 TCP, 7 other)
- 15 intentional security/architectural deviations (Won't Fix)
- 1 deliberate design difference (Prometheus metrics format)

**Test Breakdown** (2026-01-18 validation):
- **Passing**: 212 tests (84.1%)
- **Failing**: 40 tests (15.9%)
  - Security/architectural: 15 tests (won't fix)
  - Deliberate differences: 1 test (Prometheus metrics)
  - Actionable: 24 tests

**Key Discovery**:
- Many HTTP Proxy features are implemented and working in go-tartuffe integration tests
- ✅ JSON key ordering issue resolved (createSortedQueryObject ensures deterministic output)
- Remaining mountebank test failures may be due to test environment differences or response format expectations
- Need investigation and alignment with mountebank test expectations
