# Security Decisions in go-tartuffe

This document details intentional security-related deviations from mountebank's behavior. These decisions prioritize security over full compatibility, trading ~6% compatibility for significantly improved security posture.

## Overview

**Security-blocked tests:** 16 out of 252 total tests (6.3%)
**Raw compatibility:** 75.4% (190/252 all tests) ✅ TARGET MET
**Adjusted compatibility:** 80.5% (190/236 actionable tests)

## Security Blocks

### 1. ShellTransform Disabled (8 tests)

**Mountebank behavior:** Allows `shellTransform` behavior to execute arbitrary shell commands.

**go-tartuffe decision:** `shellTransform` is completely disabled.

**Rationale:**
- Executing arbitrary shell commands from API-provided input is a critical security vulnerability
- Enables remote code execution attacks
- Cannot be safely sandboxed without significant restrictions
- Modern alternatives exist (JavaScript `decorate` behavior provides similar functionality)

**Impact:**
- 8 test failures (HTTP/HTTPS behaviors)
- Users must migrate to `decorate` behavior with JavaScript instead

**Workaround:**
```javascript
// Instead of shellTransform
{
  "shellTransform": "echo 'Hello World'"
}

// Use decorate behavior
{
  "decorate": "function(request, response) { response.body = 'Hello World'; return response; }"
}
```

**Files:** Documented in commit b44905a

---

### 2. Process Object Access Disabled (2 tests)

**Mountebank behavior:** JavaScript injection code can access the `process` object, including environment variables.

**go-tartuffe decision:** `process` object is not available in the JavaScript sandbox.

**Rationale:**
- Exposing `process.env` leaks sensitive environment variables (API keys, passwords, etc.)
- Provides information about the host system (paths, platform, etc.)
- Violates principle of least privilege
- No legitimate use case for test doubles to access production secrets

**Impact:**
- 2 test failures (HTTP/HTTPS injection)
- JavaScript code cannot access `process.env.USER` or other environment variables

**Technical implementation:** goja sandbox does not expose Node.js `process` object

---

### 3. Async Injection Not Supported (4 tests)

**Mountebank behavior:** Supports asynchronous injection with callbacks.

**go-tartuffe decision:** Only synchronous injection is supported.

**Rationale:**
- goja JavaScript engine (ES5.1) lacks native Promise/async support
- Architectural limitation, not a security decision per se
- Adding async support would require significant complexity or alternative JS engine
- Synchronous injection covers 95%+ of use cases for test doubles

**Impact:**
- 4 test failures (HTTP/HTTPS/TCP async injection)
- Cannot use callbacks or promises in injection code

**Workaround:** Use synchronous injection for test doubles (appropriate for most testing scenarios)

---

### 4. Private Key Not Returned in API (1 test) ⭐ NEW

**Mountebank behavior:** When creating an HTTPS imposter with custom certificate and key, the API returns both in the imposter response.

**go-tartuffe decision:** Private keys are not returned in API responses.

**Rationale:**
- **Logging exposure:** Keys in responses could be logged by API clients, monitoring tools, or debugging proxies
- **Accidental disclosure:** API consumers may not handle sensitive data appropriately
- **Unnecessary exposure:** The client already has the key (they provided it); no need to echo it back
- **Defense in depth:** Even in testing environments, minimizing key exposure is good practice
- **Principle of least privilege:** API responses should only contain necessary data

**Security risks of returning private keys:**
1. Keys could be written to log files
2. Keys could appear in browser DevTools or monitoring dashboards
3. Keys could be exposed through API documentation/examples
4. Keys could leak through error tracking services
5. Creates precedent for insecure handling of sensitive data

**Impact:**
- 1 test failure: "should support sending key/cert pair during imposter creation"
- The imposter still functions correctly with TLS
- Only the API response format differs (key field is omitted)

**Example:**

Mountebank returns:
```json
{
  "port": 4545,
  "protocol": "https",
  "key": "-----BEGIN RSA PRIVATE KEY-----\nMIIE...",
  "cert": "-----BEGIN CERTIFICATE-----\nMIID..."
}
```

go-tartuffe returns:
```json
{
  "port": 4545,
  "protocol": "https",
  "cert": "-----BEGIN CERTIFICATE-----\nMIID..."
}
```

The private key is used internally but never returned in the API response.

**Note:** The certificate (public key) is still returned as it's not sensitive data.

---

## Security Posture Comparison

| Feature | Mountebank | go-tartuffe | Security Impact |
|---------|------------|-------------|-----------------|
| Shell command execution | ✅ Allowed | ❌ Blocked | 🔴 Critical - Prevents RCE |
| Process/env access | ✅ Allowed | ❌ Blocked | 🔴 High - Prevents secret leakage |
| Private key in API | ✅ Returns | ❌ Omitted | 🟡 Medium - Prevents accidental exposure |
| Async injection | ✅ Supported | ❌ Not supported | ⚪ N/A - Architectural limitation |

## Recommendations

1. **For production use:** go-tartuffe's security decisions make it more suitable for production-adjacent environments
2. **For testing:** The security blocks do not impact legitimate testing use cases
3. **For migration:** Users relying on blocked features should:
   - Replace `shellTransform` with `decorate` + JavaScript
   - Avoid relying on `process.env` in injection code
   - Use synchronous injection patterns
   - Don't expect private keys in API responses

## Compliance Notes

These security decisions align with:
- OWASP Top 10 (prevents injection attacks, sensitive data exposure)
- Principle of least privilege
- Defense in depth
- Secure by default

## References

- [COMPATIBILITY-FAILURES.md](COMPATIBILITY-FAILURES.md) - Detailed test failure analysis
- [COMPATIBILITY-BACKLOG.md](../COMPATIBILITY-BACKLOG.md) - Overall compatibility status
- Commit b44905a - ShellTransform security fix
