package httpx

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// HTTPClient is an interface that defines the methods required for making HTTP requests.
// This allows for easier testing and mocking of HTTP requests in unit tests.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// GenericClient is a generic HTTP client that can handle requests and responses with type safety.
// It wraps an HTTPClient and provides methods for executing typed HTTP requests.
type GenericClient[T any] struct {
	httpClient HTTPClient
	// Configuration fields for building HTTP client
	customClient          HTTPClient // If set, use this instead of building one
	maxIdleConns          *int
	idleConnTimeout       *time.Duration
	tlsHandshakeTimeout   *time.Duration
	expectContinueTimeout *time.Duration
	maxIdleConnsPerHost   *int
	timeout               *time.Duration
	maxRetries            *int
	retryBaseDelay        *time.Duration
	retryMaxDelay         *time.Duration
	retryStrategy         *Strategy
	disableKeepAlive      *bool
	proxyURL              *string      // Proxy URL (e.g., "http://proxy.example.com:8080")
	logger                *slog.Logger // Optional logger (nil = no logging)
}

// GenericClientOption is a function type for configuring the GenericClient.
type GenericClientOption[T any] func(*GenericClient[T])

// Response represents the response from an HTTP request with generic type support.
type Response[T any] struct {
	Data       T
	Headers    http.Header
	RawBody    []byte
	StatusCode int
}

// ErrorResponse represents an error response from the API.
type ErrorResponse struct {
	Message    string `json:"message,omitempty"`
	ErrorMsg   string `json:"error,omitempty"`
	Details    string `json:"details,omitempty"`
	StatusCode int    `json:"statusCode,omitempty"`
}

// Error implements the error interface for ErrorResponse.
// It returns a human-readable error message that includes the HTTP status code
// and any available error details from the API response.
func (e *ErrorResponse) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("http %d: %s", e.StatusCode, e.Message)
	}

	if e.ErrorMsg != "" {
		return fmt.Sprintf("http %d: %s", e.StatusCode, e.ErrorMsg)
	}

	return fmt.Sprintf("http %d: request failed", e.StatusCode)
}

// NewGenericClient creates a new generic HTTP client with the specified type.
// By default, it builds an HTTP client using ClientBuilder with default settings.
// Use the provided options to customize the client behavior.
// If WithHTTPClient is used, it takes precedence over all other configuration options.
func NewGenericClient[T any](options ...GenericClientOption[T]) *GenericClient[T] {
	client := &GenericClient[T]{}

	// Apply options
	for _, option := range options {
		option(client)
	}

	// If a custom HTTP client was provided, use it
	if client.customClient != nil {
		client.httpClient = client.customClient
		return client
	}

	// Otherwise, build an HTTP client using ClientBuilder with the configured options
	builder := NewClientBuilder()

	// Apply configuration if set
	if client.maxIdleConns != nil {
		builder.WithMaxIdleConns(*client.maxIdleConns)
	}

	if client.idleConnTimeout != nil {
		builder.WithIdleConnTimeout(*client.idleConnTimeout)
	}

	if client.tlsHandshakeTimeout != nil {
		builder.WithTLSHandshakeTimeout(*client.tlsHandshakeTimeout)
	}

	if client.expectContinueTimeout != nil {
		builder.WithExpectContinueTimeout(*client.expectContinueTimeout)
	}

	if client.maxIdleConnsPerHost != nil {
		builder.WithMaxIdleConnsPerHost(*client.maxIdleConnsPerHost)
	}

	if client.timeout != nil {
		builder.WithTimeout(*client.timeout)
	}

	if client.maxRetries != nil {
		builder.WithMaxRetries(*client.maxRetries)
	}

	if client.retryBaseDelay != nil {
		builder.WithRetryBaseDelay(*client.retryBaseDelay)
	}

	if client.retryMaxDelay != nil {
		builder.WithRetryMaxDelay(*client.retryMaxDelay)
	}

	if client.retryStrategy != nil {
		builder.WithRetryStrategy(*client.retryStrategy)
	}

	if client.disableKeepAlive != nil {
		builder.WithDisableKeepAlive(*client.disableKeepAlive)
	}

	if client.logger != nil {
		builder.WithLogger(client.logger)
	}

	if client.proxyURL != nil {
		builder.WithProxy(*client.proxyURL)
	}

	client.httpClient = builder.Build()
	return client
}

// WithHTTPClient configures the generic client to use a custom HTTPClient implementation.
// If httpClient is nil, the option is ignored and the client retains its current HTTPClient.
// This option takes precedence over all other configuration options.
// This is useful for using a pre-configured retry client or custom transport.
func WithHTTPClient[T any](httpClient HTTPClient) GenericClientOption[T] {
	return func(c *GenericClient[T]) {
		if httpClient != nil {
			c.customClient = httpClient
		}
	}
}

