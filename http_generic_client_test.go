package httpx

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Test data structures for generic client tests
type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Post struct {
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	UserID int    `json:"userId"`
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// TestNewGenericClient tests the creation of a new generic client
func TestNewGenericClient(t *testing.T) {
	t.Run("Default client", func(t *testing.T) {
		client := NewGenericClient[User]()

		if client == nil {
			t.Fatal("NewGenericClient returned nil")
		}

		if client.httpClient == nil {
			t.Error("httpClient should be initialized")
		}
	})

	t.Run("With custom timeout", func(t *testing.T) {
		timeout := 5 * time.Second
		client := NewGenericClient[User](
			WithTimeout[User](timeout),
		)

		if httpClient, ok := client.httpClient.(*http.Client); ok {
			if httpClient.Timeout != timeout {
				t.Errorf("Expected timeout %v, got %v", timeout, httpClient.Timeout)
			}
		} else {
			t.Error("httpClient should be *http.Client")
		}
	})

	t.Run("With custom HTTP client", func(t *testing.T) {
		customClient := &http.Client{
			Timeout: 10 * time.Second,
		}
		client := NewGenericClient[User](
			WithHTTPClient[User](customClient),
		)

		if client.httpClient != customClient {
			t.Error("Expected custom HTTP client to be used")
		}
	})

	t.Run("With nil HTTP client (should be ignored)", func(t *testing.T) {
		client := NewGenericClient[User](
			WithHTTPClient[User](nil),
		)

		if client.httpClient == nil {
			t.Error("httpClient should not be nil when nil option is provided")
		}
	})
}

// TestGenericClient_Execute tests the Execute method
func TestGenericClient_Execute(t *testing.T) {
	t.Run("Successful GET request", func(t *testing.T) {
		expectedUser := User{ID: 1, Name: "John Doe", Email: "john@example.com"}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("Expected GET method, got %s", r.Method)
			}
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(expectedUser); err != nil {
				t.Errorf("Failed to encode response: %v", err)
			}
		}))
		defer server.Close()

		client := NewGenericClient[User]()
		req, _ := http.NewRequest(http.MethodGet, server.URL, nil)

		resp, err := client.Execute(req)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
		}

		if resp.Data.ID != expectedUser.ID {
			t.Errorf("Expected ID %d, got %d", expectedUser.ID, resp.Data.ID)
		}

		if resp.Data.Name != expectedUser.Name {
			t.Errorf("Expected Name %s, got %s", expectedUser.Name, resp.Data.Name)
		}
	})

	t.Run("HTTP error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			if err := json.NewEncoder(w).Encode(map[string]string{
				"message": "User not found",
			}); err != nil {
				t.Errorf("Failed to encode response: %v", err)
			}
		}))
		defer server.Close()

		client := NewGenericClient[User]()
		req, _ := http.NewRequest(http.MethodGet, server.URL, nil)

		_, err := client.Execute(req)

		if err == nil {
			t.Fatal("Expected error for 404 response")
		}

		errorResp, ok := err.(*ErrorResponse)
		if !ok {
			t.Fatalf("Expected ErrorResponse, got %T", err)
		}

		if errorResp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status code %d, got %d", http.StatusNotFound, errorResp.StatusCode)
		}

		if !strings.Contains(errorResp.Message, "not found") {
			t.Errorf("Expected error message about 'not found', got %s", errorResp.Message)
		}
	})

	t.Run("Empty response body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()

		client := NewGenericClient[User]()
		req, _ := http.NewRequest(http.MethodGet, server.URL, nil)

		resp, err := client.Execute(req)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		if resp.StatusCode != http.StatusNoContent {
			t.Errorf("Expected status %d, got %d", http.StatusNoContent, resp.StatusCode)
		}

		if len(resp.RawBody) != 0 {
			t.Errorf("Expected empty body, got %d bytes", len(resp.RawBody))
		}
	})

	t.Run("Invalid JSON response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, err := w.Write([]byte("invalid json")); err != nil {
				t.Errorf("Failed to write response: %v", err)
			}
		}))
		defer server.Close()

		client := NewGenericClient[User]()
		req, _ := http.NewRequest(http.MethodGet, server.URL, nil)

		_, err := client.Execute(req)

		if err == nil {
			t.Fatal("Expected error for invalid JSON")
		}

		if !strings.Contains(err.Error(), "unmarshal") {
			t.Errorf("Expected unmarshal error, got: %v", err)
		}
	})
}

