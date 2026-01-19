# Mountebank Compatibility Test Failures

**Validation Date:** 2026-01-17
**Test Results:** 173 passing, 79 failing (252 total)
**Compatibility:** 68.7% raw | ~73% adjusted (excluding security/architectural blocks)

## Test Summary

| Category | Passing | Failing | Total | Pass Rate |
|----------|---------|---------|-------|-----------|
| HTTP Behaviors | 22 | 8 | 30 | 73.3% |
| HTTPS Behaviors | 22 | 8 | 30 | 73.3% |
| HTTP Injection | 6 | 6 | 12 | 50.0% |
| HTTPS Injection | 6 | 6 | 12 | 50.0% |
| HTTP Proxy | 8 | 18 | 26 | 30.8% |
| HTTP Stubs | 23 | 7 | 30 | 76.7% |
| HTTPS Stubs | 23 | 7 | 30 | 76.7% |
| HTTPS Certs | 2 | 2 | 4 | 50.0% |
| Controller | 6 | 2 | 8 | 75.0% |
| Metrics | 2 | 1 | 3 | 66.7% |
| TCP Behaviors | 0 | 2 | 2 | 0.0% |
| TCP Imposter | 6 | 3 | 9 | 66.7% |
| TCP Injection | 4 | 5 | 9 | 44.4% |
| TCP Proxy | 2 | 4 | 6 | 33.3% |
| TCP Stubs | 6 | 4 | 10 | 60.0% |
| SMTP | 3 | 1 | 4 | 75.0% |

## Detailed Failure Analysis

### Category 1: ShellTransform (Expected - Security Block)
**Status:** Won't Fix - Intentionally disabled for security
**Files:** `mbTest/api/http/httpBehaviorsTest.js`

| # | Test Name | Line | Error | Root Cause |
|---|-----------|------|-------|------------|
| 1 | should support shell transform without array for backwards compatibility | 362 | `shellTransform behavior is not supported` | Security: arbitrary command execution |
| 2 | should support array of shell transforms in order | 397 | `shellTransform behavior is not supported` | Security: arbitrary command execution |
| 3 | should compose multiple behaviors together (old interface for backwards compatibility) | 658 | `shellTransform behavior is not supported` | Old interface uses shellTransform |
| 4 | should apply multiple behaviors in sequence with repeat (new format) | 749 | `shellTransform behavior is not supported` | Sequence includes shellTransform |
| 5 | (HTTPS) should support shell transform without array | 362 | `shellTransform behavior is not supported` | Same as #1 |
| 6 | (HTTPS) should support array of shell transforms in order | 397 | `shellTransform behavior is not supported` | Same as #2 |
| 7 | (HTTPS) should compose multiple behaviors together | 658 | `shellTransform behavior is not supported` | Same as #3 |
| 8 | (HTTPS) should apply multiple behaviors in sequence with repeat | 749 | `shellTransform behavior is not supported` | Same as #4 |

**Workaround:** Use `decorate` behavior with JavaScript instead.

---

### Category 2: Fault Handling (Partial Implementation)
**Status:** Needs Fix
**Files:** `mbTest/api/http/httpFaultTest.js`

| # | Test Name | Line | Error | Root Cause |
|---|-----------|------|-------|------------|
| 9 | should do nothing when undefined fault is specified | N/A | `socket hang up` | Undefined fault should return normal response |
| 10 | (HTTPS) should do nothing when undefined fault is specified | N/A | `socket hang up` | Same as #9 |

**Fix Required:** When fault is undefined/unknown, don't close connection - return normal response.
**File to modify:** `internal/imposter/http_server.go` - fault handling logic

---

### Category 3: Request Deletion API
**Status:** Needs Fix
**Files:** `mbTest/api/http/httpImposterTest.js`

| # | Test Name | Line | Error | Root Cause |
|---|-----------|------|-------|------------|
| 11 | should return the imposter post requests-deletion | 236 | `undefined should equal []` | DELETE /imposters/:id/savedRequests not returning imposter |
| 12 | (HTTPS) should return the imposter post requests-deletion | 236 | Same | Same |

**Fix Required:** `DELETE /imposters/:id/savedRequests` should return imposter with empty requests array.
**File to modify:** `internal/api/handlers/imposter.go`

---

### Category 4: JavaScript Injection (Critical - Multiple Issues)
**Status:** Needs Fix - P0 Priority
**Files:** `mbTest/api/http/httpInjectionTest.js`

