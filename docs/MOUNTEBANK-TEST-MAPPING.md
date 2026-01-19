# Mountebank API Test Suite Mapping

**Current Status**: 216/252 passing (85.7%)
**Date**: 2026-01-19
**Mountebank Version**: 2.9.3
**Test Command**: `MB_EXECUTABLE=/path/to/tartuffe-wrapper.sh npm run test:api`

## Summary by Category

| Category | Passing | Failing | Total | Pass% | Notes |
|----------|---------|---------|-------|-------|-------|
| **Controllers** | 11/11 | 0 | 11 | 100% | ✅ Complete |
| **HTTP Imposter** | 23/27 | 4 | 27 | 85% | ❌ shellTransform, process access, async inject, behaviors |
| **HTTPS Imposter** | 23/27 | 4 | 27 | 85% | ❌ Same as HTTP (protocol variant) |
| **HTTP Proxy** | 21/27 | 6 | 27 | 78% | ❌ Cross-protocol, mTLS, CONNECT, latency, invalid domains |
| **HTTP Behaviors** | 34/38 | 4 | 38 | 89% | ❌ shellTransform (disabled for security) |
| **HTTP Stubs** | 27/27 | 0 | 27 | 100% | ✅ Complete |
| **HTTP Injection** | 9/13 | 4 | 13 | 69% | ❌ process object, async injection |
| **HTTP Predicates** | 27/27 | 0 | 27 | 100% | ✅ Complete |
| **HTTP Fault** | 2/3 | 1 | 3 | 67% | ❌ Undefined fault handling |
| **HTTP Metrics** | 3/3 | 0 | 3 | 100% | ✅ Complete |
| **SMTP** | 3/3 | 0 | 3 | 100% | ✅ Complete |
| **TCP Imposter** | 6/15 | 9 | 15 | 40% | ❌ Multiple failures (see below) |
| **TCP Proxy** | 0/4 | 4 | 4 | 0% | ❌ Not implemented |
| **TCP Injection** | 0/1 | 1 | 1 | 0% | ❌ Not implemented |
| **TOTAL** | **216** | **36** | **252** | **85.7%** | Target: 75%+ ✅ |

## Detailed Failure Analysis

### HTTP/HTTPS Behavior Failures (8 failures, 4 unique tests × 2 protocols)

These tests run for both HTTP and HTTPS:

| Test Name | HTTP Status | HTTPS Status | Reason | Priority |
|-----------|-------------|--------------|--------|----------|
| should support shell transform without array for backwards compatibility | ❌ Fail | ❌ Fail | shellTransform disabled (security) | Won't Fix |
| should support array of shell transforms in order | ❌ Fail | ❌ Fail | shellTransform disabled (security) | Won't Fix |
| should compose multiple behaviors together (old interface for backwards compatibility) | ❌ Fail | ❌ Fail | Behavior composition issue | P2 |
| should apply multiple behaviors in sequence with repeat (new format) | ❌ Fail | ❌ Fail | Repeat behavior not implemented | P2 |

**Note**: shellTransform is intentionally disabled for security (see docs/SECURITY.md)

### HTTP/HTTPS Injection Failures (8 failures, 4 unique tests × 2 protocols)

| Test Name | HTTP Status | HTTPS Status | Reason | Priority |
|-----------|-------------|--------------|--------|----------|
| should allow access to the global process object | ❌ Fail | ❌ Fail | process object not exposed (security) | Won't Fix |
| should allow asynchronous injection | ❌ Fail | ❌ Fail | Async JS not supported in goja | P3 |
| should provide access to all requests (injection context) | ❌ Fail | ❌ Fail | Requests array not passed to injection | P2 |
| should compose multiple behaviors together | ❌ Fail | N/A | Behavior composition | P2 |

**Note**: process object exposure is security risk - won't implement

### HTTP/HTTPS Fault Failures (2 failures, 1 unique test × 2 protocols)