// TestGenericClient_ConvenienceMethods tests GET, POST, PUT, DELETE, PATCH methods
func TestGenericClient_ConvenienceMethods(t *testing.T) {
	t.Run("Get method", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("Expected GET, got %s", r.Method)
			}
			if err := json.NewEncoder(w).Encode(User{ID: 1, Name: "John"}); err != nil {
				t.Errorf("Failed to encode response: %v", err)
			}
		}))
		defer server.Close()

		client := NewGenericClient[User]()
		resp, err := client.Get(server.URL)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}

		if resp.Data.ID != 1 {
			t.Errorf("Expected ID 1, got %d", resp.Data.ID)
		}
	})

	t.Run("Post method", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("Expected POST, got %s", r.Method)
			}

			var user User
			if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
				t.Errorf("Failed to decode request: %v", err)
			}
			user.ID = 123
			if err := json.NewEncoder(w).Encode(user); err != nil {
				t.Errorf("Failed to encode response: %v", err)
			}
		}))
		defer server.Close()

		client := NewGenericClient[User]()
		userData, _ := json.Marshal(User{Name: "Jane", Email: "jane@example.com"})
		resp, err := client.Post(server.URL, bytes.NewReader(userData))
		if err != nil {
			t.Fatalf("Post failed: %v", err)
		}

		if resp.Data.ID != 123 {
			t.Errorf("Expected ID 123, got %d", resp.Data.ID)
		}
	})

	t.Run("Put method", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				t.Errorf("Expected PUT, got %s", r.Method)
			}
			if err := json.NewEncoder(w).Encode(User{ID: 1, Name: "Updated"}); err != nil {
				t.Errorf("Failed to encode response: %v", err)
			}
		}))
		defer server.Close()

		client := NewGenericClient[User]()
		userData, _ := json.Marshal(User{ID: 1, Name: "Updated"})
		resp, err := client.Put(server.URL, bytes.NewReader(userData))
		if err != nil {
			t.Fatalf("Put failed: %v", err)
		}

		if resp.Data.Name != "Updated" {
			t.Errorf("Expected Name 'Updated', got %s", resp.Data.Name)
		}
	})

	t.Run("Delete method", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				t.Errorf("Expected DELETE, got %s", r.Method)
			}
			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()

		client := NewGenericClient[User]()
		resp, err := client.Delete(server.URL)
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		if resp.StatusCode != http.StatusNoContent {
			t.Errorf("Expected status %d, got %d", http.StatusNoContent, resp.StatusCode)
		}
	})

	t.Run("Patch method", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPatch {
				t.Errorf("Expected PATCH, got %s", r.Method)
			}
			if err := json.NewEncoder(w).Encode(User{ID: 1, Name: "Patched"}); err != nil {
				t.Errorf("Failed to encode response: %v", err)
			}
		}))
		defer server.Close()

		client := NewGenericClient[User]()
		patchData, _ := json.Marshal(map[string]string{"name": "Patched"})
		resp, err := client.Patch(server.URL, bytes.NewReader(patchData))
		if err != nil {
			t.Fatalf("Patch failed: %v", err)
		}

		if resp.Data.Name != "Patched" {
			t.Errorf("Expected Name 'Patched', got %s", resp.Data.Name)
		}
	})
}

// TestGenericClient_ExecuteRaw tests the ExecuteRaw method
func TestGenericClient_ExecuteRaw(t *testing.T) {
	t.Run("Returns raw response", func(t *testing.T) {
		expectedBody := "raw response body"
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, err := w.Write([]byte(expectedBody)); err != nil {
				t.Errorf("Failed to write response: %v", err)
			}
		}))
		defer server.Close()

		client := NewGenericClient[User]()
		req, _ := http.NewRequest(http.MethodGet, server.URL, nil)

		resp, err := client.ExecuteRaw(req)
		if err != nil {
			t.Fatalf("ExecuteRaw failed: %v", err)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		if string(body) != expectedBody {
			t.Errorf("Expected body %s, got %s", expectedBody, string(body))
		}
	})
}

// TestGenericClient_Do tests the Do method alias
func TestGenericClient_Do(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewEncoder(w).Encode(User{ID: 1, Name: "Test"}); err != nil {
			t.Errorf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	client := NewGenericClient[User]()
	req, _ := http.NewRequest(http.MethodGet, server.URL, nil)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Do failed: %v", err)
	}

	if resp.Data.ID != 1 {
		t.Errorf("Expected ID 1, got %d", resp.Data.ID)
	}
}

