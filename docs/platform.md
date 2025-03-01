# Platform Service Documentation

This document provides an overview of the platform service, including how to use it, how to add new endpoints, and configuration options.

## Overview

The platform service is a Go-based web service that supports multiple communication protocols for each endpoint:

- **REST**: For traditional request-response interactions
- **Server-Sent Events (SSE)**: For server-to-client streaming

Each endpoint can be accessed through both protocols, providing flexibility for different client requirements.

## Architecture

The platform is designed with a modular architecture:

- **Core Platform**: Handles routing, connection management, and protocol-specific details
- **Endpoints**: Individual modules that implement business logic
- **Interface**: A common interface that all endpoints must implement

### Directory Structure

```
cmd/platform/
  └── main.go                 # Main entry point for the service
internal/platform/
  ├── platform.go             # Core platform implementation
  ├── platform_test.go        # Tests for the platform
  └── <endpoint>/             # Directory for each endpoint
      ├── <endpoint>.go       # Endpoint implementation
      └── <endpoint>_test.go  # Tests for the endpoint
docs/
  └── platform.md             # This documentation
```

## Endpoint Interface

All endpoints must implement the following interface:

```go
type Endpoint interface {
	// HandleSSE handles Server-Sent Events
	HandleSSE(w http.ResponseWriter, r *http.Request)
}
```

The `HandleSSE` method is used for both REST and SSE requests. For REST requests, the platform sets the `max_count` parameter to 1, making it a special case of SSE.

## Adding a New Endpoint

To add a new endpoint to the platform:

1. Create a new directory under `internal/platform/<endpoint>/`
2. Implement the Endpoint interface in `<endpoint>.go`
3. Add tests in `<endpoint>_test.go`
4. Register the endpoint in `cmd/platform/main.go`

### Example: Date Endpoint

The date endpoint is provided as an example implementation. It returns the current date and time in various formats.

```go
// Create a new date endpoint with configuration
dateEndpoint := date.New(date.Config{})

// Register the endpoint with the platform
platform.RegisterEndpoint("date", dateEndpoint)
```

## Accessing Endpoints

### REST and SSE

The platform supports two types of requests:

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
    "UTC": "2023-01-01T12:00:00Z"
  }
}
```

#### SSE Example

```
GET /date/stream
```

The server will send events in the format:

```
data: {"data":{"UTC":"2023-01-01T12:00:00Z"}}
```

#### Limiting SSE Events

You can limit the number of events sent by an SSE endpoint using the `max_count` parameter:

```
GET /date/stream?max_count=5
```

This will send at most 5 events before closing the connection.

## Configuration

The platform can be configured using command-line flags:

- `--port`: The port to listen on (default: "8080")
- `--log-level`: The log level (debug, info, warn, error) (default: "info")

Example:

```
./platform --port=9000 --log-level=debug
```

## Logging

The platform uses the `slog` package for structured logging. Logs are output to stdout in JSON format.

Example log:

```json
{"time":"2023-01-01T12:00:00Z","level":"INFO","msg":"Starting platform service","port":"8080","log_level":"info"}
```

## Error Handling

Errors are logged using `slog` and returned to the client in JSON API format:

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

Error handling differs slightly between protocols:

- **REST**: HTTP status codes with JSON error responses
- **SSE**: Error events or connection closure

## Testing

Each component of the platform has corresponding tests:

- `platform_test.go`: Tests for the core platform
- `<endpoint>_test.go`: Tests for each endpoint

Run the tests using:

```
go test ./...
```

## Example Usage

### Starting the Service

```
go run cmd/platform/main.go
```

### Making Requests

#### REST

```
curl http://localhost:8080/date
```


#### SSE

Using curl:

```
curl -N http://localhost:8080/date/stream
```

Using JavaScript:

```javascript
const eventSource = new EventSource('http://localhost:8080/date/stream');

eventSource.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log(data);
};
