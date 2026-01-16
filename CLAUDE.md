# Claude Development Guide for go-tartuffe

This document contains workflow hints, validation procedures, and development guidelines for working on go-tartuffe. It preserves context for Claude AI sessions.

## Quick Reference

- **Project**: go-tartuffe - Go implementation of mountebank service virtualization
- **Branch**: feat/missing-backlog
- **Compatibility Target**: 95%+ with mountebank API test suite, 100% of the mountebank JS test suite.
- **Current Status**: **57.1% (144/252 tests passing, 108 failing)**

## Validation Workflow

### Prerequisites Check

Before running validation tests, ensure no existing tartuffe processes are running to prevent port conflicts:

```bash
# Stop any running tartuffe instances
pkill -f tartuffe || true

# Or more specifically:
killall tartuffe 2>/dev/null || true

# Verify no process is listening on port 2525 (default MB_PORT)
lsof -ti:2525 | xargs kill -9 2>/dev/null || true
```

### Running Mountebank Test Suite

The mountebank test suite validates compatibility with the original mountebank behavior.

#### Test Suite Overview

Mountebank has several test categories:

- **test:api** - API-level integration tests (**146 passing, 106 failing - 57.9%**)
  - Recent fixes: copy, lookup, repeat behaviors, composition, TCP injection (22 tests fixed)
  - Remaining gaps: TCP behaviors/proxy, HTTP proxy, edge cases
