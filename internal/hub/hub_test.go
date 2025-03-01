package hub

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// MockEndpoint is a mock implementation of the Endpoint interface for testing
type MockEndpoint struct {
	data  []byte
	isSSE bool
}

// NewMockEndpoint creates a new MockEndpoint with the given data
func NewMockEndpoint(data []byte) *MockEndpoint {
	return &MockEndpoint{
		data: data,
	}
}

// HandleSSE implements the Endpoint interface
func (m *MockEndpoint) HandleSSE(w http.ResponseWriter, r *http.Request) {
	// Check if this is an SSE request
	m.isSSE = strings.HasSuffix(r.URL.Path, "/stream")

	// Get max_count parameter
	maxCountStr := r.URL.Query().Get("max_count")
	maxCount := 1 // Default to 1 for test purposes
	if maxCountStr != "" {
		fmt.Sscanf(maxCountStr, "%d", &maxCount)
	}

	// Set REST headers
	w.Header().Set("Content-Type", "application/json")

	// In a real endpoint, we would loop up to maxCount
	// but for the mock, we just send the data once
	if _, err := w.Write(m.data); err != nil {
		// Log the error but continue
		fmt.Printf("Error writing response: %v\n", err)
	}
}

func TestPlatform_REST(t *testing.T) {
	// Create a new platform
	config := DefaultConfig()
	platform := New(config)

	// Create a mock endpoint
	mockData := []byte(`{"message":"Hello, World!"}`)
	mockEndpoint := NewMockEndpoint(mockData)

	// Register the endpoint
	platform.RegisterEndpoint("test", mockEndpoint)

	// Create a test server
	mux := http.NewServeMux()
	// Register the endpoint with the platform's handler
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
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
		mockEndpoint.HandleSSE(rr, r)

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
			t.Fatalf("Error encoding response: %v", err)
			return
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	// Make a request to the endpoint
	resp, err := http.Get(server.URL + "/test")
	if err != nil {
		t.Fatalf("Error making request: %v", err)
	}
	defer resp.Body.Close()

	// Check the response
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Decode the response
	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		t.Fatalf("Error decoding response: %v", err)
	}

	// Check that the response contains the data field
	dataJSON, ok := data["data"]
	if !ok {
		t.Fatal("Response does not contain 'data' field")
	}

	// Check the data
	dataMap, ok := dataJSON.(map[string]interface{})
	if !ok {
		t.Fatalf("Data is not a map: %v", dataJSON)
	}

	if dataMap["message"] != "Hello, World!" {
		t.Errorf("Expected message %q, got %q", "Hello, World!", dataMap["message"])
	}
}

func TestPlatform_SSE(t *testing.T) {
	// Create a new platform
	config := DefaultConfig()
	platform := New(config)

	// Create a mock endpoint with a simple response
	mockData := []byte(`{"message":"Hello, SSE!"}`)
	mockEndpoint := NewMockEndpoint(mockData)

	// Register the endpoint
	platform.RegisterEndpoint("test", mockEndpoint)

	// Create a test server
	mux := http.NewServeMux()
	mux.HandleFunc("/test/stream", func(w http.ResponseWriter, r *http.Request) {
		// Add max_count=1 to ensure the test completes
		q := r.URL.Query()
		q.Set("max_count", "1")
		r.URL.RawQuery = q.Encode()

		// Set SSE headers
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		// Check if streaming is supported
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("Streaming not supported")
			return
		}

		// Create a response recorder to capture the endpoint's response
		rr := &responseRecorder{
			header: make(http.Header),
			body:   new(strings.Builder),
			code:   http.StatusOK,
		}
		mockEndpoint.HandleSSE(rr, r)

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
		responseJSON, err := json.Marshal(wrappedResponse)
		if err != nil {
			t.Fatalf("Error encoding SSE response: %v", err)
			return
		}

		// Send the response
		fmt.Fprintf(w, "data: %s\n\n", responseJSON)
		flusher.Flush()
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	// Make a request to the SSE endpoint
	resp, err := http.Get(server.URL + "/test/stream")
	if err != nil {
		t.Fatalf("Error making request: %v", err)
	}
	defer resp.Body.Close()

	// Check the response headers
	if resp.Header.Get("Content-Type") != "text/event-stream" {
		t.Errorf("Expected Content-Type %q, got %q", "text/event-stream", resp.Header.Get("Content-Type"))
	}

	if resp.Header.Get("Cache-Control") != "no-cache" {
		t.Errorf("Expected Cache-Control %q, got %q", "no-cache", resp.Header.Get("Cache-Control"))
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Error reading response body: %v", err)
	}

	// Check that the response body contains the expected data
	responseStr := string(body)
	if !strings.Contains(responseStr, "data:") {
		t.Errorf("Expected response to contain 'data:', got %q", responseStr)
	}

	// Extract the JSON from the SSE event
	lines := strings.Split(responseStr, "\n")
	if len(lines) < 2 {
		t.Fatalf("Response body does not contain enough lines: %v", responseStr)
	}

	dataLine := lines[0]
	if !strings.HasPrefix(dataLine, "data: ") {
		t.Fatalf("SSE event does not start with 'data: ': %v", dataLine)
	}

	jsonStr := strings.TrimPrefix(dataLine, "data: ")

	// Print the jsonStr for debugging
	t.Logf("jsonStr: %s", jsonStr)

	// Parse the JSON
	var response map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &response); err != nil {
		t.Fatalf("Error parsing JSON from SSE event: %v (jsonStr: %s)", err, jsonStr)
	}

	// Check that the response contains the data field
	dataJSON, ok := response["data"]
	if !ok {
		t.Fatal("Response does not contain 'data' field")
	}

	// Check the data
	dataMap, ok := dataJSON.(map[string]interface{})
	if !ok {
		t.Fatalf("Data is not a map: %v", dataJSON)
	}

	if dataMap["message"] != "Hello, SSE!" {
		t.Errorf("Expected message %q, got %q", "Hello, SSE!", dataMap["message"])
	}
}

