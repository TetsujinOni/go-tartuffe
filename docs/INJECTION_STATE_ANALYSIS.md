# JavaScript Injection State Management - Analysis and Fix Plan

## Problem Statement

go-tartuffe's JavaScript injection (for HTTP/HTTPS) does not persist state across requests or between predicate and response evaluation within the same request. This causes ~8 mountebank tests to fail.

**Current Status**: TCP injection state works correctly (has persistent `state` map in TCPServer). HTTP injection needs the same treatment.

## Root Cause Analysis

### How Mountebank Handles State

Mountebank provides **two types of state** to injection functions:

1. **Local/Request State**: Persists within a single request lifecycle (predicate evaluation + response generation)
2. **Imposter State (Global)**: Persists across ALL requests to an imposter

#### Mountebank Injection Signature (HTTP)

```javascript
// Response injection (5 parameters):
function(request, localState, logger, callback, imposterState) {
    // localState - empty object, persists within request lifecycle
    // imposterState - global state across all requests
    imposterState.counter = (imposterState.counter || 0) + 1;
    return { statusCode: 200, body: String(imposterState.counter) };
}

// Predicate injection (3 parameters):
function(request, logger, imposterState) {
    imposterState.hits = (imposterState.hits || 0) + 1;
    return true;
}
```

### Current go-tartuffe HTTP Implementation (BROKEN)

**File**: `internal/imposter/inject.go`

```go
func (e *JSEngine) ExecuteResponse(script string, req *models.Request) (*models.IsResponse, error) {
    vm := goja.New()
    // ... setup ...

    // PROBLEM: Creates NEW empty state each time
    vm.Set("state", map[string]interface{}{})

    wrappedScript := fmt.Sprintf(`
        (function() {
            var fn = %s;
            return fn(request, state, logger);  // state is always empty!
        })()
    `, script)
    // ...
}

func (e *JSEngine) ExecutePredicate(script string, req *models.Request) (bool, error) {
    vm := goja.New()
    // ... setup ...

    // PROBLEM: No state parameter passed to predicates at all!
    wrappedScript := fmt.Sprintf(`
        (function() {
            var fn = %s;
            return fn(request, logger);  // Missing imposterState!
        })()
    `, script)
    // ...
}
```

### Current go-tartuffe TCP Implementation (WORKING)

**File**: `internal/imposter/tcp_server.go`

```go
type TCPServer struct {
    // ...
    state map[string]interface{} // Persistent state for injection scripts - CORRECT!
}

func NewTCPServer(imp *models.Imposter) (*TCPServer, error) {
    return &TCPServer{
        // ...
        state: make(map[string]interface{}), // Created once per imposter - CORRECT!
    }, nil
}

func (s *TCPServer) handleConnection(...) {
    // ...
    // Pass persistent state - CORRECT!
    injectedData, err := s.jsEngine.ExecuteTCPResponse(match.RawResponse.Inject, dataStr, s.state)
    // ...
}
```

### HTTP Server Implementation (MISSING STATE)

**File**: `internal/imposter/manager.go` (Server struct)

```go
type Server struct {
    imposter         *models.Imposter
    httpServer       *http.Server
    listener         net.Listener
    matcher          *Matcher
    proxyHandler     *ProxyHandler
    jsEngine         *JSEngine
    behaviorExecutor *BehaviorExecutor
    tlsConfig        *tls.Config
    useTLS           bool
    started          bool
    mu               sync.RWMutex
    // MISSING: state map[string]interface{} !!
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // ...
    match := s.matcher.Match(req)  // Predicates evaluated here, but no state passed
    // ...
    if match.Inject != "" {
        // PROBLEM: No state passed!
        injResp, err := s.jsEngine.ExecuteResponse(match.Inject, req)
        // ...
    }
}
```

### Matcher Implementation (MISSING STATE)

**File**: `internal/imposter/matcher.go`

