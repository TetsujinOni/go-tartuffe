# deepEquals Predicate Fix - Summary

## Overview

Implemented type-insensitive comparison for deepEquals predicates to match mountebank's behavior, where numeric and string values are coerced to strings for comparison (e.g., `1` matches `"1"`).

## Problem

Mountebank's deepEquals predicate uses type-insensitive comparison through its `forceStrings()` function. This allows predicates with numeric values to match query parameters which are always strings:

```javascript
// Mountebank test expects this to match
Predicate: { deepEquals: { query: { num: 1 } } }
Request: GET /path?num=1
```

go-tartuffe was doing strict type comparison, so integer `1` did not match string `"1"`.

## Solution

### Implementation

Added `forceToString()` function to `internal/imposter/matcher.go` that recursively converts all values to strings before comparison:

```go
func (m *Matcher) forceToString(value interface{}) interface{} {
    if value == nil {
        return "null"
    }

    switch v := value.(type) {
    case string:
        return v
    case map[string]interface{}:
        // Recursively convert map values
        result := make(map[string]interface{})
        for key, val := range v {
            result[key] = m.forceToString(val)
        }
        return result
    case []interface{}:
        // Recursively convert array elements
        result := make([]interface{}, len(v))
        for i, val := range v {
            result[i] = m.forceToString(val)
        }
        return result
    case bool:
        return fmt.Sprintf("%t", v)  // true → "true"
    case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
        return fmt.Sprintf("%d", v)  // 42 → "42"
    case float32, float64:
        return fmt.Sprintf("%v", v)
    default:
        return fmt.Sprintf("%v", v)
    }
}
```

### Integration

Updated `deepCompareValues()` to use forced string conversion:

```go
func (m *Matcher) deepCompareValues(actual, expected interface{}, opts predicateOptions) bool {
    // ... handle nil ...

    // Force both values to strings for type-insensitive comparison
    actualForced := m.forceToString(actual)
    expectedForced := m.forceToString(expected)

    // Compare using forced versions
    actualStr, actualIsStr := toString(actualForced)
    expectedStr, expectedIsStr := toString(expectedForced)

    // ... comparison logic using actualForced/expectedForced ...
}
```

**Critical Fix**: Ensured all comparison branches use `actualForced` and `expectedForced` instead of original `actual` and `expected` values.

## Test Results

### Before Fix
- **185 passing, 67 failing (73.4%)**
- deepEquals predicates with type mismatches failing

### After Fix
- **188 passing, 64 failing (74.6%)**
- **+3 tests passing**

### Tests Fixed

1. **HTTP deepEquals object predicates** - Numeric values in predicates match string query parameters
2. **HTTPS deepEquals object predicates** - Same for HTTPS protocol
3. **Additional type coercion cases** - Various scenarios with type mismatches

## Files Modified

- `internal/imposter/matcher.go`
  - Added `forceToString()` function (lines 350-389)
  - Updated `deepCompareValues()` to use forced conversion (lines 391-475)
  - Fixed all comparison branches to use forced versions

## Mountebank Compatibility

This fix brings go-tartuffe's deepEquals behavior in line with mountebank's implementation in `/src/models/predicates.js`:

```javascript
// Mountebank's forceStrings function
function forceStrings (obj) {
    if (obj === null || typeof obj === 'string') {
        return obj;
    }
    else if (Array.isArray(obj)) {
        return obj.map(forceStrings);
    }
    else if (typeof obj === 'object') {
        return Object.keys(obj).reduce((result, key) => {
            result[key] = forceStrings(obj[key]);
            return result;
        }, {});
    }
    else {
        return obj.toString();
    }
}
```

## Impact

**Positive:**
- Improved mountebank compatibility (+1.2%)
- Type-insensitive comparison more user-friendly
- Matches expected behavior for query parameters

**Neutral:**
- No performance impact (comparison is still O(n))
- Maintains backward compatibility

## Related Issues

This fix is part of the broader compatibility effort tracked in:
- [COMPATIBILITY-BACKLOG.md](../COMPATIBILITY-BACKLOG.md)
- [TEST-RESULTS-2026-01-17.md](TEST-RESULTS-2026-01-17.md)

## Future Work

The deepEquals fix is complete. Remaining predicate issues are:
- XPath array predicates (different issue)
- Gzip request decompression (different issue)
- JSON body predicate matching (different issue)

These require separate fixes beyond deepEquals type coercion.
