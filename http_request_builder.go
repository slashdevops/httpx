package httpx

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strings"
)

// RequestBuilder provides a fluent API for building HTTP requests with and without body.
type RequestBuilder struct {
	method      string
	baseURL     string
	path        string
	queryParams url.Values
	headers     map[string]string
	body        any
	bodyReader  io.Reader
	ctx         context.Context
	errors      []error
}

// NewRequestBuilder creates a new RequestBuilder with the specified base URL.
func NewRequestBuilder(baseURL string) *RequestBuilder {
	return &RequestBuilder{
		baseURL:     baseURL,
		queryParams: make(url.Values),
		headers:     make(map[string]string),
		ctx:         context.Background(),
		errors:      make([]error, 0),
	}
}

// WithMethod sets the HTTP method to the specified method.
// The method is normalized to uppercase and validated against standard HTTP methods.
func (rb *RequestBuilder) WithMethod(method string) *RequestBuilder {
	if method == "" {
		rb.addError(fmt.Errorf("http method cannot be empty"))

		return rb
	}

	method = strings.ToUpper(strings.TrimSpace(method))
	if !isValidHTTPMethod(method) {
		rb.addError(fmt.Errorf("invalid http method: %s", method))

		return rb
	}

	rb.method = method

	return rb
}

// WithMethodGET sets the HTTP method to GET.
func (rb *RequestBuilder) WithMethodGET() *RequestBuilder {
	rb.method = http.MethodGet

	return rb
}

// WithMethodPOST sets the HTTP method to POST.
func (rb *RequestBuilder) WithMethodPOST() *RequestBuilder {
	rb.method = http.MethodPost

	return rb
}

// WithMethodPUT sets the HTTP method to PUT.
func (rb *RequestBuilder) WithMethodPUT() *RequestBuilder {
	rb.method = http.MethodPut

	return rb
}

// WithMethodDELETE sets the HTTP method to DELETE.
func (rb *RequestBuilder) WithMethodDELETE() *RequestBuilder {
	rb.method = http.MethodDelete

	return rb
}

// WithMethodPATCH sets the HTTP method to PATCH.
func (rb *RequestBuilder) WithMethodPATCH() *RequestBuilder {
	rb.method = http.MethodPatch

	return rb
}

// WithMethodHEAD sets the HTTP method to HEAD.
func (rb *RequestBuilder) WithMethodHEAD() *RequestBuilder {
	rb.method = http.MethodHead

	return rb
}

// WithMethodOPTIONS sets the HTTP method to OPTIONS.
func (rb *RequestBuilder) WithMethodOPTIONS() *RequestBuilder {
	rb.method = http.MethodOptions

	return rb
}

// WithMethodTRACE sets the HTTP method to TRACE.
func (rb *RequestBuilder) WithMethodTRACE() *RequestBuilder {
	rb.method = http.MethodTrace

	return rb
}

// WithMethodCONNECT sets the HTTP method to CONNECT.
func (rb *RequestBuilder) WithMethodCONNECT() *RequestBuilder {
	rb.method = http.MethodConnect

	return rb
}

// WithPath sets the path component of the URL.
func (rb *RequestBuilder) WithPath(path string) *RequestBuilder {
	rb.path = path

	return rb
}

// WithQueryParam adds a single query parameter.
func (rb *RequestBuilder) WithQueryParam(key, value string) *RequestBuilder {
	if key == "" {
		rb.addError(fmt.Errorf("query parameter key cannot be empty"))

		return rb
	}

	if value == "" {
		rb.addError(fmt.Errorf("query parameter value for key '%s' cannot be empty", key))

		return rb
	}

	// Validate query key format
	if strings.ContainsAny(key, " \t\n\r=&") {
		rb.addError(fmt.Errorf("invalid query parameter key format: '%s' (contains invalid characters)", key))

		return rb
	}

	rb.queryParams.Add(key, value)

	return rb
}

// WithQueryParams adds multiple query parameters from a map.
func (rb *RequestBuilder) WithQueryParams(params map[string]string) *RequestBuilder {
	for key, value := range params {
		rb.queryParams.Add(key, value)
	}

	return rb
}

