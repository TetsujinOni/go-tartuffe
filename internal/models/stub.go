package models

import (
	"encoding/json"
	"sync"
)

// Stub defines matching rules and responses
type Stub struct {
	Predicates []Predicate `json:"predicates,omitempty"`
	Responses  []Response  `json:"responses"`
	Links      *StubLinks  `json:"_links,omitempty"`

	// Internal state for response cycling
	mu            sync.Mutex `json:"-"`
	responseIndex int        `json:"-"`
	repeatCount   int        `json:"-"`
}

// StubLinks contains hypermedia links for a stub
type StubLinks struct {
	Self *Link `json:"self,omitempty"`
}

// Predicate defines match conditions
type Predicate struct {
	// Operators - only one should be set
	Equals     interface{} `json:"equals,omitempty"`
	DeepEquals interface{} `json:"deepEquals,omitempty"`
	Contains   interface{} `json:"contains,omitempty"`
	StartsWith interface{} `json:"startsWith,omitempty"`
	EndsWith   interface{} `json:"endsWith,omitempty"`
	Matches    interface{} `json:"matches,omitempty"`
	Exists     interface{} `json:"exists,omitempty"`
	Not        *Predicate  `json:"not,omitempty"`
	And        []Predicate `json:"and,omitempty"`
	Or         []Predicate `json:"or,omitempty"`
	Inject     string      `json:"inject,omitempty"`

	// Options
	CaseSensitive    bool      `json:"caseSensitive,omitempty"`
	KeyCaseSensitive bool      `json:"keyCaseSensitive,omitempty"`
	Except           string    `json:"except,omitempty"`
	XPath            *Selector `json:"xpath,omitempty"`
	JSONPath         *Selector `json:"jsonpath,omitempty"`
}

// Selector for XPath or JSONPath expressions
type Selector struct {
	Selector   string            `json:"selector"`
	Namespaces map[string]string `json:"ns,omitempty"`
}

// Response defines what to return
type Response struct {
	Is        *IsResponse    `json:"is,omitempty"`
	Proxy     *ProxyResponse `json:"proxy,omitempty"`
	Inject    string         `json:"inject,omitempty"`
	Fault     string         `json:"fault,omitempty"`
	Repeat    int            `json:"repeat,omitempty"`
	Behaviors []Behavior     `json:"_behaviors,omitempty"`

	// Internal: tracks if this was parsed from shorthand format
	isShorthand bool `json:"-"`
}

// UnmarshalJSON handles the shorthand format for defaultResponse
// where {statusCode, body, headers} is equivalent to {is: {statusCode, body, headers}}
func (r *Response) UnmarshalJSON(data []byte) error {
	// First, try to unmarshal with standard Response format
	type responseAlias Response
	var standard responseAlias
	if err := json.Unmarshal(data, &standard); err != nil {
		return err
	}

	// Check if any of the response type fields are set
	if standard.Is != nil || standard.Proxy != nil || standard.Inject != "" || standard.Fault != "" {
		*r = Response(standard)
		r.isShorthand = false
		return nil
	}

	// Try to unmarshal as a shorthand IsResponse (for defaultResponse)
	var isResp IsResponse
	if err := json.Unmarshal(data, &isResp); err != nil {
		return err
	}

	// If any IsResponse fields are set, wrap it
	if isResp.StatusCode != 0 || isResp.Body != nil || isResp.Headers != nil || isResp.Data != "" {
		r.Is = &isResp
		r.Repeat = standard.Repeat
		r.Behaviors = standard.Behaviors
		r.isShorthand = true // Mark as shorthand format
		return nil
	}

	// Otherwise use the standard format (empty response)
	*r = Response(standard)
	return nil
}

// MarshalJSON serializes the response, using shorthand form if it was parsed that way
func (r Response) MarshalJSON() ([]byte, error) {
	// If this was a shorthand form and only has Is response, serialize as shorthand
	if r.isShorthand && r.Is != nil && r.Proxy == nil && r.Inject == "" && r.Fault == "" {
		return json.Marshal(r.Is)
	}

	// Otherwise use standard format
	type responseAlias Response
	return json.Marshal(responseAlias(r))
}

// Fault type constants
const (
	FaultConnectionResetByPeer = "CONNECTION_RESET_BY_PEER"
	FaultRandomDataThenClose   = "RANDOM_DATA_THEN_CLOSE"
)

