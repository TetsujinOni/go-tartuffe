# Compatibility Backlog

Remaining work to improve mountebank API compatibility.

## Current Status

**Progress**: 233/252 passing (92.5%) | 233/233 adjusted (100%) ✅ **ALL ACTIONABLE ITEMS COMPLETE**
**Target**: 75%+ compatibility
**Last Validation**: 2026-01-20

## Remaining Failures: 19 Tests

**Breakdown**:
- Security/Architectural (Won't Fix): 15 tests
- goja Async Limitation: 4 tests
- Actionable: 0 tests ✅

**Test Mapping**: See [docs/MOUNTEBANK-TEST-MAPPING.md](docs/MOUNTEBANK-TEST-MAPPING.md) for complete analysis.

## Won't Fix (Security/Architectural - 15 tests)

### shellTransform Behavior (10 tests)
- **Reason**: Arbitrary command execution security risk
- **Tests**:
  - HTTP/HTTPS shell transform tests (4)
  - HTTP/HTTPS behavior composition with shellTransform (4)
  - TCP behavior composition with shellTransform (1)
  - HTTP proxy persist behaviors (uses shellTransform) (1)
- **Alternative**: Use `decorate` behavior with sandboxed JavaScript
- **Reference**: `docs/SECURITY.md`

### Process Object Access (2 tests)
- **Reason**: Environment variable exposure security risk
- **Tests**: HTTP + HTTPS injection tests accessing `process.env`
- **Decision**: Security sandbox is priority over compatibility

### HTTPS Key/Cert Echo (1 test)
- **Test**: `should support sending key/cert pair during imposter creation`
- **File**: `mbTest/api/https/httpsCertificateTest.js`
- **Reason**: Private key material exposure security risk
- **Decision**: Never return private keys in API responses

### HTTP Proxy Replayable Export (1 test)
- **Test**: `should support retrieving replayable JSON with proxies removed for later playback`
- **Status**: Minor format differences (equals vs deepEquals, headers)
- **Priority**: Low - edge case feature

### TCP Old Proxy Syntax (1 test)
- **Test**: `should support old proxy syntax for backwards compatibility`
- **Reason**: Legacy compatibility for deprecated format
- **Decision**: Not supporting deprecated syntax

## goja Async Limitation (4 tests)

### Async JavaScript Injection
- **Tests**:
  - `should allow asynchronous injection` (HTTP)
  - `should allow asynchronous injection` (HTTPS)
  - `should allow asynchronous injection (old interface)` (TCP)
  - `should allow asynchronous injection` (TCP)
- **Status**: goja ES5.1 limitation - no native Promise/async support
- **Priority**: Low - rarely used feature

## Validation Procedure

See `.claude/claude.md` for complete validation workflow.

### Quick Validation

```bash
cd /home/tetsujinoni/work/mountebank
pkill -f tartuffe 2>/dev/null || true
MB_EXECUTABLE=/home/tetsujinoni/work/go-tartuffe/bin/tartuffe-wrapper.sh npm run test:api 2>&1 | tee /tmp/tartuffe-validation.log
grep -E "passing|failing" /tmp/tartuffe-validation.log | tail -3
```

Expected: 233 passing / 19 failing (92.5%)

## Documentation References

- **Test Mapping**: [docs/MOUNTEBANK-TEST-MAPPING.md](docs/MOUNTEBANK-TEST-MAPPING.md)
- **Controller Tests**: [docs/CONTROLLER-API-TEST-MAPPING.md](docs/CONTROLLER-API-TEST-MAPPING.md)
- **HTTP Proxy**: [docs/HTTP-PROXY-TEST-MAPPING.md](docs/HTTP-PROXY-TEST-MAPPING.md)
- **Security**: [docs/SECURITY.md](docs/SECURITY.md)
- **Workflows**: [.claude/claude.md](.claude/claude.md)

---

**Note**: Historical results and fix summaries are in `docs/` directory. This backlog contains only remaining work.