// TestGenericClient_WithRequestBuilder tests integration with RequestBuilder
func TestGenericClient_WithRequestBuilder(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("Authorization") != "Bearer token123" {
			t.Errorf("Expected Authorization header, got %s", r.Header.Get("Authorization"))
		}

		var post Post
		if err := json.NewDecoder(r.Body).Decode(&post); err != nil {
			t.Errorf("Failed to decode request: %v", err)
		}
		post.ID = 42
		if err := json.NewEncoder(w).Encode(post); err != nil {
			t.Errorf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	// Create client
	client := NewGenericClient[Post]()

	// Use RequestBuilder to create request with headers
	post := Post{Title: "Test Post", Body: "Test content", UserID: 1}
	req, err := NewRequestBuilder(server.URL).
		WithMethodPOST().
		WithPath("/posts").
		WithContentType("application/json").
		WithHeader("Authorization", "Bearer token123").
		WithJSONBody(post).
		Build()
	if err != nil {
		t.Fatalf("RequestBuilder.Build failed: %v", err)
	}

	// Execute request with client
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("client.Do failed: %v", err)
	}

	if resp.Data.ID != 42 {
		t.Errorf("Expected ID 42, got %d", resp.Data.ID)
	}

	if resp.Data.Title != "Test Post" {
		t.Errorf("Expected Title 'Test Post', got %s", resp.Data.Title)
	}
}

// TestErrorResponse_Error tests the Error method of ErrorResponse
func TestErrorResponse_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *ErrorResponse
		expected string
	}{
		{
			name: "With Message field",
			err: &ErrorResponse{
				StatusCode: 404,
				Message:    "Resource not found",
			},
			expected: "http 404: Resource not found",
		},
		{
			name: "With ErrorMsg field",
			err: &ErrorResponse{
				StatusCode: 500,
				ErrorMsg:   "Internal server error",
			},
			expected: "http 500: Internal server error",
		},
		{
			name: "Without message",
			err: &ErrorResponse{
				StatusCode: 400,
			},
			expected: "http 400: request failed",
		},
		{
			name: "Message takes precedence over ErrorMsg",
			err: &ErrorResponse{
				StatusCode: 403,
				Message:    "Forbidden",
				ErrorMsg:   "Access denied",
			},
			expected: "http 403: Forbidden",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.Error()
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestGenericClient_MultipleTypes tests using multiple typed clients
func TestGenericClient_MultipleTypes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/users") {
			if err := json.NewEncoder(w).Encode(User{ID: 1, Name: "John"}); err != nil {
				t.Errorf("Failed to encode response: %v", err)
			}
		} else if strings.Contains(r.URL.Path, "/posts") {
			if err := json.NewEncoder(w).Encode(Post{ID: 100, Title: "Test Post"}); err != nil {
				t.Errorf("Failed to encode response: %v", err)
			}
		}
	}))
	defer server.Close()

	// Create two clients with different types
	userClient := NewGenericClient[User]()
	postClient := NewGenericClient[Post]()

	// Test user client
	userResp, err := userClient.Get(server.URL + "/users/1")
	if err != nil {
		t.Fatalf("userClient.Get failed: %v", err)
	}

	if userResp.Data.Name != "John" {
		t.Errorf("Expected user name John, got %s", userResp.Data.Name)
	}

	// Test post client
	postResp, err := postClient.Get(server.URL + "/posts/100")
	if err != nil {
		t.Fatalf("postClient.Get failed: %v", err)
	}

	if postResp.Data.Title != "Test Post" {
		t.Errorf("Expected post title 'Test Post', got %s", postResp.Data.Title)
	}
}

// TestGenericClient_ContextPropagation tests context propagation
func TestGenericClient_ContextPropagation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(100 * time.Millisecond)
		if err := json.NewEncoder(w).Encode(User{ID: 1}); err != nil {
			t.Errorf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	client := NewGenericClient[User]()

	// Create request with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)

	_, err := client.Execute(req)

	if err == nil {
		t.Fatal("Expected error with cancelled context")
	}

	if !strings.Contains(err.Error(), "context") {
		t.Errorf("Expected context error, got: %v", err)
	}
}

