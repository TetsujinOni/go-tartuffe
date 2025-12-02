package imposter

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/bufbuild/protocompile"
	"github.com/bufbuild/protocompile/linker"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// ProtoLoader handles loading and parsing .proto files at runtime
type ProtoLoader struct {
	mu       sync.RWMutex
	files    linker.Files
	services map[string]protoreflect.ServiceDescriptor
	messages map[string]protoreflect.MessageDescriptor
	baseDir  string
}

// NewProtoLoader creates a new proto loader
func NewProtoLoader() *ProtoLoader {
	return &ProtoLoader{
		services: make(map[string]protoreflect.ServiceDescriptor),
		messages: make(map[string]protoreflect.MessageDescriptor),
	}
}

// LoadProtos loads proto files from the specified directory
func (l *ProtoLoader) LoadProtos(baseDir string, protoFiles []string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.baseDir = baseDir

	// Validate base directory
	if baseDir != "" {
		info, err := os.Stat(baseDir)
		if err != nil {
			return fmt.Errorf("proto directory not found: %s", baseDir)
		}
		if !info.IsDir() {
			return fmt.Errorf("proto path is not a directory: %s", baseDir)
		}
	}

	// Resolve proto file paths
	var resolvedFiles []string
	for _, f := range protoFiles {
		var fullPath string
		if filepath.IsAbs(f) {
			fullPath = f
		} else if baseDir != "" {
			fullPath = filepath.Join(baseDir, f)
		} else {
			fullPath = f
		}

		// Validate file exists
		if _, err := os.Stat(fullPath); err != nil {
			return fmt.Errorf("proto file not found: %s", fullPath)
		}

		resolvedFiles = append(resolvedFiles, fullPath)
	}

	if len(resolvedFiles) == 0 {
		return fmt.Errorf("no proto files specified")
	}

	// Create compiler with resolver
	compiler := protocompile.Compiler{
		Resolver: &protocompile.SourceResolver{
			ImportPaths: l.getImportPaths(baseDir, resolvedFiles),
		},
	}

	// Compile proto files
	files, err := compiler.Compile(context.Background(), resolvedFiles...)
	if err != nil {
		return fmt.Errorf("failed to compile proto files: %w", err)
	}

	l.files = files

	// Index services and messages
	for _, file := range files {
		l.indexFile(file)
	}

	return nil
}

// getImportPaths returns import paths for proto resolution
func (l *ProtoLoader) getImportPaths(baseDir string, files []string) []string {
	paths := make(map[string]bool)

	// Add base directory
	if baseDir != "" {
		paths[baseDir] = true
	}

	// Add directories containing proto files
	for _, f := range files {
		dir := filepath.Dir(f)
		paths[dir] = true
	}

	// Convert to slice
	var result []string
	for p := range paths {
		result = append(result, p)
	}

	return result
}

// indexFile indexes services and messages from a proto file
func (l *ProtoLoader) indexFile(file protoreflect.FileDescriptor) {
	// Index services
	services := file.Services()
	for i := 0; i < services.Len(); i++ {
		svc := services.Get(i)
		fullName := string(svc.FullName())
		l.services[fullName] = svc
	}

	// Index top-level messages
	messages := file.Messages()
	for i := 0; i < messages.Len(); i++ {
		l.indexMessage(messages.Get(i))
	}
}

// indexMessage recursively indexes a message and its nested types
func (l *ProtoLoader) indexMessage(msg protoreflect.MessageDescriptor) {
	fullName := string(msg.FullName())
	l.messages[fullName] = msg

	// Index nested messages
	nested := msg.Messages()
	for i := 0; i < nested.Len(); i++ {
		l.indexMessage(nested.Get(i))
	}
}

// GetService returns a service descriptor by full name
func (l *ProtoLoader) GetService(fullName string) (protoreflect.ServiceDescriptor, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	svc, ok := l.services[fullName]
	return svc, ok
}

// GetMessage returns a message descriptor by full name
func (l *ProtoLoader) GetMessage(fullName string) (protoreflect.MessageDescriptor, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	msg, ok := l.messages[fullName]
	return msg, ok
}

// GetAllServices returns all loaded service descriptors
func (l *ProtoLoader) GetAllServices() []protoreflect.ServiceDescriptor {
	l.mu.RLock()
	defer l.mu.RUnlock()

	result := make([]protoreflect.ServiceDescriptor, 0, len(l.services))
	for _, svc := range l.services {
		result = append(result, svc)
	}
	return result
}

// GetMethod returns a method descriptor by service and method name
func (l *ProtoLoader) GetMethod(serviceName, methodName string) (protoreflect.MethodDescriptor, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	svc, ok := l.services[serviceName]
	if !ok {
		return nil, false
	}

	methods := svc.Methods()
	for i := 0; i < methods.Len(); i++ {
		m := methods.Get(i)
		if string(m.Name()) == methodName {
			return m, true
		}
	}

	return nil, false
}

// GetMethodByFullName returns a method descriptor by full method path (e.g., "/package.Service/Method")
func (l *ProtoLoader) GetMethodByFullName(fullMethod string) (protoreflect.MethodDescriptor, bool) {
	// Parse full method name (format: /package.Service/Method)
	fullMethod = strings.TrimPrefix(fullMethod, "/")
	parts := strings.Split(fullMethod, "/")
	if len(parts) != 2 {
		return nil, false
	}

	return l.GetMethod(parts[0], parts[1])
}

// GetFiles returns all loaded file descriptors
func (l *ProtoLoader) GetFiles() linker.Files {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.files
}

// ServiceInfo contains information about a service
type ServiceInfo struct {
	FullName string
	Methods  []MethodInfo
}

// MethodInfo contains information about a method
type MethodInfo struct {
	Name            string
	InputType       string
	OutputType      string
	ClientStreaming bool
	ServerStreaming bool
}

// ListServices returns information about all loaded services
func (l *ProtoLoader) ListServices() []ServiceInfo {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var result []ServiceInfo
	for name, svc := range l.services {
		info := ServiceInfo{
			FullName: name,
		}

		methods := svc.Methods()
		for i := 0; i < methods.Len(); i++ {
			m := methods.Get(i)
			info.Methods = append(info.Methods, MethodInfo{
				Name:            string(m.Name()),
				InputType:       string(m.Input().FullName()),
				OutputType:      string(m.Output().FullName()),
				ClientStreaming: m.IsStreamingClient(),
				ServerStreaming: m.IsStreamingServer(),
			})
		}

		result = append(result, info)
	}

	return result
}
