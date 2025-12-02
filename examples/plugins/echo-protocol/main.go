// Echo Protocol Plugin for go-tartuffe
//
// This is an example out-of-process protocol plugin that implements
// a simple echo protocol. It can be used as a template for creating
// custom protocol plugins.
//
// Build:
//   go build -o echo-plugin
//
// Usage in protocols.json:
//   {
//     "echo": {
//       "createCommand": "/path/to/echo-plugin"
//     }
//   }
//
// Create an echo imposter:
//   curl -X POST http://localhost:2525/imposters -d '{
//     "protocol": "echo",
//     "port": 3000,
//     "stubs": [{
//       "responses": [{
//         "is": { "data": "Hello from echo!" }
//       }]
//     }]
//   }'
//
// Test it:
//   echo "test" | nc localhost 3000

package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Config is passed as the last command-line argument
type Config struct {
	Port            int             `json:"port"`
	Protocol        string          `json:"protocol"`
	Name            string          `json:"name"`
	RecordRequests  bool            `json:"recordRequests"`
	Stubs           []Stub          `json:"stubs"`
	DefaultResponse json.RawMessage `json:"defaultResponse,omitempty"`
	CallbackURL     string          `json:"callbackURL"`
}

type Stub struct {
	Predicates []json.RawMessage `json:"predicates"`
	Responses  []Response        `json:"responses"`
}

type Response struct {
	Is    map[string]interface{} `json:"is,omitempty"`
	Proxy map[string]interface{} `json:"proxy,omitempty"`
}

// StartupMessage is sent to stdout when the plugin starts
type StartupMessage struct {
	Port     int                    `json:"port"`
	PID      int                    `json:"pid"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// CallbackRequest is sent to go-tartuffe for stub matching
type CallbackRequest struct {
	Request map[string]interface{} `json:"request"`
}

// CallbackResponse is received from go-tartuffe
type CallbackResponse struct {
	Response  map[string]interface{} `json:"response,omitempty"`
	StubIndex int                    `json:"stubIndex"`
	Matched   bool                   `json:"matched"`
}

func main() {
	log.SetPrefix("[echo-plugin] ")
	log.SetFlags(log.Ltime)

	// Get config from last argument
	if len(os.Args) < 2 {
		log.Fatal("config argument required")
	}

	var config Config
	if err := json.Unmarshal([]byte(os.Args[len(os.Args)-1]), &config); err != nil {
		log.Fatalf("failed to parse config: %v", err)
	}

	// Start TCP listener
	addr := fmt.Sprintf(":%d", config.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	defer listener.Close()

	// Output startup message
	startup := StartupMessage{
		Port: config.Port,
		PID:  os.Getpid(),
		Metadata: map[string]interface{}{
			"version": "1.0.0",
			"name":    "echo-protocol",
		},
	}
	startupJSON, _ := json.Marshal(startup)
	fmt.Println(string(startupJSON))

	log.Printf("echo plugin listening on port %d", config.Port)

	// Handle shutdown
	done := make(chan struct{})
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("shutting down...")
		listener.Close()
		close(done)
	}()

	// Accept connections
	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-done:
				return
			default:
				log.Printf("accept error: %v", err)
				continue
			}
		}

		go handleConnection(conn, &config)
	}
}

func handleConnection(conn net.Conn, config *Config) {
	defer conn.Close()

	remoteAddr := conn.RemoteAddr().String()
	log.Printf("connection from %s", remoteAddr)

	reader := bufio.NewReader(conn)

	for {
		// Set read deadline
		conn.SetReadDeadline(time.Now().Add(30 * time.Second))

		// Read line
		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				log.Printf("read error: %v", err)
			}
			return
		}

		// Build request for callback
		request := map[string]interface{}{
			"requestFrom": remoteAddr,
			"data":        line,
			"timestamp":   time.Now().Format(time.RFC3339),
		}

		// Call back to go-tartuffe for stub matching
		response := callbackToCore(config.CallbackURL, request)

		// Send response
		var responseData string
		if response != nil && response.Matched {
			if data, ok := response.Response["data"].(string); ok {
				responseData = data
			}
		}

		if responseData == "" {
			// Default echo behavior
			responseData = "ECHO: " + line
		}

		if _, err := conn.Write([]byte(responseData)); err != nil {
			log.Printf("write error: %v", err)
			return
		}
	}
}

func callbackToCore(callbackURL string, request map[string]interface{}) *CallbackResponse {
	if callbackURL == "" {
		return nil
	}

	callbackReq := CallbackRequest{Request: request}
	body, err := json.Marshal(callbackReq)
	if err != nil {
		log.Printf("callback marshal error: %v", err)
		return nil
	}

	resp, err := http.Post(callbackURL, "application/json", bytes.NewReader(body))
	if err != nil {
		log.Printf("callback error: %v", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("callback returned %d", resp.StatusCode)
		return nil
	}

	var callbackResp CallbackResponse
	if err := json.NewDecoder(resp.Body).Decode(&callbackResp); err != nil {
		log.Printf("callback decode error: %v", err)
		return nil
	}

	return &callbackResp
}