// TestGenericClient_AllConfigurationOptions tests all the new configuration options
func TestGenericClient_AllConfigurationOptions(t *testing.T) {
	t.Run("With all configuration options", func(t *testing.T) {
		timeout := 15 * time.Second
		maxRetries := 5
		baseDelay := 1 * time.Second
		maxDelay := 10 * time.Second

		client := NewGenericClient[User](
			WithTimeout[User](timeout),
			WithMaxRetries[User](maxRetries),
			WithRetryBaseDelay[User](baseDelay),
			WithRetryMaxDelay[User](maxDelay),
			WithRetryStrategy[User](JitterBackoffStrategy),
			WithMaxIdleConns[User](50),
			WithIdleConnTimeout[User](60*time.Second),
			WithTLSHandshakeTimeout[User](5*time.Second),
			WithExpectContinueTimeout[User](2*time.Second),
			WithMaxIdleConnsPerHost[User](25),
			WithDisableKeepAlive[User](false),
		)

		if client == nil {
			t.Fatal("NewGenericClient returned nil")
		}

		if client.httpClient == nil {
			t.Fatal("httpClient should be initialized")
		}

		// Verify the client was built with an HTTP client
		if httpClient, ok := client.httpClient.(*http.Client); ok {
			if httpClient.Timeout != timeout {
				t.Errorf("Expected timeout %v, got %v", timeout, httpClient.Timeout)
			}

			// Verify the transport is configured
			if httpClient.Transport == nil {
				t.Error("Transport should be configured")
			}
		} else {
			t.Error("Expected *http.Client as underlying client")
		}
	})

	t.Run("WithRetryStrategyAsString", func(t *testing.T) {
		client := NewGenericClient[User](
			WithRetryStrategyAsString[User]("fixed"),
			WithMaxRetries[User](3),
		)

		if client == nil {
			t.Fatal("NewGenericClient returned nil")
		}

		if client.httpClient == nil {
			t.Fatal("httpClient should be initialized")
		}
	})

	t.Run("WithHTTPClient takes precedence", func(t *testing.T) {
		customClient := &http.Client{
			Timeout: 99 * time.Second,
		}

		client := NewGenericClient[User](
			WithTimeout[User](15*time.Second),  // Should be ignored
			WithMaxRetries[User](5),            // Should be ignored
			WithHTTPClient[User](customClient), // Should take precedence
		)

		if client == nil {
			t.Fatal("NewGenericClient returned nil")
		}

		if client.httpClient != customClient {
			t.Error("Expected custom HTTP client to be used")
		}

		if httpClient, ok := client.httpClient.(*http.Client); ok {
			if httpClient.Timeout != 99*time.Second {
				t.Errorf("Expected timeout 99s, got %v", httpClient.Timeout)
			}
		}
	})

	t.Run("Default client with no options", func(t *testing.T) {
		client := NewGenericClient[User]()

		if client == nil {
			t.Fatal("NewGenericClient returned nil")
		}

		if client.httpClient == nil {
			t.Fatal("httpClient should be initialized")
		}

		// Should have default timeout from ClientBuilder
		if httpClient, ok := client.httpClient.(*http.Client); ok {
			if httpClient.Timeout != DefaultTimeout {
				t.Errorf("Expected default timeout %v, got %v", DefaultTimeout, httpClient.Timeout)
			}
		}
	})

	t.Run("Invalid values should use ClientBuilder defaults", func(t *testing.T) {
		// Use out-of-range values
		client := NewGenericClient[User](
			WithTimeout[User](0),         // Invalid, should use default
			WithMaxRetries[User](999),    // Invalid, should use default
			WithMaxIdleConns[User](9999), // Invalid, should use default
		)

		if client == nil {
			t.Fatal("NewGenericClient returned nil")
		}

		if client.httpClient == nil {
			t.Fatal("httpClient should be initialized")
		}

		// ClientBuilder should have applied defaults
		if httpClient, ok := client.httpClient.(*http.Client); ok {
			// Should have been corrected to default
			if httpClient.Timeout != DefaultTimeout {
				t.Errorf("Expected default timeout %v, got %v", DefaultTimeout, httpClient.Timeout)
			}
		}
	})

	t.Run("Mix of options and WithHTTPClient", func(t *testing.T) {
		// Build a client using ClientBuilder
		httpClient := NewClientBuilder().
			WithMaxRetries(3).
			WithRetryStrategy(ExponentialBackoffStrategy).
			WithTimeout(10 * time.Second).
			Build()

		// Use it with GenericClient
		client := NewGenericClient[User](
			WithHTTPClient[User](httpClient),
		)

		if client == nil {
			t.Fatal("NewGenericClient returned nil")
		}

		if client.httpClient != httpClient {
			t.Error("Expected the built HTTP client to be used")
		}
	})
}

