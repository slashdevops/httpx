package httpx

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

// Test data structures
type TestData struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

func TestNewRequestBuilder(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		wantURL string
	}{
		{
			name:    "simple URL",
			baseURL: "https://api.example.com",
			wantURL: "https://api.example.com",
		},
		{
			name:    "URL with path",
			baseURL: "https://api.example.com/v1",
			wantURL: "https://api.example.com/v1",
		},
		{
			name:    "URL with trailing slash",
			baseURL: "https://api.example.com/",
			wantURL: "https://api.example.com/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb := NewRequestBuilder(tt.baseURL)

			if rb == nil {
				t.Fatal("NewRequestBuilder returned nil")
			}

			if rb.baseURL != tt.wantURL {
				t.Errorf("NewRequestBuilder() baseURL = %v, want %v", rb.baseURL, tt.wantURL)
			}

			if rb.queryParams == nil {
				t.Error("NewRequestBuilder() queryParams not initialized")
			}

			if rb.headers == nil {
				t.Error("NewRequestBuilder() headers not initialized")
			}

			if rb.ctx == nil {
				t.Error("NewRequestBuilder() context not initialized")
			}
		})
	}
}

func TestRequestBuilder_HTTPMethods(t *testing.T) {
	rb := NewRequestBuilder("https://api.example.com")

	tests := []struct {
		name     string
		function func() *RequestBuilder
		expected string
	}{
		{"GET", rb.WithMethodGET, http.MethodGet},
		{"POST", rb.WithMethodPOST, http.MethodPost},
		{"PUT", rb.WithMethodPUT, http.MethodPut},
		{"DELETE", rb.WithMethodDELETE, http.MethodDelete},
		{"PATCH", rb.WithMethodPATCH, http.MethodPatch},
		{"HEAD", rb.WithMethodHEAD, http.MethodHead},
		{"OPTIONS", rb.WithMethodOPTIONS, http.MethodOptions},
		{"TRACE", rb.WithMethodTRACE, http.MethodTrace},
		{"CONNECT", rb.WithMethodCONNECT, http.MethodConnect},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.function()
			if result.method != tt.expected {
				t.Errorf("%s() method = %v, want %v", tt.name, result.method, tt.expected)
			}
			// Ensure fluent interface returns same instance
			if result != rb {
				t.Errorf("%s() returned different instance", tt.name)
			}
		})
	}
}

func TestRequestBuilder_HTTPMethod(t *testing.T) {
	tests := []struct {
		name        string
		method      string
		expectError bool
	}{
		{"Valid GET", "GET", false},
		{"Valid POST", "POST", false},
		{"Valid lowercase get", "get", false},
		{"Valid with spaces", " PUT ", false},
		{"Empty method", "", true},
		{"Invalid method", "INVALID", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb := NewRequestBuilder("https://api.example.com")
			rb.WithMethod(tt.method)

			if tt.expectError {
				if !rb.HasErrors() {
					t.Errorf("Expected error for method %s, but got none", tt.method)
				}
			} else {
				if rb.HasErrors() {
					t.Errorf("Unexpected error for method %s: %v", tt.method, rb.GetErrors())
				}
				expectedMethod := strings.ToUpper(strings.TrimSpace(tt.method))
				if rb.method != expectedMethod {
					t.Errorf("Expected method %s, got %s", expectedMethod, rb.method)
				}
			}
		})
	}
}

func TestRequestBuilder_HTTPMethodCustom(t *testing.T) {
	rb := NewRequestBuilder("https://api.example.com")

	customMethod := "CUSTOM"
	result := rb.WithMethod(customMethod)

	// Custom methods are now validated and should cause an error
	if !result.HasErrors() {
		t.Error("HTTPMethod() should reject invalid custom method")
	}

	if result != rb {
		t.Error("HTTPMethod() returned different instance")
	}
}

func TestRequestBuilder_WithPath(t *testing.T) {
	rb := NewRequestBuilder("https://api.example.com")

	path := "/users/123"
	result := rb.WithPath(path)

	if result.path != path {
		t.Errorf("WithPath() path = %v, want %v", result.path, path)
	}

	if result != rb {
		t.Error("WithPath() returned different instance")
	}
}

