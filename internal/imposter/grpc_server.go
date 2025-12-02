package imposter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/TetsujinOni/go-tartuffe/internal/models"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

// GRPCServer represents a gRPC imposter server
type GRPCServer struct {
	imposter         *models.Imposter
	grpcServer       *grpc.Server
	listener         net.Listener
	protoLoader      *ProtoLoader
	matcher          *GRPCMatcher
	jsEngine         *JSEngine
	behaviorExecutor *BehaviorExecutor
	started          bool
	stopping         bool
	mu               sync.RWMutex
}

// NewGRPCServer creates a new gRPC imposter server
func NewGRPCServer(imp *models.Imposter) (*GRPCServer, error) {
	loader := NewProtoLoader()

	// Load proto files
	if len(imp.ProtoFiles) == 0 {
		return nil, fmt.Errorf("protoFiles must be specified for gRPC imposter")
	}

	if err := loader.LoadProtos(imp.ProtoDirectory, imp.ProtoFiles); err != nil {
		return nil, fmt.Errorf("failed to load proto files: %w", err)
	}

	jsEngine := NewJSEngine()

	return &GRPCServer{
		imposter:         imp,
		protoLoader:      loader,
		matcher:          NewGRPCMatcher(imp, loader),
		jsEngine:         jsEngine,
		behaviorExecutor: NewBehaviorExecutor(jsEngine),
	}, nil
}

// Start starts the gRPC server
func (s *GRPCServer) Start() error {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return fmt.Errorf("server already started")
	}
	s.started = true
	s.mu.Unlock()

	// Create listener
	addr := fmt.Sprintf("%s:%d", s.imposter.Host, s.imposter.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		s.mu.Lock()
		s.started = false
		s.mu.Unlock()
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}
	s.listener = listener

	// Create gRPC server with unknown service handler
	s.grpcServer = grpc.NewServer(
		grpc.UnknownServiceHandler(s.handleUnknown),
	)

	// Enable reflection if configured
	if s.imposter.EnableReflection {
		reflection.Register(s.grpcServer)
	}

	// Start serving
	go func() {
		if err := s.grpcServer.Serve(listener); err != nil {
			s.mu.RLock()
			stopping := s.stopping
			s.mu.RUnlock()
			if !stopping {
				fmt.Printf("gRPC server error: %v\n", err)
			}
		}
	}()

	return nil
}

// Stop stops the gRPC server
func (s *GRPCServer) Stop(ctx context.Context) error {
	s.mu.Lock()
	if !s.started {
		s.mu.Unlock()
		return nil
	}
	s.stopping = true
	s.started = false
	s.mu.Unlock()

	done := make(chan struct{})
	go func() {
		s.grpcServer.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		s.grpcServer.Stop()
		return ctx.Err()
	}
}

// handleUnknown handles all unknown service calls (our dynamic handler)
func (s *GRPCServer) handleUnknown(srv interface{}, stream grpc.ServerStream) error {
	ctx := stream.Context()

	fullMethod, ok := grpc.Method(ctx)
	if !ok {
		return status.Error(codes.Internal, "unable to get method from context")
	}

	method, ok := s.protoLoader.GetMethodByFullName(fullMethod)
	if !ok {
		return status.Errorf(codes.Unimplemented, "method %s not found in loaded protos", fullMethod)
	}

	// Route to appropriate handler based on streaming type
	isClientStream := method.IsStreamingClient()
	isServerStream := method.IsStreamingServer()

	switch {
	case !isClientStream && !isServerStream:
		return s.handleUnary(ctx, stream, method, fullMethod)
	case !isClientStream && isServerStream:
		return s.handleServerStream(ctx, stream, method, fullMethod)
	case isClientStream && !isServerStream:
		return s.handleClientStream(ctx, stream, method, fullMethod)
	default:
		return s.handleBidiStream(ctx, stream, method, fullMethod)
	}
}

