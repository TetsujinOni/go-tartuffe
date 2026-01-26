# Claude Code Project Guidelines

## Project Overview

**Project**: go-tartuffe - Go implementation of mountebank service virtualization
**Branch**: patch/fix-stub-pattern-matching
**Compatibility Target**: 75%+ with mountebank API test suite
**Current Status**: **92.1% (232/252 tests) | 100% adjusted (all actionable items complete)**

**Recent Achievements**:
- Predicate Map Matching: Fixed contains/startsWith/endsWith predicates on headers/query with map patterns
- HTTP Proxy CONNECT: Loopback tunnel support for HTTPS proxy scenarios
- TCP Packet Splitting: Each TCP packet recorded as separate request
- TCP Proxy: Non-TCP protocol rejection with proper error format
- SMTP: Fixed html field serialization (always include even when empty)

## Backlog Management Philosophy

### What Belongs in a Backlog

A backlog should contain **ONLY remaining work** - items that are not yet done.

**Include:**
- Outstanding bugs and issues
- Missing features
- Known gaps in functionality
- Prioritized work items
- Validation procedures for tracking progress

**DO NOT Include:**
- "Recent Fixes" sections
- "What's Working" lists (unless needed for context)
- Progress history (except final current status)
- Any celebration of completed work
- Changelogs or release notes

### Rationale

Backlogs are **forward-looking planning documents**. They help developers understand what needs to be done next. Completed work belongs in:
- Fix summary documents (e.g., `docs/HTTP-STUB-FIX-SUMMARY.md`)
- Historical documentation (e.g., `docs/TEST-RESULTS-2026-01-18.md`)
- Git commit messages
- Release notes
- Changelog files

### Maintaining Backlogs

When updating backlogs:

1. **Remove completed items** - Move them to historical documentation
2. **Keep status concise** - Current progress metrics only
3. **Focus on gaps** - What's broken or missing
4. **Prioritize** - Help developers know what to work on next

### Example Structure

**Good backlog:**
```markdown
# Project Backlog

## Current Status
- 85% compatibility (214/252 tests)
- 22 actionable failures remaining

## Remaining Work

### High Priority (13 items)
- TCP Behaviors
- ...

### Medium Priority (9 items)
- Edge cases
- ...
```

**Bad backlog:**
```markdown
# Project Backlog

## Recent Fixes (Don't do this!)
- ✅ Fixed JSON predicates (+7 tests)
- ✅ Fixed gzip support (+2 tests)
...
```

### When to Create Fix Summaries

After completing a significant fix or feature:

1. Create a dedicated fix summary document in `docs/`
   - Example: `docs/HTTP-PROXY-TEST-MAPPING.md`
   - Include: problem description, solution, code examples, impact

2. Update the backlog to remove completed items

3. Do not reference the fix summary from the backlog. It is never needed.

This keeps backlogs clean while preserving detailed information about fixes in dedicated documentation.

## Validation Workflow

### Prerequisites Check

Before running validation tests, ensure no existing tartuffe processes are running to prevent port conflicts:

```bash
# Stop any running tartuffe instances
pkill -f tartuffe || true

# Verify no process is listening on port 2525 (default MB_PORT)
lsof -ti:2525 | xargs kill -9 2>/dev/null || true
```

### Running Mountebank Test Suite

The mountebank test suite validates compatibility with the original mountebank behavior.

**CRITICAL:** The mountebank tests must use the `MB_EXECUTABLE` environment variable to test against go-tartuffe instead of the default mountebank binary.

#### Test Suite Overview

Mountebank has several test categories:

- **test:api** - API-level integration tests (232/252 passing - 92.1%)
  - Won't fix (20 tests):
    - Security: shellTransform (4), process object (2)
    - Architectural: replayable export (2), old proxy syntax, HTTPS key/cert creation, proxy behavior persistence
    - goja limitation: async injection (4)
    - TCP behaviors with composition (4)