func TestRequestBuilder_WithQueryParam(t *testing.T) {
	rb := NewRequestBuilder("https://api.example.com")

	result := rb.WithQueryParam("key1", "value1").WithQueryParam("key2", "value2")

	if len(result.queryParams) != 2 {
		t.Errorf("WithQueryParam() queryParams length = %v, want %v", len(result.queryParams), 2)
	}

	if result.queryParams.Get("key1") != "value1" {
		t.Errorf("WithQueryParam() key1 = %v, want %v", result.queryParams.Get("key1"), "value1")
	}

	if result.queryParams.Get("key2") != "value2" {
		t.Errorf("WithQueryParam() key2 = %v, want %v", result.queryParams.Get("key2"), "value2")
	}

	if result != rb {
		t.Error("WithQueryParam() returned different instance")
	}
}

func TestRequestBuilder_QueryParam_SpecialCharacters(t *testing.T) {
	tests := []struct {
		name          string
		key           string
		value         string
		expectedInURL string
		description   string
	}{
		{
			name:          "spaces in value",
			key:           "query",
			value:         "hello world",
			expectedInURL: "query=hello+world",
			description:   "spaces should be encoded as +",
		},
		{
			name:          "equals sign",
			key:           "filter",
			value:         "a=b",
			expectedInURL: "filter=a%3Db",
			description:   "= should be encoded as %3D",
		},
		{
			name:          "ampersand",
			key:           "data",
			value:         "a&b",
			expectedInURL: "data=a%26b",
			description:   "&& should be encoded as %26",
		},
		{
			name:          "question mark",
			key:           "test",
			value:         "what?",
			expectedInURL: "test=what%3F",
			description:   "? should be encoded as %3F",
		},
		{
			name:          "double quotes",
			key:           "quote",
			value:         `"test"`,
			expectedInURL: "quote=%22test%22",
			description:   `" should be encoded as %22`,
		},
		{
			name:          "single quotes",
			key:           "quote",
			value:         "'test'",
			expectedInURL: "quote=%27test%27",
			description:   "' should be encoded as %27",
		},
		{
			name:          "plus sign",
			key:           "math",
			value:         "1+1",
			expectedInURL: "math=1%2B1",
			description:   "+ should be encoded as %2B",
		},
		{
			name:          "JQL-like query",
			key:           "jql",
			value:         `project = "TEST" AND type = Requirement`,
			expectedInURL: "jql=project+%3D+%22TEST%22+AND+type+%3D+Requirement",
			description:   "complex JQL query should be properly encoded",
		},
		{
			name:          "parentheses",
			key:           "expr",
			value:         "(a,b,c)",
			expectedInURL: "expr=%28a%2Cb%2Cc%29",
			description:   "parentheses and commas should be encoded",
		},
		{
			name:          "multiple special characters",
			key:           "complex",
			value:         `a=b&c=d?e="f"`,
			expectedInURL: `complex=a%3Db%26c%3Dd%3Fe%3D%22f%22`,
			description:   "multiple special characters should all be encoded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb := NewRequestBuilder("https://api.example.com").
				WithMethodGET().
				WithQueryParam(tt.key, tt.value)

			req, err := rb.Build()
			if err != nil {
				t.Fatalf("Build() failed: %v", err)
			}

			actualQuery := req.URL.RawQuery
			if actualQuery != tt.expectedInURL {
				t.Errorf("%s\nGot:      %s\nExpected: %s", tt.description, actualQuery, tt.expectedInURL)
			}
		})
	}
}

func TestRequestBuilder_QueryParams(t *testing.T) {
	rb := NewRequestBuilder("https://api.example.com")

	params := map[string]string{
		"param1": "value1",
		"param2": "value2",
		"param3": "value3",
	}

	result := rb.WithQueryParams(params)

	if len(result.queryParams) != 3 {
		t.Errorf("QueryParams() queryParams length = %v, want %v", len(result.queryParams), 3)
	}

	for key, expectedValue := range params {
		if result.queryParams.Get(key) != expectedValue {
			t.Errorf("QueryParams() %s = %v, want %v", key, result.queryParams.Get(key), expectedValue)
		}
	}

	if result != rb {
		t.Error("QueryParams() returned different instance")
	}
}

