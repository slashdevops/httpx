package httpx

import (
	"context"
	"strings"
	"testing"
)

// TestRequestBuilder_ErrorHandling tests the error accumulation pattern
func TestRequestBuilder_ErrorHandling(t *testing.T) {
	t.Run("HasErrors and GetErrors", func(t *testing.T) {
		rb := NewRequestBuilder("https://api.example.com")

		if rb.HasErrors() {
			t.Error("New builder should not have errors")
		}

		if len(rb.GetErrors()) != 0 {
			t.Error("New builder should have empty error slice")
		}

		// Add an error by using invalid input
		rb.WithMethod("")

		if !rb.HasErrors() {
			t.Error("Builder should have errors after invalid method")
		}

		errors := rb.GetErrors()
		if len(errors) != 1 {
			t.Errorf("Expected 1 error, got %d", len(errors))
		}
	})

	t.Run("Multiple errors accumulate", func(t *testing.T) {
		rb := NewRequestBuilder("https://api.example.com")
		rb.WithMethod("")
		rb.WithHeader("", "value")
		rb.WithQueryParam("", "value")
		rb.WithBasicAuth("", "")

		if !rb.HasErrors() {
			t.Error("Builder should have errors")
		}

		errors := rb.GetErrors()
		if len(errors) < 4 {
			t.Errorf("Expected at least 4 errors, got %d", len(errors))
		}
	})

	t.Run("Build fails with accumulated errors", func(t *testing.T) {
		rb := NewRequestBuilder("https://api.example.com")
		rb.WithMethodGET()
		rb.WithHeader("", "value") // Invalid header

		_, err := rb.Build()
		if err == nil {
			t.Error("Build() should fail with accumulated errors")
		}
	})
}

// TestRequestBuilder_Reset tests the Reset method
func TestRequestBuilder_Reset(t *testing.T) {
	rb := NewRequestBuilder("https://api.example.com")

	// Configure builder
	rb.WithMethodPOST().
		WithPath("/users").
		WithHeader("Content-Type", "application/json").
		WithQueryParam("foo", "bar").
		WithBasicAuth("user", "pass")

	// Add an error
	rb.WithHeader("", "value")

	if !rb.HasErrors() {
		t.Error("Builder should have errors before reset")
	}

	// Reset
	rb.Reset()

	if rb.HasErrors() {
		t.Error("Builder should not have errors after reset")
	}

	if rb.method != "" {
		t.Error("Method should be cleared after reset")
	}

	if rb.path != "" {
		t.Error("Path should be cleared after reset")
	}

	if len(rb.headers) != 0 {
		t.Error("Headers should be cleared after reset")
	}

	if len(rb.queryParams) != 0 {
		t.Error("Query params should be cleared after reset")
	}

	if rb.ctx == nil {
		t.Error("Context should be initialized after reset")
	}
}