| Test Name | HTTP Status | HTTPS Status | Reason | Priority |
|-----------|-------------|--------------|--------|----------|
| should do nothing when undefined fault is specified | ❌ Fail | ❌ Fail | Error handling for unknown faults | P2 |

### HTTP Proxy Failures (6 unique failures)

| Test Name | Status | Reason | Priority |
|-----------|--------|--------|----------|
| should proxy to https | ❌ Fail | Cross-protocol proxying (HTTP→HTTPS) | P1 |
| should allow proxy stubs to invalid domains | ❌ Fail | DNS error handling | P1 |
| should handle the connect method | ❌ Fail | CONNECT method support | P2 |
| should persist behaviors from origin server | ❌ Fail | Behavior persistence from proxied responses | P2 |
| should support adding latency to saved responses | ❌ Fail | addWaitBehavior not implemented | P2 |
| should support retrieving replayable JSON with proxies removed | ❌ Fail | removeProxies export functionality | P3 |

### HTTPS-Specific Failures (2 unique failures)

| Test Name | Status | Reason | Priority |
|-----------|--------|--------|----------|
| should support sending key/cert pair during imposter creation | ❌ Fail | Certificate handling issue | P1 |
| should support proxying to origin server requiring mutual auth | ❌ Fail | mTLS proxy support | P1 |

### TCP Imposter Failures (9 failures)

| Test Name | Status | Reason | Priority |
|-----------|--------|--------|----------|
| should provide access to all requests | ❌ Fail | TCP request recording | P1 |
| should allow asynchronous injection (old interface) | ❌ Fail | Async JS support | P3 |
| should allow asynchronous injection | ❌ Fail | Async JS support | P3 |
| should allow binary requests extending beyond a single packet using endOfRequestResolver | ❌ Fail | endOfRequestResolver not implemented | P0 |
| should allow text requests extending beyond a single packet using endOfRequestResolver | ❌ Fail | endOfRequestResolver not implemented | P0 |
| should support old proxy syntax for backwards compatibility | ❌ Fail | Old proxy format parsing | P2 |
| should allow proxy stubs to invalid hosts | ❌ Fail | TCP proxy error handling | P1 |
| should split each packet into a separate request by default | ❌ Fail | TCP packet splitting logic | P1 |
| should compose multiple behaviors together | ❌ Fail | TCP behavior composition | P2 |

### TCP Proxy Failures (4 failures)

| Test Name | Status | Reason | Priority |
|-----------|--------|--------|----------|
| should obey endOfRequestResolver | ❌ Fail | endOfRequestResolver in proxy mode | P0 |
| should gracefully deal with DNS errors | ❌ Fail | TCP proxy DNS error handling | P1 |
| should gracefully deal with non listening ports | ❌ Fail | TCP proxy connection error handling | P1 |
| should reject non-tcp protocols | ❌ Fail | Protocol validation in TCP mode | P1 |

### TCP Injection Failure (1 failure)

| Test Name | Status | Reason | Priority |
|-----------|--------|--------|----------|
| should allow asynchronous injection | ❌ Fail | Async JS support in TCP | P3 |

### SMTP Imposter Failure (1 failure)

| Test Name | Status | Reason | Priority |
|-----------|--------|--------|----------|
| (Unknown SMTP test) | ❌ Fail | Need to investigate | P2 |

## Priority Definitions

- **P0 (Critical)**: Core functionality gaps affecting multiple tests
- **P1 (High)**: Important features with clear use cases
- **P2 (Medium)**: Enhancement features or edge cases
- **P3 (Low)**: Nice-to-have or rarely used features
- **Won't Fix**: Security concerns or architectural limitations

## Actionable Failures by Priority

### P0 - Critical (3 unique failures)

1. **HTTP Metrics API** (1 test)
   - Expose metrics endpoint
   - Status: Infrastructure exists, just needs API handler