func TestRequestBuilder_WithHeader(t *testing.T) {
	rb := NewRequestBuilder("https://api.example.com")

	result := rb.WithHeader("Content-Type", "application/json").WithHeader("Accept", "application/json")

	if len(result.headers) != 2 {
		t.Errorf("WithHeader() headers length = %v, want %v", len(result.headers), 2)
	}

	if result.headers["Content-Type"] != "application/json" {
		t.Errorf("WithHeader() Content-Type = %v, want %v", result.headers["Content-Type"], "application/json")
	}

	if result.headers["Accept"] != "application/json" {
		t.Errorf("WithHeader() Accept = %v, want %v", result.headers["Accept"], "application/json")
	}

	if result != rb {
		t.Error("WithHeader() returned different instance")
	}
}

func TestRequestBuilder_Headers(t *testing.T) {
	rb := NewRequestBuilder("https://api.example.com")

	headers := map[string]string{
		"Content-Type": "application/json",
		"Accept":       "application/json",
		"User-Agent":   "test-client/1.0",
	}

	result := rb.WithHeaders(headers)

	if len(result.headers) != 3 {
		t.Errorf("Headers() headers length = %v, want %v", len(result.headers), 3)
	}

	for key, expectedValue := range headers {
		if result.headers[key] != expectedValue {
			t.Errorf("Headers() %s = %v, want %v", key, result.headers[key], expectedValue)
		}
	}

	if result != rb {
		t.Error("Headers() returned different instance")
	}
}

func TestRequestBuilder_WithBasicAuth(t *testing.T) {
	rb := NewRequestBuilder("https://api.example.com")

	username := "user"
	password := "pass"
	result := rb.WithBasicAuth(username, password)

	expectedAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(username+":"+password))

	if result.headers["Authorization"] != expectedAuth {
		t.Errorf("WithBasicAuth() Authorization = %v, want %v", result.headers["Authorization"], expectedAuth)
	}

	if result != rb {
		t.Error("WithBasicAuth() returned different instance")
	}
}

func TestRequestBuilder_WithBearerAuth(t *testing.T) {
	rb := NewRequestBuilder("https://api.example.com")

	token := "abc123token"
	result := rb.WithBearerAuth(token)

	expectedAuth := "Bearer " + token

	if result.headers["Authorization"] != expectedAuth {
		t.Errorf("WithBearerAuth() Authorization = %v, want %v", result.headers["Authorization"], expectedAuth)
	}

	if result != rb {
		t.Error("WithBearerAuth() returned different instance")
	}
}

func TestRequestBuilder_WithUserAgent(t *testing.T) {
	rb := NewRequestBuilder("https://api.example.com")

	userAgent := "test-client/1.0"
	result := rb.WithUserAgent(userAgent)

	if result.headers["User-Agent"] != userAgent {
		t.Errorf("WithUserAgent() User-Agent = %v, want %v", result.headers["User-Agent"], userAgent)
	}

	if result != rb {
		t.Error("WithUserAgent() returned different instance")
	}
}

func TestRequestBuilder_WithContentType(t *testing.T) {
	rb := NewRequestBuilder("https://api.example.com")

	contentType := "application/json"
	result := rb.WithContentType(contentType)

	if result.headers["Content-Type"] != contentType {
		t.Errorf("WithContentType() Content-Type = %v, want %v", result.headers["Content-Type"], contentType)
	}

	if result != rb {
		t.Error("WithContentType() returned different instance")
	}
}