// handleUnary handles a unary RPC call
func (s *GRPCServer) handleUnary(ctx context.Context, stream grpc.ServerStream, method protoreflect.MethodDescriptor, fullMethod string) error {
	inputMsg := dynamicpb.NewMessage(method.Input())

	if err := stream.RecvMsg(inputMsg); err != nil {
		return status.Errorf(codes.Internal, "failed to receive message: %v", err)
	}

	grpcReq, err := s.createGRPCRequest(ctx, inputMsg, fullMethod)
	if err != nil {
		return err
	}

	s.recordRequest(ctx, grpcReq)

	match := s.matcher.Match(grpcReq, method)

	return s.sendUnaryResponse(stream, method, match, grpcReq)
}

// handleServerStream handles server streaming RPC
func (s *GRPCServer) handleServerStream(ctx context.Context, stream grpc.ServerStream, method protoreflect.MethodDescriptor, fullMethod string) error {
	inputMsg := dynamicpb.NewMessage(method.Input())

	if err := stream.RecvMsg(inputMsg); err != nil {
		return status.Errorf(codes.Internal, "failed to receive message: %v", err)
	}

	grpcReq, err := s.createGRPCRequest(ctx, inputMsg, fullMethod)
	if err != nil {
		return err
	}

	s.recordRequest(ctx, grpcReq)

	match := s.matcher.Match(grpcReq, method)

	return s.sendStreamingResponse(stream, method, match, grpcReq)
}

// handleClientStream handles client streaming RPC
func (s *GRPCServer) handleClientStream(ctx context.Context, stream grpc.ServerStream, method protoreflect.MethodDescriptor, fullMethod string) error {
	// Collect all client messages
	var messages []map[string]interface{}
	inputDesc := method.Input()

	for {
		inputMsg := dynamicpb.NewMessage(inputDesc)
		err := stream.RecvMsg(inputMsg)
		if err == io.EOF {
			break
		}
		if err != nil {
			return status.Errorf(codes.Internal, "failed to receive message: %v", err)
		}

		jsonBytes, err := protojson.Marshal(inputMsg)
		if err != nil {
			continue
		}

		var msgMap map[string]interface{}
		if err := json.Unmarshal(jsonBytes, &msgMap); err != nil {
			continue
		}
		messages = append(messages, msgMap)
	}

	// Create request with all messages (use first message for matching, store all)
	var messageMap map[string]interface{}
	if len(messages) > 0 {
		messageMap = messages[0]
	} else {
		messageMap = make(map[string]interface{})
	}

	serviceName, methodName := parseFullMethod(fullMethod)
	md, _ := metadata.FromIncomingContext(ctx)
	metadataMap := make(map[string][]string)
	for k, v := range md {
		metadataMap[k] = v
	}

	grpcReq := &models.GRPCRequest{
		Service:   serviceName,
		Method:    methodName,
		Message:   messageMap,
		Metadata:  metadataMap,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	s.recordRequest(ctx, grpcReq)

	match := s.matcher.Match(grpcReq, method)

	return s.sendUnaryResponse(stream, method, match, grpcReq)
}

// handleBidiStream handles bidirectional streaming RPC
func (s *GRPCServer) handleBidiStream(ctx context.Context, stream grpc.ServerStream, method protoreflect.MethodDescriptor, fullMethod string) error {
	inputDesc := method.Input()
	outputDesc := method.Output()
	serviceName, methodName := parseFullMethod(fullMethod)

	md, _ := metadata.FromIncomingContext(ctx)
	metadataMap := make(map[string][]string)
	for k, v := range md {
		metadataMap[k] = v
	}

	// Process each incoming message and respond
	for {
		inputMsg := dynamicpb.NewMessage(inputDesc)
		err := stream.RecvMsg(inputMsg)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return status.Errorf(codes.Internal, "failed to receive message: %v", err)
		}

		jsonBytes, err := protojson.Marshal(inputMsg)
		if err != nil {
			continue
		}

		var messageMap map[string]interface{}
		if err := json.Unmarshal(jsonBytes, &messageMap); err != nil {
			continue
		}

		grpcReq := &models.GRPCRequest{
			Service:   serviceName,
			Method:    methodName,
			Message:   messageMap,
			Metadata:  metadataMap,
			Timestamp: time.Now().Format(time.RFC3339),
		}

		s.recordRequest(ctx, grpcReq)

		match := s.matcher.Match(grpcReq, method)

		if match.Response == nil {
			continue
		}

		// Apply behaviors
		resp, err := s.applyBehaviors(match, grpcReq)
		if err != nil {
			return err
		}

		// Check for error status
		if resp.StatusCode != 0 {
			return status.Error(codes.Code(resp.StatusCode), resp.StatusMessage)
		}

		// Send response
		outputMsg := dynamicpb.NewMessage(outputDesc)
		if resp.Body != nil {
			bodyBytes, _ := json.Marshal(resp.Body)
			protojson.Unmarshal(bodyBytes, outputMsg)
		}

		if err := stream.SendMsg(outputMsg); err != nil {
			return status.Errorf(codes.Internal, "failed to send message: %v", err)
		}
	}
}