// TestGenericClient_OptionsIntegration tests that options actually affect behavior
func TestGenericClient_OptionsIntegration(t *testing.T) {
	t.Run("Configured client is usable", func(t *testing.T) {
		client := NewGenericClient[User](
			WithTimeout[User](5*time.Second),
			WithMaxRetries[User](2),
			WithRetryStrategy[User](FixedDelayStrategy),
		)

		if client == nil {
			t.Fatal("NewGenericClient returned nil")
		}

		// The client should be functional (we won't make actual requests,
		// but we can verify it's properly constructed)
		if client.httpClient == nil {
			t.Fatal("httpClient should be initialized")
		}

		// Verify we can call methods without panicking
		if httpClient, ok := client.httpClient.(*http.Client); ok {
			if httpClient.Transport == nil {
				t.Error("Transport should be configured")
			}
		}
	})
}

func TestGenericClient_WithProxy(t *testing.T) {
	t.Run("Create client with proxy URL", func(t *testing.T) {
		proxyURL := "http://proxy.example.com:8080"
		client := NewGenericClient[User](
			WithProxy[User](proxyURL),
		)

		if client == nil {
			t.Fatal("NewGenericClient returned nil")
		}

		if client.proxyURL == nil {
			t.Fatal("proxyURL should be set")
		}

		if *client.proxyURL != proxyURL {
			t.Errorf("Expected proxy URL %s, got %s", proxyURL, *client.proxyURL)
		}
	})

	t.Run("Create client with HTTPS proxy", func(t *testing.T) {
		proxyURL := "https://secure-proxy.example.com:3128"
		client := NewGenericClient[User](
			WithProxy[User](proxyURL),
		)

		if client == nil {
			t.Fatal("NewGenericClient returned nil")
		}

		if client.proxyURL == nil || *client.proxyURL != proxyURL {
			t.Errorf("Expected proxy URL %s", proxyURL)
		}
	})

	t.Run("Create client without proxy", func(t *testing.T) {
		client := NewGenericClient[User]()

		if client == nil {
			t.Fatal("NewGenericClient returned nil")
		}

		if client.proxyURL != nil {
			t.Error("proxyURL should be nil when not configured")
		}
	})

	t.Run("Create client with empty proxy (disable)", func(t *testing.T) {
		client := NewGenericClient[User](
			WithProxy[User](""),
		)

		if client == nil {
			t.Fatal("NewGenericClient returned nil")
		}

		if client.proxyURL == nil {
			t.Fatal("proxyURL should be set (even if empty)")
		}

		if *client.proxyURL != "" {
			t.Error("Expected empty proxy URL")
		}
	})

	t.Run("Build HTTP client with proxy", func(t *testing.T) {
		proxyURL := "http://proxy.example.com:8080"
		client := NewGenericClient[User](
			WithProxy[User](proxyURL),
		)

		if client.httpClient == nil {
			t.Fatal("httpClient should be initialized")
		}

		// Verify the transport has proxy configured
		if httpClient, ok := client.httpClient.(*http.Client); ok {
			if rt, ok := httpClient.Transport.(*retryTransport); ok {
				if transport, ok := rt.Transport.(*http.Transport); ok {
					if transport.Proxy == nil {
						t.Error("Expected Proxy to be configured")
					}
				} else {
					t.Error("Expected *http.Transport")
				}
			} else {
				t.Error("Expected *retryTransport")
			}
		} else {
			t.Error("Expected *http.Client")
		}
	})

	t.Run("Combine proxy with other options", func(t *testing.T) {
		proxyURL := "http://proxy.example.com:8080"
		client := NewGenericClient[User](
			WithProxy[User](proxyURL),
			WithTimeout[User](10*time.Second),
			WithMaxRetries[User](3),
			WithRetryStrategy[User](ExponentialBackoffStrategy),
		)

		if client == nil {
			t.Fatal("NewGenericClient returned nil")
		}

		if client.proxyURL == nil || *client.proxyURL != proxyURL {
			t.Error("Proxy URL should be configured")
		}

		if client.timeout == nil || *client.timeout != 10*time.Second {
			t.Error("Timeout should be configured")
		}

		if client.maxRetries == nil || *client.maxRetries != 3 {
			t.Error("MaxRetries should be configured")
		}
	})
}
