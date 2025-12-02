package models

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"strconv"
	"time"
)

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
	RecordRequests       bool                  `json:"recordRequests,omitempty"`
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

	// Internal fields (not serialized)
	NumberOfRequests int `json:"numberOfRequests"`
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
	return json.Marshal(body)
}
