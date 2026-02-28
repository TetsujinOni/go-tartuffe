package models

import (
	"bytes"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"strconv"
	"sync"
	"time"
)

// Buffer pool for reducing allocations during JSON marshaling
var bufPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

func getBuf() *bytes.Buffer {
	return bufPool.Get().(*bytes.Buffer)
}

func putBuf(buf *bytes.Buffer) {
	buf.Reset()
	bufPool.Put(buf)
}

// EndOfRequestResolver defines how to determine the end of a TCP request
type EndOfRequestResolver struct {
	Inject string `json:"inject,omitempty"` // JavaScript function: (requestData, logger) => boolean
}

// Imposter represents a mock server instance
type Imposter struct {
	Port                 int                   `json:"port"`
	Protocol             string                `json:"protocol"`
	Name                 string                `json:"name,omitempty"`
	Host                 string                `json:"host,omitempty"` // Hostname/IP to bind to (empty = all interfaces)
	Mode                 string                `json:"mode,omitempty"` // For TCP: "text" or "binary"
	RecordRequests       bool                  `json:"recordRequests"`
	AllowCORS            bool                  `json:"allowCORS,omitempty"`            // Enable CORS preflight support
	EndOfRequestResolver *EndOfRequestResolver `json:"endOfRequestResolver,omitempty"` // For TCP: custom request boundary detection
	Stubs                []Stub                `json:"stubs,omitempty"`
	DefaultResponse      *Response             `json:"defaultResponse,omitempty"`
	Requests             []Request             `json:"requests,omitempty"`
	TCPRequests          []TCPRequest          `json:"tcpRequests,omitempty"`  // For TCP protocol
	SMTPRequests         []SMTPRequest         `json:"smtpRequests,omitempty"` // For SMTP protocol
	GRPCRequests         []GRPCRequest         `json:"grpcRequests,omitempty"` // For gRPC protocol
	Links                *Links                `json:"_links,omitempty"`

	// gRPC configuration
	ProtoFiles       []string        `json:"protoFiles,omitempty"`       // .proto files to load
	ProtoDirectory   string          `json:"protoDirectory,omitempty"`   // Base directory for proto files
	Services         []ServiceConfig `json:"services,omitempty"`         // Services to expose (nil = all)
	EnableReflection bool            `json:"enableReflection,omitempty"` // Enable gRPC reflection API

	// HTTPS/TLS configuration (input fields)
	Key                string   `json:"key,omitempty"`                // Private key PEM (not returned in API responses)
	Cert               string   `json:"cert,omitempty"`               // Certificate PEM
	MutualAuth         bool     `json:"mutualAuth,omitempty"`         // Request client certificates
	RejectUnauthorized bool     `json:"rejectUnauthorized,omitempty"` // Validate client certs against CA
	Ca                 []string `json:"ca,omitempty"`                 // CA certificates for client validation
	Ciphers            string   `json:"ciphers,omitempty"`            // TLS cipher suite

	// HTTPS certificate metadata (output fields - extracted from cert)
	CertificateFingerprint string `json:"certificateFingerprint,omitempty"` // SHA-256 fingerprint
	CommonName             string `json:"commonName,omitempty"`             // Certificate CN
	ValidFrom              string `json:"validFrom,omitempty"`              // Not Before date
	ValidTo                string `json:"validTo,omitempty"`                // Not After date

	// Internal fields (conditionally serialized)
	NumberOfRequests *int `json:"numberOfRequests,omitempty"`
}

// TCPRequest represents a recorded TCP request
type TCPRequest struct {
	RequestFrom string `json:"requestFrom,omitempty"`
	Data        string `json:"data"`
	Timestamp   string `json:"timestamp,omitempty"`
}