| # | Test Name | Line | Error | Root Cause |
|---|-----------|------|-------|------------|
| 13 | should allow javascript predicate for matching (old interface) | 42 | `'' !== 'MATCHED'` | Old interface predicate not evaluating correctly |
| 14 | should allow synchronous javascript injection for responses (old interface) | 113 | `'undefined INJECTED' !== 'GET INJECTED'` | `request.method` undefined in old interface |
| 15 | should allow synchronous javascript injection for responses | 128 | `undefined !== 'close'` | Response headers not being set |
| 16 | should share state with predicate and response injection (old interface) | N/A | `Cannot read property 'calls' of undefined` | imposterState not passed to old interface |
| 17 | should allow access to the global process object | N/A | `process is not defined` | **Expected - Security block** |
| 18 | should allow asynchronous injection | 299 | `500 !== true` | **Expected - ES5.1 lacks async** |
| 19-24 | (HTTPS versions of 13-18) | Same | Same | Same issues |

**Analysis:**
- #13, 14, 16: Old interface (`request, logger`, `request, state, logger`) needs fixes
- #15: Response injection needs to set headers properly
- #17: **Won't Fix** - Security sandbox prevents process access
- #18: **Won't Fix** - goja ES5.1 lacks Promise/async support

**Files to modify:** `internal/imposter/inject.go`, `internal/imposter/matcher.go`

---

### Category 5: Metrics Endpoint (Format Mismatch)
**Status:** Needs Fix
**Files:** `mbTest/api/http/metricsTest.js`

| # | Test Name | Line | Error | Root Cause |
|---|-----------|------|-------|------------|
| 25 | should return imposter metrics after imposters calls | N/A | Regex mismatch | Prometheus format doesn't have `mb_predicate_match_duration_seconds` |

**Fix Required:** Add mountebank-compatible metrics or update test expectations.
**Note:** go-tartuffe uses Prometheus format; mountebank expects custom format.
**File to modify:** `internal/metrics/metrics.go`

---

### Category 6: HTTP Proxy (Significant Gaps)
**Status:** Needs Fix - High Priority
**Files:** `mbTest/api/http/httpProxyStubTest.js`

