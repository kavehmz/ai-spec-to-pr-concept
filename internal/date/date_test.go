package date

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDateEndpoint_HandleSSE_REST(t *testing.T) {
	// Create a new date endpoint with default config
	endpoint := New(Config{})

	// Create a request for REST with max_count=1
	req, err := http.NewRequest("GET", "/date", nil)
	if err != nil {
		t.Fatalf("Error creating request: %v", err)
	}
	q := req.URL.Query()
	q.Set("max_count", "1")
	req.URL.RawQuery = q.Encode()

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Handle the request
	endpoint.HandleSSE(rr, req)

	// Decode the response
	var dateResponse DateResponse
	if err := json.NewDecoder(rr.Body).Decode(&dateResponse); err != nil {
		t.Fatalf("Error decoding response: %v", err)
	}

	// Check that the UTC field is not empty
	if dateResponse.UTC == "" {
		t.Error("Expected UTC to be non-empty")
	}

	// Parse the UTC time
	utc, err := time.Parse(time.RFC3339, dateResponse.UTC)
	if err != nil {
		t.Fatalf("Error parsing UTC time: %v", err)
	}

	// Check that the UTC time is in UTC
	if utc.Location().String() != "UTC" {
		t.Errorf("Expected UTC time to be in UTC, got %v", utc.Location().String())
	}
}

func TestDateEndpoint_HandleSSE_Stream(t *testing.T) {
	// Create a new date endpoint with default config
	endpoint := New(Config{})

	// Create a request for SSE with max_count=1
	req, err := http.NewRequest("GET", "/date/stream", nil)
	if err != nil {
		t.Fatalf("Error creating request: %v", err)
	}
	q := req.URL.Query()
	q.Set("max_count", "1")
	req.URL.RawQuery = q.Encode()

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Create a channel to signal when the test is done
	done := make(chan bool)

	// Start the handler in a goroutine
	go func() {
		endpoint.HandleSSE(rr, req)
		done <- true
	}()

	// Wait for the handler to finish or timeout
	select {
	case <-done:
		// Handler finished
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for SSE handler to finish")
	}

	// Decode the response
	var dateResponse DateResponse
	if err := json.NewDecoder(rr.Body).Decode(&dateResponse); err != nil {
		t.Fatalf("Error decoding response: %v", err)
	}

	// Check that the UTC field is not empty
	if dateResponse.UTC == "" {
		t.Error("Expected UTC to be non-empty")
	}

	// Parse the UTC time
	utc, err := time.Parse(time.RFC3339, dateResponse.UTC)
	if err != nil {
		t.Fatalf("Error parsing UTC time: %v", err)
	}

	// Check that the UTC time is in UTC
	if utc.Location().String() != "UTC" {
		t.Errorf("Expected UTC time to be in UTC, got %v", utc.Location().String())
	}
}