// GRPCRequest represents a recorded gRPC request
type GRPCRequest struct {
	RequestFrom string                 `json:"requestFrom,omitempty"`
	Service     string                 `json:"service"`            // Full service name (package.Service)
	Method      string                 `json:"method"`             // RPC method name
	Message     map[string]interface{} `json:"message"`            // Deserialized request as JSON
	Metadata    map[string][]string    `json:"metadata,omitempty"` // gRPC metadata (like headers)
	Timestamp   string                 `json:"timestamp,omitempty"`
}

// ServiceConfig defines which services/methods to expose for gRPC
type ServiceConfig struct {
	Name    string   `json:"name"`              // Full service name (package.Service)
	Methods []string `json:"methods,omitempty"` // Specific methods (nil = all methods)
}

// Links contains hypermedia links for REST discoverability
type Links struct {
	Self  *Link `json:"self,omitempty"`
	Stubs *Link `json:"stubs,omitempty"`
}

// Link is a single hypermedia link
type Link struct {
	Href string `json:"href"`
}

// ExtractCertMetadata extracts metadata from the certificate PEM
func (imp *Imposter) ExtractCertMetadata() {
	if imp.Cert == "" {
		return
	}

	block, _ := pem.Decode([]byte(imp.Cert))
	if block == nil {
		return
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return
	}

	// SHA-256 fingerprint
	fingerprint := sha256.Sum256(cert.Raw)
	imp.CertificateFingerprint = fmt.Sprintf("%X", fingerprint[:])

	// Common Name
	imp.CommonName = cert.Subject.CommonName

	// Validity dates in ISO 8601 format
	imp.ValidFrom = cert.NotBefore.UTC().Format(time.RFC3339)
	imp.ValidTo = cert.NotAfter.UTC().Format(time.RFC3339)
}

