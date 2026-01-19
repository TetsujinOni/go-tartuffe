# JSON Predicate Fixes - Summary (2026-01-18)

## Overview

Fixed three critical issues with JSON body predicate matching to achieve 75%+ mountebank compatibility.

## Problems Fixed

### 1. equals Predicate - Array Element Matching

**Problem:** When checking if a scalar value matches an array field, go-tartuffe only checked the first element.

**Mountebank behavior:** Check if ANY element in the array matches.

**Example:**
```javascript
// Predicate
{ equals: { body: { arr: 3 } } }

// Request body
{ "arr": [1, 2, 3] }

// Expected: MATCH (because 3 is in the array)
// Before fix: NO MATCH (only checked arr[0] = 1)
// After fix: MATCH ✅
```

**Fix:** Updated `jsonContains()` to iterate through all array elements:

```go
// Before (line 913-917):
if actualArr, ok := actual.([]interface{}); ok {
    if len(actualArr) > 0 {
        return m.jsonContains(actualArr[0], expected, opts)  // Only checks first element
    }
    return false
}

// After (line 914-920):
if actualArr, ok := actual.([]interface{}); ok {
    for _, elem := range actualArr {
        if m.jsonContains(elem, expected, opts) {
            return true  // Returns true if ANY element matches
        }
    }
    return false
}
```

### 2. deepEquals Predicate - Array Order Insensitivity

**Problem:** Arrays had to match in exact order.

**Mountebank behavior:** Arrays can be in any order as long as they contain the same elements.

**Example:**
```javascript
// Predicate
{ deepEquals: { body: { arr: [2, 1, 3] } } }

// Request body
{ "arr": [3, 2, 1] }

// Expected: MATCH (same elements, different order)
// Before fix: NO MATCH (order didn't match)
// After fix: MATCH ✅
```

**Fix:** Updated `deepEqualJSON()` to use order-insensitive array comparison:

```go
// Before (line 530-535):
for i, ev := range expectedArr {
    if !m.deepEqualJSON(actualArr[i], ev, opts) {  // Index-based comparison
        return false
    }
}

// After (line 531-547):
// Check if every expected element exists in actual array (order-insensitive)
// Track which actual elements have been matched
matched := make([]bool, len(actualArr))
for _, ev := range expectedArr {
    found := false
    for i, av := range actualArr {
        if !matched[i] && m.deepEqualJSON(av, ev, opts) {
            matched[i] = true
            found = true
            break
        }
    }
    if !found {
        return false
    }
}
```

### 3. exists Predicate - JSON Body Key Checking

**Problem:** exists predicate couldn't check if keys exist within JSON bodies.

**Mountebank behavior:** Supports checking JSON object keys like `{ exists: { body: { key: true } } }`.

**Example:**
```javascript
// Predicate
{ exists: { body: { firstName: true, lastName: false } } }

// Request body
{ "firstName": "John", "age": 30 }

// Expected: MATCH (firstName exists, lastName doesn't)
// Before fix: Didn't parse body as JSON
// After fix: MATCH ✅
```

**Fix:** Added special handling for body field in `evaluateExists()`:

```go
// Added (line 633-658):
if strings.ToLower(field) == "body" {
    // Parse body as JSON and check if keys exist
    bodyStr := fmt.Sprintf("%v", req.Body)
    var bodyParsed map[string]interface{}
    if err := json.Unmarshal([]byte(bodyStr), &bodyParsed); err == nil {
        for jsonKey, jsonShouldExist := range nestedMap {
            expected, _ := jsonShouldExist.(bool)
            _, exists := bodyParsed[jsonKey]

            // Try case-insensitive if needed
            if !exists && !opts.keyCaseSensitive {
                for k := range bodyParsed {
                    if strings.EqualFold(k, jsonKey) {
                        exists = true
                        break
                    }
                }
            }

            if exists != expected {
                return false
            }
        }
        continue
    }
}
```

## Test Results

### Before Initial JSON Predicate Fixes
- **188 passing, 64 failing (252 total)**
- **74.6% raw compatibility**
- **79.7% adjusted compatibility**

### After Initial JSON Predicate Fixes
- **190 passing, 62 failing (252 total)**
- **75.4% raw compatibility** ✅ **TARGET MET!**
- **80.5% adjusted compatibility**
- **+2 tests passing**

### After HTTP Stub Comprehensive Fixes (2026-01-18)
- **196 passing, 56 failing (252 total)**
- **77.8% raw compatibility** ✅ **TARGET EXCEEDED!**
- **83.1% adjusted compatibility**
- **+8 tests total (+6 net from initial fixes)**

### Tests Fixed

#### Initial JSON Predicate Fixes:
1. **"should support treating the body as a JSON object for predicate matching" (HTTP)** ✅
2. **"should support treating the body as a JSON object for predicate matching" (HTTPS)** ✅

Both tests now pass because all four predicates in the test work correctly:
- `equals: { body: { key: 'value' } }` - Matches nested object
- `equals: { body: { arr: 3 } }` - Matches element in array (ANY element)
- `deepEquals: { body: { key: 'value', arr: [2, 1, 3] } }` - Matches with different array order
- `matches: { body: { key: '^v' } }` - Regex matching on nested value

#### HTTP Stub Comprehensive Fixes:
3. **Gzip request predicate matching (HTTP)** ✅
4. **Gzip request predicate matching (HTTPS)** ✅
5. **XPath array predicates (HTTP)** ✅
6. **XPath array predicates (HTTPS)** ✅
7. **Stub validation errors (HTTP)** ✅
8. **Stub validation errors (HTTPS)** ✅

## Files Modified

- `internal/imposter/matcher.go`
  - `jsonContains()` - Lines 911-920: Array element iteration
  - `deepEqualJSON()` - Lines 521-548: Order-insensitive array matching
  - `evaluateExists()` - Lines 632-658: JSON body key existence checking

## Impact

**Positive:**
- Exceeded 75% compatibility target (77.8% raw, 83.1% adjusted)
- JSON predicates now fully functional for real-world use cases
- Array handling matches mountebank behavior
- Gzip request handling enables testing compressed APIs
- XPath array predicates enable XML testing with multiple matches
- Stub validation provides better error messages

**Compatibility:**
- No breaking changes
- Maintains backward compatibility
- More lenient matching (arrays order-insensitive)
- Enhanced functionality (gzip decompression, XPath arrays)

## Related Documentation

- [COMPATIBILITY-BACKLOG.md](../COMPATIBILITY-BACKLOG.md) - Updated compatibility tracking
- [JSON_PREDICATES_ANALYSIS.md](JSON_PREDICATES_ANALYSIS.md) - Detailed analysis of JSON predicate features
- [TEST-RESULTS-2026-01-17.md](TEST-RESULTS-2026-01-17.md) - Previous test results

## Future Work

All major JSON predicate issues have been resolved! Remaining gaps (40 tests total):
- HTTP/HTTPS Proxy issues (18 tests) - ProxyAlways, predicate generators, CONNECT method
- TCP implementation gaps (13 tests) - Behaviors, proxy, requests arrays
- HTTP faults/metrics/controller (6 tests) - Undefined fault, metrics format
- Other minor issues (3 tests)

The remaining gaps are not related to JSON predicates or HTTP stubs - they are proxy and TCP features.
