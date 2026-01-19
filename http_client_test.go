package httpx

import (
	"net/http"
	"reflect"
	"testing"
	"time"
)

// Helper functions to replace testify assertions
func assertEqual(t *testing.T, expected, actual any) {
	t.Helper()
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Expected %v, got %v", expected, actual)
	}
}

func assertTrue(t *testing.T, condition bool) {
	t.Helper()
	if !condition {
		t.Error("Expected condition to be true")
	}
}

func assertNotNil(t *testing.T, value any) {
	t.Helper()
	if value == nil {
		t.Error("Expected value to be non-nil")
	}
}

func TestClientBuilder_WithMethods(t *testing.T) {
	builder := NewClientBuilder()

	// Test valid settings
	builder.WithMaxIdleConns(50).
		WithIdleConnTimeout(60 * time.Second).
		WithTLSHandshakeTimeout(5 * time.Second).
		WithExpectContinueTimeout(2 * time.Second).
		WithDisableKeepAlive(true).
		WithMaxIdleConnsPerHost(50).
		WithTimeout(10 * time.Second).
		WithMaxRetries(5).
		WithRetryBaseDelay(100 * time.Millisecond).
		WithRetryMaxDelay(5 * time.Second).
		WithRetryStrategy(FixedDelayStrategy)

	client := builder.client
	// Assert that the With... methods *set* the values on the internal client struct
	assertEqual(t, 50, client.maxIdleConns)
	assertEqual(t, 60*time.Second, client.idleConnTimeout)
	assertEqual(t, 5*time.Second, client.tlsHandshakeTimeout)
	assertEqual(t, 2*time.Second, client.expectContinueTimeout)
	assertTrue(t, client.disableKeepAlive)
	assertEqual(t, 50, client.maxIdleConnsPerHost)
	assertEqual(t, 10*time.Second, client.timeout)
	assertEqual(t, 5, client.maxRetries)
	assertEqual(t, 100*time.Millisecond, client.retryBaseDelay) // Check the value *set* by WithRetryBaseDelay
	assertEqual(t, 5*time.Second, client.retryMaxDelay)
	// Check that the strategy type was set correctly
	assertEqual(t, FixedDelayStrategy, client.retryStrategyType) // Check the type *set* by WithRetryStrategy

	// Test invalid settings (should use defaults or adjusted values)
	builder = NewClientBuilder() // Reset builder
	builder.WithMaxIdleConns(0). // Invalid, use default
					WithIdleConnTimeout(0).                   // Invalid, use default
					WithTLSHandshakeTimeout(0).               // Invalid, use default
					WithExpectContinueTimeout(0).             // Invalid, use default
					WithMaxIdleConnsPerHost(0).               // Invalid, use default
					WithTimeout(0).                           // Invalid, use default
					WithMaxRetries(0).                        // Invalid, use default
					WithRetryBaseDelay(1 * time.Millisecond). // Invalid, use default
					WithRetryMaxDelay(50 * time.Millisecond). // Invalid, use default
					WithRetryStrategy("invalid")              // Invalid strategy type

	client = builder.client
	// Assert that the *invalid* values were set by the With... methods (before Build validation)
	assertEqual(t, 0, client.maxIdleConns)
	assertEqual(t, 0*time.Second, client.idleConnTimeout)
	assertEqual(t, 0*time.Second, client.tlsHandshakeTimeout)
	assertEqual(t, 0*time.Second, client.expectContinueTimeout)
	assertEqual(t, 0, client.maxIdleConnsPerHost)
	assertEqual(t, 0*time.Second, client.timeout)
	assertEqual(t, 0, client.maxRetries)
	assertEqual(t, 1*time.Millisecond, client.retryBaseDelay)
	assertEqual(t, 50*time.Millisecond, client.retryMaxDelay)
	// Check that the invalid strategy type was set
	assertEqual(t, Strategy("invalid"), client.retryStrategyType)
}

func TestClientBuilder_Build(t *testing.T) {
	baseDelay := 200 * time.Millisecond
	maxDelay := 2 * time.Second
	maxRetries := 4

	builder := NewClientBuilder().
		WithMaxIdleConns(55).
		WithIdleConnTimeout(65 * time.Second).
		WithTLSHandshakeTimeout(6 * time.Second).
		WithExpectContinueTimeout(3 * time.Second).
		WithDisableKeepAlive(true).
		WithMaxIdleConnsPerHost(55).
		WithTimeout(15 * time.Second).
		WithMaxRetries(maxRetries).
		WithRetryBaseDelay(baseDelay). // Invalid, should be corrected to default
		WithRetryMaxDelay(maxDelay).
		WithRetryStrategy(JitterBackoffStrategy)

	httpClient := builder.Build()

	// Verify the HTTP client was built
	assertNotNil(t, httpClient)
	assertNotNil(t, httpClient.Transport)

	// Verify timeout
	assertEqual(t, 15*time.Second, httpClient.Timeout)

	// Test the transport is a retry transport
	if retryTrans, ok := httpClient.Transport.(*retryTransport); ok {
		assertEqual(t, maxRetries, retryTrans.MaxRetries)
		assertNotNil(t, retryTrans.RetryStrategy)

		// Test that the underlying transport has the right settings
		if baseTrans, ok := retryTrans.Transport.(*http.Transport); ok {
			assertEqual(t, 55, baseTrans.MaxIdleConns)
			assertEqual(t, 65*time.Second, baseTrans.IdleConnTimeout)
			assertEqual(t, 6*time.Second, baseTrans.TLSHandshakeTimeout)
			assertEqual(t, 3*time.Second, baseTrans.ExpectContinueTimeout)
			assertTrue(t, baseTrans.DisableKeepAlives)
			assertEqual(t, 55, baseTrans.MaxIdleConnsPerHost)
		} else {
			t.Error("Expected underlying transport to be *http.Transport")
		}
	} else {
		t.Error("Expected transport to be *retryTransport")
	}
}

