// Package date implements the date endpoint for the hub service.
// It provides current date and time information in various formats.
package date

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// DateResponse represents the response from the date endpoint
type DateResponse struct {
	// According to the specification, the date endpoint should return the UTC field
	UTC string `json:"UTC"` // 16 bytes
}

// Config represents the configuration for the date endpoint
type Config struct {
	// Add any date-specific configuration options here
}

// Endpoint implements the hub.Endpoint interface for the date endpoint
type Endpoint struct {
	config Config
}

// New creates a new Endpoint with the given configuration
func New(config Config) *Endpoint {
	return &Endpoint{
		config: config,
	}
}

// HandleSSE handles both REST and SSE requests for the date endpoint
// The hub will handle the differences between REST and SSE
func (d *Endpoint) HandleSSE(w http.ResponseWriter, r *http.Request) {
	// Get max_count parameter (for SSE) - hub.go will default to 3600 if not provided
	maxCountStr := r.URL.Query().Get("max_count")
	var maxCount int
	if maxCountStr != "" {
		fmt.Sscanf(maxCountStr, "%d", &maxCount)
	} else {
		maxCount = 3600 // Default value
	}

	// Create a context that is canceled when the client disconnects
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// Handle client disconnect
	go func() {
		<-ctx.Done()
		slog.Info("Context canceled for date endpoint")
	}()

	// Send events to client
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	count := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Check if we've reached the max count
			if count >= maxCount {
				return
			}

			// Get the current time
			now := time.Now()

			// Create the response
			response := DateResponse{
				UTC: now.UTC().Format(time.RFC3339),
			}

			// Encode the response
			responseData, err := json.Marshal(response)
			if err != nil {
				slog.Error("Error encoding response", "error", err)
				return
			}

			// Write the response
			// The hub will handle wrapping this in a "data" field
			// and handle the protocol-specific formatting
			_, err = w.Write(responseData)
			if err != nil {
				slog.Error("Error writing response", "error", err)
				return
			}

			// For SSE, the hub will handle flushing
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}

			// Increment the count
			count++
		}
	}
}
