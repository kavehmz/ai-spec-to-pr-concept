# Hub Service Documentation

This document provides an overview of the hub service, including how to use it, how to add new endpoints, and configuration options.

## Overview

The hub service is a Go-based web service that supports multiple communication protocols for each endpoint:

- **REST**: For traditional request-response interactions
- **Server-Sent Events (SSE)**: For server-to-client streaming

Each endpoint can be accessed through both protocols, providing flexibility for different client requirements.

## Architecture

The hub is designed with a modular architecture:

- **Core Hub**: Handles routing, connection management, and protocol-specific details
- **Endpoints**: Individual modules that implement business logic
- **Interface**: A common interface that all endpoints must implement

### Directory Structure

```
cmd/hub/
  └── main.go                 # Main entry point for the service
internal/hub/
  ├── hub.go                  # Core hub implementation
  ├── hub_test.go             # Tests for the hub
  └── <endpoint>/             # Directory for each endpoint
      ├── <endpoint>.go       # Endpoint implementation
      └── <endpoint>_test.go  # Tests for the endpoint
docs/
  └── hub.md                  # This documentation
```

## Endpoint Interface

All endpoints must implement the following interface:

```go
type Endpoint interface {
	// HandleSSE handles Server-Sent Events
	HandleSSE(w http.ResponseWriter, r *http.Request)
}
```

The `HandleSSE` method is used for both REST and SSE requests. For REST requests, the hub sets the `max_count` parameter to 1, making it a special case of SSE.

## Adding a New Endpoint

To add a new endpoint to the hub:

1. Create a new directory under `internal/hub/<endpoint>/`
2. Implement the Endpoint interface in `<endpoint>.go`
3. Add tests in `<endpoint>_test.go`
4. Register the endpoint in `cmd/hub/main.go`

### Example: Date Endpoint

The date endpoint is provided as an example implementation. It returns the current date and time in UTC format.

```go
// Create a new date endpoint with configuration
dateEndpoint := date.New(date.Config{})

// Register the endpoint with the hub
hub.RegisterEndpoint("date", dateEndpoint)
```

## Accessing Endpoints

### REST and SSE

The hub supports two types of requests:

1. **REST**: One-time request/response, accessible at `/<endpoint>`. This is a special case of SSE with `max_count=1`.
2. **SSE**: Server-Sent Events for streaming data, accessible at `/<endpoint>/stream`.

#### REST Example

```
GET /date
```

Response:

```json
{
  "data": {
    "UTC": "2025-02-27T12:31:34Z"
  }
}
```

#### SSE Example

```
GET /date/stream
```

The server will send events in the format:

```
data: {"data":{"UTC":"2025-02-27T12:31:34Z"}}
```

#### Limiting SSE Events

You can limit the number of events sent by an SSE endpoint using the `max_count` parameter:

```
GET /date/stream?max_count=5
```

This will send at most 5 events before closing the connection. If not specified, the default value is 3600.

## Configuration

The hub can be configured using command-line flags:

- `--port`: The port to listen on (default: "8080")
- `--log-level`: The log level (debug, info, warn, error) (default: "info")

Example:

```
./hub --port=9000 --log-level=debug
```

## Logging

The hub uses the `slog` package for structured logging. Logs are output to stdout in JSON format.

Example log:

```json
{"time":"2025-02-27T12:31:34Z","level":"INFO","msg":"Starting hub service","port":"8080","log_level":"info"}
```

## Response Format

All responses follow a standardized format:

### Success Response

```json
{
  "data": <endpoint_return_json>
}
```

The endpoint's response is wrapped in a `data` field. The response can be either a simple value or a JSON object.

### Error Response

```json
{
  "errors": [
    {
      "status": "422",
      "title": "Invalid Attribute",
      "detail": "First name must contain at least two characters."
    }
  ]
}
```

## Error Handling

Errors are logged using `slog` and returned to the client in JSON API format.

Error handling differs slightly between protocols:

- **REST**: HTTP status codes with JSON error responses
- **SSE**: Error events or connection closure

## Testing

Each component of the hub has corresponding tests:

- `hub_test.go`: Tests for the core hub
- `<endpoint>_test.go`: Tests for each endpoint

Run the tests using:

```
go test ./...
```

## Example Usage

### Starting the Service

```
go run cmd/hub/main.go
```

### Making Requests

#### REST

```
curl http://localhost:8080/date
```

Response:
```json
{"data":{"UTC":"2025-02-27T12:31:34Z"}}
```

#### SSE

Using curl:

```
curl -N http://localhost:8080/date/stream
```

Response stream:
```
data: {"data":{"UTC":"2025-02-27T12:31:34Z"}}
data: {"data":{"UTC":"2025-02-27T12:31:35Z"}}
data: {"data":{"UTC":"2025-02-27T12:31:36Z"}}
...
```

Using JavaScript:

```javascript
const eventSource = new EventSource('http://localhost:8080/date/stream');

eventSource.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log(data);
};