- **test:js** - JavaScript client tests (3 passing, 0 failing - 100%)
- **test:cli** - CLI tests (won't fix - different CLI implementation)
- **test:web** - Web UI tests (won't fix - different UI)
- **test:unit** - Mountebank's internal unit tests (not applicable)

#### Full Validation Procedure

**IMPORTANT:** The mountebank tests must use the `MB_EXECUTABLE` environment variable to test against go-tartuffe instead of the default mountebank binary.

```bash
# 1. Ensure clean state
cd /home/tetsujinoni/work/go-tartuffe
pkill -f tartuffe || true

# 2. Build latest version
go build -o bin/tartuffe ./cmd/tartuffe

# 3. Run Go unit tests (should all pass)
go test ./internal/... ./cmd/...
# Expected: All tests pass in ~5 seconds

# 4. Run mountebank API tests against go-tartuffe
cd /home/tetsujinoni/work/mountebank
MB_EXECUTABLE=/home/tetsujinoni/work/go-tartuffe/bin/tartuffe-wrapper.sh npm run test:api
# Current: 146 passing, 106 failing (252 total) = 57.9%

# 5. Run mountebank JavaScript tests against go-tartuffe
MB_EXECUTABLE=/home/tetsujinoni/work/go-tartuffe/bin/tartuffe-wrapper.sh npm run test:js
# Expected: 3 passing, 0 failing

# 6. Clean up
pkill -f tartuffe || true
```

#### Quick Validation (API tests only)

**Note:** Run this synchronously (not in background) to see results immediately.

```bash
cd /home/tetsujinoni/work/mountebank
pkill -f tartuffe 2>/dev/null || true
MB_EXECUTABLE=/home/tetsujinoni/work/go-tartuffe/bin/tartuffe-wrapper.sh npm run test:api 2>&1 | tee /tmp/tartuffe-validation.log
```

To check just the summary:
```bash
grep -E "(passing|failing|pending)" /tmp/tartuffe-validation.log | tail -5
```

#### Validation Notes

**Critical:** Without setting `MB_EXECUTABLE`, the mountebank tests will use the original Node.js mountebank binary instead of go-tartuffe, resulting in incorrect validation (all tests passing with original mountebank).

**Setting MB_EXECUTABLE:**
- Points mountebank tests to use tartuffe binary via wrapper script
- The wrapper script (`tartuffe-wrapper.sh`) handles command compatibility (e.g., `mb restart`)
- Must be an absolute path to the wrapper script

#### Test Results Interpretation

**Current status (as of 2026-01-16 end-of-session):**
- **test:api**: **146 passing, 106 failing (252 total) = 57.9%**
- **test:js**: Not yet tested
- **Target**: 75%+ passing excluding the shellTransform tests - **MAKING PROGRESS**

**Recent fixes** (this session):
- ✅ **Copy behavior** (6 tests) - Fixed array parsing, token replacement
- ✅ **Lookup behavior** (6 tests) - Fixed xpath/jsonpath with namespaces
- ✅ **Repeat behavior** (6 tests) - Fixed response cycling logic
- ✅ **Behavior composition** (2 tests) - Fixed "behaviors" vs "_behaviors" parsing
- ✅ **TCP injection** (2 tests) - Fixed by passing requestData via VM.Set instead of string interpolation
- **Total: +22 tests fixed**

**Remaining failure categories**:
1. **shellTransform** (6 tests) - Expected failure (security block) - Won't Fix
2. **TCP behaviors** (~6 tests) - Decorate/behaviors not working in TCP context
3. **TCP proxy** (~5 tests) - endOfRequestResolver and error handling issues
4. **HTTP proxy** (many tests) - Various proxy functionality gaps
5. **Response format** (multiple) - Missing fields (savedRequests, numberOfRequests vs recordRequests)
6. **Various edge cases** (~68 tests) - Case-sensitive headers, gzip support, xpath predicates in matchers, etc.

### Running Go Tests

```bash
cd /home/tetsujinoni/work/go-tartuffe

# Run all tests
go test ./internal/... ./cmd/...

# Run specific package tests
go test ./internal/imposter -v

# Run specific test
go test ./internal/imposter -run TestWait -v

# Run with coverage
go test -cover ./internal/...

# Run with race detection
go test -race ./internal/...
```

## Development Workflow

### Making Changes

1. **Create/update tests first** (TDD approach)
2. **Implement the feature**
3. **Run Go tests** to verify implementation
4. **Run mountebank tests** to verify compatibility
5. **Update documentation** (COMPATIBILITY-BACKLOG.md, CLAUDE.md)
6. **Commit with descriptive message**

### Test-Driven Development Pattern

```bash
# 1. Create test file
vim internal/imposter/behaviors_xxx_test.go

# 2. Run test (should fail)
go test ./internal/imposter -run TestXxx -v

# 3. Implement feature
vim internal/imposter/behaviors.go

# 4. Run test again (should pass)
go test ./internal/imposter -run TestXxx -v

# 5. Run full suite
go test ./internal/... ./cmd/...
```

### Commit Message Format

```
<type>: <subject>

<body with details>

<optional footer>

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
```

**Types:**
- `feat:` - New feature
- `fix:` - Bug fix
- `docs:` - Documentation only
- `test:` - Adding tests
- `refactor:` - Code refactoring
- `perf:` - Performance improvement
- `chore:` - Maintenance

## Key Files and Locations

### Source Code

```
go-tartuffe/
├── cmd/tartuffe/main.go          # CLI entry point
├── internal/
│   ├── api/
│   │   └── handlers/             # HTTP API handlers
│   ├── imposter/
│   │   ├── behaviors.go          # Behavior implementations (wait, decorate, copy)
│   │   ├── http_server.go        # HTTP protocol implementation
│   │   ├── tcp_server.go         # TCP protocol implementation
│   │   ├── inject.go             # JavaScript injection engine
│   │   └── proxy.go              # Proxy behavior
│   └── models/
│       ├── imposter.go           # Imposter data model
│       ├── stub.go               # Stub and behavior models
│       └── request.go            # Request model
└── bin/
    └── tartuffe-wrapper.sh       # Wrapper for mountebank tests
```

### Test Files

```
go-tartuffe/internal/imposter/
├── behaviors_wait_test.go        # Wait behavior tests (4 functions)
├── behaviors_decorate_test.go    # Decorate behavior tests (6 functions)
├── behaviors_copy_test.go        # Copy behavior tests (4 functions)
├── behaviors_repeat_test.go      # Repeat behavior tests (placeholder)
├── http_test.go                  # HTTP protocol tests
├── tcp_test.go                   # TCP protocol tests
└── inject_test.go                # Injection tests
```

### Documentation

```
go-tartuffe/
├── COMPATIBILITY-BACKLOG.md      # Test results and remaining gaps
├── CLAUDE.md                     # This file - workflow hints
├── docs/
│   ├── SECURITY.md              # Security decisions (shellTransform)
│   ├── TEST-HARNESS-FIX.md      # Test harness setup
│   ├── BEHAVIOR-FIX.md          # Behavior implementation notes
│   └── IMPLEMENTATION-PLAN.md   # TDD strategy
└── .claude/plans/               # Claude planning sessions
```

## Common Tasks

### Adding a New Behavior

1. **Create test file:**
   ```bash
   vim internal/imposter/behaviors_xxx_test.go
   ```

2. **Add test cases:**
   ```go
   func TestXxxBasic(t *testing.T) {
       jsEngine := NewJSEngine()
       executor := NewBehaviorExecutor(jsEngine)
       // ... test implementation
   }
   ```

3. **Implement behavior in behaviors.go:**
   ```go
   func (e *BehaviorExecutor) executeXxx(...) error {
       // implementation
   }
   ```

4. **Wire into Execute() method:**
   ```go
   if behavior.Xxx != nil {
       if err = e.executeXxx(...); err != nil {
           return nil, fmt.Errorf("xxx behavior error: %w", err)
       }
   }
   ```

### Debugging Test Failures

1. **Run with verbose output:**
   ```bash
   go test ./internal/imposter -run TestXxx -v
   ```

2. **Add debug output:**
   ```go
   t.Logf("Debug: value=%v", value)
   ```

3. **Check mountebank expected behavior:**
   ```bash
   cd /home/tetsujinoni/work/mountebank
   grep -r "should do something" mbTest/
   ```

4. **Run single mountebank test:**
   ```bash
   cd mbTest
   npx mocha api/http/httpStubTest.js -g "specific test name"
   ```

### Updating Compatibility Status

After implementing features and running validation:

1. **Update COMPATIBILITY-BACKLOG.md:**
   - Update pass rate
   - Move features from "Missing" to "Completed"
   - Update remaining failure areas

2. **Commit the update:**
   ```bash
   git add COMPATIBILITY-BACKLOG.md
   git commit -m "docs: update compatibility backlog with X% pass rate"
   ```

## Environment Variables

### Mountebank Test Environment

```bash
export MB_EXECUTABLE=/home/tetsujinoni/work/go-tartuffe/bin/tartuffe-wrapper.sh
export MB_PORT=2525
```

These are typically set by the mountebank test harness automatically.

## Known Issues and Workarounds

### Port Conflicts in Tests

**Problem:** Multiple tests try to use the same ports, causing EADDRINUSE errors.

**Workaround:**
```bash
# Before running tests
pkill -f tartuffe 2>/dev/null || true
lsof -ti:2525 | xargs kill -9 2>/dev/null || true
```

### Test Harness Pidfile Issues

**Problem:** Pidfile not cleaned up between test runs.

**Solution:** Already fixed in commit 8be1a34. The stop command now exits successfully when pidfile doesn't exist.

### JavaScript Function Context

**Problem:** JavaScript functions need request/response objects.

**Solution:** Always create request object in executeXxxFunction():
```go
requestObj := map[string]interface{}{
    "method":  req.Method,
    "path":    req.Path,
    "query":   req.Query,
    "headers": req.Headers,
    "body":    req.Body,
}
vm.Set("request", requestObj)
```

## Security Considerations

### shellTransform Behavior - DISABLED

The `shellTransform` behavior is intentionally disabled for security reasons:
- Allows arbitrary command execution
- Creates command injection vulnerabilities
- Unrestricted system access

**Alternative:** Use `decorate` behavior with JavaScript for response transformations.

See [docs/SECURITY.md](docs/SECURITY.md) for details.

### JavaScript Sandboxing

JavaScript code runs in goja (ES5.1) VM with:
- ✅ No access to Node.js `require()` or filesystem
- ✅ No access to `process.env`
- ✅ Limited to safe JavaScript operations
- ✅ Timeout protection (configurable)

## Performance Notes

### Test Execution Times

- **Go unit tests**: ~5 seconds for full suite
- **Mountebank API tests**: ~14 seconds (181 passing)
- **Individual Go test**: <100ms (most <10ms)
- **Individual behavior**: <10ms overhead

### Optimization Tips

- Use `jsEngine := NewJSEngine()` once per test, not per scenario
- Reuse BehaviorExecutor when possible
- Avoid unnecessary JSON marshaling/unmarshaling

## Useful Commands

### Git Operations

```bash
# View recent commits
git log --oneline -10

# View changes in a file
git diff internal/imposter/behaviors.go

# View specific commit
git show 06c71be

# Create new branch
git checkout -b feat/new-feature
```

### Code Search

```bash
# Find all references to a function
grep -r "executeWait" internal/

# Find all test files
find . -name "*_test.go"

# Find TODO comments
grep -r "TODO" internal/
```

### Process Management

```bash
# Find tartuffe processes
ps aux | grep tartuffe

# Kill all tartuffe processes
pkill -f tartuffe

# Check what's using port 2525
lsof -i:2525

# Kill process on specific port
lsof -ti:2525 | xargs kill -9
```

## Troubleshooting

### "cannot find package" errors

```bash
# Download dependencies
go mod download

# Tidy up go.mod
go mod tidy
```

### "address already in use" errors

```bash
# Kill processes on common ports
for port in 2525 2526 2527; do
    lsof -ti:$port | xargs kill -9 2>/dev/null || true
done
```

### Build failures

```bash
# Clean build cache
go clean -cache

# Rebuild from scratch
rm -rf bin/tartuffe
go build -o bin/tartuffe ./cmd/tartuffe
```

## Achievement Status

### Compatibility Target: MAKING PROGRESS

**Target**: 75%+ compatibility
**Current**: **57.9% (146/252 tests passing)**

### Feature Status

**Working features**:
- ✅ Wait behavior - static and dynamic latency
- ✅ Decorate behavior - JavaScript post-processing
- ✅ Copy behavior - regex, xpath, jsonpath extraction and token replacement
- ✅ Lookup behavior - CSV lookup with xpath/jsonpath selectors and namespaces
- ✅ Repeat behavior - response cycling
- ✅ Behavior composition - multiple behaviors in sequence (new format)
- ✅ HTTP/HTTPS basic stubs - is responses, basic predicates
- ✅ TCP basic stubs - basic forwarding and responses
- ✅ HTTPS with mutual authentication
- ✅ SMTP - basic functionality
- 🔒 ShellTransform - **DISABLED for security** (6 tests intentionally fail)

**Features with gaps**:
- ✅ TCP injection - **FIXED** (now working with VM.Set approach)
- ❌ TCP behaviors - Decorate not working in TCP context (~6 tests failing)
- ❌ TCP proxy - endOfRequestResolver issues (5+ tests failing)
- ❌ HTTP proxy - Multiple gaps (many tests failing)
- ❌ Response format - Missing API fields (savedRequests, numberOfRequests)
- ❌ Various edge cases - gzip, xpath predicates in matchers, case-sensitive headers (~68 tests)

### ShellTransform Security Note:

ShellTransform is intentionally disabled (6 tests fail) as it allows arbitrary command execution, creating a critical command injection vulnerability. Users should use the `decorate` behavior with sandboxed JavaScript instead. See [docs/SECURITY.md](../docs/SECURITY.md).

## Additional Resources

### Mountebank Documentation

- Website: http://www.mbtest.dev
- GitHub: https://github.com/mountebank-testing/mountebank
- API Docs: http://www.mbtest.dev/docs/api/overview

### Go Resources

- Goja (JavaScript engine): https://github.com/dop251/goja
- Testing: https://pkg.go.dev/testing

## Session Continuity

When resuming work:

1. Read COMPATIBILITY-BACKLOG.md for current status
2. Read this file (CLAUDE.md) for workflows
3. Run validation to establish baseline
4. Check recent commits: `git log --oneline -10`
5. Review open TODOs in code: `grep -r "TODO" internal/`

---

**Last Updated**: 2026-01-16 (End of session - TCP injection fixed)
**Current Compatibility**: **57.9% (146/252 passing, 106 failing)**
**Branch**: feat/missing-backlog
**Status**: Making progress toward 75%+ target (+22 tests this session)