// WithTimeout sets the request timeout for the generic client.
// Uses ClientBuilder validation and defaults if the value is out of range.
func WithTimeout[T any](timeout time.Duration) GenericClientOption[T] {
	return func(c *GenericClient[T]) {
		c.timeout = &timeout
	}
}

// WithMaxIdleConns sets the maximum number of idle connections.
// Uses ClientBuilder validation and defaults if the value is out of range.
func WithMaxIdleConns[T any](maxIdleConns int) GenericClientOption[T] {
	return func(c *GenericClient[T]) {
		c.maxIdleConns = &maxIdleConns
	}
}

// WithIdleConnTimeout sets the idle connection timeout.
// Uses ClientBuilder validation and defaults if the value is out of range.
func WithIdleConnTimeout[T any](idleConnTimeout time.Duration) GenericClientOption[T] {
	return func(c *GenericClient[T]) {
		c.idleConnTimeout = &idleConnTimeout
	}
}

// WithTLSHandshakeTimeout sets the TLS handshake timeout.
// Uses ClientBuilder validation and defaults if the value is out of range.
func WithTLSHandshakeTimeout[T any](tlsHandshakeTimeout time.Duration) GenericClientOption[T] {
	return func(c *GenericClient[T]) {
		c.tlsHandshakeTimeout = &tlsHandshakeTimeout
	}
}

// WithExpectContinueTimeout sets the expect continue timeout.
// Uses ClientBuilder validation and defaults if the value is out of range.
func WithExpectContinueTimeout[T any](expectContinueTimeout time.Duration) GenericClientOption[T] {
	return func(c *GenericClient[T]) {
		c.expectContinueTimeout = &expectContinueTimeout
	}
}

// WithDisableKeepAlive sets whether to disable keep-alive.
func WithDisableKeepAlive[T any](disableKeepAlive bool) GenericClientOption[T] {
	return func(c *GenericClient[T]) {
		c.disableKeepAlive = &disableKeepAlive
	}
}

// WithMaxIdleConnsPerHost sets the maximum number of idle connections per host.
// Uses ClientBuilder validation and defaults if the value is out of range.
func WithMaxIdleConnsPerHost[T any](maxIdleConnsPerHost int) GenericClientOption[T] {
	return func(c *GenericClient[T]) {
		c.maxIdleConnsPerHost = &maxIdleConnsPerHost
	}
}

// WithMaxRetries sets the maximum number of retry attempts.
// Uses ClientBuilder validation and defaults if the value is out of range.
func WithMaxRetries[T any](maxRetries int) GenericClientOption[T] {
	return func(c *GenericClient[T]) {
		c.maxRetries = &maxRetries
	}
}

// WithRetryBaseDelay sets the base delay for retry strategies.
// Uses ClientBuilder validation and defaults if the value is out of range.
func WithRetryBaseDelay[T any](baseDelay time.Duration) GenericClientOption[T] {
	return func(c *GenericClient[T]) {
		c.retryBaseDelay = &baseDelay
	}
}

// WithRetryMaxDelay sets the maximum delay for retry strategies.
// Uses ClientBuilder validation and defaults if the value is out of range.
func WithRetryMaxDelay[T any](maxDelay time.Duration) GenericClientOption[T] {
	return func(c *GenericClient[T]) {
		c.retryMaxDelay = &maxDelay
	}
}

// WithRetryStrategy sets the retry strategy type (fixed, jitter, or exponential).
// Uses ClientBuilder validation and defaults if the value is invalid.
func WithRetryStrategy[T any](strategy Strategy) GenericClientOption[T] {
	return func(c *GenericClient[T]) {
		c.retryStrategy = &strategy
	}
}

// WithRetryStrategyAsString sets the retry strategy type from a string.
// Valid values: "fixed", "jitter", "exponential".
// Uses ClientBuilder validation and defaults if the value is invalid.
func WithRetryStrategyAsString[T any](strategy string) GenericClientOption[T] {
	return func(c *GenericClient[T]) {
		s := Strategy(strategy)
		c.retryStrategy = &s
	}
}

// WithLogger sets the logger for logging HTTP operations (retries, errors, etc.).
// Pass nil to disable logging (default behavior).
func WithLogger[T any](logger *slog.Logger) GenericClientOption[T] {
	return func(c *GenericClient[T]) {
		c.logger = logger
	}
}

// WithProxy sets the proxy URL for HTTP requests.
// The proxy URL should be in the format "http://proxy.example.com:8080" or "https://proxy.example.com:8080".
// Pass an empty string to disable proxy (default behavior).
func WithProxy[T any](proxyURL string) GenericClientOption[T] {
	return func(c *GenericClient[T]) {
		c.proxyURL = &proxyURL
	}
}