func TestRequestBuilder_Accept(t *testing.T) {
	tests := []struct {
		name   string
		accept string
	}{
		{
			name:   "application/json",
			accept: "application/json",
		},
		{
			name:   "application/xml",
			accept: "application/xml",
		},
		{
			name:   "text/html",
			accept: "text/html",
		},
		{
			name:   "multiple types with quality",
			accept: "application/json, text/html;q=0.9, */*;q=0.8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb := NewRequestBuilder("https://api.example.com")
			result := rb.WithAccept(tt.accept)

			if result.headers["Accept"] != tt.accept {
				t.Errorf("Accept() Accept = %v, want %v", result.headers["Accept"], tt.accept)
			}

			if result != rb {
				t.Error("Accept() returned different instance")
			}

			// Build and verify the request header is set
			req, err := result.WithMethodGET().Build()
			if err != nil {
				t.Fatalf("Build() failed: %v", err)
			}

			if req.Header.Get("Accept") != tt.accept {
				t.Errorf("Built request Accept header = %v, want %v", req.Header.Get("Accept"), tt.accept)
			}
		})
	}
}

func TestRequestBuilder_WithJSONBody(t *testing.T) {
	rb := NewRequestBuilder("https://api.example.com")

	testData := TestData{Name: "test", Value: 42}
	result := rb.WithJSONBody(testData)

	if result.body != testData {
		t.Errorf("WithJSONBody() body = %v, want %v", result.body, testData)
	}

	if result.bodyReader != nil {
		t.Error("WithJSONBody() bodyReader should be nil when body is set")
	}

	if result.headers["Content-Type"] != "application/json" {
		t.Errorf("WithJSONBody() Content-Type = %v, want %v", result.headers["Content-Type"], "application/json")
	}

	if result != rb {
		t.Error("WithJSONBody() returned different instance")
	}
}

func TestRequestBuilder_RawBody(t *testing.T) {
	rb := NewRequestBuilder("https://api.example.com")

	body := strings.NewReader("raw body content")
	result := rb.WithRawBody(body)

	if result.bodyReader != body {
		t.Errorf("RawBody() bodyReader = %v, want %v", result.bodyReader, body)
	}

	if result.body != nil {
		t.Error("RawBody() body should be nil when bodyReader is set")
	}

	if result != rb {
		t.Error("RawBody() returned different instance")
	}
}

func TestRequestBuilder_WithStringBody(t *testing.T) {
	rb := NewRequestBuilder("https://api.example.com")

	bodyContent := "string body content"
	result := rb.WithStringBody(bodyContent)

	if result.bodyReader == nil {
		t.Error("WithStringBody() bodyReader should not be nil")
	}

	if result.body != nil {
		t.Error("WithStringBody() body should be nil when bodyReader is set")
	}

	// Read the content to verify
	if result.bodyReader != nil {
		content, err := io.ReadAll(result.bodyReader)
		if err != nil {
			t.Fatalf("Failed to read bodyReader: %v", err)
		}
		if string(content) != bodyContent {
			t.Errorf("WithStringBody() content = %v, want %v", string(content), bodyContent)
		}
	}

	if result != rb {
		t.Error("WithStringBody() returned different instance")
	}
}

func TestRequestBuilder_BytesBody(t *testing.T) {
	rb := NewRequestBuilder("https://api.example.com")

	bodyContent := []byte("bytes body content")
	result := rb.WithBytesBody(bodyContent)

	if result.bodyReader == nil {
		t.Error("BytesBody() bodyReader should not be nil")
	}

	if result.body != nil {
		t.Error("BytesBody() body should be nil when bodyReader is set")
	}

	// Read the content to verify
	if result.bodyReader != nil {
		content, err := io.ReadAll(result.bodyReader)
		if err != nil {
			t.Fatalf("Failed to read bodyReader: %v", err)
		}
		if !bytes.Equal(content, bodyContent) {
			t.Errorf("BytesBody() content = %v, want %v", content, bodyContent)
		}
	}

	if result != rb {
		t.Error("BytesBody() returned different instance")
	}
}

type contextKey string

func TestRequestBuilder_Context(t *testing.T) {
	rb := NewRequestBuilder("https://api.example.com")

	ctx := context.WithValue(context.Background(), contextKey("testkey"), "value")
	result := rb.WithContext(ctx)

	if result.ctx != ctx {
		t.Errorf("Context() ctx = %v, want %v", result.ctx, ctx)
	}

	if result != rb {
		t.Error("Context() returned different instance")
	}
}