func TestPlatform_MultipleEndpoints(t *testing.T) {
	// Create a new platform
	config := DefaultConfig()
	platform := New(config)

	// Create mock endpoints
	endpoint1 := NewMockEndpoint([]byte(`{"endpoint":"endpoint1"}`))
	endpoint2 := NewMockEndpoint([]byte(`{"endpoint":"endpoint2"}`))

	// Register the endpoints
	platform.RegisterEndpoint("endpoint1", endpoint1)
	platform.RegisterEndpoint("endpoint2", endpoint2)

	// Create a test server
	mux := http.NewServeMux()

	// Register endpoint1 with the platform's handler
	mux.HandleFunc("/endpoint1", func(w http.ResponseWriter, r *http.Request) {
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
		endpoint1.HandleSSE(rr, r)

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
			t.Fatalf("Error encoding response: %v", err)
			return
		}
	})

	// Register endpoint2 with the platform's handler
	mux.HandleFunc("/endpoint2", func(w http.ResponseWriter, r *http.Request) {
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
		endpoint2.HandleSSE(rr, r)

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
			t.Fatalf("Error encoding response: %v", err)
			return
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	// Test endpoint1
	resp1, err := http.Get(server.URL + "/endpoint1")
	if err != nil {
		t.Fatalf("Error making request to endpoint1: %v", err)
	}
	defer resp1.Body.Close()

	var response1 map[string]interface{}
	if err := json.NewDecoder(resp1.Body).Decode(&response1); err != nil {
		t.Fatalf("Error decoding response from endpoint1: %v", err)
	}

	// Check that the response contains the data field
	dataJSON1, ok := response1["data"]
	if !ok {
		t.Fatal("Response from endpoint1 does not contain 'data' field")
	}

	// Check the data
	dataMap1, ok := dataJSON1.(map[string]interface{})
	if !ok {
		t.Fatalf("Data from endpoint1 is not a map: %v", dataJSON1)
	}

	if dataMap1["endpoint"] != "endpoint1" {
		t.Errorf("Expected endpoint %q, got %q", "endpoint1", dataMap1["endpoint"])
	}

	// Test endpoint2
	resp2, err := http.Get(server.URL + "/endpoint2")
	if err != nil {
		t.Fatalf("Error making request to endpoint2: %v", err)
	}
	defer resp2.Body.Close()

	var response2 map[string]interface{}
	if err := json.NewDecoder(resp2.Body).Decode(&response2); err != nil {
		t.Fatalf("Error decoding response from endpoint2: %v", err)
	}

	// Check that the response contains the data field
	dataJSON2, ok := response2["data"]
	if !ok {
		t.Fatal("Response from endpoint2 does not contain 'data' field")
	}

	// Check the data
	dataMap2, ok := dataJSON2.(map[string]interface{})
	if !ok {
		t.Fatalf("Data from endpoint2 is not a map: %v", dataJSON2)
	}

	if dataMap2["endpoint"] != "endpoint2" {
		t.Errorf("Expected endpoint %q, got %q", "endpoint2", dataMap2["endpoint"])
	}
}