func TestStrategyString(t *testing.T) {
	assertEqual(t, "fixed", FixedDelayStrategy.String())
	assertEqual(t, "jitter", JitterBackoffStrategy.String())
	assertEqual(t, "exponential", ExponentialBackoffStrategy.String())
	assertEqual(t, "unknown", Strategy("unknown").String())
}

func TestClientError(t *testing.T) {
	err := &ClientError{Message: "test error"}
	assertEqual(t, "test error", err.Error())
}

func TestClientBuilder_WithRetryStrategyAsString(t *testing.T) {
	tests := []struct {
		name          string
		inputStrategy string
		expectedType  Strategy
		expectWarning bool // Although we can't directly test logs here, good to note
	}{
		{
			name:          "Valid Fixed Strategy",
			inputStrategy: "fixed",
			expectedType:  FixedDelayStrategy,
			expectWarning: false,
		},
		{
			name:          "Valid Jitter Strategy",
			inputStrategy: "jitter",
			expectedType:  JitterBackoffStrategy,
			expectWarning: false,
		},
		{
			name:          "Valid Exponential Strategy",
			inputStrategy: "exponential",
			expectedType:  ExponentialBackoffStrategy,
			expectWarning: false,
		},
		{
			name:          "Invalid Strategy",
			inputStrategy: "invalid-strategy",
			expectedType:  ExponentialBackoffStrategy, // Should default
			expectWarning: true,
		},
		{
			name:          "Empty Strategy",
			inputStrategy: "",
			expectedType:  ExponentialBackoffStrategy, // Should default
			expectWarning: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewClientBuilder() // Start fresh for each test case
			builder.WithRetryStrategyAsString(tt.inputStrategy)

			// Assert that the correct strategy *type* was set on the internal client struct
			assertEqual(t, tt.expectedType, builder.client.retryStrategyType)

			// Note: We expect a warning log for invalid strategies, but testing logs
			// usually requires more setup (e.g., capturing log output).
			// This test focuses on the functional outcome (correct strategy type set).
		})
	}
}

func TestClientBuilder_WithProxy(t *testing.T) {
	t.Run("Valid proxy URL", func(t *testing.T) {
		builder := NewClientBuilder()
		proxyURL := "http://proxy.example.com:8080"

		builder.WithProxy(proxyURL)

		assertEqual(t, proxyURL, builder.client.proxyURL)
	})

	t.Run("HTTPS proxy URL", func(t *testing.T) {
		builder := NewClientBuilder()
		proxyURL := "https://secure-proxy.example.com:3128"

		builder.WithProxy(proxyURL)

		assertEqual(t, proxyURL, builder.client.proxyURL)
	})

	t.Run("Empty proxy URL (disable proxy)", func(t *testing.T) {
		builder := NewClientBuilder()
		builder.WithProxy("")

		assertEqual(t, "", builder.client.proxyURL)
	})

	t.Run("Proxy with authentication", func(t *testing.T) {
		builder := NewClientBuilder()
		proxyURL := "http://user:pass@proxy.example.com:8080"

		builder.WithProxy(proxyURL)

		assertEqual(t, proxyURL, builder.client.proxyURL)
	})
}

func TestClientBuilder_Build_WithProxy(t *testing.T) {
	t.Run("Build with valid proxy", func(t *testing.T) {
		builder := NewClientBuilder()
		proxyURL := "http://proxy.example.com:8080"

		client := builder.WithProxy(proxyURL).Build()

		assertNotNil(t, client)
		assertNotNil(t, client.Transport)

		// Verify that the transport has proxy configured
		if rt, ok := client.Transport.(*retryTransport); ok {
			if transport, ok := rt.Transport.(*http.Transport); ok {
				assertNotNil(t, transport.Proxy)
			} else {
				t.Error("Expected *http.Transport")
			}
		} else {
			t.Error("Expected *retryTransport")
		}
	})

	t.Run("Build without proxy", func(t *testing.T) {
		builder := NewClientBuilder()

		client := builder.Build()

		assertNotNil(t, client)
		assertNotNil(t, client.Transport)

		// Verify that the transport has no proxy configured (nil)
		if rt, ok := client.Transport.(*retryTransport); ok {
			if transport, ok := rt.Transport.(*http.Transport); ok {
				// Proxy should be nil (not configured)
				if transport.Proxy != nil {
					t.Error("Expected Proxy to be nil when not configured")
				}
			}
		}
	})

	t.Run("Build with invalid proxy URL", func(t *testing.T) {
		builder := NewClientBuilder()
		invalidProxyURL := "://invalid-url"

		client := builder.WithProxy(invalidProxyURL).Build()

		// Should still build successfully, but proxy will be ignored
		assertNotNil(t, client)

		// Verify that the transport has no proxy configured due to parse error
		if rt, ok := client.Transport.(*retryTransport); ok {
			if transport, ok := rt.Transport.(*http.Transport); ok {
				if transport.Proxy != nil {
					t.Error("Expected Proxy to be nil when invalid URL provided")
				}
			}
		}
	})
}