| # | Test Name | Line | Error | Root Cause |
|---|-----------|------|-------|------------|
| 26 | should proxy to https | 81 | `502 !== 400` | HTTPS proxy returning wrong status |
| 27 | should update the host header to the origin server | 109 | `200 !== 400` | Host header not updated |
| 28 | should allow proxy stubs to invalid domains | 121 | `502 !== 500` | Wrong error status for DNS failure |
| 29 | should handle the connect method | N/A | `Bad response: 401` | CONNECT method not implemented |
| 30 | should allow programmatic creation of predicates | 289 | `undefined should equal [...]` | Predicate generators not working |
| 31 | should record new stubs with multiple responses behind proxy resolver in proxyAlways mode | 328 | `Cannot read 'body'` | ProxyAlways mode broken |
| 32 | should capture responses together in proxyAlways mode even with complex predicateGenerators | 371 | `Cannot read 'body'` | Same as #31 |
| 33 | should match entire object graphs | N/A | `socket hang up` | deepEquals in proxy context failing |
| 34 | should persist behaviors from origin server | 488 | `shellTransform not supported` | **Expected** - uses shellTransform |
| 35 | should support adding latency to saved responses based on how long the origin server took to respond | 522 | `Cannot read '0'` | Wait behavior from proxy broken |
| 36 | should support retrieving replayable JSON with proxies removed for later playback | N/A | Deep equal mismatch | Replayable format incorrect |
| 37 | should support returning binary data from origin server based on content encoding | N/A | Array mismatch | Binary data proxy broken |
| 38 | should persist decorated proxy responses and only run decorator once | N/A | Timeout | Decorated proxy hanging |
| 39 | should inject proxy headers if specified | 728 | `undefined == 'http://www.google.com'` | InjectHeaders not implemented |
| 40 | should add decorate behaviors to newly created response | N/A | Timeout | Decorate on proxy hanging |
| 41 | DELETE /imposters/:id/requests should delete proxy stubs but not other stubs | 839 | `404 !== 200` | Delete endpoint not implemented |
| 42 | should not add = at end of of query key missing = in original request (issue #410) | 871 | `/path?WSDL= !== /path?WSDL` | Query string handling bug |
| 43 | should save JSON bodies as JSON instead of text (issue #656) | 901 | Object vs string | JSON body serialization |

**Files to modify:** `internal/imposter/proxy.go`, `internal/imposter/http_server.go`

---

### Category 7: HTTP Stubs (JSON Predicate Issues)
**Status:** Needs Fix - Medium Priority
**Files:** `mbTest/api/http/httpStubTest.js`

| # | Test Name | Line | Error | Root Cause |
|---|-----------|------|-------|------------|
| 44 | should correctly handle deepEquals object predicates | 154 | `'' !== 'second stub'` | deepEquals with objects failing |
| 45 | should support JSON bodies | 201 | `"[object Object]" is not valid JSON` | JSON body not being stringified |
| 46 | should support treating the body as a JSON object for predicate matching | 228 | `'' !== 'SUCCESS'` | JSON body predicate not matching |
| 47 | should handle JSON null values | 339 | `"[object Object]" is not valid JSON` | Same as #45 |
| 48 | should support array predicates with xpath | 369 | `'' !== 'SUCCESS'` | XPath array predicate failing |
| 49 | should support predicate from gzipped request (issue #499) | 446 | `'' !== 'SUCCESS'` | Gzip decompression not implemented |
| 50 | should provide a good error message when adding stub with missing information | 589 | `200 !== 400` | Validation not returning error |
| 51-57 | (HTTPS versions of 44-50) | Same | Same | Same issues |

**Files to modify:** `internal/imposter/matcher.go`, `internal/imposter/http_server.go`

---

### Category 8: HTTPS Certificates
**Status:** Partial - #58 Won't Fix (Security), #59 Needs Fix
**Files:** `mbTest/api/https/httpsCertificateTest.js`

| # | Test Name | Line | Error | Root Cause |
|---|-----------|------|-------|------------|
| 58 | should support sending key/cert pair during imposter creation | N/A | `undefined !== '-----BEGIN RSA...'` | **Won't Fix** - Private keys should not be returned in API responses (security) |
| 59 | should support proxying to origin server requiring mutual auth | 87 | TLS verification failed | mTLS proxy not configured correctly |

**Rationale for #58:**
Returning private keys in API responses poses a significant security risk. While mountebank returns the certificate and key in the imposter response for convenience, this violates the principle of least privilege and creates potential exposure vectors:
- Keys could be logged inadvertently
- Keys could be exposed through monitoring/debugging tools
- Keys could leak through API consumers that don't handle sensitive data properly

go-tartuffe deliberately omits private keys from API responses as a security hardening measure. The imposter still functions correctly with the provided certificate; it just doesn't echo the sensitive key material back to the client.

**Files to modify (for #59):** `internal/imposter/proxy.go`

---

### Category 9: Controller/Delete All
**Status:** Needs Fix
**Files:** `mbTest/api/controllers/imposterControllerTest.js`

| # | Test Name | Line | Error | Root Cause |
|---|-----------|------|-------|------------|
| 60 | deletes all imposters and returns replayable body | N/A | Response structure mismatch | Replayable format incorrect |
| 61 | supports returning a non-replayable body with proxies removed | N/A | Response structure mismatch | Non-replayable format incorrect |

**Files to modify:** `internal/api/handlers/imposter.go`

---

### Category 10: SMTP
**Status:** Needs Fix
**Files:** `mbTest/api/smtp/smtpImposterTest.js`

| # | Test Name | Line | Error | Root Cause |
|---|-----------|------|-------|------------|
| 62 | should provide access to all requests | 52 | `Cannot read 'forEach' of undefined` | Requests array not being returned |

**Files to modify:** `internal/imposter/smtp_server.go`

---

### Category 11: TCP Behaviors
**Status:** Needs Fix
**Files:** `mbTest/api/tcp/tcpBehaviorsTest.js`

| # | Test Name | Line | Error | Root Cause |
|---|-----------|------|-------|------------|
| 63 | should support decorating response from origin server | 41 | `'ORIGIN' !== 'ORIGIN DECORATED'` | TCP proxy decorate not working |
| 64 | should compose multiple behaviors together | 81 | Token substitution failed | Copy behavior not working in TCP |

**Files to modify:** `internal/imposter/tcp_server.go`, `internal/imposter/behaviors.go`

---

### Category 12: TCP Imposter
**Status:** Needs Fix
**Files:** `mbTest/api/tcp/tcpImposterTest.js`

| # | Test Name | Line | Error | Root Cause |
|---|-----------|------|-------|------------|
| 65 | should provide access to all requests | 35 | `Cannot read 'map' of undefined` | Requests array not being returned |
| 66 | should reflect default mode | N/A | Missing `mode` field | Mode not included in response |

**Files to modify:** `internal/imposter/tcp_server.go`, `internal/api/handlers/imposter.go`

---

### Category 13: TCP Injection
**Status:** Needs Fix
**Files:** `mbTest/api/tcp/tcpInjectionTest.js`

| # | Test Name | Line | Error | Root Cause |
|---|-----------|------|-------|------------|
| 67 | should allow synchronous javascript injection for responses (old interface) | 51 | `'undefined INJECTED'` | `request.data` undefined in old interface |
| 68 | should allow asynchronous injection (old interface) | N/A | Timeout | **Expected** - ES5.1 lacks async |
| 69 | should allow asynchronous injection | N/A | Timeout | **Expected** - ES5.1 lacks async |
| 70 | should allow binary requests extending beyond a single packet using endOfRequestResolver | 194 | `Cannot read 'length'` | endOfRequestResolver broken |
| 71 | should allow text requests extending beyond a single packet using endOfRequestResolver | 222 | `Cannot read 'length'` | Same as #70 |

**Files to modify:** `internal/imposter/inject.go`, `internal/imposter/tcp_server.go`

---

### Category 14: TCP Proxy
**Status:** Needs Fix
**Files:** `mbTest/api/tcp/tcpProxyTest.js`

| # | Test Name | Line | Error | Root Cause |
|---|-----------|------|-------|------------|
| 72 | should obey endOfRequestResolver | N/A | Timeout | endOfRequestResolver not working |
| 73 | should gracefully deal with DNS errors | N/A | Timeout | DNS error handling missing |
| 74 | should gracefully deal with non listening ports | N/A | Timeout | Connection error handling missing |
| 75 | should reject non-tcp protocols | N/A | Timeout | Protocol validation missing |

**Files to modify:** `internal/imposter/tcp_server.go`

---

### Category 15: TCP Stubs
**Status:** Needs Fix
**Files:** `mbTest/api/tcp/tcpStubTest.js`

| # | Test Name | Line | Error | Root Cause |
|---|-----------|------|-------|------------|
| 76 | should return 400 if uses matches predicate with binary mode | 92 | `201 !== 400` | Validation not rejecting matches+binary |
| 77 | should support old proxy syntax for backwards compatibility | 117 | `400 !== 201` | Old proxy syntax not parsed |
| 78 | should allow proxy stubs to invalid hosts | N/A | Timeout | Error handling for invalid hosts |
| 79 | should split each packet into a separate request by default | 171 | `Cannot read 'reduce'` | Packet splitting broken |

**Files to modify:** `internal/imposter/tcp_server.go`

---

## Priority Fix Order

### P0 - Critical (Blocking Core Functionality)
1. **JavaScript Injection** (#13-16, 19-22) - Old interface broken
2. **JSON Body Predicates** (#44-47, 51-54) - Common use case
3. **Request Access** (#62, 65) - SMTP/TCP requests array

### P1 - High Priority
4. **HTTP Proxy** (#26-32, 36-43) - Significant feature gaps
5. **TCP Behaviors** (#63-64) - Core TCP functionality
6. **TCP Injection** (#67, 70-71) - Old interface and endOfRequestResolver

### P2 - Medium Priority
7. **Fault Handling** (#9-10) - Undefined fault behavior
8. **Request Deletion** (#11-12) - API completeness
9. **HTTPS mTLS Proxy** (#59) - Mutual TLS proxy support
10. **TCP Proxy** (#72-75) - Error handling

### Won't Fix (Security/Architectural)
- **ShellTransform** (#1-8, 34) - Security risk (arbitrary command execution)
- **Process Object** (#17, 23) - Security sandbox (prevents environment variable access)
- **Async Injection** (#18, 24, 68, 69) - ES5.1 limitation (goja lacks Promise/async)
- **Private Key Return** (#58) - Security hardening (keys shouldn't be returned in API responses)

## Files Needing Modification (by priority)

| Priority | File | Issues |
|----------|------|--------|
| P0 | `internal/imposter/inject.go` | #13-16, 19-22, 67 |
| P0 | `internal/imposter/matcher.go` | #44-47, 51-54 |
| P1 | `internal/imposter/proxy.go` | #26-32, 36-43 |
| P1 | `internal/imposter/tcp_server.go` | #63-79 |
| P1 | `internal/imposter/behaviors.go` | #63-64 |
| P2 | `internal/imposter/http_server.go` | #9-10, 45, 47, 49 |
| P2 | `internal/api/handlers/imposter.go` | #11-12, 60-61 |
| P2 | `internal/imposter/smtp_server.go` | #62 |
| P2 | `internal/imposter/https_server.go` | #58-59 |