func TestRequestBuilder_Build_Success(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *RequestBuilder
		validate func(t *testing.T, req *http.Request)
	}{
		{
			name: "simple GET request",
			setup: func() *RequestBuilder {
				return NewRequestBuilder("https://api.example.com").WithMethodGET()
			},
			validate: func(t *testing.T, req *http.Request) {
				if req.Method != http.MethodGet {
					t.Errorf("Expected method GET, got %s", req.Method)
				}
				if req.URL.String() != "https://api.example.com" {
					t.Errorf("Expected URL https://api.example.com, got %s", req.URL.String())
				}
			},
		},
		{
			name: "POST with JSON body",
			setup: func() *RequestBuilder {
				testData := TestData{Name: "test", Value: 42}
				return NewRequestBuilder("https://api.example.com").
					WithMethodPOST().
					WithPath("/users").
					WithJSONBody(testData)
			},
			validate: func(t *testing.T, req *http.Request) {
				if req.Method != http.MethodPost {
					t.Errorf("Expected method POST, got %s", req.Method)
				}
				if req.URL.String() != "https://api.example.com/users" {
					t.Errorf("Expected URL https://api.example.com/users, got %s", req.URL.String())
				}
				if req.Header.Get("Content-Type") != "application/json" {
					t.Errorf("Expected Content-Type application/json, got %s", req.Header.Get("Content-Type"))
				}

				// Verify body content
				if req.Body != nil {
					bodyBytes, err := io.ReadAll(req.Body)
					if err != nil {
						t.Fatalf("Failed to read request body: %v", err)
					}

					var testData TestData
					if err := json.Unmarshal(bodyBytes, &testData); err != nil {
						t.Fatalf("Failed to unmarshal JSON body: %v", err)
					}

					if testData.Name != "test" || testData.Value != 42 {
						t.Errorf("Unexpected body content: %+v", testData)
					}
				}
			},
		},
		{
			name: "GET with query parameters and headers",
			setup: func() *RequestBuilder {
				return NewRequestBuilder("https://api.example.com").
					WithMethodGET().
					WithPath("/search").
					WithQueryParam("q", "golang").
					WithQueryParam("limit", "10").
					WithHeader("Accept", "application/json").
					WithUserAgent("test-client/1.0")
			},
			validate: func(t *testing.T, req *http.Request) {
				expected := "https://api.example.com/search?limit=10&q=golang"
				if req.URL.String() != expected {
					t.Errorf("Expected URL %s, got %s", expected, req.URL.String())
				}
				if req.Header.Get("Accept") != "application/json" {
					t.Errorf("Expected Accept header application/json, got %s", req.Header.Get("Accept"))
				}
				if req.Header.Get("User-Agent") != "test-client/1.0" {
					t.Errorf("Expected User-Agent test-client/1.0, got %s", req.Header.Get("User-Agent"))
				}
			},
		},
		{
			name: "PUT with string body",
			setup: func() *RequestBuilder {
				return NewRequestBuilder("https://api.example.com").
					WithMethodPUT().
					WithPath("/data").
					WithStringBody("plain text content").
					WithContentType("text/plain")
			},
			validate: func(t *testing.T, req *http.Request) {
				if req.Method != http.MethodPut {
					t.Errorf("Expected method PUT, got %s", req.Method)
				}
				if req.Header.Get("Content-Type") != "text/plain" {
					t.Errorf("Expected Content-Type text/plain, got %s", req.Header.Get("Content-Type"))
				}
			},
		},
		{
			name: "request with basic auth",
			setup: func() *RequestBuilder {
				return NewRequestBuilder("https://api.example.com").
					WithMethodGET().
					WithBasicAuth("user", "pass")
			},
			validate: func(t *testing.T, req *http.Request) {
				expected := "Basic " + base64.StdEncoding.EncodeToString([]byte("user:pass"))
				if req.Header.Get("Authorization") != expected {
					t.Errorf("Expected Authorization %s, got %s", expected, req.Header.Get("Authorization"))
				}
			},
		},
		{
			name: "request with bearer auth",
			setup: func() *RequestBuilder {
				return NewRequestBuilder("https://api.example.com").
					WithMethodGET().
					WithBearerAuth("token123")
			},
			validate: func(t *testing.T, req *http.Request) {
				expected := "Bearer token123"
				if req.Header.Get("Authorization") != expected {
					t.Errorf("Expected Authorization %s, got %s", expected, req.Header.Get("Authorization"))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb := tt.setup()
			req, err := rb.Build()
			if err != nil {
				t.Fatalf("Build() failed: %v", err)
			}

			if req == nil {
				t.Fatal("Build() returned nil request")
			}

			tt.validate(t, req)
		})
	}
}