// createGRPCRequest creates a GRPCRequest from a dynamic message
func (s *GRPCServer) createGRPCRequest(ctx context.Context, inputMsg *dynamicpb.Message, fullMethod string) (*models.GRPCRequest, error) {
	jsonBytes, err := protojson.Marshal(inputMsg)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal request to JSON: %v", err)
	}

	var messageMap map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &messageMap); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to parse request JSON: %v", err)
	}

	md, _ := metadata.FromIncomingContext(ctx)
	metadataMap := make(map[string][]string)
	for k, v := range md {
		metadataMap[k] = v
	}

	serviceName, methodName := parseFullMethod(fullMethod)

	return &models.GRPCRequest{
		Service:   serviceName,
		Method:    methodName,
		Message:   messageMap,
		Metadata:  metadataMap,
		Timestamp: time.Now().Format(time.RFC3339),
	}, nil
}

// recordRequest records the request if configured
func (s *GRPCServer) recordRequest(ctx context.Context, grpcReq *models.GRPCRequest) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.imposter.RecordRequests {
		if p, ok := getPeerAddress(ctx); ok {
			grpcReq.RequestFrom = p
		}
		s.imposter.GRPCRequests = append(s.imposter.GRPCRequests, *grpcReq)
	}
	s.imposter.NumberOfRequests++
}

// applyBehaviors applies behaviors to the response
func (s *GRPCServer) applyBehaviors(match *GRPCMatchResult, grpcReq *models.GRPCRequest) (*models.IsResponse, error) {
	resp := match.Response
	if resp == nil {
		return &models.IsResponse{}, nil
	}

	// Apply behaviors if present
	if len(match.Behaviors) > 0 {
		// Convert gRPC request to HTTP-like request for behavior executor
		httpReq := s.grpcToHTTPRequest(grpcReq)

		var err error
		resp, err = s.behaviorExecutor.Execute(httpReq, resp, match.Behaviors)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "behavior error: %v", err)
		}
	}

	return resp, nil
}

