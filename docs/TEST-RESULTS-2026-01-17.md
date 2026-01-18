# Test Results - 2026-01-17

## Summary

**Total Tests**: 252
**Passing**: 188 (74.6%)
**Failing**: 64 (25.4%)
**Adjusted (excluding security blocks)**: 188/236 (79.7%)

## Test Progress

| Date | Passing | Failing | Percentage |
|------|---------|---------|------------|
| Initial | 173 | 79 | 68.7% |
| After JS injection fix | 181 | 71 | 71.8% |
| After JSON body fix | 185 | 67 | 73.4% |
| After deepEquals fix | **188** | **64** | **74.6%** |

## Failure Breakdown

### Security/Architectural Blocks (Won't Fix) - 16 tests

These failures are intentional security improvements or architectural limitations:

| # | Test | Category | Rationale |
|---|------|----------|-----------|
| 1-8 | shellTransform (HTTP/HTTPS) | Security | Arbitrary command execution risk |
| 13-14 | Process object access (HTTP/HTTPS) | Security | Prevents environment variable leakage |
| 15-16 | Async injection (HTTP/HTTPS) | Architectural | goja ES5.1 lacks Promise/async |
| 44 | Private key return (HTTPS) | Security | Prevents key exposure in API responses |
| 53-54 | Async injection (TCP) | Architectural | goja ES5.1 lacks Promise/async |

**Total: 16 tests (6.3% of total)**

### Actionable Failures - 48 tests

These are implementation gaps that could be fixed:

#### HTTP/HTTPS Proxy Issues - 18 tests

| # | Test | Issue |
|---|------|-------|
| 18 | Proxy to HTTPS | Wrong status code (502 vs 400) |
| 19 | Host header update | Not updating host header |
| 20 | Invalid domains | Wrong status code (502 vs 500) |
| 21 | CONNECT method | Not implemented |
| 22 | Predicate generators | Not working |
| 23-24 | ProxyAlways mode | Broken |
| 25 | Match object graphs | JSON key ordering issue |
| 26 | Persist behaviors | shellTransform dependency |
| 27 | Latency persistence | TypeError accessing undefined |
| 28 | Replayable JSON | Wrong predicate type (equals vs deepEquals) |
| 29 | Binary data via encoding | Proxy error reading response |
| 30 | Decorated proxy | Timeout |
| 31 | Inject headers | Not implemented |
| 32 | Decorate new response | Timeout |
| 33 | DELETE savedRequests | 404 error |
| 34 | Query string (issue #410) | Adds = incorrectly |
| 35 | JSON body formatting | Spacing mismatch |

#### HTTP/HTTPS Stub Issues - 8 tests

| # | Test | Issue |
|---|------|-------|
| 36, 40 | JSON body predicates (HTTP/HTTPS) | Not matching |
| 37, 41 | XPath array predicates (HTTP/HTTPS) | Not matching |
| 38, 42 | Gzip decompression (HTTP/HTTPS) | Not implemented |
| 39, 43 | Stub validation (HTTP/HTTPS) | Not returning 400 |

#### TCP Implementation Gaps - 13 tests

| # | Test | Issue |
|---|------|-------|
| 49 | Decorate behavior | Not working |
| 50 | Copy behavior | Token substitution failing |
| 51 | Access requests | Requests array undefined |
| 52 | Default mode | Missing mode, requests, stubs fields |
| 55-56 | endOfRequestResolver | Requests array undefined |
| 57 | Proxy endOfRequestResolver | Timeout |
| 58-59 | DNS/connection errors | Timeout |
| 60 | Protocol validation | Timeout |
| 61 | Matches validation | Not returning 400 |
| 62 | Old proxy syntax | Parse error |
| 63 | Invalid hosts | Timeout |
| 64 | Packet splitting | Requests array undefined |

#### Other Issues - 9 tests

| # | Test | Issue |
|---|------|-------|
| 9-10 | Undefined fault (HTTP/HTTPS) | Should return normal response, not hang up |
| 11-12 | DELETE savedRequests (HTTP/HTTPS) | Not returning requests array |
| 17 | Metrics format | Prometheus vs mountebank format |
| 46-47 | DELETE all imposters | Missing stubs and requests fields |
| 48 | SMTP requests | Requests array undefined |

## Analysis by Priority

### P0 - Critical (0 tests)
None - all critical functionality is working

### P1 - High Priority (18 tests)
- HTTP/HTTPS Proxy issues - Major feature for testing scenarios

### P2 - Medium Priority (21 tests)
- TCP implementation gaps (13 tests) - Important for TCP protocol support
- HTTP/HTTPS Stub issues (8 tests) - Advanced predicate features

### P3 - Low Priority (9 tests)
- Fault handling, metrics format, API response formats

## Files Needing Modification

### High Priority
- `internal/imposter/proxy.go` - HTTP proxy issues
- `internal/imposter/tcp_server.go` - TCP behaviors and proxy

### Medium Priority
- `internal/imposter/matcher.go` - XPath arrays, gzip decompression, validation
- `internal/imposter/behaviors.go` - TCP copy/decorate behaviors

### Low Priority
- `internal/imposter/manager.go` - Undefined fault handling, API responses
- `internal/api/handlers/*.go` - DELETE endpoints, metrics format

## Security Posture

go-tartuffe has made **intentional security improvements** over mountebank:

1. **shellTransform disabled** - Prevents arbitrary command execution
2. **Process object blocked** - Prevents environment variable leakage
3. **Private keys not returned** - Prevents key exposure in API responses

These 16 security blocks improve security at the cost of 6.3% compatibility.

**Adjusted compatibility excluding security blocks: 79.7%**

## Recommendations

1. **Focus on HTTP Proxy** (18 tests) - Most impactful for users
2. **Fix TCP basics** (requests arrays, mode field) - Quick wins
3. **Consider security trade-offs** - Current posture is more secure
4. **Metrics format** - Prometheus format is industry standard, acceptable deviation

## Conclusion

go-tartuffe has achieved **74.6% raw compatibility** and **79.7% adjusted compatibility** with mountebank while maintaining better security posture. The remaining gaps are primarily in:
- HTTP proxy advanced features (ProxyAlways, predicate generators)
- TCP protocol completeness (behaviors, proxy)
- Advanced predicate features (XPath arrays, gzip)

Core functionality is solid with 188/252 tests passing.