func TestRequestBuilder_Build_Errors(t *testing.T) {
	tests := []struct {
		name    string
		setup   func() *RequestBuilder
		wantErr string
	}{
		{
			name: "missing HTTP method",
			setup: func() *RequestBuilder {
				return NewRequestBuilder("https://api.example.com")
			},
			wantErr: "HTTP method must be specified",
		},
		{
			name: "invalid base URL",
			setup: func() *RequestBuilder {
				return NewRequestBuilder("://invalid-url").WithMethodGET()
			},
			wantErr: "invalid base URL",
		},
		{
			name: "JSON marshal error",
			setup: func() *RequestBuilder {
				// Create a struct with an invalid JSON field (channel)
				invalidData := make(chan int)
				return NewRequestBuilder("https://api.example.com").
					WithMethodPOST().
					WithJSONBody(invalidData)
			},
			wantErr: "failed to marshal JSON body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb := tt.setup()
			req, err := rb.Build()

			if err == nil {
				t.Fatalf("Build() expected error containing %q, got nil", tt.wantErr)
			}

			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Build() error = %v, want error containing %q", err, tt.wantErr)
			}

			if req != nil {
				t.Error("Build() should return nil request on error")
			}
		})
	}
}

func TestRequestBuilder_Build_GetBody(t *testing.T) {
	testData := TestData{Name: "test", Value: 42}
	rb := NewRequestBuilder("https://api.example.com").
		WithMethodPOST().
		WithJSONBody(testData)

	req, err := rb.Build()
	if err != nil {
		t.Fatalf("Build() failed: %v", err)
	}

	// Test GetBody function for retry support
	if req.GetBody == nil {
		t.Error("Build() should set GetBody for JSON requests")
		return
	}

	// Call GetBody to get a new body reader
	bodyReader, err := req.GetBody()
	if err != nil {
		t.Fatalf("GetBody() failed: %v", err)
	}

	// Read and verify the body content
	bodyBytes, err := io.ReadAll(bodyReader)
	if err != nil {
		t.Fatalf("Failed to read body from GetBody(): %v", err)
	}

	var decodedData TestData
	if err := json.Unmarshal(bodyBytes, &decodedData); err != nil {
		t.Fatalf("Failed to unmarshal JSON from GetBody(): %v", err)
	}

	if decodedData != testData {
		t.Errorf("GetBody() returned different data: got %+v, want %+v", decodedData, testData)
	}

	// Close the reader
	if err := bodyReader.Close(); err != nil {
		t.Errorf("Failed to close body reader: %v", err)
	}
}