- **test:js** - JavaScript client tests (passing)
- **test:cli** - CLI tests (won't fix - different CLI implementation)
- **test:web** - Web UI tests (won't fix - different UI)
- **test:unit** - Mountebank's internal unit tests (not applicable)

#### Full Validation Procedure

```bash
# 1. Ensure clean state
cd /home/tetsujinoni/work/go-tartuffe
pkill -f tartuffe || true

# 2. Build latest version
go build -o bin/tartuffe ./cmd/tartuffe

# 3. Run Go integration tests (should all pass)
go test ./test/integration/... -v
# Expected: 238+ tests passing in ~55 seconds

# 4. Run mountebank API tests against go-tartuffe
cd /home/tetsujinoni/work/mountebank
MB_EXECUTABLE=/home/tetsujinoni/work/go-tartuffe/bin/tartuffe-wrapper.sh npm run test:api
# Current: 232 passing, 20 failing (252 total) = 92.1%

# 5. Clean up
pkill -f tartuffe || true
```

#### Quick Validation (API tests only)

```bash
cd /home/tetsujinoni/work/mountebank
# Only kill tartuffe if port 2525 is in use
lsof -ti:2525 >/dev/null 2>&1 && pkill -f tartuffe 2>/dev/null || true
MB_EXECUTABLE=/home/tetsujinoni/work/go-tartuffe/bin/tartuffe-wrapper.sh npm run test:api 2>&1 | tee /tmp/tartuffe-validation.log
```

To check just the summary:
```bash
grep -E "(passing|failing|pending)" /tmp/tartuffe-validation.log | tail -5
```

#### Full Test Suite Output Handling

**CRITICAL:** When running the full mountebank test suite:

1. **NEVER use `tail` on the output** - truncated output loses valuable failure details
2. **Always capture full logs** using `tee` or redirect to a file:
   ```bash
   MB_EXECUTABLE=/home/tetsujinoni/work/go-tartuffe/bin/tartuffe-wrapper.sh npm run test:api 2>&1 | tee /tmp/tartuffe-validation.log
   ```
3. **Analyze the log file** instead of streaming output:
   ```bash
   # Get pass/fail summary
   grep -E "^\s+[0-9]+ (passing|failing)" /tmp/tartuffe-validation.log

   # List all failing tests
   grep -E "^\s+[0-9]+\)" /tmp/tartuffe-validation.log

   # Find specific test failures
   grep -A 20 "should specific test name" /tmp/tartuffe-validation.log
   ```

This ensures no information is lost during context compaction or output truncation.

#### Validation Notes

**Critical:** Without setting `MB_EXECUTABLE`, the mountebank tests will use the original Node.js mountebank binary instead of go-tartuffe, resulting in incorrect validation (all tests passing with original mountebank).

**Setting MB_EXECUTABLE:**
- Points mountebank tests to use tartuffe binary via wrapper script
- The wrapper script (`tartuffe-wrapper.sh`) handles command compatibility
- Must be an absolute path to the wrapper script

### Running Go Tests

```bash
cd /home/tetsujinoni/work/go-tartuffe

# Run all integration tests
go test ./test/integration/... -v

# Run all unit tests
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
5. **Update documentation** (create fix summary in docs/, update COMPATIBILITY-BACKLOG.md)
6. **Commit with descriptive message**

### Test-Driven Development Pattern

```bash
# 1. Create test file
vim test/integration/feature_test.go

# 2. Run test (should fail)
go test ./test/integration -run TestFeature -v

# 3. Implement feature
vim internal/imposter/feature.go

# 4. Run test again (should pass)
go test ./test/integration -run TestFeature -v

# 5. Run full suite
go test ./test/integration/...
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
│   │   ├── behaviors.go          # Behavior implementations
│   │   ├── http_server.go        # HTTP protocol implementation
│   │   ├── tcp_server.go         # TCP protocol implementation
│   │   ├── inject.go             # JavaScript injection engine
│   │   ├── proxy.go              # Proxy behavior
│   │   ├── matcher.go            # Predicate matching
│   │   └── selectors.go          # JSONPath/XPath selectors
│   ├── models/
│   │   ├── imposter.go           # Imposter data model
│   │   ├── stub.go               # Stub and behavior models
│   │   └── request.go            # Request model
│   └── metrics/
│       └── metrics.go            # Prometheus metrics
└── bin/
    └── tartuffe-wrapper.sh       # Wrapper for mountebank tests
```

### Test Files

```
go-tartuffe/test/integration/
├── http_stub_test.go             # HTTP stub tests
├── http_injection_test.go        # HTTP injection tests
├── http_injection_state_test.go  # Injection state tests
├── http_proxy_always_test.go     # ProxyAlways mode tests (13 tests)
├── http_proxy_edge_cases_test.go # Proxy edge cases (5 tests)
├── proxy_inject_test.go          # Basic proxy tests (4 tests)
├── tcp_injection_test.go         # TCP injection tests
├── tcp_imposter_test.go          # TCP imposter tests
├── advanced_predicate_test.go    # Complex predicate tests
├── jsonpath_predicates_test.go   # JSONPath selector tests
└── ...                           # 228 total test functions
```

### Documentation

```
go-tartuffe/
├── COMPATIBILITY-BACKLOG.md      # Remaining work only (backlog)
├── .claude/
│   └── claude.md                 # This file - workflows & guidelines
└── docs/
    ├── HTTP-PROXY-TEST-MAPPING.md     # HTTP Proxy test coverage (89%)
    ├── TCP-TEST-MAPPING.md            # TCP test mapping
    ├── BASELINE-UPDATE-2026-01-18.md  # Latest baseline update
    ├── SECURITY.md                    # Security decisions
    ├── TEST-HARNESS-FIX.md            # Test harness setup
    ├── BEHAVIOR-FIX.md                # Behavior implementation
    └── TEST-RESULTS-*.md              # Historical test results
```

## Common Tasks

### Adding a New Feature with Tests

1. **Create test file:**
   ```bash
   vim test/integration/feature_test.go
   ```

2. **Add test cases:**
   ```go
   func TestFeatureBasic(t *testing.T) {
       defer cleanup(t)

       // Create imposter
       resp, body, err := post("/imposters", map[string]interface{}{
           "protocol": "http",
           "port":     8000,
           // ... configuration
       })

       // Test implementation
       // ... assertions
   }
   ```

3. **Implement feature in internal/imposter/**

4. **Run tests to verify:**
   ```bash
   go test ./test/integration -run TestFeature -v
   ```

### Debugging Test Failures

1. **Run with verbose output:**
   ```bash
   go test ./test/integration -run TestXxx -v
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
   cd /home/tetsujinoni/work/mountebank/mbTest
   npx mocha api/http/httpProxyStubTest.js -g "specific test name"
   ```

### Updating Documentation After Implementation

After completing a feature:

1. **Create fix summary in docs/:**
   ```bash
   vim docs/FEATURE-NAME-FIX-SUMMARY.md
   ```
   Include: problem, solution, code examples, test coverage impact

2. **Update COMPATIBILITY-BACKLOG.md:**
   - Remove completed items from actionable failures
   - Update overall progress percentage
   - Add brief note about completed feature in "Recent Improvements" if significant

3. **Update test mapping if applicable:**
   - Update docs/HTTP-PROXY-TEST-MAPPING.md or TCP-TEST-MAPPING.md
   - Mark tests as covered with test function names

4. **Commit:**
   ```bash
   git add docs/ COMPATIBILITY-BACKLOG.md
   git commit -m "docs: update after feature X implementation"
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
```

### JavaScript Function Context

**Problem:** JavaScript functions need proper request/response objects.

**Solution:** Always create request object with all required fields:
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

### JavaScript Buffer API

**Problem:** String interpolation can cause Buffer.toString() scoping issues.

**Solution:** Use VM.Set() to pass data:
```go
vm.Set("requestData", requestData)
// Then in JS: Buffer.from(requestData, 'utf8')
```

## Security Considerations

### shellTransform Behavior - DISABLED

The `shellTransform` behavior is intentionally disabled for security reasons:
- Allows arbitrary command execution
- Creates command injection vulnerabilities
- Unrestricted system access

**Alternative:** Use `decorate` behavior with JavaScript for response transformations.

See [docs/SECURITY.md](../docs/SECURITY.md) for details.

### JavaScript Sandboxing

JavaScript code runs in goja (ES5.1) VM with:
- ✅ No access to Node.js `require()` or filesystem
- ✅ No access to `process.env`
- ✅ Limited to safe JavaScript operations
- ✅ Timeout protection (configurable)

## Performance Notes

### Test Execution Times

- **Go integration tests**: ~50 seconds for 228 tests (309 including sub-tests)
- **Mountebank API tests**: ~15-20 seconds for 252 tests
- **Individual Go test**: <100ms (most <10ms)

### Optimization Tips

- Use `jsEngine := NewJSEngine()` once per test suite
- Reuse BehaviorExecutor when possible
- Avoid unnecessary JSON marshaling/unmarshaling

## Useful Commands

### Git Operations

```bash
# View recent commits
git log --oneline -10

# View changes in a file
git diff internal/imposter/proxy.go

# View specific commit
git show <commit-hash>

# Create new branch
git checkout -b feat/new-feature
```

### Code Search

```bash
# Find all references to a function
grep -r "ProxyHandler" internal/

# Find all test files
find test/integration -name "*_test.go"

# Find TODO comments
grep -r "TODO" internal/
```

### Process Management

**IMPORTANT: pkill Exit Code Behavior**

`pkill` has non-zero exit codes in multiple situations:
- **Exit code 1**: No matching processes found
- **Exit code 144**: Processes were killed (128 + SIGTERM signal 16) - **this is expected/success**

Because of these exit codes:
- **NEVER chain commands after pkill with `&&`** - the chain will abort regardless of whether processes were killed or not found
- **pkill must be the TERMINAL command** or use `|| true` to suppress the exit code
- **Exit code 144 is NOT an error** - it indicates processes were successfully terminated

```bash
# WRONG - will abort the chain in ALL cases (exit 1 if none, exit 144 if killed)
pkill -f tartuffe && go test ./...  # ❌ Never reaches go test

# CORRECT - suppress exit code with || true
pkill -f tartuffe || true           # ✅ Always succeeds
go test ./...                        # ✅ Runs regardless

# CORRECT - use semicolon if you don't care about pkill result
pkill -f tartuffe 2>/dev/null; go test ./...  # ✅ Always continues

# CORRECT - pkill as terminal command (no chaining needed)
pkill -f tartuffe 2>/dev/null || true  # ✅ Safe, always returns 0

# CORRECT - check if processes exist first
pgrep -f tartuffe >/dev/null && pkill -f tartuffe || true  # ✅ Safe
```

**Validation Workflow Note:** When running validation with process cleanup, always use separate commands or `|| true`:
```bash
# Kill any existing tartuffe processes
pkill -f tartuffe 2>/dev/null || true

# Then run validation
cd /home/tetsujinoni/work/mountebank
MB_EXECUTABLE=/home/tetsujinoni/work/go-tartuffe/bin/tartuffe-wrapper.sh npm run test:api
```

**Common Process Management Commands:**

```bash
# Find tartuffe processes
ps aux | grep tartuffe

# Kill all tartuffe processes (safe - always succeeds)
pkill -f tartuffe 2>/dev/null || true

# Check what's using port 2525
lsof -i:2525

# Kill process on specific port
lsof -ti:2525 | xargs kill -9 2>/dev/null || true
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
for port in 2525 2526 2527 8000 8100 8200; do
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

### Test failures after changes

```bash
# Run specific test with verbose output
go test ./test/integration -run TestHTTPProxy_ProxyAlways -v

# Check for port conflicts
pkill -f tartuffe || true

# Re-run full suite
go test ./test/integration/...
```

## Test Results Documentation

### Current Test Results
- Store in `docs/TEST-RESULTS-YYYY-MM-DD.md`
- Include detailed failure analysis

### Fix Summaries
- Store in `docs/FEATURE-NAME-FIX-SUMMARY.md`
- Include before/after comparisons
- Document code changes with line numbers
- Explain rationale and impact

### Compatibility Tracking
- Main backlog: `COMPATIBILITY-BACKLOG.md` (remaining work only)
- Test mappings: `docs/HTTP-PROXY-TEST-MAPPING.md`, `docs/TCP-TEST-MAPPING.md`
- Baseline updates: `docs/BASELINE-UPDATE-*.md`
- Historical results: `docs/TEST-RESULTS-*.md` (archived)

## Additional Resources

### Mountebank Documentation

- Website: http://www.mbtest.org
- GitHub: https://github.com/bbyars/mountebank
- API Docs: http://www.mbtest.org/docs/api/overview

### Go Resources

- Goja (JavaScript engine): https://github.com/dop251/goja
- Testing: https://pkg.go.dev/testing

## Session Continuity

When resuming work:

1. Read COMPATIBILITY-BACKLOG.md for current status (remaining work)
2. Read this file (.claude/claude.md) for workflows
3. Check recent commits: `git log --oneline -10`
4. Review test mappings in docs/ for specific feature areas
5. Run validation to establish current baseline:
   ```bash
   cd /home/tetsujinoni/work/go-tartuffe
   go test ./test/integration/... -v
   ```

## Context Continuity

When processing user requests with limited context remaining:
- Create "extended-context-summary.md" summarizing the request and plan
- Include key decisions, code changes, and rationale
- Reference this for continuity in next session