// WithHeader sets a single header.
func (rb *RequestBuilder) WithHeader(key, value string) *RequestBuilder {
	if key == "" {
		rb.addError(fmt.Errorf("header key cannot be empty"))

		return rb
	}

	if value == "" {
		rb.addError(fmt.Errorf("header value for key '%s' cannot be empty", key))

		return rb
	}

	// Validate header key format
	if strings.ContainsAny(key, " \t\n\r") {
		rb.addError(fmt.Errorf("invalid header key format: '%s' (contains whitespace)", key))

		return rb
	}

	rb.headers[key] = value

	return rb
}

// WithHeaders sets multiple headers from a map.
func (rb *RequestBuilder) WithHeaders(headers map[string]string) *RequestBuilder {
	maps.Copy(rb.headers, headers)

	return rb
}

// WithBasicAuth sets the Authorization header for basic authentication.
func (rb *RequestBuilder) WithBasicAuth(username, password string) *RequestBuilder {
	if username == "" {
		rb.addError(fmt.Errorf("username for basic auth cannot be empty"))

		return rb
	}

	if password == "" {
		rb.addError(fmt.Errorf("password for basic auth cannot be empty"))

		return rb
	}

	rb.headers["Authorization"] = "Basic " + basicAuth(username, password)

	return rb
}

// WithBearerAuth sets the Authorization header for bearer token authentication.
func (rb *RequestBuilder) WithBearerAuth(token string) *RequestBuilder {
	if token == "" {
		rb.addError(fmt.Errorf("bearer token cannot be empty"))

		return rb
	}

	rb.headers["Authorization"] = "Bearer " + token

	return rb
}

// WithUserAgent sets the User-Agent header.
// The user agent is trimmed and validated to ensure it:
// - is non-empty after trimming
// - does not exceed 500 characters
// - does not contain control characters (\r, \n, \t)
func (rb *RequestBuilder) WithUserAgent(userAgent string) *RequestBuilder {
	if userAgent == "" {
		rb.addError(fmt.Errorf("user-agent cannot be empty"))

		return rb
	}

	// Trim whitespace
	userAgent = strings.TrimSpace(userAgent)
	if userAgent == "" {
		rb.addError(fmt.Errorf("user-agent cannot be empty after trimming whitespace"))

		return rb
	}

	// Validate length
	if len(userAgent) > 500 {
		rb.addError(fmt.Errorf("user-agent is too long (max 500 characters), got %d characters", len(userAgent)))

		return rb
	}

	// Validate that User-Agent doesn't contain control characters
	if strings.ContainsAny(userAgent, "\r\n\t") {
		rb.addError(fmt.Errorf("user-agent cannot contain control characters (\\r, \\n, \\t)"))

		return rb
	}

	rb.headers["User-Agent"] = userAgent

	return rb
}

// WithContentType sets the Content-Type header.
func (rb *RequestBuilder) WithContentType(contentType string) *RequestBuilder {
	return rb.WithHeader("Content-Type", contentType)
}

// WithAccept sets the Accept header.
func (rb *RequestBuilder) WithAccept(accept string) *RequestBuilder {
	return rb.WithHeader("Accept", accept)
}

// WithJSONBody sets the request body as JSON and sets the appropriate Content-Type header.
func (rb *RequestBuilder) WithJSONBody(body any) *RequestBuilder {
	rb.body = body
	rb.bodyReader = nil
	rb.WithContentType("application/json")

	return rb
}

// WithRawBody sets the request body from an io.Reader.
func (rb *RequestBuilder) WithRawBody(body io.Reader) *RequestBuilder {
	rb.bodyReader = body
	rb.body = nil

	return rb
}

// WithStringBody sets the request body from a string.
func (rb *RequestBuilder) WithStringBody(body string) *RequestBuilder {
	rb.bodyReader = strings.NewReader(body)
	rb.body = nil

	return rb
}