func Test_basicAuth(t *testing.T) {
	tests := []struct {
		name     string
		username string
		password string
		want     string
	}{
		{
			name:     "simple credentials",
			username: "user",
			password: "pass",
			want:     base64.StdEncoding.EncodeToString([]byte("user:pass")),
		},
		{
			name:     "empty username",
			username: "",
			password: "pass",
			want:     base64.StdEncoding.EncodeToString([]byte(":pass")),
		},
		{
			name:     "empty password",
			username: "user",
			password: "",
			want:     base64.StdEncoding.EncodeToString([]byte("user:")),
		},
		{
			name:     "both empty",
			username: "",
			password: "",
			want:     base64.StdEncoding.EncodeToString([]byte(":")),
		},
		{
			name:     "special characters",
			username: "user@domain.com",
			password: "p@ss:w0rd!",
			want:     base64.StdEncoding.EncodeToString([]byte("user@domain.com:p@ss:w0rd!")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := basicAuth(tt.username, tt.password)
			if got != tt.want {
				t.Errorf("basicAuth() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_base64Encode(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want string
	}{
		{
			name: "simple text",
			data: []byte("hello world"),
			want: base64.StdEncoding.EncodeToString([]byte("hello world")),
		},
		{
			name: "empty data",
			data: []byte(""),
			want: "",
		},
		{
			name: "binary data",
			data: []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD},
			want: base64.StdEncoding.EncodeToString([]byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}),
		},
		{
			name: "unicode characters",
			data: []byte("Hello, 世界! 🌍"),
			want: base64.StdEncoding.EncodeToString([]byte("Hello, 世界! 🌍")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := base64Encode(tt.data)
			if got != tt.want {
				t.Errorf("base64Encode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRequestBuilder_JiraJQLQuery(t *testing.T) {
	tests := []struct {
		name          string
		jql           string
		expectedInURL string
		description   string
	}{
		{
			name:          "simple JQL query",
			jql:           `project = "TEST"`,
			expectedInURL: `jql=project+%3D+%22TEST%22`,
			description:   "simple JQL with project filter",
		},
		{
			name:          "JQL with AND operator",
			jql:           `project = "TEST" AND type = Requirement`,
			expectedInURL: `jql=project+%3D+%22TEST%22+AND+type+%3D+Requirement`,
			description:   "JQL with AND operator should encode spaces and equals signs",
		},
		{
			name:          "JQL with IN clause",
			jql:           `status IN (Accepted, Active)`,
			expectedInURL: `jql=status+IN+%28Accepted%2C+Active%29`,
			description:   "JQL with IN clause should encode parentheses and commas",
		},
		{
			name:          "JQL with NOT IN clause",
			jql:           `status NOT IN (Obsolete, Cancelled)`,
			expectedInURL: `jql=status+NOT+IN+%28Obsolete%2C+Cancelled%29`,
			description:   "JQL with NOT IN clause",
		},
		{
			name:          "JQL with date comparison",
			jql:           `statusCategoryChangedDate >= '2024-01-01'`,
			expectedInURL: `jql=statusCategoryChangedDate+%3E%3D+%272024-01-01%27`,
			description:   "JQL with date and >= operator should encode properly",
		},
		{
			name:          "complex JQL query",
			jql:           `project = "TEST" AND type = Requirement AND status NOT IN (Obsolete, Cancelled) AND status IN (Accepted, Active) AND statusCategoryChangedDate >= '2024-01-01' AND statusCategoryChangedDate < '2024-02-01' ORDER BY statusCategoryChangedDate DESC`,
			expectedInURL: `jql=project+%3D+%22TEST%22+AND+type+%3D+Requirement+AND+status+NOT+IN+%28Obsolete%2C+Cancelled%29+AND+status+IN+%28Accepted%2C+Active%29+AND+statusCategoryChangedDate+%3E%3D+%272024-01-01%27+AND+statusCategoryChangedDate+%3C+%272024-02-01%27+ORDER+BY+statusCategoryChangedDate+DESC`,
			description:   "complex JQL query with multiple clauses",
		},
		{
			name:          "JQL with special characters in text",
			jql:           `summary ~ "bug & issue"`,
			expectedInURL: `jql=summary+~+%22bug+%26+issue%22`,
			description:   "JQL with ampersand in text search",
		},
		{
			name:          "JQL with < operator",
			jql:           `created < '2024-01-01'`,
			expectedInURL: `jql=created+%3C+%272024-01-01%27`,
			description:   "JQL with less-than operator",
		},
		{
			name:          "JQL with <= operator",
			jql:           `created <= '2024-01-01'`,
			expectedInURL: `jql=created+%3C%3D+%272024-01-01%27`,
			description:   "JQL with less-than-or-equal operator",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb := NewRequestBuilder("https://company.atlassian.net/rest/api/3").
				WithMethodGET().
				WithPath("/search").
				WithQueryParam("jql", tt.jql)

			req, err := rb.Build()
			if err != nil {
				t.Fatalf("Build() failed: %v", err)
			}

			actualQuery := req.URL.RawQuery
			if actualQuery != tt.expectedInURL {
				t.Errorf("%s\nJQL:      %s\nGot:      %s\nExpected: %s",
					tt.description, tt.jql, actualQuery, tt.expectedInURL)
			}

			// Verify the URL can be parsed back correctly
			parsedURL, err := req.URL.Parse(req.URL.String())
			if err != nil {
				t.Fatalf("Failed to parse built URL: %v", err)
			}

			// Verify we can extract the JQL parameter back
			actualJQL := parsedURL.Query().Get("jql")
			if actualJQL != tt.jql {
				t.Errorf("Round-trip failed:\nOriginal: %s\nAfter:    %s", tt.jql, actualJQL)
			}
		})
	}
}

// TestRequestBuilder_NewHTTPMethods tests the newly added HTTP method convenience functions
func TestRequestBuilder_NewHTTPMethods(t *testing.T) {
	tests := []struct {
		name     string
		builder  func() *RequestBuilder
		expected string
	}{
		{
			name:     "HEAD method",
			builder:  func() *RequestBuilder { return NewRequestBuilder("https://api.example.com").WithMethodHEAD() },
			expected: http.MethodHead,
		},
		{
			name:     "OPTIONS method",
			builder:  func() *RequestBuilder { return NewRequestBuilder("https://api.example.com").WithMethodOPTIONS() },
			expected: http.MethodOptions,
		},
		{
			name:     "TRACE method",
			builder:  func() *RequestBuilder { return NewRequestBuilder("https://api.example.com").WithMethodTRACE() },
			expected: http.MethodTrace,
		},
		{
			name:     "CONNECT method",
			builder:  func() *RequestBuilder { return NewRequestBuilder("https://api.example.com").WithMethodCONNECT() },
			expected: http.MethodConnect,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb := tt.builder()

			// Check the method is set correctly
			if rb.method != tt.expected {
				t.Errorf("Expected method %s, got %s", tt.expected, rb.method)
			}

			// Build the request and verify
			req, err := rb.WithPath("/test").Build()
			if err != nil {
				t.Fatalf("Build() failed: %v", err)
			}

			if req.Method != tt.expected {
				t.Errorf("Built request method = %s, want %s", req.Method, tt.expected)
			}
		})
	}
}

// TestRequestBuilder_AllHTTPMethodsIntegration tests all HTTP methods in a single integration test
func TestRequestBuilder_AllHTTPMethodsIntegration(t *testing.T) {
	baseURL := "https://api.example.com"
	path := "/resource"

	methods := map[string]struct {
		builder  func(*RequestBuilder) *RequestBuilder
		expected string
	}{
		"GET":     {builder: (*RequestBuilder).WithMethodGET, expected: http.MethodGet},
		"POST":    {builder: (*RequestBuilder).WithMethodPOST, expected: http.MethodPost},
		"PUT":     {builder: (*RequestBuilder).WithMethodPUT, expected: http.MethodPut},
		"DELETE":  {builder: (*RequestBuilder).WithMethodDELETE, expected: http.MethodDelete},
		"PATCH":   {builder: (*RequestBuilder).WithMethodPATCH, expected: http.MethodPatch},
		"HEAD":    {builder: (*RequestBuilder).WithMethodHEAD, expected: http.MethodHead},
		"OPTIONS": {builder: (*RequestBuilder).WithMethodOPTIONS, expected: http.MethodOptions},
		"TRACE":   {builder: (*RequestBuilder).WithMethodTRACE, expected: http.MethodTrace},
		"CONNECT": {builder: (*RequestBuilder).WithMethodCONNECT, expected: http.MethodConnect},
	}

	for name, tc := range methods {
		t.Run(name, func(t *testing.T) {
			rb := NewRequestBuilder(baseURL)
			rb = tc.builder(rb)
			rb = rb.WithPath(path)

			req, err := rb.Build()
			if err != nil {
				t.Fatalf("Build() failed for %s: %v", name, err)
			}

			if req.Method != tc.expected {
				t.Errorf("Method = %s, want %s", req.Method, tc.expected)
			}

			if req.URL.Path != path {
				t.Errorf("Path = %s, want %s", req.URL.Path, path)
			}

			if req.URL.Host != "api.example.com" {
				t.Errorf("Host = %s, want api.example.com", req.URL.Host)
			}
		})
	}
}
