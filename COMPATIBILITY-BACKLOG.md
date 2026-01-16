# Tartuffe Compatibility Backlog

Remaining gaps from mountebank mbTest suite validation against go-tartuffe.

## Current Status

**Mountebank Test Harness**: ✅ Working
**Overall Progress**: 71.5% compatibility (181/253 passing, 72 failing)
**Last Updated**: 2026-01-16

**Recent Fixes**:
- ✅ **Wait, Decorate, Copy behaviors** (commit 06c71be) - **+135 tests passing!**
- ✅ Content-Type handling for text/plain responses (commit fc977b8)
- ✅ Test harness pidfile exit handling (commit 8be1a34)
- ✅ HTTP stub predicates (deepEquals, exists, AND logic) - already implemented
- ✅ HTTP stub CRUD operations - already implemented

### Test Results Analysis

**Mountebank Test Suite (API tests only)**: 181 passing, 72 failing (253 total)

**Improvement**: +135 tests passing (293% increase from previous 46 passing)

**Major Remaining Failure Areas**:
- Port conflicts and connection issues (~30 failures) - EADDRINUSE, ECONNREFUSED errors
- HTTP/HTTPS Proxy edge cases (~15 failures) - query parameter handling, specific proxy scenarios
- TCP protocol (~12 failures) - port conflicts, some edge cases
- HTTP/HTTPS advanced features (~10 failures) - CORS, specific stub/imposter features
- Other edge cases (~5 failures)

**Won't Fix** (security/architectural): shellTransform (4 tests), Node.js features (4 tests), CLI (17 tests), Web UI (5 tests)

**Working Features** (181 passing tests):
- ✅ **Wait behavior** - static and dynamic (JavaScript function) latency
- ✅ **Decorate behavior** - JavaScript post-processing of responses
- ✅ **Copy behavior** - regex and JSONPath extraction from request to response
- ✅ **Repeat behavior** - already implemented at stub level
- ✅ Basic imposter creation/deletion (POST/DELETE /imposters)
- ✅ Metrics endpoint (3 tests)
- ✅ HTTP fault injection (CONNECTION_RESET_BY_PEER, RANDOM_DATA_THEN_CLOSE)
- ✅ HTTPS with default certs and mutual auth
- ✅ Content-Type default handling for JSON
- ✅ Basic stub matching and predicates
- ✅ Controller PUT /imposters (overwrite all)
- ✅ Basic GET /imposters/:id operations
- ✅ 404 handling for non-existent imposters
- ✅ Most HTTP/HTTPS behavior combinations

## Remaining Gaps

### Critical Priority (P0)

#### HTTP/HTTPS Behaviors - MOSTLY COMPLETE ✅
**Impact**: High - advanced response transformation features

**Completed Features** (commit 06c71be):
- ✅ `wait` behavior - static and dynamic latency (all tests passing)
- ✅ `decorate` behavior - JavaScript post-processing (all tests passing)
- ✅ `copy` behavior - regex and JSONPath extraction (all tests passing)
- ✅ `repeat` behavior - already implemented at stub level
- ✅ Behavior composition - multiple behaviors in sequence

**Still Missing**:
- `lookup` behavior - lookup from CSV file (~2-3 tests estimated)
  - Implementation exists in behaviors.go but may need testing/fixes

**Files**:
- `internal/imposter/behaviors.go` - Wait, decorate, copy implemented
- `internal/models/stub.go` - Repeat implemented

**Note**: `shellTransform` moved to Won't Fix (security risk - see docs/SECURITY.md)

**Status**: ~90% complete, lookup behavior remaining

#### HTTP/HTTPS Injection (~24 failing tests)
**Impact**: High - dynamic request/response logic

