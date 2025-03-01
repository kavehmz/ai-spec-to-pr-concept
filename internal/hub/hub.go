// Package hub implements a web service hub that supports REST and SSE.
// It provides a common interface for endpoints to implement and handles the communication
// details for each protocol.
package hub

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// Config represents the configuration for the hub
type Config struct {
	Port     string // Default: "8080"
	LogLevel string // Default: "info"
}

// DefaultConfig returns a Config with default values
func DefaultConfig() Config {
	return Config{
		Port:     "8080",
		LogLevel: "info",
	}
}

// Error represents an error in the JSON API format
type Error struct {
	Status string `json:"status"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

// ErrorResponse represents the error response in the JSON API format
type ErrorResponse struct {
	Errors []Error `json:"errors"`
}

// Endpoint is the interface that each endpoint must implement
type Endpoint interface {
	// HandleSSE handles Server-Sent Events
	// The endpoint should return its data directly, and the hub will wrap it in a "data" field
	HandleSSE(w http.ResponseWriter, r *http.Request)
}

// WriteError writes an error response in the JSON API format
func WriteError(w http.ResponseWriter, status int, title, detail string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	response := ErrorResponse{
		Errors: []Error{
			{
				Status: fmt.Sprintf("%d", status),
				Title:  title,
				Detail: detail,
			},
		},
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("Error encoding error response", "error", err)
		http.Error(w, "Error encoding error response", http.StatusInternalServerError)
	}
}

// DataResponse represents the response in the JSON API format
type DataResponse struct {
	Data interface{} `json:"data"`
}

// responseRecorder is a simple implementation of http.ResponseWriter for capturing responses
type responseRecorder struct {
	header http.Header
	body   *strings.Builder
	code   int
}

// Header returns the header map that will be sent by WriteHeader
func (r *responseRecorder) Header() http.Header {
	return r.header
}

// Write writes the data to the response body
func (r *responseRecorder) Write(b []byte) (int, error) {
	return r.body.Write(b)
}

// WriteHeader sends an HTTP response header with the provided status code
func (r *responseRecorder) WriteHeader(statusCode int) {
	r.code = statusCode
}

// BodyString returns the response body as a string
func (r *responseRecorder) BodyString() string {
	return r.body.String()
}

// BodyBytes returns the response body as a byte slice
func (r *responseRecorder) BodyBytes() []byte {
	return []byte(r.body.String())
}

// customResponseWriter is a custom implementation of http.ResponseWriter that sends
// the response to a channel instead of writing it directly
type customResponseWriter struct {
	http.ResponseWriter
	responseChan chan<- []byte
}

// Write sends the data to the response channel
func (w *customResponseWriter) Write(b []byte) (int, error) {
	// Send a copy of the data to the channel
	data := make([]byte, len(b))
	copy(data, b)
	w.responseChan <- data
	return len(b), nil
}

// Hub represents the web service hub
type Hub struct {
	endpoints map[string]Endpoint // 8 bytes
	config    Config              // 32 bytes
	mu        sync.RWMutex        // 8 bytes
}

// New creates a new Hub with the given configuration
func New(config Config) *Hub {
	return &Hub{
		config:    config,
		endpoints: make(map[string]Endpoint),
	}
}

// RegisterEndpoint registers an endpoint with the hub
func (p *Hub) RegisterEndpoint(name string, endpoint Endpoint) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.endpoints[name] = endpoint
}

// getMaxCount extracts the max_count parameter from the request
// If not provided, returns 3600 (default)
func getMaxCount(r *http.Request) int {
	maxCountStr := r.URL.Query().Get("max_count")
	if maxCountStr == "" {
		return 3600 // Default value
	}

	var maxCount int
	_, err := fmt.Sscanf(maxCountStr, "%d", &maxCount)
	if err != nil || maxCount <= 0 {
		slog.Debug("Invalid max_count parameter", "value", maxCountStr)
		return 3600 // Invalid value, use default
	}

	return maxCount
}

// Start starts the hub server
func (p *Hub) Start() error {
	// Set up logging
	logLevel := getLogLevel(p.config.LogLevel)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)

	// Set up HTTP server
	mux := http.NewServeMux()

	// Register endpoints
	p.mu.RLock()
	for name, endpoint := range p.endpoints {
		endpointName := name // Create a new variable to avoid closure issues
		endpointHandler := endpoint

		// REST endpoint (special case of SSE with max_count=1)
		mux.HandleFunc("/"+endpointName, func(w http.ResponseWriter, r *http.Request) {
			slog.Info("Received REST request", "endpoint", endpointName, "method", r.Method, "path", r.URL.Path)

			// Set max_count=1 for REST requests
			q := r.URL.Query()
			q.Set("max_count", "1")
			r.URL.RawQuery = q.Encode()

			// Create a response recorder to capture the endpoint's response
			rr := &responseRecorder{
				header: make(http.Header),
				body:   new(strings.Builder),
				code:   http.StatusOK,
			}
			endpointHandler.HandleSSE(rr, r)

			// Copy the headers from the recorder to the response writer
			for k, v := range rr.Header() {
				w.Header()[k] = v
			}

			// Set the content type to application/json for REST
			w.Header().Set("Content-Type", "application/json")

			// Check if the response is an error
			if rr.code != http.StatusOK {
				w.WriteHeader(rr.code)
				w.Write(rr.BodyBytes())
				return
			}

			// Parse the response body
			var responseData interface{}
			if err := json.Unmarshal(rr.BodyBytes(), &responseData); err != nil {
				// If the response is not valid JSON, wrap it as a string
				responseData = rr.BodyString()
			}

			// Wrap the response in a data field
			wrappedResponse := DataResponse{
				Data: responseData,
			}

			// Encode the wrapped response
			if err := json.NewEncoder(w).Encode(wrappedResponse); err != nil {
				slog.Error("Error encoding response", "error", err)
				http.Error(w, "Error encoding response", http.StatusInternalServerError)
				return
			}
		})

		// SSE endpoint
		mux.HandleFunc("/"+endpointName+"/stream", func(w http.ResponseWriter, r *http.Request) {
			slog.Info("Received SSE request", "endpoint", endpointName, "method", r.Method, "path", r.URL.Path)

			// Set SSE headers
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")
			w.Header().Set("Access-Control-Allow-Origin", "*")

			// Check if streaming is supported
			flusher, ok := w.(http.Flusher)
			if !ok {
				slog.Error("Streaming not supported")
				http.Error(w, "Streaming not supported", http.StatusInternalServerError)
				return
			}

			// Create a channel to receive responses from the endpoint
			responseChan := make(chan []byte)

			// Start the endpoint handler in a goroutine
			go func() {
				// Create a custom response writer that captures the response
				customWriter := &customResponseWriter{
					ResponseWriter: w,
					responseChan:   responseChan,
				}

				// Call the endpoint handler
				endpointHandler.HandleSSE(customWriter, r)
				close(responseChan)
			}()

			// Process responses from the endpoint
			for responseData := range responseChan {
				// Parse the response body
				var responseObj interface{}
				if err := json.Unmarshal(responseData, &responseObj); err != nil {
					// If the response is not valid JSON, wrap it as a string
					responseObj = string(responseData)
				}

				// Wrap the response in a data field
				wrappedResponse := DataResponse{
					Data: responseObj,
				}

				// Encode the wrapped response
				wrappedData, err := json.Marshal(wrappedResponse)
				if err != nil {
					slog.Error("Error encoding SSE response", "error", err)
					continue
				}

				// Send the response as an SSE event
				fmt.Fprintf(w, "data: %s\n\n", wrappedData)
				flusher.Flush()
			}
		})
	}
	p.mu.RUnlock()

	// Start server with timeouts
	addr := ":" + p.config.Port
	slog.Info("Starting server", "port", p.config.Port)

	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return server.ListenAndServe()
}

// getLogLevel converts a string log level to a slog.Level
func getLogLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
