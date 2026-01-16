# Tartuffe Compatibility Backlog

Remaining gaps from mountebank mbTest suite validation against go-tartuffe.

## Current Status

**Mountebank Test Harness**: ✅ Working
**Overall Progress**: 73% compatibility (171/235 passing, 64 failing)
**Last Updated**: 2026-01-16

### Test Suite Summary

| Test Suite | Passing | Failing | Pass Rate | Status |
|------------|---------|---------|-----------|---------|
| **HTTP Behaviors** | 50 | 0 | 100% | ✅ Complete |
| **HTTP Injection** | 28 | 0 | 100% | ✅ Complete |
| **HTTP Metrics** | 3 | 0 | 100% | ✅ Complete |
| **SMTP** | 2 | 0 | 100% | ✅ Complete |
| **TCP** | 29 | 5 | 85% | 🔄 Mostly complete |
| **Imposters Controller** | 7 | 3 | 70% | 🔄 Good coverage |
| **HTTP Fault** | 4 | 2 | 67% | 🔄 Good coverage |
| **HTTPS** | 2 | 2 | 50% | 🔄 Core features work |
| **HTTP Imposter** | 12 | 16 | 43% | ⚠️ Needs work |
| **HTTP Stub** | 8 | 46 | 15% | ⚠️ Needs significant work |
| **CLI** | 0 | 17 | 0% | ❌ Won't fix |
| **Total** | **171** | **64** | **73%** | **64 remaining issues** |

## Remaining Gaps

### Critical Priority (P0)

#### HTTP Stub Issues (~46 failing tests)
**Impact**: High - core stub matching and response features

**Known Issues**:
- Content-Type handling for non-JSON responses (many test failures)
- Complex predicate matching:
  - deepEquals with nested structures (4 tests)
  - exists operator (2 tests)
  - Multiple predicates with AND logic (3 tests)
- XPath extraction in predicates (2 tests)
- Null value handling (2 tests)
- Gzip request handling (1 test)
- Stub CRUD operations:
  - Overwriting single stub (1 test)
  - Deleting single stub (1 test)
  - Adding single stub (2 tests)
  - Validation errors for bad stub data (1 test)

**Files to investigate**:
- `internal/imposter/http.go` - Content-Type response handling
- `internal/imposter/predicates.go` - Predicate evaluation logic
- `internal/api/handlers/stubs.go` - Stub CRUD operations

**Estimated effort**: 2-3 days

### High Priority (P1)

#### HTTP Imposter Issues (~16 failing tests)
**Impact**: Medium - imposter management features

**Known Issues**:
- Auto-assign port functionality (when port not provided) (2 tests)
- DELETE /imposters response format:
  - Missing recordRequests field (1 test)
  - Missing requests array (1 test)
  - Replayable format edge cases (1 test)
- GET /imposters response format differences (2 tests)

**Files to investigate**:
- `internal/api/handlers/imposters.go` - Port assignment, DELETE response
- `internal/api/handlers/imposter.go` - GET imposter response format

**Estimated effort**: 1-2 days

#### Imposters Controller Issues (~3 failing tests)
**Impact**: Medium - API consistency

**Known Issues**:
- Response format differences in controller operations
- Missing fields in certain response scenarios

**Files to investigate**:
- `internal/api/handlers/imposters.go` - Controller response formatting

**Estimated effort**: 0.5-1 day

### Medium Priority (P2)

#### TCP Remaining Issues (~5 failing tests)
**Impact**: Low - edge cases, TCP mostly works

**Known Issues**:
- Binary mode with `matches` predicate - should return 400 error (1 test)
- Request recording format differences (2 tests)
- Proxy edge cases with port conflicts (2 tests)

**Files to investigate**:
- `internal/imposter/tcp_server.go` - Predicate validation, request recording
- `internal/imposter/predicates.go` - Binary mode validation

**Estimated effort**: 0.5-1 day

#### HTTPS Proxy Issues (~2 failing tests)
**Impact**: Low - specific proxy scenarios

**Known Issues**:
- Proxy to HTTPS origins with mutual auth (1 test)
- Certificate field persistence in API responses (1 test)

**Files to investigate**:
- `internal/imposter/proxy.go` - HTTPS proxy handling
- `internal/api/handlers/imposter.go` - Certificate field serialization

**Estimated effort**: 0.5 day

#### HTTP Fault Injection (~2 failing tests)
**Impact**: Low - advanced fault scenarios

**Known Issues**:
- Specific fault timing scenarios
- Connection handling edge cases

**Files to investigate**:
- `internal/imposter/http.go` - Fault injection implementation

**Estimated effort**: 0.5 day

### Won't Fix (Expected Differences)

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

Based on impact and effort, recommended order:

1. **HTTP Stub Content-Type** (P0) - Quick win, fixes many tests
2. **HTTP Stub Predicates** (P0) - deepEquals, exists, AND logic
3. **HTTP Imposter Auto-assign Port** (P1) - Important feature
4. **HTTP Stub CRUD Operations** (P0) - API completeness
5. **Imposters Controller Fixes** (P1) - API consistency
6. **TCP Binary Mode Validation** (P2) - Edge case fix
7. **HTTPS Proxy** (P2) - Low priority, works for most cases

## Validation Workflow

### Running Mountebank Tests

```bash
cd /home/tetsujinoni/work/mountebank
npm run test:api

# Current results: 66 passing, 186 failing
# Target: Reduce failures to ~20 (Node.js + CLI + Web UI)
```

### Running Go Tests

```bash
go test ./internal/... ./cmd/...
# All tests should pass (~8 seconds)
```

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