```go
func (m *Matcher) evaluatePredicate(pred *models.Predicate, req *models.Request) bool {
    // ...
    if pred.Inject != "" {
        return m.evaluateInject(pred.Inject, req)
    }
    // ...
}

func (m *Matcher) evaluateInject(script string, req *models.Request) bool {
    engine := NewJSEngine()  // PROBLEM: Creates new engine each time!
    result, err := engine.ExecutePredicate(script, req)  // No state passed!
    // ...
}
```

## Required Changes

### 1. Add State to Server struct

**File**: `internal/imposter/manager.go`

```go
type Server struct {
    // ... existing fields ...
    injectionState map[string]interface{} // ADD: Persistent state for injection scripts
}

func NewServer(imp *models.Imposter, useTLS bool) (*Server, error) {
    // ... existing code ...
    srv := &Server{
        // ... existing fields ...
        injectionState: make(map[string]interface{}), // ADD
    }
    // ...
}
```

### 2. Modify Matcher to Accept State

**File**: `internal/imposter/matcher.go`

```go
type Matcher struct {
    imposter *models.Imposter
    state    map[string]interface{} // ADD: Reference to imposter state
    jsEngine *JSEngine              // ADD: Shared JS engine for state persistence
}

func NewMatcher(imp *models.Imposter) *Matcher {
    return &Matcher{
        imposter: imp,
        state:    nil,              // Will be set by SetState()
        jsEngine: NewJSEngine(),
    }
}

// ADD: Method to set state reference
func (m *Matcher) SetState(state map[string]interface{}) {
    m.state = state
}

func (m *Matcher) evaluateInject(script string, req *models.Request) bool {
    // Use shared engine and pass state
    result, err := m.jsEngine.ExecutePredicate(script, req, m.state)
    // ...
}
```

### 3. Modify JSEngine to Accept State

**File**: `internal/imposter/inject.go`

```go
// ExecutePredicate now accepts imposter state
func (e *JSEngine) ExecutePredicate(script string, req *models.Request, imposterState map[string]interface{}) (bool, error) {
    vm := goja.New()
    // ... setup ...

    // Ensure state is not nil
    if imposterState == nil {
        imposterState = make(map[string]interface{})
    }

    vm.Set("imposterState", imposterState)

    // Match mountebank signature: function(request, logger, imposterState)
    wrappedScript := fmt.Sprintf(`
        (function() {
            var fn = %s;
            return fn(request, logger, imposterState);
        })()
    `, script)
    // ...
}

// ExecuteResponse now accepts imposter state
func (e *JSEngine) ExecuteResponse(script string, req *models.Request, imposterState map[string]interface{}) (*models.IsResponse, error) {
    vm := goja.New()
    // ... setup ...

    if imposterState == nil {
        imposterState = make(map[string]interface{})
    }

    // Local state for request-scoped persistence
    localState := make(map[string]interface{})

    vm.Set("state", localState)           // Request-local state
    vm.Set("imposterState", imposterState) // Global imposter state

    // Match mountebank signature (simplified without callback):
    // function(request, localState, logger, callback, imposterState)
    // We use: function(request, state, logger, imposterState)
    wrappedScript := fmt.Sprintf(`
        (function() {
            var fn = %s;
            // Try new interface with imposterState
            try {
                return fn(request, state, logger, imposterState);
            } catch (e) {
                // Fallback to old interface without imposterState
                return fn(request, state, logger);
            }
        })()
    `, script)
    // ...
}
```

### 4. Wire State in Server.ServeHTTP

**File**: `internal/imposter/manager.go`

```go
func NewServer(imp *models.Imposter, useTLS bool) (*Server, error) {
    injectionState := make(map[string]interface{})
    jsEngine := NewJSEngine()

    matcher := NewMatcher(imp)
    matcher.SetState(injectionState)
    matcher.SetJSEngine(jsEngine)  // Share engine for state

    srv := &Server{
        imposter:         imp,
        matcher:          matcher,
        jsEngine:         jsEngine,
        injectionState:   injectionState,
        // ... other fields ...
    }
    // ...
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // ...
    if match.Inject != "" {
        // Pass imposter state
        injResp, err := s.jsEngine.ExecuteResponse(match.Inject, req, s.injectionState)
        // ...
    }
    // ...
}
```

