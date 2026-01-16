# Behavior Test Failures Fix

## Problem Statement

All 50 mountebank behavior tests were failing with the error:
```
json: cannot unmarshal object into Go struct field responseAlias._behaviors of type []models.Behavior
```

**Root Cause**: Mountebank's API accepts `_behaviors` in **two formats**:
1. **Object format** (single behavior): `{"_behaviors": {"wait": 1000}}`
2. **Array format** (multiple behaviors): `{"_behaviors": [{"wait": 1000}]}`

The original implementation only supported the array format, causing all behavior tests to fail when they used the object format.

## Solution

Modified the `Response.UnmarshalJSON()` method in [internal/models/stub.go](../internal/models/stub.go) to:
1. Parse the JSON into a raw `map[string]interface{}`
2. Detect if `_behaviors` is an object or array
3. If it's an object, convert it to a single-element array
4. Continue with normal unmarshaling

This allows both formats to work seamlessly while maintaining backward compatibility.

## Implementation Details

### Modified Files

**[internal/models/stub.go](../internal/models/stub.go:67-126)**
- Updated `Response.UnmarshalJSON()` to handle both object and array formats
- Added normalization logic before standard unmarshaling

### Test Coverage

**Unit Tests** - [internal/models/stub_test.go](../internal/models/stub_test.go)
- `TestBehaviorUnmarshalSingleObject` - Tests all object format variations
- `TestImposterWithBehaviors` - Tests full imposter creation with behaviors
- `TestBehaviorMarshalRoundTrip` - Tests serialization/deserialization

**Integration Tests** - [internal/api/handlers/behaviors_test.go](../internal/api/handlers/behaviors_test.go)
- `TestCreateImposterWithBehaviorsObject` - Tests imposter creation via HTTP API
- `TestCreateImposterWithBehaviorsRejectsInvalid` - Tests error handling

**Updated Integration Tests** - [test/integration/api_test.go](../test/integration/api_test.go)
- Fixed tests to handle absolute URLs (from Fix #1)
- Updated `doRequest()` to handle both relative and absolute URLs

## Examples

### Object Format (Now Supported)
```json
{
  "protocol": "http",
  "port": 3000,
  "stubs": [{
    "responses": [{
      "is": {"statusCode": 200, "body": "OK"},
      "_behaviors": {"wait": 1000}
    }]
  }]
}
```

### Multiple Fields in Object
```json
{
  "responses": [{
    "is": {"body": "test"},
    "_behaviors": {"wait": 500, "repeat": 2}
  }]
}
```

### Array Format (Already Supported)
```json
{
  "responses": [{
    "is": {"body": "test"},
    "_behaviors": [{"wait": 1000}]
  }]
}
```

## Behavior Types Supported

All mountebank behavior types are now properly parsed from object format:

- **wait**: Adds latency (number or function string)
- **repeat**: Repeats response N times
- **decorate**: Post-processes response (function string)
- **copy**: Copies values from request to response
- **lookup**: Looks up values from external data
- **shellTransform**: Executes shell command (when allowed)

## Testing

Run the behavior-specific tests:
```bash
# Unit tests
go test ./internal/models -run TestBehavior -v

# Integration tests
go test ./internal/api/handlers -run TestCreateImposterWithBehaviors -v

# All tests
go test ./...
```

## Impact

**Before Fix**: 0/50 behavior tests passing (0%)
**After Fix**: All behavior parsing issues resolved

This fix unlocks:
- 50 HTTP behavior tests
- Better mountebank API compatibility
- Support for real-world mountebank configurations

## API Compatibility

This change is **100% backward compatible**:
- Existing configurations using array format continue to work
- New configurations can use the simpler object format
- Error handling remains unchanged

## Related Fixes

This was part of a series of mountebank compatibility fixes:
1. ✅ Fix #1: Absolute URLs in `_links`
2. ✅ Fix #3: Error response format with source field
3. ✅ Fix #2: DELETE /imposters replayable format default
4. ✅ **Fix #4: Behaviors JSON parsing** (this fix)

## Next Steps

With behaviors working, the next priorities are:
- Fix #5: Injection tests (similar parsing issues expected)
- Fix #6: TCP protocol improvements
- Fix #7-8: HTTPS/SMTP protocol-specific issues
