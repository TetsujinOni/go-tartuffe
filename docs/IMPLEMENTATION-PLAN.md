# Mountebank Compatibility Implementation Plan

## Overview

This document outlines the plan to fix behavior test failures in go-tartuffe, focusing on a test-driven approach using lightweight Go tests instead of the heavyweight mountebank mbTest mocha harness.

## Strategy

### 1. Test-Driven Development
- Create Go unit tests that reproduce the defects
- Fix the implementation
- Verify with Go tests
- Validate with mountebank tests (optional)

### 2. Benefits of Go Tests

**Speed**: Go tests run in milliseconds vs. seconds for mountebank tests
```
Go unit test:     0.006s
Go integration:   0.267s
Mountebank test:  2-10s per test
```

**Simplicity**: Direct testing without process management
```go
// Simple Go test
func TestBehaviorParsing(t *testing.T) {
    var resp Response
    json.Unmarshal([]byte(`{"_behaviors": {"wait": 1000}}`), &resp)
    // Assert expectations
}
```

**Debuggability**: Standard Go debugging tools work immediately

**CI/CD**: Faster feedback loops in continuous integration

## Fix #4: Behaviors JSON Parsing

### Problem
Mountebank accepts `_behaviors` as both object and array:
```json
// Object format (single behavior)
{"_behaviors": {"wait": 1000}}

// Array format (multiple behaviors)
{"_behaviors": [{"wait": 1000}]}
```

Go's JSON unmarshaler expects a single type (array), causing all 50 behavior tests to fail.

### Solution

**File Modified**: [internal/models/stub.go](../internal/models/stub.go:67-126)

Custom `UnmarshalJSON` that:
1. Parses JSON to raw map
2. Detects `_behaviors` format
3. Normalizes object → array
4. Continues standard unmarshaling

### Test Implementation

**Unit Tests**: [internal/models/stub_test.go](../internal/models/stub_test.go)
```go
func TestBehaviorUnmarshalSingleObject(t *testing.T) {
    // Tests object format: {"_behaviors": {"wait": 1000}}
    // Tests array format: {"_behaviors": [{"wait": 1000}]}
    // Tests multiple fields: {"_behaviors": {"wait": 500, "repeat": 2}}
    // Tests copy, lookup, decorate, etc.
}
```

**Integration Tests**: [internal/api/handlers/behaviors_test.go](../internal/api/handlers/behaviors_test.go)
```go
func TestCreateImposterWithBehaviorsObject(t *testing.T) {
    // Tests full imposter creation via HTTP API
    // Tests all behavior types
    // Tests error handling
}
```

### Results

```
=== RUN   TestBehaviorUnmarshalSingleObject
=== RUN   TestBehaviorUnmarshalSingleObject/single_wait_behavior_as_object
=== RUN   TestBehaviorUnmarshalSingleObject/single_decorate_behavior_as_object
=== RUN   TestBehaviorUnmarshalSingleObject/multiple_fields_in_behavior_object
=== RUN   TestBehaviorUnmarshalSingleObject/copy_behavior_as_object
--- PASS: TestBehaviorUnmarshalSingleObject (0.00s)
PASS
ok  	github.com/TetsujinOni/go-tartuffe/internal/models	0.006s
```

**Impact**: Unlocks 50 failing behavior tests

## Testing Workflow

### 1. Create Failing Tests
```bash
# Write test that reproduces the issue
cat > internal/models/feature_test.go <<EOF
func TestFeature(t *testing.T) {
    // Test that currently fails
}
EOF

# Verify it fails
go test ./internal/models -run TestFeature
```

### 2. Implement Fix
```bash
# Make changes to implementation
# Fix should make test pass
```

### 3. Verify Fix
```bash
# Run specific test
go test ./internal/models -run TestFeature -v

# Run all tests (ensure no regressions)
go test ./...

# Optional: Verify with mountebank
MB_EXECUTABLE="./bin/tartuffe-wrapper.sh" npm run test:api
```

## Test Organization

```
go-tartuffe/
├── internal/
│   ├── models/
│   │   ├── stub.go              # Implementation
│   │   └── stub_test.go         # Unit tests
│   └── api/
│       └── handlers/
│           ├── imposters.go      # HTTP handlers
│           └── behaviors_test.go # Integration tests
└── test/
    └── integration/
        └── api_test.go           # End-to-end tests
```

## Next Priorities

Following the same test-driven approach:

### Fix #5: Injection Tests (18/22 failing)
**Root Cause**: Similar to behaviors - likely parsing issues

**Approach**:
1. Create Go test reproducing injection parsing
2. Add custom UnmarshalJSON if needed
3. Handle goja vs Node.js differences
4. Verify with Go tests

**Expected**: Similar pattern to behaviors fix

### Fix #6: TCP Protocol (18/26 failing)
**Root Cause**: Binary mode, predicate matching, keepalive

**Approach**:
1. Create Go tests for each TCP feature
2. Fix binary mode handling
3. Improve predicate matching
4. Add keepalive support
5. Test each feature independently

### Fix #7-8: HTTPS/SMTP
**Approach**: Protocol-specific testing with Go's native HTTP/SMTP clients

## Advantages Over Mountebank Tests

| Aspect | Go Tests | Mountebank Tests |
|--------|----------|------------------|
| Speed | 0.006s | 2-10s |
| Setup | None | Process management |
| Debugging | Native Go tools | Node.js debugging |
| CI/CD | Fast feedback | Slower pipeline |
| Dependencies | Go stdlib | Node.js, npm, mountebank |
| Isolation | Per-function | Per-process |
| Coverage | Line-level | Integration-level |

## When to Use Each

**Go Unit Tests**:
- Parsing/unmarshaling logic
- Data structure validation
- Business logic
- Fast iteration during development

**Go Integration Tests**:
- HTTP handler behavior
- Multi-component interactions
- Error handling flows

**Mountebank Tests**:
- Final validation
- Compatibility verification
- Release confidence
- Bug reports from users

## Conclusion

This approach provides:
- ✅ Faster development cycles
- ✅ Better debugging experience
- ✅ Higher confidence in fixes
- ✅ Easier maintenance
- ✅ Better CI/CD integration

All while maintaining full mountebank compatibility as the ultimate validation.