// Execute performs an HTTP request and returns a typed response.
// It executes the request, reads the response body,
// and unmarshals the JSON response into the generic type T.
// Returns an error if the HTTP status code is >= 400.
func (c *GenericClient[T]) Execute(req *http.Request) (*Response[T], error) {
	// Log raw request details
	if c.logger != nil {
		c.logger.Debug("Executing HTTP request",
			"method", req.Method,
			"url", req.URL.String(),
		)
		c.logger.Debug("Request headers",
			"headers", req.Header,
		)

		// Log request body if present
		if req.Body != nil {
			body, err := io.ReadAll(req.Body)
			req.Body.Close()
			if err == nil {
				c.logger.Debug("Request body (raw)",
					"body", string(body),
					"length", len(body),
				)
				// Restore the body for actual request
				req.Body = io.NopCloser(strings.NewReader(string(body)))
			}
		}
	}

	// Execute the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute http request: %w", err)
	}
	defer resp.Body.Close()

	// Log raw response details
	if c.logger != nil {
		c.logger.Debug("Received HTTP response",
			"status", resp.Status,
			"status_code", resp.StatusCode,
			"url", req.URL.String(),
			"method", req.Method,
		)
		c.logger.Debug("Response headers",
			"headers", resp.Header,
		)
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	// Log raw response body
	if c.logger != nil {
		c.logger.Debug("Response body (raw)",
			"body", string(body),
			"length", len(body),
			"content_type", resp.Header.Get("Content-Type"),
		)
	}

	// Check for HTTP errors
	if resp.StatusCode >= 400 {
		return nil, c.handleErrorResponse(resp.StatusCode, body)
	}

	// Parse the response
	response := &Response[T]{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		RawBody:    body,
	}

	// Unmarshal JSON response if body is not empty
	if len(body) > 0 {
		if err := json.Unmarshal(body, &response.Data); err != nil {
			return nil, fmt.Errorf("unmarshal response json: %w", err)
		}
	}

	return response, nil
}

// ExecuteRaw performs an HTTP request and returns the raw response without unmarshaling.
// This is useful when you need direct access to the http.Response, such as for streaming
// or when the response is not JSON.
func (c *GenericClient[T]) ExecuteRaw(req *http.Request) (*http.Response, error) {
	// Execute the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}

	return resp, nil
}

// Do performs an HTTP request and returns a typed response.
// This method is designed to work seamlessly with the RequestBuilder.
// It's an alias for Execute but with a more familiar name for those used to http.Client.Do().
func (c *GenericClient[T]) Do(req *http.Request) (*Response[T], error) {
	return c.Execute(req)
}

// Get performs a GET request to the specified URL and returns a typed response.
func (c *GenericClient[T]) Get(url string) (*Response[T], error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create GET request: %w", err)
	}

	return c.Execute(req)
}

// Post performs a POST request with the specified body and returns a typed response.
func (c *GenericClient[T]) Post(url string, body io.Reader) (*Response[T], error) {
	req, err := http.NewRequest(http.MethodPost, url, body)
	if err != nil {
		return nil, fmt.Errorf("create POST request: %w", err)
	}

	return c.Execute(req)
}

// Put performs a PUT request with the specified body and returns a typed response.
func (c *GenericClient[T]) Put(url string, body io.Reader) (*Response[T], error) {
	req, err := http.NewRequest(http.MethodPut, url, body)
	if err != nil {
		return nil, fmt.Errorf("create PUT request: %w", err)
	}

	return c.Execute(req)
}

// Delete performs a DELETE request and returns a typed response.
func (c *GenericClient[T]) Delete(url string) (*Response[T], error) {
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create DELETE request: %w", err)
	}

	return c.Execute(req)
}

// Patch performs a PATCH request with the specified body and returns a typed response.
func (c *GenericClient[T]) Patch(url string, body io.Reader) (*Response[T], error) {
	req, err := http.NewRequest(http.MethodPatch, url, body)
	if err != nil {
		return nil, fmt.Errorf("create PATCH request: %w", err)
	}

	return c.Execute(req)
}

// handleErrorResponse handles HTTP error responses.
// It attempts to unmarshal the error response as JSON, and if that fails,
// uses the raw body as the error message.
func (c *GenericClient[T]) handleErrorResponse(statusCode int, body []byte) error {
	errorResp := &ErrorResponse{
		StatusCode: statusCode,
	}

	// Try to unmarshal error response
	if len(body) > 0 {
		if err := json.Unmarshal(body, errorResp); err != nil {
			// If unmarshaling fails, use raw body as message
			errorResp.Message = string(body)
		}
	}

	// Set default message if none provided
	if errorResp.Message == "" && errorResp.ErrorMsg == "" {
		errorResp.Message = http.StatusText(statusCode)
	}

	return errorResp
}