// WithBytesBody sets the request body from a byte slice.
func (rb *RequestBuilder) WithBytesBody(body []byte) *RequestBuilder {
	rb.bodyReader = bytes.NewReader(body)
	rb.body = nil

	return rb
}

// WithContext sets the context for the request.
func (rb *RequestBuilder) WithContext(ctx context.Context) *RequestBuilder {
	if ctx == nil {
		rb.addError(fmt.Errorf("context cannot be nil"))
		return rb
	}

	rb.ctx = ctx

	return rb
}

// Build creates an *http.Request from the builder configuration.
// Returns an error if any validation fails.
func (rb *RequestBuilder) Build() (*http.Request, error) {
	// Check for any errors accumulated during building
	if len(rb.errors) > 0 {
		return nil, fmt.Errorf("request builder errors: %v", rb.errors)
	}

	// Validate method
	if rb.method == "" {
		return nil, fmt.Errorf("HTTP method must be specified")
	}

	// Build URL
	u, err := url.Parse(rb.baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	// Validate URL has scheme and host
	if u.Scheme == "" {
		return nil, fmt.Errorf("base URL must include a scheme (http or https)")
	}

	if u.Host == "" {
		return nil, fmt.Errorf("base URL must include a host")
	}

	// Validate scheme
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("unsupported url scheme: %s (only http and https are supported)", u.Scheme)
	}

	// Add path
	if rb.path != "" {
		u.Path = strings.TrimSuffix(u.Path, "/") + "/" + strings.TrimPrefix(rb.path, "/")
	}

	// Add query parameters
	if len(rb.queryParams) > 0 {
		q := u.Query()

		for key, values := range rb.queryParams {
			for _, value := range values {
				q.Add(key, value)
			}
		}

		u.RawQuery = q.Encode()
	}

	// Prepare body
	var bodyReader io.Reader
	if rb.body != nil {
		jsonData, err := json.Marshal(rb.body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal JSON body: %w", err)
		}

		bodyReader = bytes.NewReader(jsonData)
	} else if rb.bodyReader != nil {
		bodyReader = rb.bodyReader
	}

	// Create request
	req, err := http.NewRequestWithContext(rb.ctx, rb.method, u.String(), bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	for key, value := range rb.headers {
		req.Header.Set(key, value)
	}

	// Set GetBody for retry support if we have a body
	if bodyReader != nil && rb.body != nil {
		// For JSON bodies, we can recreate the body
		req.GetBody = func() (io.ReadCloser, error) {
			jsonData, err := json.Marshal(rb.body)
			if err != nil {
				return nil, err
			}

			return io.NopCloser(bytes.NewReader(jsonData)), nil
		}
	}

	return req, nil
}

// basicAuth encodes username and password for basic authentication.
func basicAuth(username, password string) string {
	auth := username + ":" + password

	return base64Encode([]byte(auth))
}

// base64Encode encodes bytes to base64 string.
func base64Encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// addError adds an error to the error collection.
func (rb *RequestBuilder) addError(err error) {
	if err != nil {
		rb.errors = append(rb.errors, err)
	}
}

// GetErrors returns all accumulated errors during the building process.
func (rb *RequestBuilder) GetErrors() []error {
	return rb.errors
}

// HasErrors returns true if there are any accumulated errors.
func (rb *RequestBuilder) HasErrors() bool {
	return len(rb.errors) > 0
}

// Reset clears all errors and resets the builder to a clean state.
func (rb *RequestBuilder) Reset() *RequestBuilder {
	rb.errors = make([]error, 0)
	rb.method = ""
	rb.path = ""
	rb.queryParams = make(url.Values)
	rb.headers = make(map[string]string)
	rb.body = nil
	rb.bodyReader = nil
	rb.ctx = context.Background()

	return rb
}

// isValidHTTPMethod checks if the provided method is a valid HTTP method.
func isValidHTTPMethod(method string) bool {
	validMethods := []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodDelete,
		http.MethodPatch,
		http.MethodHead,
		http.MethodOptions,
		http.MethodTrace,
		http.MethodConnect,
	}

	return slices.Contains(validMethods, method)
}
