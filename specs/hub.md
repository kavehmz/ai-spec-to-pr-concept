You are an AI tasked with generating a Go-based web service hub that supports REST, and Server-Sent Events (SSE) for each endpoint. 
Follow the specifications below to ensure the generated code meets all requirements.

This file is `specs/hub.md` and describing the `hub` package implementation of our code. All paths are based on projects root that contains `go.mod`.

`hub` will implement the general logic for our API. Hub will collect and registers endpoint modudes and handles connections and other connection complexities. This will enable us to implement the endpoints without much complexity with focus on fucntionality they provide.

Note: review `CODING_GUIDELINE.md` as the general guideline for how we code.

### Specifications

1. **Service Architecture:**
   - The service must be able to handle multiple endpoints, each supporting REST, and SSE.
   - REST requests should receive one response per request.
   - SSE connections can remain open for multiple messages.
   - REST endpoints should be accessible at `/<endpoint>`.
   - SSE endpoints should be accessible at `/<endpoint>/stream`.
   - We want to be able to pass a max_count parameter to stream to limit the max number of returned records.
   - if max_count is not set, it defaults to 3600.
   - REST and SSE will be handled by the same function. REST will be an spcial case for SSE with max_count set to 1.


2. **Endpoint Handling:**
   - Each endpoint is defined in its own package under `internal/hub/<endpoint>/`.
   - For example, the `/date` endpoint will be in `internal/date/date.go`.
   - You will implement the /date endpoint as an example and starting point so others can follow the pattern. 
   - Date will return the current date as "2009-11-10T23:00:00Z".

3. **Interface Definition:**
   - Define an interface that each endpoint must implement.
   - The interface should include methods for handling REST, and SSE requests.
   - methods:

```go
type Endpoint interface {
	// HandleSSE handles Server-Sent Events
	HandleSSE(w http.ResponseWriter, r *http.Request)
}
```

   - SSE and REST will use the same function but SSE and REST have slightly different protocols. HandleSSE will handle the requests and also use of max_count and how it loops and send how many requests it returns and if needed stops the streaming if needed all are handled in HandleSSE. But the parts that SSE has a different output headers and output, e.g its format is data: <DATA>, will be handled in `hub.go`. It is better to keep the endpoint implementation as simple as possible.
   - Because REST will be a special case of SSE with max_count = 1, in the endpoints we should only see the SSE implementation. Other details for difference between REST and SSE if needed should be handled in` hub.go` or overriden there.

4. **Communication:**
   - Response are always in JSON format.
   - End points result itself can be JSON or not, `<endpoint_return_json>`. For example date endpoint can return a simple date lile `2009-11-10T23:00:00Z` or a JSON value  {"utc": "2009-11-10T23:00:00Z"}. in Either case we follow jsonapi specifications and `hub` will wrap the result and add other parts and return a complete result like this:

   ```json
   {"data": "2009-11-10T23:00:00Z"}
   or
   {"data": {"utc": "2009-11-10T23:00:00Z"}}
   ```

   - If there was an error we want to return this format:

   ```json
      {
      "errors": [
         {
            "status": "422",
            "title":  "Invalid Attribute",
            "detail": "First name must contain at least two characters."
         }
      ]
      }
   ```

5. **Logging:**
   - Utilize the `slog` package for logging.
   - Output logs to stdout in JSON format.
   - Log key events, such as:
     - Server startup and shutdown.
     - Incoming requests.
     - Errors.

6. **Error Handling:**
   - Return raw errors (not structured JSON).
   - Log errors using `slog`.

7. **Configuration:**
   - Use Go structs for configuration with default values (e.g., port, log level).
   - Example configuration struct:
     ```go
     type Config struct {
         Port     string // Default: "8080"
         LogLevel string // Default: "info"
     }
     ```
   - We should be able to pass the config to endpoint imeplmentations while initiating their objects.

8. **Project Structure:**
   - Main file: `cmd/hub/main.go`
   - Lib file: `internal/hub/hub.go`
   - Endpoint implementations: `internal/hub/<endpoint>/<endpoint>.go`
   - The hub itself will be defined at`internal/hub/hub.go` along its test file. We include this file in `cmd/hub/main.go`.


9. **Documentation:**
   - Generate Markdown documentation for the service, including:
     - Overview of the hub.
     - How to add new endpoints.
     - Configuration options.
     - Usage examples.
   - documentation will be at `docs/hub.md`

### Instructions

- **Step 0:** Create the main service file at `cmd/hub/main.go`. This hold the main package to run the service.
- **Step 1:** Implementation for the service and its test will be at `internal/hub/hub.go` and  `internal/hub/hub_test.go`
- **Step 2:** Define the interface that each endpoint must implement.
- **Step 3:** Implement the logic to handle REST, and SSE requests.
   - REST: Use `http.HandlerFunc` for handling.
   - SSE: Implement SSE using `http.ResponseWriter` with appropriate headers.
- **Step 4:** Set up logging using `slog` with JSON output to stdout.
- **Step 5:** Implement basic error handling and logging.
- **Step 6:** Define a configuration struct with default values.
- **Step 7:** Ensure the service can be extended to include multiple endpoints.
- **Step 8:** Generate Markdown documentation for the service.

### Notes

- Do not implement specific endpoints in this prompt; focus on the hub.
- Ensure the code is modular and follows Go best practices.
- Use native Go libraries when possible instead of 3rd party libraries.
- Ensure consistency by adhering to the defined interface and project structure.
- Practice defensive coding:
  - Validate configuration values.
  - Handle errors gracefully.
  - Log key events and errors.
- To find how the current setup work it is better if you read content of go.mod first.
- After applying the changes make sure test pass.
- The example Date endpoint: This endpoint returns the time.RFC3339 for its results.
- I expect `curl http://localhost:8080/date` and `curl http://localhost:8080/date/stream` both to return a reply like: `{"data":{"UTC":"2025-02-27T12:31:34Z"}}`

### File related to this spefication:
The project can hold many files which might be unrelated to your tasks.
For this specification you will deal with the following files:

File you will own and change if needed:
- cmd/hub/main.go
- docs/hub.md
- internal/hub/hub.go
- internal/hub/hub_test.go
- internal/date/date.go
- internal/date/date_test.go

File you will review to learn about details you need and dependencies:
- go.mod
