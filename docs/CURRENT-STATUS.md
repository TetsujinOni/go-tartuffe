# go-tartuffe Current Status (2026-01-18)

## Compatibility Summary

| Metric | Value |
|--------|-------|
| **Raw Compatibility** | 77.8% (196/252 tests) ✅ |
| **Adjusted Compatibility** | 83.1% (196/236 tests, excluding security blocks) |
| **Target** | 75%+ |
| **Status** | ✅ **TARGET EXCEEDED!** |

## Recent Fixes (2026-01-17 to 2026-01-18)

1. ✅ **HTTP Stub Comprehensive Fixes** (+7 net tests, 2026-01-18)
   - Gzip request decompression - Handles Content-Encoding: gzip
   - XPath array predicates - Returns JSON arrays for multiple nodes
   - Order-insensitive array matching - Arrays match regardless of order
   - Stub validation - AddStub validates 'stub' field presence

2. ✅ **JSON Predicate Enhancements** (+1 net test, 2026-01-18)
   - contains, startsWith, endsWith predicates now support JSON bodies
   - matches predicate case-insensitive by default, respects caseSensitive option
   - Top-level array handling in equals predicate
   - except option now works with JSON values
   - caseSensitive option properly affects both keys and values

3. ✅ **JavaScript Injection Compatibility** (+8 tests, 2026-01-17)
   - Old interface: `function(request, state, logger, callback, imposterState)`
   - New interface: `function(config)`
   - Config object flattening for backward compatibility
   - State persistence across requests

4. ✅ **JSON Body Serialization** (+4 tests, 2026-01-17)
   - Object bodies converted to JSON strings early in processing
   - Removed automatic `Content-Type: application/json` header
   - Matches mountebank behavior

5. ✅ **deepEquals Type Coercion** (+3 tests, 2026-01-17)
   - Type-insensitive comparison using `forceToString()`
   - Integer `1` matches string `"1"`
   - Recursive conversion for nested objects/arrays

6. ✅ **Connection: close Header** (2026-01-17)
   - Set by default per mountebank behavior
   - Prevents connection reuse issues

## What's Working

### Core Functionality ✅
- HTTP/HTTPS basic stubs and responses
- TCP basic stubs and binary data
- SMTP basic functionality
- Predicate matching (equals, deepEquals, contains, matches, exists)
- JavaScript injection (sync only, both interfaces)
- Auto-assign ports (port=0)

### Behaviors ✅
- **Wait** - Static and dynamic latency
- **Wait as function** - Dynamic calculation
- **Decorate** - JavaScript post-processing
- **Copy** - Regex, XPath, JSONPath extraction
- **Lookup** - CSV lookup with selectors
- **Repeat** - Response cycling

### Security ✅
- **Fault injection** - CONNECTION_RESET_BY_PEER, RANDOM_DATA_THEN_CLOSE
- **CORS handling** - allowCORS option
- **Metrics** - Prometheus-format /metrics endpoint
- **HTTPS mTLS** - Mutual TLS authentication

## Security Improvements Over Mountebank

| Feature | Mountebank | go-tartuffe | Benefit |
|---------|------------|-------------|---------|
| shellTransform | ✅ Enabled | ❌ Disabled | Prevents RCE attacks |
| process.env access | ✅ Allowed | ❌ Blocked | Prevents secret leakage |
| Private key in API | ✅ Returns | ❌ Omitted | Prevents key exposure |
| Async injection | ✅ Supported | ❌ Not supported | ES5.1 limitation |

**Total security blocks:** 16 tests (6.3%)

## Remaining Gaps

**40 actionable failures** remain (down from initial 79). See [COMPATIBILITY-BACKLOG.md](../COMPATIBILITY-BACKLOG.md) for detailed breakdown.

**Summary by priority:**
- High Priority: 18 tests (HTTP/HTTPS Proxy)
- Medium Priority: 13 tests (TCP implementation)
- Low Priority: 9 tests (Faults, metrics, controller APIs)

**16 intentional security/architectural blocks** (Won't Fix):
- ShellTransform (8 tests)
- Process object access (2 tests)
- Async injection (5 tests)
- Private key return (1 test)

## Validation Workflow

```bash
# Build latest
cd /home/tetsujinoni/work/go-tartuffe
go build -o bin/tartuffe ./cmd/tartuffe

# Run mountebank tests against tartuffe
cd /home/tetsujinoni/work/mountebank
MB_EXECUTABLE=/home/tetsujinoni/work/go-tartuffe/bin/tartuffe-wrapper.sh npm run test:api

# Expected: 196 passing, 56 failing (77.8% raw, 83.1% adjusted)
```

## Documentation

- [COMPATIBILITY-BACKLOG.md](../COMPATIBILITY-BACKLOG.md) - Remaining work tracking
- [HTTP-STUB-FIX-SUMMARY.md](HTTP-STUB-FIX-SUMMARY.md) - HTTP stub fixes (2026-01-18)
- [JSON-PREDICATE-FIX-SUMMARY.md](JSON-PREDICATE-FIX-SUMMARY.md) - JSON predicate fixes
- [SECURITY-DECISIONS.md](SECURITY-DECISIONS.md) - Security trade-offs
- [TEST-RESULTS-2026-01-17.md](TEST-RESULTS-2026-01-17.md) - Historical test results

## Conclusion

go-tartuffe has **exceeded target compatibility (77.8% raw, 83.1% adjusted)** ✅ with significant **security improvements** over mountebank. The remaining gaps are primarily in advanced proxy features and TCP protocol completeness. Core functionality including JSON predicates, HTTP stubs, gzip support, and XPath is production-ready.