// MarshalJSON implements custom JSON marshaling to ensure requests and stubs arrays are always present
func (imp *Imposter) MarshalJSON() ([]byte, error) {
	// Use a more efficient approach that builds the JSON structure directly
	// instead of marshal->unmarshal->modify->marshal

	// Create a temporary struct that mirrors Imposter but allows us to control serialization
	type ImposterJSON struct {
		Port                   int                   `json:"port"`
		Protocol               string                `json:"protocol"`
		Name                   string                `json:"name,omitempty"`
		Host                   string                `json:"host,omitempty"`
		Mode                   string                `json:"mode,omitempty"`
		RecordRequests         bool                  `json:"recordRequests"`
		AllowCORS              bool                  `json:"allowCORS,omitempty"`
		EndOfRequestResolver   *EndOfRequestResolver `json:"endOfRequestResolver,omitempty"`
		Stubs                  []Stub                `json:"stubs"`
		DefaultResponse        *Response             `json:"defaultResponse,omitempty"`
		Requests               interface{}           `json:"requests,omitempty"`
		Links                  *Links                `json:"_links,omitempty"`
		ProtoFiles             []string              `json:"protoFiles,omitempty"`
		ProtoDirectory         string                `json:"protoDirectory,omitempty"`
		Services               []ServiceConfig       `json:"services,omitempty"`
		EnableReflection       bool                  `json:"enableReflection,omitempty"`
		Cert                   string                `json:"cert,omitempty"`
		Key                    string                `json:"key,omitempty"`
		MutualAuth             bool                  `json:"mutualAuth,omitempty"`
		RejectUnauthorized     bool                  `json:"rejectUnauthorized,omitempty"`
		Ca                     []string              `json:"ca,omitempty"`
		Ciphers                string                `json:"ciphers,omitempty"`
		CertificateFingerprint string                `json:"certificateFingerprint,omitempty"`
		CommonName             string                `json:"commonName,omitempty"`
		ValidFrom              string                `json:"validFrom,omitempty"`
		ValidTo                string                `json:"validTo,omitempty"`
		NumberOfRequests       *int                  `json:"numberOfRequests,omitempty"`
	}

	result := ImposterJSON{
		Port:                   imp.Port,
		Protocol:               imp.Protocol,
		Name:                   imp.Name,
		Host:                   imp.Host,
		Mode:                   imp.Mode,
		RecordRequests:         imp.RecordRequests,
		AllowCORS:              imp.AllowCORS,
		EndOfRequestResolver:   imp.EndOfRequestResolver,
		DefaultResponse:        imp.DefaultResponse,
		Links:                  imp.Links,
		ProtoFiles:             imp.ProtoFiles,
		ProtoDirectory:         imp.ProtoDirectory,
		Services:               imp.Services,
		EnableReflection:       imp.EnableReflection,
		Cert:                   imp.Cert,
		Key:                    imp.Key,
		MutualAuth:             imp.MutualAuth,
		RejectUnauthorized:     imp.RejectUnauthorized,
		Ca:                     imp.Ca,
		Ciphers:                imp.Ciphers,
		CertificateFingerprint: imp.CertificateFingerprint,
		CommonName:             imp.CommonName,
		ValidFrom:              imp.ValidFrom,
		ValidTo:                imp.ValidTo,
		NumberOfRequests:       imp.NumberOfRequests,
	}

	// Ensure stubs is never nil (required for mountebank compatibility)
	if imp.Stubs != nil {
		result.Stubs = imp.Stubs
	} else {
		result.Stubs = []Stub{}
	}

	// Determine which requests field to use based on protocol
	// Mountebank uses "requests" for all protocols
	switch imp.Protocol {
	case "tcp":
		if imp.TCPRequests != nil {
			if len(imp.TCPRequests) == 0 {
				result.Requests = []interface{}{}
			} else {
				result.Requests = imp.TCPRequests
			}
		}
	case "smtp":
		if imp.SMTPRequests != nil {
			if len(imp.SMTPRequests) == 0 {
				result.Requests = []interface{}{}
			} else {
				result.Requests = imp.SMTPRequests
			}
		}
	case "grpc":
		if imp.GRPCRequests != nil {
			if len(imp.GRPCRequests) == 0 {
				result.Requests = []interface{}{}
			} else {
				result.Requests = imp.GRPCRequests
			}
		}
	default:
		// HTTP/HTTPS - use Requests field
		if imp.Requests != nil {
			if len(imp.Requests) == 0 {
				result.Requests = []interface{}{}
			} else {
				result.Requests = imp.Requests
			}
		}
	}

	return json.Marshal(result)
}

// ToJSON serializes the imposter with options
func (imp *Imposter) ToJSON(options SerializeOptions) ([]byte, error) {
	// Create a copy for serialization
	out := *imp

	// Add hypermedia links
	out.Links = &Links{
		Self:  &Link{Href: "/imposters/" + itoa(imp.Port)},
		Stubs: &Link{Href: "/imposters/" + itoa(imp.Port) + "/stubs"},
	}

	// In replayable mode, exclude requests
	if options.Replayable {
		out.Requests = nil
		out.TCPRequests = nil
		out.SMTPRequests = nil
		out.GRPCRequests = nil
	}

	// Remove proxy stubs if requested
	if options.RemoveProxies && len(out.Stubs) > 0 {
		filtered := make([]Stub, 0, len(out.Stubs))
		for _, stub := range out.Stubs {
			if !stub.IsProxyStub() {
				filtered = append(filtered, stub)
			}
		}
		out.Stubs = filtered
	}

	// For HTTPS imposters, never return the private key
	// Keep cert for transparency but remove key
	out.Key = ""

	return json.Marshal(out)
}

// SerializeOptions controls JSON serialization
type SerializeOptions struct {
	Replayable    bool
	RemoveProxies bool
}

// Simple int to string
func itoa(i int) string {
	return strconv.Itoa(i)
}

// MarshalBody marshals a body value to JSON bytes
func MarshalBody(body interface{}) ([]byte, error) {
	// Use MarshalIndent to match mountebank's pretty-printed JSON format
	return json.MarshalIndent(body, "", "    ")
}
