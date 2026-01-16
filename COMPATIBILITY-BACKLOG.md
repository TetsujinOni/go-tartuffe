# Tartuffe Compatibility Backlog

Remaining gaps from mountebank mbTest suite validation against go-tartuffe.

## Current Status

**Mountebank Test Harness**: ✅ Working (with MB_EXECUTABLE correctly set)
**Overall Progress**: **49.2% compatibility (124/252 passing, 128 failing)**
**Last Updated**: 2026-01-16 (Accurate validation with correct binary)

**Known Working**:
- ✅ **Wait behavior** - Static and dynamic latency working
- ✅ **Decorate behavior** - JavaScript post-processing working
- ✅ **HTTP/HTTPS basic stubs** - Simple is responses and predicates
- ✅ **TCP basic stubs** - Basic forwarding and binary data
- ✅ **HTTPS mutual auth** - mTLS working correctly
- ✅ **SMTP basic** - Basic SMTP functionality
- ✅ Test harness integration - MB_EXECUTABLE workflow established
- 🔒 **ShellTransform disabled** (commit b44905a) - Security fix for command injection vulnerability

### Test Results Analysis

**Mountebank Test Suite (API tests only)**: **124 passing, 128 failing (252 total) = 49.2%**

**Major Failure Categories** (128 failing tests):

1. **Repeat behavior** (6 tests) - Not cycling responses correctly
   - Expected first response to repeat N times before moving to second response
   - Currently advancing to next response immediately

2. **Copy behavior** (6 tests) - Invalid JSON parse errors on imposter creation
   - Tests with regex, xpath, jsonpath copy all fail with "Unable to parse body as JSON"

3. **Lookup behavior** (6 tests) - Invalid JSON parse errors on imposter creation
   - CSV file lookup tests fail with "Unable to parse body as JSON"

4. **Behavior composition** (6 tests) - Invalid JSON parse errors
   - Multiple behaviors in sequence fail to parse

5. **ShellTransform** (4 tests) - **Expected failure (security block)**
   - Intentionally disabled for security (arbitrary command execution risk)

6. **TCP injection** (~8 tests) - JavaScript injection not working in TCP context
   - Predicate injection, response injection, state management all failing

7. **TCP proxy** (~5 tests) - endOfRequestResolver and error handling issues
   - Binary requests with custom resolvers failing
   - DNS error handling not working

8. **HTTP proxy** (many tests) - Multiple proxy functionality gaps
   - ProxyOnce mode issues
   - ProxyAlways mode issues
   - Predicate generators not working
   - Mutual auth proxy issues

9. **Response format** (multiple tests) - API response missing fields
   - `savedRequests` field missing/undefined
   - `numberOfRequests` vs `recordRequests` mismatch
   - Case-sensitive headers not preserved

10. **Various edge cases** (multiple tests)
    - Gzip request handling
    - XPath predicates
    - Auto-assign port issues
    - Stub overwrite operations
    - DELETE operations with replayable bodies

**Won't Fix** (architectural):
- CLI tests - Different CLI implementation
- Web UI tests - Different UI implementation

## Remaining Gaps (Significant)

### Status: IN PROGRESS - 49.2% compatibility

With 124/252 tests passing (49.2%), go-tartuffe has significant work remaining to reach the 75%+ target. The following sections detail feature status.

### Partially Working Features

#### HTTP/HTTPS Behaviors - ⚠️ PARTIAL
- ✅ `wait` behavior - static and dynamic latency WORKING
- ✅ `decorate` behavior - JavaScript post-processing WORKING
- ❌ `copy` behavior - Invalid JSON parse errors (6 tests failing)
- ❌ `lookup` behavior - Invalid JSON parse errors (6 tests failing)
- ❌ `repeat` behavior - Not cycling correctly (6 tests failing)
- 🔒 `shellTransform` behavior - **DISABLED for security** (4 tests failing intentionally)
- ❌ Behavior composition - Invalid JSON parse errors (6 tests failing)

#### HTTP/HTTPS Injection - ⚠️ MOSTLY WORKING
- ✅ Basic injection working for some tests
- ❌ Multiple injection tests failing (need detailed analysis)

#### HTTP/HTTPS Proxy - ❌ NEEDS WORK
- ❌ ProxyOnce mode - recording/replay issues (multiple tests failing)
- ❌ ProxyAlways mode - issues (multiple tests failing)
- ❌ Predicate generators - not working (tests failing)
- ❌ Mutual auth proxying - issues (tests failing)
- ⚠️ Basic proxy may work for simple cases

#### TCP Protocol - ❌ MAJOR GAPS
- ❌ TCP behaviors - decorate not working, composition failing (2 tests)
- ❌ TCP injection - predicates, responses, state all failing (~8 tests)
- ⚠️ TCP proxy - basic forwarding works, but endOfRequestResolver issues (~5 tests)
- ❌ Binary requests with custom resolvers failing
- ❌ DNS error handling not working

#### Other Features - ⚠️ MIXED
- ⚠️ HTTP/HTTPS fault injection - some working, some failing
- ✅ SMTP basic functionality - WORKING
- ❌ Metrics endpoints - need verification
- ⚠️ CORS support - need verification
- ⚠️ Controller operations - some DELETE issues, savedRequests missing
- ✅ HTTPS with mutual authentication - WORKING
- ❌ Auto-assign ports - failing in some contexts
- ❌ Case-sensitive header handling - NOT working (undefined headers)
- ❌ Request recording and savedRequests - field missing/undefined

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
**Current**: **49.2% compatibility (124/252 tests)**

go-tartuffe has substantial work remaining to achieve the 75%+ compatibility target. Current validation shows:

**Test Breakdown**:
- ✅ 124 passing - Basic features working (wait, decorate, basic stubs, HTTPS, SMTP)
- ❌ 128 failing - Significant gaps in behaviors, TCP, proxy, and edge cases
  - 4 failures are intentional (shellTransform security block)
  - 124 failures need investigation and fixes

**Priority Areas for Improvement**:
1. Fix JSON parsing errors for copy/lookup/composition behaviors (18 tests)
2. Fix repeat behavior cycling logic (6 tests)
3. Implement TCP injection support (8+ tests)
4. Fix TCP proxy endOfRequestResolver (5+ tests)
5. Implement HTTP proxy modes (ProxyOnce, ProxyAlways, predicate generators)
6. Fix API response format issues (savedRequests, numberOfRequests, case-sensitive headers)
7. Various edge cases (gzip, xpath, stub operations)

**Security Note**: The 4 shellTransform test failures are intentional. ShellTransform allows arbitrary command execution which poses a critical security vulnerability. Users should use the `decorate` behavior with sandboxed JavaScript instead.

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
# Current: 124 passing, 128 failing (252 total) = 49.2%
# Target: 75%+ passing - NOT YET ACHIEVED

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