// IsResponse is a static response definition
type IsResponse struct {
	StatusCode    int                    `json:"statusCode,omitempty"`
	StatusMessage string                 `json:"statusMessage,omitempty"` // For gRPC error message
	Headers       map[string]interface{} `json:"headers,omitempty"`       // Can be string or []string for multi-value
	Body          interface{}            `json:"body,omitempty"`
	Data          string                 `json:"data,omitempty"` // For TCP protocol
	Mode          string                 `json:"_mode,omitempty"`

	// gRPC streaming support
	Stream []interface{} `json:"stream,omitempty"` // Array of messages for server streaming
}

// ProxyResponse defines proxy behavior
type ProxyResponse struct {
	To                     string            `json:"to"`
	Mode                   string            `json:"mode,omitempty"`
	PredicateGenerators    []PredicateGen    `json:"predicateGenerators,omitempty"`
	AddWaitBehavior        bool              `json:"addWaitBehavior,omitempty"`
	AddDecorateBehavior    string            `json:"addDecorateBehavior,omitempty"`
	InjectHeaders          map[string]string `json:"injectHeaders,omitempty"`

	// mTLS options for proxy requests
	Cert           string `json:"cert,omitempty"`           // Client certificate PEM
	Key            string `json:"key,omitempty"`            // Private key PEM
	Ciphers        string `json:"ciphers,omitempty"`        // TLS cipher suites
	SecureProtocol string `json:"secureProtocol,omitempty"` // TLS version (TLSv1, TLSv1.1, TLSv1.2, TLSv1.3)
}

// PredicateGen defines how to generate predicates from proxied requests
type PredicateGen struct {
	Matches       interface{} `json:"matches,omitempty"`
	CaseSensitive bool        `json:"caseSensitive,omitempty"`
	XPath         *Selector   `json:"xpath,omitempty"`
	JSONPath      *Selector   `json:"jsonpath,omitempty"`
}

// Behavior modifies response handling
type Behavior struct {
	Wait           interface{} `json:"wait,omitempty"`
	Repeat         int         `json:"repeat,omitempty"`
	Copy           *Copy       `json:"copy,omitempty"`
	Lookup         *Lookup     `json:"lookup,omitempty"`
	Decorate       string      `json:"decorate,omitempty"`
	ShellTransform string      `json:"shellTransform,omitempty"`
}

// Copy behavior copies values from request to response
type Copy struct {
	From  interface{} `json:"from"`
	Into  string      `json:"into"`
	Using *Using      `json:"using,omitempty"`
}

// Lookup behavior looks up values from external data
type Lookup struct {
	Key         interface{} `json:"key"`
	FromDataSource *DataSource `json:"fromDataSource"`
	Into        string      `json:"into"`
	Using       *Using      `json:"using,omitempty"`
}

// DataSource for lookup behavior
type DataSource struct {
	CSV *CSVSource `json:"csv,omitempty"`
}

// CSVSource defines CSV file data source
type CSVSource struct {
	Path           string `json:"path"`
	KeyColumn      string `json:"keyColumn"`
	Delimiter      string `json:"delimiter,omitempty"`
}

// Using defines value extraction method
type Using struct {
	Method   string            `json:"method"`
	Selector string            `json:"selector,omitempty"`
	NS       map[string]string `json:"ns,omitempty"`
	Options  *UsingOptions     `json:"options,omitempty"`
}

// UsingOptions for extraction methods
type UsingOptions struct {
	IgnoreCase   bool   `json:"ignoreCase,omitempty"`
	Multiline    bool   `json:"multiline,omitempty"`
}

// NextResponse returns the next response in the cycle
func (s *Stub) NextResponse() *Response {
	if len(s.Responses) == 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	resp := &s.Responses[s.responseIndex]

	// Handle repeat
	repeat := resp.Repeat
	if repeat == 0 {
		repeat = 1
	}

	s.repeatCount++
	if s.repeatCount >= repeat {
		s.repeatCount = 0
		s.responseIndex = (s.responseIndex + 1) % len(s.Responses)
	}

	return resp
}

// IsProxyStub returns true if this stub was generated from a proxy
func (s *Stub) IsProxyStub() bool {
	for _, r := range s.Responses {
		if r.Proxy != nil {
			return true
		}
	}
	return false
}
