# JSON Body Serialization Fix

## Problem

When returning responses with object bodies (not strings), go-tartuffe was returning `[object Object]` instead of properly serialized JSON. This affected 4 tests in the mountebank test suite.

### Example of the Issue

**Request to mountebank:**
```json
{
  "port": 4545,
  "protocol": "http",
  "stubs": [{
    "responses": [{
      "is": {
        "body": {
          "key": "value",
          "arr": [1, 2]
        }
      }
    }]
  }]
}
```

**Expected response body:**
```json
{"key":"value","arr":[1,2]}
```

**Actual response body (before fix):**
```
[object Object]
```

### Root Cause

The issue had two parts:

1. **Object bodies weren't being converted to JSON strings early enough** - We were keeping them as `map[string]interface{}` until the final write step
2. **Content-Type header was being set incorrectly** - We set `Content-Type: application/json` for object bodies, which caused the mountebank test client to automatically parse the body as JSON (turning it into an object), then when tests tried to `JSON.parse()` it again, they got `[object Object]`

### Mountebank's Behavior

Looking at mountebank's source ([baseHttpServer.js:34-37](https://github.com/bbyars/mountebank/blob/master/src/models/http/baseHttpServer.js#L34-L37)):

```javascript
if (isObject(response.body)) {
    // Support JSON response bodies
    response.body = JSON.stringify(response.body, null, 4);
}
```

Mountebank:
- Converts object bodies to JSON strings **early** in response processing
- Does NOT automatically set Content-Type header
- Uses 4-space indentation for pretty-printing

## Solution

### Changes Made

#### 1. Added `normalizeResponse()` in [matcher.go](../internal/imposter/matcher.go)

```go
// normalizeResponse normalizes a response body, converting objects to JSON strings
// This matches mountebank's behavior of converting object bodies to JSON before processing
func (m *Matcher) normalizeResponse(resp *models.IsResponse) *models.IsResponse {
	// Make a copy to avoid modifying the original
	normalized := *resp

	// If body is an object (not a string), convert it to JSON
	if normalized.Body != nil {
		switch normalized.Body.(type) {
		case string, []byte:
			// Already a string or bytes, no conversion needed
		default:
			// It's an object - convert to JSON string
			if jsonBytes, err := json.Marshal(normalized.Body); err == nil {
				normalized.Body = string(jsonBytes)
			}
		}
	}

	return &normalized
}
```

This function:
- Takes an `IsResponse` and returns a normalized copy
- Converts object bodies to JSON strings using `json.Marshal`
- Leaves string/byte bodies unchanged
- Is called when preparing a response from a stub (line 126)

#### 2. Removed automatic Content-Type setting in [manager.go](../internal/imposter/manager.go)

**Before:**
```go
// Set content-type if not set and we have a body
if w.Header().Get("Content-Type") == "" && resp != nil && resp.Body != nil {
	// Default to text/plain for string bodies, application/json for objects
	switch resp.Body.(type) {
	case string, []byte:
		w.Header().Set("Content-Type", "text/plain")
	default:
		w.Header().Set("Content-Type", "application/json")
	}
}
```

**After:**
```go
// Note: We do NOT automatically set Content-Type header for body
// Mountebank doesn't set it automatically either - it must be explicitly
// set in the response headers if desired. This matches mountebank behavior
// and prevents issues with test clients that auto-parse application/json responses.
```

Why this matters:
- The mountebank test client auto-parses responses with `Content-Type: application/json`
- When a body is already a JSON string and the client parses it, subsequent `JSON.parse()` calls fail
- By not setting Content-Type automatically, we match mountebank's behavior exactly

### Test Results

**Before fix:**
- 181 passing, 71 failing (71.8% raw)
- JSON body tests failing with `"[object Object]" is not valid JSON`

**After fix:**
- 185 passing, 67 failing (73.4% raw, 78.4% adjusted)
- All JSON body tests passing (+4 tests):
  - ✅ should support JSON bodies (HTTP)
  - ✅ should support JSON bodies (HTTPS)
  - ✅ should handle JSON null values (HTTP)
  - ✅ should handle JSON null values (HTTPS)

## Files Modified

1. **[internal/imposter/matcher.go](../internal/imposter/matcher.go)**
   - Added `normalizeResponse()` function (lines 140-160)
   - Updated `selectResponse()` to call normalization (line 126)

2. **[internal/imposter/manager.go](../internal/imposter/manager.go)**
   - Removed automatic Content-Type header setting (lines 841-844)
   - Added comment explaining why we don't set it

## Testing

### Manual Test
```bash
# Start tartuffe
./bin/tartuffe --port 2525

# Create imposter with object body
curl -X POST http://localhost:2525/imposters \
  -H "Content-Type: application/json" \
  -d '{
    "port": 4545,
    "protocol": "http",
    "stubs": [{
      "responses": [{
        "is": {
          "body": {"key": "value", "arr": [1, 2]}
        }
      }]
    }]
  }'

# Test the response
curl http://localhost:4545/
# Output: {"arr":[1,2],"key":"value"}
```

### Mountebank Test Suite
```bash
cd /home/tetsujinoni/work/mountebank
MB_EXECUTABLE=/home/tetsujinoni/work/go-tartuffe/bin/tartuffe-wrapper.sh npm run test:api
```

**Result:** 185 passing, 67 failing

## Related Issues

This fix resolves test failures documented in:
- [COMPATIBILITY-FAILURES.md](COMPATIBILITY-FAILURES.md) - Tests #45, #47 (HTTP + HTTPS variants)
- [COMPATIBILITY-BACKLOG.md](../COMPATIBILITY-BACKLOG.md) - Category 4: JSON/Predicates

## Notes

- We use standard `json.Marshal` (compact format), while mountebank uses pretty-printing with 4 spaces
- This difference is cosmetic and doesn't affect functionality
- The key insight is that object-to-JSON conversion must happen **before** behaviors run, not during final response writing
- Content-Type header should only be set if explicitly specified in the response definition
