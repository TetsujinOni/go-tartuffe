# Tartuffe Compatibility Backlog

Remaining gaps from mountebank mbTest suite validation against go-tartuffe.

## Current Status

**Mountebank Test Harness**: ✅ Working
**Overall Progress**: **98.4% compatibility (248/253 passing, 4 failing, 1 skipped)**
**Last Updated**: 2026-01-16 (Final validation with security fix)

**Recent Fixes**:
- ✅ **Wait, Decorate, Copy behaviors** (commit 06c71be) - Dynamic latency, JavaScript post-processing, regex/JSONPath extraction
- ✅ **All HTTP/HTTPS injection tests** - Predicate, response, state management
- ✅ **All TCP injection tests** - Predicate, response, async support
- ✅ **All proxy tests** - HTTP, HTTPS, TCP, ProxyOnce, ProxyAlways, mutual auth
- ✅ **Lookup behavior** - CSV file lookups with key transformations
- ✅ **Repeat behavior** - Already implemented at stub level
- ✅ Content-Type handling for text/plain responses (commit fc977b8)
- ✅ Test harness pidfile exit handling (commit 8be1a34)
- ✅ **Port conflict resolution** - Cleanup procedures working correctly
- 🔒 **ShellTransform disabled** (commit b44905a) - Security fix for command injection vulnerability

### Test Results Analysis

**Mountebank Test Suite (API tests only)**: **248 passing, 4 failing (shellTransform security block)**

**Improvement**: +67 tests from previous validation (+202 from initial baseline)
- Previous: 181 passing, 72 failing (71.5%)
- Current: 248 passing, 4 failing, 1 skipped (98.4%)

**All Feature Areas Passing (except security-blocked shellTransform)**:
- ✅ HTTP/HTTPS Behaviors - wait, decorate, repeat, copy, lookup
- ✅ HTTP/HTTPS Injection - predicates, responses, state management
- ✅ HTTP/HTTPS Proxy - forwarding, ProxyOnce, ProxyAlways, predicate generators
- ✅ HTTP/HTTPS Stubs - deepEquals, predicates, CRUD operations
- ✅ HTTP/HTTPS Fault injection - all fault types
- ✅ TCP Behaviors - decorate, composition
- ✅ TCP Injection - predicates, responses, async, state
- ✅ TCP Proxy - forwarding, binary data, DNS errors
- ✅ SMTP - basic functionality
- ✅ Metrics - all metrics endpoints
- ✅ CORS - preflight and headers
- ✅ Controller operations - GET, POST, PUT, DELETE
- ✅ HTTPS with mutual auth

**Won't Fix** (security/architectural):
- 🔒 **ShellTransform** (4 tests) - Security risk: arbitrary command execution
- Node.js features (require(), process.env in some contexts)
- CLI tests (17) - Different CLI implementation
- Web UI (5) - Different UI implementation

**ShellTransform Investigation - RESOLVED**:

The shellTransform mystery has been solved. Initial validation showing 252/253 passing used an **outdated binary** built before the security fix:

**Timeline**:
- Jan 16 02:57 - Binary built with shellTransform implementation
- Jan 16 03:12 - Commit b44905a disabled shellTransform for security
- Jan 16 04:30 - First validation used OLD binary (252 passing)
- Jan 16 04:48 - Binary rebuilt with security block
- Jan 16 04:49 - Verification confirmed shellTransform now fails correctly

**Current behavior**: Attempting to use shellTransform returns:
```
behavior error: shellTransform behavior is not supported (security risk)
```

See [docs/SECURITY.md](docs/SECURITY.md) for security rationale and migration guide to `decorate` behavior.

## Remaining Gaps (Minimal)

### Status: COMPLETE ✅

With 248/253 tests passing (98.4%), go-tartuffe has achieved feature parity with mountebank for all tested API functionality, excluding the intentionally disabled shellTransform behavior for security reasons.

### Completed Features (All Tests Passing)

#### HTTP/HTTPS Behaviors - ✅ COMPLETE (except shellTransform)
- ✅ `wait` behavior - static and dynamic latency
- ✅ `decorate` behavior - JavaScript post-processing (secure alternative to shellTransform)
- ✅ `copy` behavior - regex, xpath, and JSONPath extraction
- ✅ `lookup` behavior - CSV file lookups with key transformations
- ✅ `repeat` behavior - implemented at stub level
- 🔒 `shellTransform` behavior - **DISABLED for security** (4 failing tests expected)
- ✅ Behavior composition - multiple behaviors in sequence

