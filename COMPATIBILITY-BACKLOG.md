# Tartuffe Compatibility Backlog

Remaining gaps from mountebank mbTest suite validation against go-tartuffe.

## Current Status

**Mountebank Test Harness**: ✅ Working (with MB_EXECUTABLE correctly set)
**Overall Progress**: **60.7% raw (153/252 tests) | 63.2% adjusted (153/242 excluding security blocks)**
**Last Updated**: 2026-01-16 (After header case preservation fix)

**Known Working**:
- ✅ **Wait behavior** - Static and dynamic latency working (HTTP/HTTPS only)
- ✅ **Decorate behavior** - JavaScript post-processing working
- ✅ **Copy behavior** - Regex, xpath, jsonpath extraction and token replacement working
- ✅ **Lookup behavior** - CSV lookup with xpath and jsonpath selectors working
- ✅ **Repeat behavior** - Response cycling working correctly
- ✅ **Behavior composition** - Multiple behaviors in sequence (new format) working
- ✅ **HTTP/HTTPS basic stubs** - Simple is responses and predicates
- ✅ **TCP basic stubs** - Basic forwarding and binary data
- ✅ **TCP injection** - Predicate and response injection working (VM.Set fix)
- ✅ **TCP behaviors** - Full BehaviorExecutor integration working
- ✅ **HTTPS mutual auth** - mTLS working correctly
- ✅ **SMTP basic** - Basic SMTP functionality
- ✅ **Response format** - recordRequests and numberOfRequests fields correct
- ✅ Test harness integration - MB_EXECUTABLE workflow established
- 🔒 **ShellTransform disabled** (commit b44905a) - Security fix for command injection vulnerability
- 🔒 **Process object disabled** - Security sandbox prevents access to `process.env` and system information

### Test Results Analysis

**Mountebank Test Suite (API tests only)**: **153 passing, 99 failing (252 total)**
- Raw: 60.7% (153/252)
- Adjusted: 63.2% (153/242 excluding ~10 security-blocked tests)

**Recent Fixes** (2026-01-16):
- ✅ **TCP injection** - Fixed by passing requestData via VM.Set (commit 631a9cc)
- ✅ **TCP behaviors** - Integrated full BehaviorExecutor (commit a849142)
- ✅ **Response format** - Fixed recordRequests/numberOfRequests fields (commit 611363e)
- ✅ **Stub overwrite URLs** - Fixed absolute URLs in PUT /imposters/{id}/stubs/{index} (commit af2fcc3)
- ✅ **Header case preservation** - Headers now saved with canonical case (commit 79605cd)

**Major Remaining Failure Categories** (99 failing tests):

1. **ShellTransform** (~6 tests) - **Expected failure (security block)**
   - Intentionally disabled for security (arbitrary command execution risk)
   - Composition tests involving shellTransform will fail

2. **Process object access** (~4 tests) - **Expected failure (security block)**
   - `process.env` and system information access blocked in JavaScript sandbox
   - Prevents information disclosure and environment variable leakage

3. **JavaScript Injection** (~16 tests) - State management and async issues
   - State sharing between predicate/response injection failing
   - Asynchronous injection issues

4. **HTTP Proxy** (~20 tests) - Multiple proxy functionality gaps
   - ProxyOnce/ProxyAlways mode issues
   - Predicate generators not working
   - Decorated proxy responses failing
   - Binary data from origin server issues

5. **TCP Protocol** (~15 tests) - Various TCP-specific issues
   - endOfRequestResolver edge cases
   - DNS error handling
   - Packet splitting behavior
   - Binary mode predicates

6. **CORS** (~6 tests) - Preflight request handling
   - allowCORS option not fully working
   - Preflight requests returning wrong status

7. **Faults** (~6 tests) - Connection fault injection
   - CONNECTION_RESET_BY_PEER not implemented
   - RANDOM_DATA_THEN_CLOSE not implemented

8. **JSON/Predicates** (~15 tests) - Complex predicate matching
   - deepEquals object handling
   - JSON body parsing issues
   - xpath array predicates
   - gzip request handling

9. **API/Controller** (~10 tests) - Various API issues
   - Auto-assign port not working
   - Stub overwrite operations
   - Metrics endpoint issues
   - Case-sensitive headers

10. **HTTPS** (~5 tests) - Certificate handling
    - Key/cert pair during creation
    - Mutual auth proxying

**Won't Fix** (architectural):
- CLI tests - Different CLI implementation
- Web UI tests - Different UI implementation

## Remaining Gaps (Significant)

### Status: IN PROGRESS - 63.2% adjusted compatibility

With 153/242 actionable tests passing (63.2% adjusted, excluding security blocks), go-tartuffe is making progress toward the 75%+ target. The following sections detail feature status.

### Partially Working Features

