# Tartuffe Compatibility Backlog

Remaining gaps from mountebank mbTest suite validation against go-tartuffe.

## Current Status

**Mountebank Test Harness**: ✅ Working (with MB_EXECUTABLE correctly set)
**Overall Progress**: **57.1% compatibility (144/252 passing, 108 failing)**
**Last Updated**: 2026-01-16 (After behavior composition and xpath lookup fixes)

**Known Working**:
- ✅ **Wait behavior** - Static and dynamic latency working
- ✅ **Decorate behavior** - JavaScript post-processing working
- ✅ **Copy behavior** - Regex, xpath, jsonpath extraction and token replacement working
- ✅ **Lookup behavior** - CSV lookup with xpath and jsonpath selectors working
- ✅ **Repeat behavior** - Response cycling working correctly
- ✅ **Behavior composition** - Multiple behaviors in sequence (new format) working
- ✅ **HTTP/HTTPS basic stubs** - Simple is responses and predicates
- ✅ **TCP basic stubs** - Basic forwarding and binary data
- ✅ **HTTPS mutual auth** - mTLS working correctly
- ✅ **SMTP basic** - Basic SMTP functionality
- ✅ Test harness integration - MB_EXECUTABLE workflow established
- 🔒 **ShellTransform disabled** (commit b44905a) - Security fix for command injection vulnerability

### Test Results Analysis

**Mountebank Test Suite (API tests only)**: **144 passing, 108 failing (252 total) = 57.1%**

**Recent Fixes** (Session ending 2026-01-16):
- ✅ **Copy behavior** - Fixed array parsing, token replacement (6 tests fixed)
- ✅ **Lookup behavior** - Fixed array parsing, xpath/jsonpath/CSV integration (6 tests fixed)
- ✅ **Repeat behavior** - Fixed response cycling logic (6 tests fixed)
- ✅ **Behavior composition** - Fixed "behaviors" vs "_behaviors" parsing (2 tests fixed)

**Progress**: +20 tests (from 124 to 144 passing) = **7.9% improvement**

**Major Remaining Failure Categories** (108 failing tests):

1. **ShellTransform** (6 tests) - **Expected failure (security block)**
   - Intentionally disabled for security (arbitrary command execution risk)
   - 4 old interface composition tests + 2 new format tests with shellTransform

2. **TCP injection** (~8 tests) - JavaScript injection not working in TCP context
   - Predicate injection, response injection, state management all failing

3. **TCP proxy** (~5 tests) - endOfRequestResolver and error handling issues
   - Binary requests with custom resolvers failing
   - DNS error handling not working

4. **HTTP proxy** (many tests) - Multiple proxy functionality gaps
   - ProxyOnce mode issues
   - ProxyAlways mode issues
   - Predicate generators not working
   - Mutual auth proxy issues

5. **Response format** (multiple tests) - API response missing fields
   - `savedRequests` field missing/undefined
   - `numberOfRequests` vs `recordRequests` mismatch
   - Case-sensitive headers not preserved

6. **Various edge cases** (remaining ~70 tests)
   - Gzip request handling
   - XPath predicates
   - Auto-assign port issues
   - Stub overwrite operations
   - DELETE operations with replayable bodies
   - TCP behaviors and edge cases

**Won't Fix** (architectural):
- CLI tests - Different CLI implementation
- Web UI tests - Different UI implementation

## Remaining Gaps (Significant)

### Status: IN PROGRESS - 57.1% compatibility

With 144/252 tests passing (57.1%), go-tartuffe is making progress toward the 75%+ target. The following sections detail feature status.

### Partially Working Features

#### HTTP/HTTPS Behaviors - ✅ MOSTLY WORKING
- ✅ `wait` behavior - static and dynamic latency WORKING
- ✅ `decorate` behavior - JavaScript post-processing WORKING
- ✅ `copy` behavior - Regex, xpath, jsonpath extraction WORKING
- ✅ `lookup` behavior - CSV lookup with xpath/jsonpath WORKING
- ✅ `repeat` behavior - Response cycling WORKING
- 🔒 `shellTransform` behavior - **DISABLED for security** (6 tests failing intentionally)
- ✅ Behavior composition (new format) - Multiple behaviors in sequence WORKING
- ❌ Behavior composition (old format with shellTransform) - Expected to fail (security)

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
**Current**: **57.1% compatibility (144/252 tests)**

go-tartuffe is making significant progress toward the 75%+ compatibility target. Current validation shows:

**Test Breakdown**:
- ✅ 144 passing - Core behaviors, stubs, protocols working (wait, decorate, copy, lookup, repeat, composition, HTTP/HTTPS/TCP/SMTP)
- ❌ 108 failing - Remaining gaps in TCP injection/proxy, HTTP proxy, edge cases
  - 6 failures are intentional (shellTransform security block)
  - 102 failures need investigation and fixes

**Recent Progress** (2026-01-16 session):
- Fixed copy behavior: array parsing, token replacement (6 tests)
- Fixed lookup behavior: xpath/jsonpath with namespaces, CSV integration (6 tests)
- Fixed repeat behavior: response cycling logic (6 tests)
- Fixed behavior composition: "behaviors" vs "_behaviors" field handling (2 tests)
- **Total: +20 tests (7.9% improvement)**

**Priority Areas for Remaining Work**:
1. Implement TCP injection support (8+ tests)
2. Fix TCP proxy endOfRequestResolver (5+ tests)
3. Implement HTTP proxy modes (ProxyOnce, ProxyAlways, predicate generators) (many tests)
4. Fix API response format issues (savedRequests, numberOfRequests, case-sensitive headers)
5. Various edge cases (gzip, xpath predicates in matchers, stub operations) (~70 tests)

**Security Note**: The 6 shellTransform test failures are intentional. ShellTransform allows arbitrary command execution which poses a critical security vulnerability. Users should use the `decorate` behavior with sandboxed JavaScript instead.

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
# Current: 144 passing, 108 failing (252 total) = 57.1%
# Target: 75%+ passing - MAKING PROGRESS

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