**Missing Features**:
- Predicate injection - JavaScript predicates for matching (4 tests)
- Response injection - JavaScript response generation (4 tests)
- State management in injection - persist state across requests (6 tests)
- `process.env` access in injection (2 tests - may be won't fix)
- Async injection support (2 tests - may be won't fix)

**Files to check**:
- `internal/imposter/inject.go` - JavaScript injection execution
- Go tests show injection works but mbTest failures suggest compatibility issues

**Estimated effort**: 2-3 days

### High Priority (P1)

#### HTTP/HTTPS Proxy (~10 failing tests)
**Impact**: Medium - proxy/record/replay functionality

**Missing Features**:
- Basic proxy forwarding to origin (2 tests)
- Proxy to HTTPS origins (1 test)
- Proxy headers and request information (2 tests)
- ProxyOnce with recording (implied by failures)
- Invalid domain handling (1 test)
- Mutual auth proxying (1 test)

**Files to check**:
- `internal/imposter/proxy.go` - HTTP proxy implementation exists
- May need integration with HTTP server request handling

**Estimated effort**: 2-3 days

#### HTTP/HTTPS Stub/Imposter Issues (~30+ failing tests)
**Impact**: Medium - HTTP-specific features

**Known Issues**:
- CORS support (8 tests - preflight, headers, etc.)
- Auto-assign port when not provided (2 tests)
- Case-sensitive header handling (2 tests)
- Request recording and numberOfRequests (2 tests)
- DELETE /imposters response format - replayable body with proxies (2 tests)
- DELETE /imposters/:id/savedRequests (2 tests)
- Various stub matching edge cases

**Files to investigate**:
- `internal/imposter/http_server.go` - CORS, headers, request recording
- `internal/api/handlers/imposters.go` - Auto-port, DELETE responses

**Estimated effort**: 2-3 days

#### Controller Operations (~7 failing tests)
**Impact**: Medium - API response formats

**Known Issues**:
- DELETE /imposters - replayable body format (2 tests)
- PUT /imposters - response format differences (1 test)
- GET /imposters - format differences (implied)

**Files to investigate**:
- `internal/api/handlers/imposters.go` - Response serialization

**Estimated effort**: 1 day

### Medium Priority (P2)

#### TCP Issues (~10+ failing tests)
**Impact**: Low-Medium - TCP proxy and injection

**Missing Features**:
- TCP proxy forwarding (6 tests)
- TCP injection in predicates and responses (multiple tests)
- TCP request recording edge cases
- Port conflict handling

**Files to check**:
- `internal/imposter/tcp_server.go` - Proxy and injection integration
- `internal/imposter/tcp_proxy_test.go` - Go tests exist and pass
- `internal/imposter/tcp_injection_test.go` - Go tests exist and pass

**Note**: Go tests for TCP proxy and injection pass, but mbTest failures suggest compatibility or integration issues

**Estimated effort**: 1-2 days

#### HTTP Fault (~1 failing test)
**Impact**: Low - edge case

**Known Issues**:
- Undefined fault behavior (should do nothing)

**Files to check**:
- `internal/imposter/http_server.go` - Fault handling

**Estimated effort**: 0.5 day

### Won't Fix (Expected Differences)

#### shellTransform Behavior (4 test failures)
**Reason**: Security Risk - Arbitrary Command Execution

The `shellTransform` behavior is **intentionally disabled** for security:
- Allows arbitrary shell command execution
- Creates command injection vulnerabilities
- Unrestricted system access

**Alternative**: Use `decorate` behavior with JavaScript for response transformations.

See [docs/SECURITY.md](docs/SECURITY.md) for migration guide and security details.

#### Node.js-Specific Features (4 test failures)
These are architectural differences, not bugs:
- `require()` for Node modules - tartuffe uses goja (ES5.1), not Node.js
- `process.env` access - different runtime environment
- Async callback injection - tartuffe is synchronous
- Custom Node.js formatters - tartuffe uses Go plugins

#### CLI Tests (17 failures)
- Process management differences
- Different CLI implementation approach
- **Recommendation**: Users should use API directly

#### Web UI Tests (5 failures)
- go-tartuffe has different web UI implementation

## Next Steps

Based on impact and effort, recommended implementation order:

1. **HTTP Behaviors** (P0) - Most impactful, ~44 test fixes
   - Start with `wait` and `decorate` (most commonly used)
   - Then `copy`, `lookup`, `repeat`
   - Finally behavior composition
   - Note: `shellTransform` disabled for security (4 tests won't fix)

2. **HTTP Injection** (P0) - Dynamic behavior, ~24 test fixes
   - Predicate injection
   - Response injection
   - State management compatibility

3. **HTTP Proxy** (P1) - Recording/replay, ~10 test fixes
   - Basic proxy forwarding
   - ProxyOnce mode
   - HTTPS proxy support

4. **HTTP Stub/Imposter Features** (P1) - ~30 test fixes
   - CORS support (8 tests)
   - Auto-assign port (2 tests)
   - Request recording improvements
   - DELETE response formats

5. **TCP Integration** (P2) - ~10 test fixes
   - Connect Go test implementations to mbTest scenarios
   - Investigate compatibility gaps

**Total Estimated Effort**: 12-18 days to reach ~75%+ compatibility

## Validation Workflow

### Prerequisites

Before running validation tests, stop any existing tartuffe processes to prevent port conflicts:

```bash
# Stop all tartuffe processes
pkill -f tartuffe 2>/dev/null || true

# Or kill processes on specific ports
for port in 2525 2526 2527; do
    lsof -ti:$port | xargs kill -9 2>/dev/null || true
done
```

### Running Mountebank Tests

Mountebank has several test suites. For go-tartuffe validation, focus on API and JavaScript tests:

```bash
cd /home/tetsujinoni/work/mountebank

# Stop any running instances first
pkill -f tartuffe 2>/dev/null || true

# API-level integration tests (primary validation)
npm run test:api
# Current: 181 passing, 72 failing (253 total) = 71.5%
# Target: 190+ passing (~75% compatibility)

# JavaScript client tests (secondary validation)
npm run test:js
# Tests the JavaScript client library against go-tartuffe
```

**Note:** Skip `test:cli` and `test:web` - go-tartuffe has different CLI/UI implementations.

### Running Go Tests

```bash
cd /home/tetsujinoni/work/go-tartuffe

# Run all tests
go test ./internal/... ./cmd/...
# Expected: All tests pass (~5 seconds)

# Run specific behavior tests
go test ./internal/imposter -run "Test(Wait|Decorate|Copy)" -v

# Run with coverage
go test -cover ./internal/...
```

### Full Validation Procedure

For complete validation before commits:

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
npm run test:api
npm run test:js

# 5. Clean up
pkill -f tartuffe || true
```

**See [CLAUDE.md](CLAUDE.md) for detailed workflow hints and troubleshooting.**

## References

### Documentation
- [Test Harness Fix](docs/TEST-HARNESS-FIX.md) - Mountebank test setup
- [Behavior Fix](docs/BEHAVIOR-FIX.md) - Object/array parsing
- [Injection Compatibility](docs/INJECTION-COMPATIBILITY.md) - JavaScript limitations
- [Protocol Fixes](docs/PROTOCOL-FIXES.md) - TCP/SMTP/HTTPS
- [Fix Summary](docs/FIX-SUMMARY.md) - API response formats
- [Implementation Plan](docs/IMPLEMENTATION-PLAN.md) - TDD strategy
- [Migration Plan](/.claude/plans/curried-gathering-galaxy.md) - Phase breakdown

### Test Environment

- **MB_EXECUTABLE**: `/home/tetsujinoni/work/go-tartuffe/bin/tartuffe-wrapper.sh`
- **MB_PORT**: 2525
- **mountebank version**: 2.9.3
- **go-tartuffe branch**: feat/missing-backlog
- **Last validation**: 2026-01-16