// TestRequestBuilder_ValidationErrors tests validation on various methods
func TestRequestBuilder_ValidationErrors(t *testing.T) {
	t.Run("Empty header key", func(t *testing.T) {
		rb := NewRequestBuilder("https://api.example.com")
		rb.WithHeader("", "value")

		if !rb.HasErrors() {
			t.Error("Should have error for empty header key")
		}
	})

	t.Run("Empty header value", func(t *testing.T) {
		rb := NewRequestBuilder("https://api.example.com")
		rb.WithHeader("X-Test", "")

		if !rb.HasErrors() {
			t.Error("Should have error for empty header value")
		}
	})

	t.Run("Header key with whitespace", func(t *testing.T) {
		rb := NewRequestBuilder("https://api.example.com")
		rb.WithHeader("X Test", "value")

		if !rb.HasErrors() {
			t.Error("Should have error for header key with whitespace")
		}
	})

	t.Run("Empty query key", func(t *testing.T) {
		rb := NewRequestBuilder("https://api.example.com")
		rb.WithQueryParam("", "value")

		if !rb.HasErrors() {
			t.Error("Should have error for empty query key")
		}
	})

	t.Run("Empty query value", func(t *testing.T) {
		rb := NewRequestBuilder("https://api.example.com")
		rb.WithQueryParam("key", "")

		if !rb.HasErrors() {
			t.Error("Should have error for empty query value")
		}
	})

	t.Run("Query key with invalid characters", func(t *testing.T) {
		rb := NewRequestBuilder("https://api.example.com")
		rb.WithQueryParam("key=value", "test")

		if !rb.HasErrors() {
			t.Error("Should have error for query key with invalid characters")
		}
	})

	t.Run("Empty username for basic auth", func(t *testing.T) {
		rb := NewRequestBuilder("https://api.example.com")
		rb.WithBasicAuth("", "password")

		if !rb.HasErrors() {
			t.Error("Should have error for empty username")
		}
	})

	t.Run("Empty password for basic auth", func(t *testing.T) {
		rb := NewRequestBuilder("https://api.example.com")
		rb.WithBasicAuth("user", "")

		if !rb.HasErrors() {
			t.Error("Should have error for empty password")
		}
	})

	t.Run("Empty bearer token", func(t *testing.T) {
		rb := NewRequestBuilder("https://api.example.com")
		rb.WithBearerAuth("")

		if !rb.HasErrors() {
			t.Error("Should have error for empty bearer token")
		}
	})

	t.Run("Empty user agent", func(t *testing.T) {
		rb := NewRequestBuilder("https://api.example.com")
		rb.WithUserAgent("")

		if !rb.HasErrors() {
			t.Error("Should have error for empty user agent")
		}
	})

	t.Run("User agent with only whitespace", func(t *testing.T) {
		rb := NewRequestBuilder("https://api.example.com")
		rb.WithUserAgent("   ")

		if !rb.HasErrors() {
			t.Error("Should have error for whitespace-only user agent")
		}
	})

	t.Run("User agent too long", func(t *testing.T) {
		rb := NewRequestBuilder("https://api.example.com")
		longUA := strings.Repeat("a", 501)
		rb.WithUserAgent(longUA)

		if !rb.HasErrors() {
			t.Error("Should have error for user agent exceeding 500 characters")
		}
	})

	t.Run("User agent with control characters", func(t *testing.T) {
		rb := NewRequestBuilder("https://api.example.com")
		rb.WithUserAgent("MyApp/1.0\nMalicious")

		if !rb.HasErrors() {
			t.Error("Should have error for user agent with control characters")
		}
	})

	t.Run("Nil context", func(t *testing.T) {
		rb := NewRequestBuilder("https://api.example.com")
		rb.WithContext(context.TODO())

		if rb.HasErrors() {
			t.Error("Should not have error for context.TODO()")
		}
	})
}

// TestRequestBuilder_BuildValidation tests Build-time validation
func TestRequestBuilder_BuildValidation(t *testing.T) {
	t.Run("Missing method", func(t *testing.T) {
		rb := NewRequestBuilder("https://api.example.com")

		_, err := rb.Build()
		if err == nil {
			t.Error("Build() should fail without method")
		}
	})

	t.Run("Invalid base URL scheme", func(t *testing.T) {
		rb := NewRequestBuilder("ftp://example.com")
		rb.WithMethodGET()

		_, err := rb.Build()
		if err == nil {
			t.Error("Build() should fail with invalid URL scheme")
		}
	})

	t.Run("Missing URL scheme", func(t *testing.T) {
		rb := NewRequestBuilder("example.com")
		rb.WithMethodGET()

		_, err := rb.Build()
		if err == nil {
			t.Error("Build() should fail without URL scheme")
		}
	})

	t.Run("Missing URL host", func(t *testing.T) {
		rb := NewRequestBuilder("http://")
		rb.WithMethodGET()

		_, err := rb.Build()
		if err == nil {
			t.Error("Build() should fail without URL host")
		}
	})
}

// TestRequestBuilder_HTTPMethodValidation tests HTTPMethod validation
func TestRequestBuilder_HTTPMethodValidation(t *testing.T) {
	tests := []struct {
		name        string
		method      string
		expectError bool
		expected    string
	}{
		{"Valid GET", "GET", false, "GET"},
		{"Valid POST", "POST", false, "POST"},
		{"Valid lowercase get", "get", false, "GET"},
		{"Valid with spaces", " PUT ", false, "PUT"},
		{"Empty method", "", true, ""},
		{"Invalid method", "INVALID", true, ""},
		{"Invalid method INVALID2", "SOMETHING", true, ""},
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
				if rb.method != tt.expected {
					t.Errorf("Expected method %s, got %s", tt.expected, rb.method)
				}
			}
		})
	}
}