#### HTTP/HTTPS Injection - ✅ COMPLETE
- ✅ Predicate injection - JavaScript predicates for matching
- ✅ Response injection - JavaScript response generation
- ✅ State management in injection - persist state across requests
- ✅ `process.env` access in injection contexts
- ✅ Async injection support

#### HTTP/HTTPS Proxy - ✅ COMPLETE
- ✅ Basic proxy forwarding to HTTP origins
- ✅ Proxy to HTTPS origins
- ✅ ProxyOnce mode with recording and replay
- ✅ ProxyAlways mode with multiple responses
- ✅ Predicate generators for programmatic predicate creation
- ✅ Proxy headers injection
- ✅ Mutual auth proxying
- ✅ Binary data proxying
- ✅ Query parameter handling

#### TCP Protocol - ✅ COMPLETE
- ✅ TCP behaviors - decorate, composition
- ✅ TCP injection - predicates, responses, async, state management
- ✅ TCP proxy - forwarding, binary data, error handling
- ✅ Request recording and numberOfRequests
- ✅ Custom endOfRequestResolver

#### Other Features - ✅ COMPLETE
- ✅ HTTP/HTTPS fault injection - all fault types
- ✅ SMTP basic functionality
- ✅ Metrics endpoints
- ✅ CORS support
- ✅ Controller operations - GET, POST, PUT, DELETE
- ✅ HTTPS with mutual authentication
- ✅ Auto-assign ports
- ✅ Case-sensitive header handling
- ✅ Request recording and savedRequests

### Architectural Differences (Expected)

#### Node.js-Specific Features
These are architectural differences, not compatibility gaps:
- `require()` for Node modules - go-tartuffe uses goja (ES5.1), not Node.js
- Some `process.env` access patterns - different runtime environment
- Custom Node.js formatters - go-tartuffe uses Go plugins

#### CLI Tests (Not Applicable)
- go-tartuffe has different CLI implementation
- Users should use the API directly
- These tests validate mountebank's specific CLI, not API functionality

#### Web UI Tests (Not Applicable)
- go-tartuffe has different web UI implementation
- These tests validate mountebank's specific UI, not API functionality

## Achievement Summary

**Target**: 75%+ compatibility
**Achieved**: **98.4% compatibility (248/253 tests)** 🎉

go-tartuffe has achieved full feature parity with mountebank for all API functionality tested in the mountebank test suite, with the exception of shellTransform which is intentionally disabled for security.

**Test Breakdown**:
- ✅ 248 passing - All implemented features working correctly
- ❌ 4 failing - ShellTransform tests (intentionally blocked for security)
- ⏭️ 1 skipped - Test infrastructure difference

**Security Trade-off**: The 4 shellTransform test failures are intentional and documented. The feature allows arbitrary command execution which poses a critical security vulnerability. Users should migrate to the `decorate` behavior with sandboxed JavaScript.

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

**CRITICAL:** Set `MB_EXECUTABLE` environment variable to test against go-tartuffe. Without it, tests will run against the original Node.js mountebank binary.

Mountebank has several test suites. For go-tartuffe validation, focus on API and JavaScript tests:

```bash
cd /home/tetsujinoni/work/mountebank

# Stop any running instances first
pkill -f tartuffe 2>/dev/null || true

# API-level integration tests (primary validation)
MB_EXECUTABLE=/home/tetsujinoni/work/go-tartuffe/bin/tartuffe-wrapper.sh npm run test:api
# Current: 248 passing, 4 failing, 1 skipped (253 total) = 98.4%
# Target: 75%+ passing - EXCEEDED!

# JavaScript client tests (secondary validation)
MB_EXECUTABLE=/home/tetsujinoni/work/go-tartuffe/bin/tartuffe-wrapper.sh npm run test:js
# Tests the JavaScript client library against go-tartuffe
```

**Important Notes:**
- Skip `test:cli` and `test:web` - go-tartuffe has different CLI/UI implementations
- `MB_EXECUTABLE` must point to `tartuffe-wrapper.sh` (not `tartuffe` directly) for command compatibility
- Without `MB_EXECUTABLE`, you'll get 252/252 passing (testing original mountebank, not go-tartuffe!)

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

# 4. Run mountebank validation (MUST use MB_EXECUTABLE)
cd /home/tetsujinoni/work/mountebank
MB_EXECUTABLE=/home/tetsujinoni/work/go-tartuffe/bin/tartuffe-wrapper.sh npm run test:api
MB_EXECUTABLE=/home/tetsujinoni/work/go-tartuffe/bin/tartuffe-wrapper.sh npm run test:js

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
