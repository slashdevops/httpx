// Package httpx provides comprehensive utilities for building and executing HTTP requests
// with advanced features including fluent request building, automatic retry logic,
// and type-safe generic clients.
//
// Requirements: Go 1.22 or higher is required to use this package.
//
// Zero Dependencies: This package is built entirely using the Go standard library,
// with no external dependencies. This ensures maximum reliability, security, and
// minimal maintenance overhead for your projects.
//
// This package is designed to simplify HTTP client development in Go by providing:
//   - A fluent, chainable API for building HTTP requests with validation
//   - Type-safe HTTP clients using Go generics for automatic JSON marshaling
//   - Configurable retry logic with multiple backoff strategies
//   - Comprehensive error handling and validation
//   - Production-ready defaults with full customization support
//   - Zero external dependencies - built purely with Go standard library
//
// # Quick Start
//
// Build and execute a simple GET request:
//
//	req, err := httpx.NewRequestBuilder("https://api.example.com").
//	    WithMethodGET().
//	    Path("/users/123").
//	    Header("Accept", "application/json").
//	    Build()
//
// Use type-safe generic client:
//
//	type User struct {
//	    ID    int    `json:"id"`
//	    Name  string `json:"name"`
//	}
//
//	client := httpx.NewGenericClient[User](
//	    httpx.//	)
//	response, err := client.Get("/users/123")
//	fmt.Printf("User: %s\n", response.Data.Name)
//
// # Request Builder
//
// The RequestBuilder provides a fluent API for constructing HTTP requests with
// comprehensive input validation and error accumulation. All inputs are validated
// before the request is built, ensuring early error detection.
//
// Basic usage:
//
//	req, err := httpx.NewRequestBuilder("https://api.example.com").
//	    WithMethodPOST().
//	    Path("/users").
//	    QueryParam("notify", "true").
//	    Header("Content-Type", "application/json").
//	    BearerAuth("your-token-here").
//	    JSONBody(user).
//	    Build()
//
// Request builder features:
//   - HTTP methods: GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS, TRACE, CONNECT
//   - Convenience methods: WithMethodGET, WithMethodPOST, WithMethodPUT, WithMethodDELETE, WithMethodPATCH, WithMethodHEAD, WithMethodOPTIONS, WithMethodTRACE, WithMethodCONNECT
//   - Query parameters with automatic URL encoding and validation
//   - Custom headers with format validation
//   - Authentication: Basic Auth and Bearer Token with validation
//   - Multiple body formats: JSON (auto-marshal), string, bytes, io.Reader
//   - Context support for timeouts and cancellation
//   - Input validation with error accumulation
//   - Detailed error messages indicating what failed
//   - Reset and reuse builder
//
// Validation features:
//
// The builder validates all inputs and accumulates errors, allowing you to
// detect multiple issues at once:
//
//	builder := httpx.NewRequestBuilder("https://api.example.com")
//	builder.HTTPMethod("")           // Error: empty method
//	builder.WithHeader("", "value")      // Error: empty header key
//	builder.WithQueryParam("key=", "val") // Error: invalid character in key
//
//	// Check accumulated errors
//	if builder.HasErrors() {
//	    for _, err := range builder.GetErrors() {
//	        log.Printf("Validation error: %v", err)
//	    }
//	}
//
//	// Or let Build() report all errors
//	req, err := builder.Build() // Returns all accumulated errors
//
// Builder reuse:
//
//	builder := httpx.NewRequestBuilder("https://api.example.com")
//	req1, _ := builder.WithWithMethodGET().WithPath("/users").Build()
//	builder.Reset() // Clear state
//	req2, _ := builder.WithWithMethodPOST().WithPath("/posts").Build()
//
// # Generic HTTP Client
//
// The GenericClient provides type-safe HTTP requests using Go generics with
// automatic JSON marshaling and unmarshaling. This eliminates the need for
// manual type assertions and reduces boilerplate code.
//
// Basic usage:
//
//	type User struct {
//	    ID    int    `json:"id"`
//	    Name  string `json:"name"`
//	    Email string `json:"email"`
//	}
//
//	client := httpx.NewGenericClient[User](
//	    httpx.WithTimeout[User](10*time.Second),
//	    httpx.WithMaxRetries[User](3),
//	    httpx.WithRetryStrategy[User](httpx.ExponentialBackoffStrategy),
//	)
//
//	// GET request - response.Data is strongly typed as User
//	response, err := client.Get("/users/1")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("User: %s (%s)\n", response.Data.Name, response.Data.Email)
//
// Generic client features:
//   - Type-safe responses with automatic JSON unmarshaling
//   - Compile-time type checking for response data
//   - Convenience methods: Get, Post, Put, Delete, Patch
//   - Execute method for custom requests (works with RequestBuilder)
//   - ExecuteRaw for non-JSON responses (images, files, etc.)
//   - Flexible configuration via option pattern
//   - Built-in retry logic with configurable strategies
//   - Connection pooling and timeout configuration
//   - TLS handshake and idle connection timeout settings
//   - Structured error responses with ErrorResponse type
//   - Full integration with ClientBuilder for complex configurations
//   - Debug logging support (uses slog)
//
// Configuration options:
//   - WithTimeout: Set request timeout
//   - WithMaxRetries: Set maximum retry attempts
//   - WithRetryStrategy: Configure retry strategy (fixed, jitter, exponential)
//   - WithRetryBaseDelay: Set base delay for retry strategies
//   - WithRetryMaxDelay: Set maximum delay for retry strategies
//   - WithMaxIdleConns: Set maximum idle connections
//   - WithIdleConnTimeout: Set idle connection timeout
//   - WithTLSHandshakeTimeout: Set TLS handshake timeout
//   - WithExpectContinueTimeout: Set expect continue timeout
//   - WithMaxIdleConnsPerHost: Set maximum idle connections per host
//   - WithDisableKeepAlive: Disable HTTP keep-alive
//   - WithProxy: Configure HTTP/HTTPS proxy server
//   - WithHTTPClient: Use a pre-configured HTTP client (takes precedence)
//
// Integration with RequestBuilder:
//
//	req, err := httpx.NewRequestBuilder("https://api.example.com").
//	    WithMethodPOST().
//	    Path("/users").
//	    ContentType("application/json").
//	    Header("X-Request-ID", "unique-123").
//	    JSONBody(newUser).
//	    Build()
//
//	response, err := client.Execute(req) // Type-safe execution
//
// Multiple typed clients:
//
//	userClient := httpx.NewGenericClient[User](...)
//	postClient := httpx.NewGenericClient[Post](...)
//
//	user, _ := userClient.Get("/users/1")
//	posts, _ := postClient.Get(fmt.Sprintf("/users/%d/posts", user.Data.ID))
//
// Error handling:
//
//	response, err := client.Get("/users/999")
//	if err != nil {
//	    if apiErr, ok := err.(*httpx.ErrorResponse); ok {
//	        // Structured API error
//	        fmt.Printf("API Error %d: %s\n", apiErr.StatusCode, apiErr.Message)
//	    } else {
//	        // Network error, parsing error, etc.
//	        log.Printf("Request failed: %v\n", err)
//	    }
//	}
//
// # Retry Logic
//
// The package provides transparent retry logic that automatically retries failed
// requests using configurable backoff strategies. Retry logic preserves all
// request properties including headers and authentication.
//
// What gets retried:
//   - Network errors (connection failures, timeouts)
//   - HTTP 5xx server errors (500-599)
//   - HTTP 429 (Too Many Requests)
//
// What does NOT get retried:
//   - HTTP 4xx client errors (except 429)
//   - HTTP 2xx/3xx successful responses
//   - Requests without GetBody (non-replayable)
//
// Available retry strategies:
//
//  1. Exponential Backoff (recommended for most use cases):
//
//     strategy := httpx.ExponentialBackoff(500*time.Millisecond, 10*time.Second)
//     // Wait times: 500ms → 1s → 2s → 4s → 8s (capped at maxDelay)
//
//  2. Fixed Delay (useful for predictable retry timing):
//
//     strategy := httpx.FixedDelay(1*time.Second)
//     // Wait times: 1s → 1s → 1s
//
//  3. Jitter Backoff (prevents thundering herd problem):
//
//     strategy := httpx.JitterBackoff(500*time.Millisecond, 10*time.Second)
//     // Wait times: random(0-500ms) → random(0-1s) → random(0-2s)
//
// Direct usage (advanced):
//
//	client := httpx.NewHTTPRetryClient(
//	    httpx.WithMaxRetriesRetry(3),
//	    httpx.WithRetryStrategyRetry(httpx.ExponentialBackoff(500*time.Millisecond, 10*time.Second)),
//	    httpx.WithBaseTransport(http.DefaultTransport),
//	)
//
// # Client Builder
//
// The ClientBuilder provides a fluent API for configuring HTTP clients with
// retry logic, timeouts, and connection pooling. All settings are validated
// and default to production-ready values if out of range.
//
// Basic configuration:
//
//	client := httpx.NewClientBuilder().
//	    WithTimeout(30 * time.Second).
//	    WithMaxRetries(3).
//	    WithRetryStrategy(httpx.ExponentialBackoffStrategy).
//	    Build()
//
// Advanced configuration:
//
//	client := httpx.NewClientBuilder().
//	    // Timeouts
//	    WithTimeout(30 * time.Second).
//	    WithIdleConnTimeout(90 * time.Second).
//	    WithTLSHandshakeTimeout(10 * time.Second).
//	    WithExpectContinueTimeout(1 * time.Second).
//
//	    // Connection pooling
//	    WithMaxIdleConns(100).
//	    WithMaxIdleConnsPerHost(10).
//	    WithDisableKeepAlive(false).
//
//	    // Retry configuration
//	    WithMaxRetries(3).
//	    WithRetryStrategy(httpx.ExponentialBackoffStrategy).
//	    WithRetryBaseDelay(500 * time.Millisecond).
//	    WithRetryMaxDelay(10 * time.Second).
//
//	    // Proxy configuration
//	    WithProxy("http://proxy.example.com:8080").
//
//	    Build()
//
// Combine with GenericClient:
//
//	retryClient := httpx.NewClientBuilder().
//	    WithMaxRetries(3).
//	    WithRetryStrategy(httpx.ExponentialBackoffStrategy).
//	    Build()
//
//	client := httpx.NewGenericClient[User](
//	    httpx.WithHTTPClient[User](retryClient),
//	    httpx.//	)
//
// Default values (validated and adjusted if out of range):
//   - Timeout: 5 seconds (valid: 1s-30s)
//   - MaxRetries: 3 (valid: 1-10)
//   - RetryBaseDelay: 500ms (valid: 300ms-5s)
//   - RetryMaxDelay: 10s (valid: 300ms-120s)
//   - MaxIdleConns: 100 (valid: 1-200)
//   - IdleConnTimeout: 90s (valid: 1s-120s)
//   - TLSHandshakeTimeout: 10s (valid: 1s-15s)
//
// # Error Handling
//
// The package provides comprehensive error handling with specific error types:
//
//  1. ErrorResponse (from GenericClient):
//     Structured API errors with status code and message
//
//  2. ClientError:
//     Generic HTTP client operation errors
//
//  3. RequestBuilder validation errors:
//     Accumulated during building, reported at Build() time
//
// Error handling examples:
//
//	// Generic client errors
//	response, err := client.Get("/users/999")
//	if err != nil {
//	    if apiErr, ok := err.(*httpx.ErrorResponse); ok {
//	        switch apiErr.StatusCode {
//	        case 404:
//	            log.Printf("Not found: %s", apiErr.Message)
//	        case 401:
//	            log.Printf("Unauthorized: %s", apiErr.Message)
//	        case 429:
//	            log.Printf("Rate limited: %s", apiErr.Message)
//	        default:
//	            log.Printf("API error %d: %s", apiErr.StatusCode, apiErr.Message)
//	        }
//	    } else {
//	        log.Printf("Network error: %v", err)
//	    }
//	}
//
//	// Builder validation errors
//	builder := httpx.NewRequestBuilder("https://api.example.com")
//	builder.WithHeader("", "value") // Invalid
//	if builder.HasErrors() {
//	    for _, err := range builder.GetErrors() {
//	        log.Printf("Validation: %v", err)
//	    }
//	}
//
// # Best Practices
//
//  1. Use type-safe clients for JSON APIs:
//
//     client := httpx.NewGenericClient[User](...)
//     response, err := client.Get("/users/1")
//     // response.Data is User, not interface{}
//
//  2. Configure retry logic for production:
//
//     retryClient := httpx.NewClientBuilder().
//     WithMaxRetries(3).
//     WithRetryStrategy(httpx.ExponentialBackoffStrategy).
//     Build()
//
//  3. Reuse HTTP clients (they're safe for concurrent use):
//
//     client := httpx.NewGenericClient[User](...)
//     // Use from multiple goroutines safely
//
//  4. Use contexts for timeouts and cancellation:
//
//     ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//     defer cancel()
//     req, _ := builder.WithContext(ctx).Build()
//
//  5. Validate before building:
//
//     if builder.HasErrors() {
//     // Handle validation errors
//     }
//
//  6. Handle API errors appropriately:
//
//     if apiErr, ok := err.(*httpx.ErrorResponse); ok {
//     // Handle specific status codes
//     }
//
// # Proxy Configuration
//
// The package provides comprehensive proxy support for all HTTP clients.
// Proxy configuration works transparently across all client types and
// supports both HTTP and HTTPS proxies with optional authentication.
//
// Basic proxy configuration with ClientBuilder:
//
//	client := httpx.NewClientBuilder().
//	    WithProxy("http://proxy.example.com:8080").
//	    Build()
//
// HTTPS proxy:
//
//	client := httpx.NewClientBuilder().
//	    WithProxy("https://secure-proxy.example.com:3128").
//	    Build()
//
// Proxy with authentication:
//
//	client := httpx.NewClientBuilder().
//	    WithProxy("http://username:password@proxy.example.com:8080").
//	    Build()
//
// Proxy with GenericClient:
//
//	client := httpx.NewGenericClient[User](
//	    httpx.WithProxy[User]("http://proxy.example.com:8080"),
//	    httpx.WithTimeout[User](10*time.Second),
//	    httpx.WithMaxRetries[User](3),
//	)
//
// Proxy with retry client:
//
//	client := httpx.NewHTTPRetryClient(
//	    httpx.WithProxyRetry("http://proxy.example.com:8080"),
//	    httpx.WithMaxRetriesRetry(5),
//	)
//
// Disable proxy (override environment variables):
//
//	client := httpx.NewClientBuilder().
//	    WithProxy(""). // Empty string disables proxy
//	    Build()
//
// Common proxy ports:
//   - HTTP proxy: 8080, 3128, 8888
//   - HTTPS proxy: 3128, 8443
//   - SOCKS proxy: 1080 (not directly supported, use custom transport)
//
// The proxy configuration:
//   - Works transparently with all request types
//   - Preserves all headers and authentication
//   - Compatible with retry logic
//   - Supports connection pooling
//   - Respects timeout settings
//   - Validates proxy URL format
//   - Falls back gracefully on invalid URLs
//
// # Thread Safety
//
// All utilities in this package are safe for concurrent use across multiple goroutines:
//   - RequestBuilder instances should not be shared between goroutines
//   - GenericClient instances are safe for concurrent use
//   - HTTP clients built by ClientBuilder are safe for concurrent use
//   - Retry logic preserves request immutability
//
// Example concurrent usage:
//
//	client := httpx.NewGenericClient[User](...)
//
//	var wg sync.WaitGroup
//	for i := 1; i <= 10; i++ {
//	    wg.Add(1)
//	    go func(id int) {
//	        defer wg.Done()
//	        user, err := client.Get(fmt.Sprintf("/users/%d", id))
//	        // Process user
//	    }(i)
//	}
//	wg.Wait()
//
// # Debugging
//
// The package uses slog for debug logging. Enable debug logging to see:
//   - Request details (method, URL, headers, body)
//   - Response details (status, headers, body)
//   - Retry attempts and delays
//
// Enable debug logging:
//
//	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
//	    Level: slog.LevelDebug,
//	}))
//	slog.SetDefault(logger)
//
// # Documentation
//
// You can view the full documentation and examples locally by running:
//
//	go doc -http=:8080
//
// Then navigate to http://localhost:8080/pkg/github.com/slashdevops/httpx/
// in your browser to browse the complete documentation, examples, and source code.
//
// # See Also
//
// For complete examples and API reference, see the README.md file or visit:
// https://pkg.go.dev/github.com/slashdevops/httpx
package httpx