// grpcToHTTPRequest converts a gRPC request to an HTTP-like request for behaviors
func (s *GRPCServer) grpcToHTTPRequest(grpcReq *models.GRPCRequest) *models.Request {
	// Convert metadata to headers
	headers := make(map[string]string)
	for k, v := range grpcReq.Metadata {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	// Convert message to body string
	bodyBytes, _ := json.Marshal(grpcReq.Message)

	return &models.Request{
		Method:  "POST",
		Path:    "/" + grpcReq.Service + "/" + grpcReq.Method,
		Headers: headers,
		Body:    string(bodyBytes),
	}
}

// sendUnaryResponse sends a single response
func (s *GRPCServer) sendUnaryResponse(stream grpc.ServerStream, method protoreflect.MethodDescriptor, match *GRPCMatchResult, grpcReq *models.GRPCRequest) error {
	if match.Response == nil {
		return status.Error(codes.Unimplemented, "no matching stub found")
	}

	// Apply behaviors
	resp, err := s.applyBehaviors(match, grpcReq)
	if err != nil {
		return err
	}

	// Check for error status code
	if resp.StatusCode != 0 {
		code := codes.Code(resp.StatusCode)
		msg := resp.StatusMessage
		if msg == "" {
			msg = code.String()
		}
		return status.Error(code, msg)
	}

	// Create and send response message
	outputMsg := dynamicpb.NewMessage(method.Output())

	if resp.Body != nil {
		bodyBytes, err := json.Marshal(resp.Body)
		if err != nil {
			return status.Errorf(codes.Internal, "failed to marshal response body: %v", err)
		}

		if err := protojson.Unmarshal(bodyBytes, outputMsg); err != nil {
			return status.Errorf(codes.Internal, "failed to convert response to protobuf: %v", err)
		}
	}

	if err := stream.SendMsg(outputMsg); err != nil {
		return status.Errorf(codes.Internal, "failed to send response: %v", err)
	}

	return nil
}

// sendStreamingResponse sends multiple responses for server streaming
func (s *GRPCServer) sendStreamingResponse(stream grpc.ServerStream, method protoreflect.MethodDescriptor, match *GRPCMatchResult, grpcReq *models.GRPCRequest) error {
	if match.Response == nil {
		return status.Error(codes.Unimplemented, "no matching stub found")
	}

	// Apply behaviors first
	resp, err := s.applyBehaviors(match, grpcReq)
	if err != nil {
		return err
	}

	// Check for error status code
	if resp.StatusCode != 0 {
		code := codes.Code(resp.StatusCode)
		msg := resp.StatusMessage
		if msg == "" {
			msg = code.String()
		}
		return status.Error(code, msg)
	}

	outputDesc := method.Output()

	// Check if we have streaming responses
	if len(resp.Stream) > 0 {
		// Send each message in the stream
		for _, item := range resp.Stream {
			outputMsg := dynamicpb.NewMessage(outputDesc)

			bodyBytes, err := json.Marshal(item)
			if err != nil {
				continue
			}

			if err := protojson.Unmarshal(bodyBytes, outputMsg); err != nil {
				continue
			}

			if err := stream.SendMsg(outputMsg); err != nil {
				return status.Errorf(codes.Internal, "failed to send stream message: %v", err)
			}
		}
	} else if resp.Body != nil {
		// Fall back to single response if no stream array
		outputMsg := dynamicpb.NewMessage(outputDesc)

		bodyBytes, err := json.Marshal(resp.Body)
		if err != nil {
			return status.Errorf(codes.Internal, "failed to marshal response body: %v", err)
		}

		if err := protojson.Unmarshal(bodyBytes, outputMsg); err != nil {
			return status.Errorf(codes.Internal, "failed to convert response to protobuf: %v", err)
		}

		if err := stream.SendMsg(outputMsg); err != nil {
			return status.Errorf(codes.Internal, "failed to send response: %v", err)
		}
	}

	return nil
}

// GetImposter returns the imposter configuration
func (s *GRPCServer) GetImposter() *models.Imposter {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.imposter
}

// UpdateStubs updates the stubs for this imposter
func (s *GRPCServer) UpdateStubs(stubs []models.Stub) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.imposter.Stubs = stubs
}

// parseFullMethod parses "/package.Service/Method" into service and method names
func parseFullMethod(fullMethod string) (string, string) {
	if len(fullMethod) > 0 && fullMethod[0] == '/' {
		fullMethod = fullMethod[1:]
	}

	for i := len(fullMethod) - 1; i >= 0; i-- {
		if fullMethod[i] == '/' {
			return fullMethod[:i], fullMethod[i+1:]
		}
	}

	return "", fullMethod
}

// getPeerAddress extracts the peer address from context
func getPeerAddress(ctx context.Context) (string, bool) {
	if p, ok := peer.FromContext(ctx); ok {
		return p.Addr.String(), true
	}
	return "", false
}