#### HTTP/HTTPS Behaviors - ✅ MOSTLY WORKING
- ✅ `wait` behavior - static and dynamic latency WORKING
- ✅ `decorate` behavior - JavaScript post-processing WORKING
- ✅ `copy` behavior - Regex, xpath, jsonpath extraction WORKING
- ✅ `lookup` behavior - CSV lookup with xpath/jsonpath WORKING
- ✅ `repeat` behavior - Response cycling WORKING
- 🔒 `shellTransform` behavior - **DISABLED for security** (~6 tests failing intentionally)
- ✅ Behavior composition (new format) - Multiple behaviors in sequence WORKING
- ❌ Behavior composition (old format with shellTransform) - Expected to fail (security)
- ❌ Wait behavior as function - Not working (~3 tests)

#### HTTP/HTTPS Injection - ⚠️ PARTIAL
- ✅ Basic synchronous injection working
- ❌ State management between requests failing (~8 tests)
- ❌ Asynchronous injection not working (~4 tests)
- 🔒 `process` object access **DISABLED for security** (~4 tests failing intentionally)

#### HTTP/HTTPS Proxy - ❌ NEEDS WORK (~20 tests)
- ❌ ProxyOnce mode - recording/replay issues
- ❌ ProxyAlways mode - issues with multiple responses
- ❌ Predicate generators - not working
- ❌ Decorated proxy responses - not persisting correctly
- ❌ Mutual auth proxying - issues
- ⚠️ Basic proxy may work for simple cases

#### TCP Protocol - ⚠️ PARTIAL (~15 tests)
- ✅ TCP behaviors - Full BehaviorExecutor integrated (copy, decorate, etc.)
- ✅ TCP injection - Predicate and response injection WORKING
- ⚠️ TCP proxy - basic forwarding works
- ❌ endOfRequestResolver edge cases failing
- ❌ DNS error handling not working
- ❌ Packet splitting behavior issues
- ❌ Binary mode with matches predicate failing

#### Faults - ❌ NOT IMPLEMENTED (~6 tests)
- ❌ CONNECTION_RESET_BY_PEER fault
- ❌ RANDOM_DATA_THEN_CLOSE fault
- ❌ Undefined fault handling

#### CORS - ❌ NOT WORKING (~6 tests)
- ❌ allowCORS option not functioning
- ❌ Preflight requests not handled correctly

#### JSON/Predicates - ⚠️ PARTIAL (~15 tests)
- ❌ deepEquals with objects failing
- ❌ JSON body parsing issues
- ❌ xpath array predicates
- ❌ gzip request decompression

#### Other Features - ⚠️ MIXED
- ✅ SMTP basic functionality - WORKING
- ✅ HTTPS with mutual authentication - WORKING
- ✅ Response format (recordRequests, numberOfRequests) - FIXED
- ❌ Metrics endpoints - failing
- ❌ Auto-assign ports - failing
- ✅ Case-sensitive header handling - FIXED
- ✅ Stub overwrite PUT operations - FIXED

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
**Current**: **63.2% adjusted (153/242 actionable tests) | 60.7% raw (153/252 total)**

go-tartuffe is making progress toward the 75%+ compatibility target. Current validation shows:

**Test Breakdown**:
- ✅ 153 passing - Core behaviors, stubs, protocols working
- ❌ 99 failing - Remaining gaps across multiple categories
  - ~10 failures are intentional (security blocks: shellTransform ~6, process object ~4)
  - ~89 failures need investigation and fixes

**Failure Category Summary** (99 tests):
| Category | Est. Tests | Priority |
|----------|-----------|----------|
| HTTP Proxy | ~20 | High |
| JavaScript Injection (state/async) | ~16 | Medium |
| TCP Protocol (edge cases) | ~15 | Medium |
| JSON/Predicates | ~15 | Medium |
| API/Controller | ~6 | Low |
| ShellTransform | ~6 | Won't Fix (security) |
| CORS | ~6 | Low |
| Faults | ~6 | Low |
| HTTPS | ~5 | Low |
| Process object access | ~4 | Won't Fix (security) |

**Recent Progress** (2026-01-16):
- ✅ TCP injection: VM.Set fix for Buffer support (commit 631a9cc)
- ✅ TCP behaviors: Full BehaviorExecutor integration (commit a849142)
- ✅ Response format: recordRequests/numberOfRequests fields (commit 611363e)
- ✅ Stub overwrite URLs: Fixed absolute URLs in PUT response (commit af2fcc3)
- ✅ Header case preservation: Canonical case for saved headers (commit 79605cd)

**Priority Areas for Next Session**:
1. **HTTP Proxy** (~20 tests) - ProxyOnce/ProxyAlways modes, predicate generators
2. **JavaScript Injection** (~16 tests) - State persistence, async support
3. **JSON/Predicates** (~15 tests) - deepEquals, JSON body parsing, gzip
4. **CORS** (~6 tests) - allowCORS option, preflight handling

**Security Note**: The ~10 security-related test failures are intentional:
- **ShellTransform (~6 tests)**: Allows arbitrary command execution - critical vulnerability. Use `decorate` behavior with sandboxed JavaScript instead.
- **Process object (~4 tests)**: Exposes `process.env` and system information - information disclosure risk. Environment-specific logic should be handled outside the mock server.

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