## Mountebank Test Cases That Should Pass After Fix

### From `responseResolverTest.js`:

1. **Test #1072**: "should allow injection request state across calls to resolve"
   - Counter increments from 1 to 2 across calls

2. **Test #1091**: "should allow injection imposter state across calls to resolve"
   - Global state persists: `barbar1` then `barbar2`

### From `predicates/injectTest.js`:

3. **Test #51**: "should allow changing the state in the injection"
   - `imposterState.foo` changes from `bar` to `barbar`

### Additional HTTP Injection Tests (~8 total):

4. State counter across multiple HTTP requests
5. State sharing between predicate and response injection
6. State isolation between different imposters
7. State reset on imposter deletion
8. State with decorate behavior

## Implementation Order

1. **Add state field to Server struct** (manager.go)
2. **Add state to Matcher** (matcher.go)
3. **Modify JSEngine.ExecutePredicate** to accept state (inject.go)
4. **Modify JSEngine.ExecuteResponse** to accept imposter state (inject.go)
5. **Wire state in NewServer** (manager.go)
6. **Wire state in ServeHTTP** (manager.go)
7. **Add integration tests** for HTTP injection state
8. **Verify existing TCP tests still pass**

## Test Plan

### Unit Tests

```go
// inject_test.go
func TestJSEngine_ExecuteResponse_WithState(t *testing.T) {
    engine := NewJSEngine()
    state := make(map[string]interface{})
    req := &models.Request{Method: "GET", Path: "/"}

    script := `function(request, localState, logger, imposterState) {
        imposterState.counter = (imposterState.counter || 0) + 1;
        return { statusCode: 200, body: String(imposterState.counter) };
    }`

    resp1, _ := engine.ExecuteResponse(script, req, state)
    resp2, _ := engine.ExecuteResponse(script, req, state)

    assert.Equal(t, "1", resp1.Body)
    assert.Equal(t, "2", resp2.Body)
}

func TestJSEngine_ExecutePredicate_WithState(t *testing.T) {
    engine := NewJSEngine()
    state := make(map[string]interface{})
    req := &models.Request{Method: "GET", Path: "/"}

    script := `function(request, logger, imposterState) {
        imposterState.hits = (imposterState.hits || 0) + 1;
        return imposterState.hits < 3;
    }`

    result1, _ := engine.ExecutePredicate(script, req, state)
    result2, _ := engine.ExecutePredicate(script, req, state)
    result3, _ := engine.ExecutePredicate(script, req, state)

    assert.True(t, result1)  // hits=1 < 3
    assert.True(t, result2)  // hits=2 < 3
    assert.False(t, result3) // hits=3 >= 3
}
```

### Integration Tests

```go
// integration/http_injection_state_test.go
func TestHTTPInjectionState_PersistsAcrossRequests(t *testing.T) {
    // Create imposter with injection that uses counter
    // Make 3 requests
    // Verify responses are "1", "2", "3"
}

func TestHTTPInjectionState_SharedBetweenPredicateAndResponse(t *testing.T) {
    // Create imposter with predicate injection that sets state
    // Response injection reads that state
    // Verify state flows from predicate to response
}
```

## Files to Modify

1. `internal/imposter/inject.go` - Add state parameters
2. `internal/imposter/matcher.go` - Add state field and methods
3. `internal/imposter/manager.go` - Wire state in Server
4. `internal/imposter/inject_test.go` - Add state tests
5. NEW: `test/integration/http_injection_state_test.go` - Integration tests

## Backward Compatibility

The changes maintain backward compatibility:
- Scripts without state parameter will still work (ignored parameter)
- Existing scripts using `state` (localState) continue to work
- New `imposterState` parameter is additive

## Risk Assessment

- **Low Risk**: TCP injection already uses this pattern successfully
- **Medium Risk**: Matcher changes could affect predicate evaluation timing
- **Mitigation**: Comprehensive test coverage before/after changes