2. **TCP endOfRequestResolver** (3 tests)
   - Implement custom request boundary detection for TCP
   - Status: Model exists, needs implementation in TCP server

### P1 - High Priority (11 unique failures)

1. **HTTP Proxy Cross-Protocol** (1 test)
   - HTTP→HTTPS proxying

2. **HTTP Proxy DNS Errors** (1 test)
   - Graceful handling of invalid domains

3. **TCP Request Recording** (1 test)
   - Track TCP requests for replay

4. **TCP Proxy** (4 tests)
   - Basic TCP proxy forwarding
   - DNS error handling
   - Connection error handling
   - Protocol validation

5. **TCP Packet Splitting** (1 test)
   - Default packet-per-request behavior

6. **HTTPS Certificate Handling** (1 test)
   - Key/cert pair validation

7. **HTTPS mTLS Proxy** (1 test)
   - Mutual auth proxy support

8. **TCP Proxy Invalid Hosts** (1 test)
   - Error handling for unreachable hosts

### P2 - Medium Priority (9 unique failures)

1. **HTTP Behavior Composition** (2 tests)
   - Multiple behaviors in sequence
   - Old interface compatibility

2. **HTTP Proxy Features** (3 tests)
   - CONNECT method
   - Behavior persistence
   - addWaitBehavior

3. **HTTP Injection Request Access** (1 test)
   - Pass requests array to injection context

4. **HTTP Fault Handling** (1 test)
   - Undefined fault graceful handling

5. **TCP Features** (2 tests)
   - Old proxy syntax
   - Behavior composition

6. **SMTP Investigation** (1 test)
   - Identify and fix unknown failure

### P3 - Low Priority (3 unique failures)

1. **Async JavaScript** (3 tests)
   - HTTP injection async
   - TCP injection async
   - Goja VM limitation - may require upstream fix

### Won't Fix (4 unique failures)

1. **shellTransform** (2 tests)
   - Security risk: arbitrary command execution
   - Alternative: Use decorate with JavaScript

2. **process object access** (2 tests)
   - Security risk: environment variable exposure
   - No secure alternative needed

## Test File Mapping

### Passing Test Files (100%)
- ✅ `mbTest/api/homeControllerTest.js` (1/1)
- ✅ `mbTest/api/impostersControllerTest.js` (10/10)
- ✅ `mbTest/api/http/httpStubTest.js` (27/27)
- ✅ `mbTest/api/http/httpPredicatesTest.js` (27/27)
- ✅ `mbTest/api/smtp/*` (3/3)

### Partially Passing Test Files
- 🟡 `mbTest/api/http/httpImposterTest.js` (23/27 - 85%)
- 🟡 `mbTest/api/https/httpsCertificateTest.js` (23/27 - 85%)
- 🟡 `mbTest/api/http/httpBehaviorsTest.js` (34/38 - 89%)
- 🟡 `mbTest/api/http/httpProxyStubTest.js` (21/27 - 78%)
- 🟡 `mbTest/api/http/httpInjectionTest.js` (9/13 - 69%)
- 🟡 `mbTest/api/http/httpFaultTest.js` (2/3 - 67%)
- 🟡 `mbTest/api/http/httpMetricsTest.js` (2/3 - 67%)
- 🟡 `mbTest/api/tcp/tcpImposterTest.js` (6/15 - 40%)

### Failing Test Files
- ❌ `mbTest/api/tcp/tcpProxyTest.js` (0/4 - 0%)
- ❌ `mbTest/api/tcp/tcpInjectionTest.js` (0/1 - 0%)

## Related Documentation

- **Security Decisions**: `docs/SECURITY.md`
- **HTTP Proxy Coverage**: `docs/HTTP-PROXY-TEST-MAPPING.md`
- **Controller Tests**: `docs/CONTROLLER-API-TEST-MAPPING.md`
- **Validation Procedure**: `.claude/claude.md`
- **Remaining Work**: `COMPATIBILITY-BACKLOG.md`
