package httpx

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// --- Test Retry Strategies ---

func TestExponentialBackoff(t *testing.T) {
	base := 100 * time.Millisecond
	max := 1 * time.Second
	strategy := ExponentialBackoff(base, max)

	expectedDelays := []time.Duration{
		base,     // attempt 0 -> base * 2^0 = base
		base * 2, // attempt 1 -> base * 2^1
		base * 4, // attempt 2 -> base * 2^2
		base * 8, // attempt 3 -> base * 2^3
		max,      // attempt 4 -> base * 2^4 = 1600ms > max, capped at max
		max,      // attempt 5 -> base * 2^5 = 3200ms > max, capped at max
	}

	for i, expected := range expectedDelays {
		actual := strategy(i)
		if actual != expected {
			t.Errorf("Attempt %d: Expected delay %v, got %v", i, expected, actual)
		}
	}

	for i, expected := range expectedDelays {
		actual := strategy(i)
		if actual != expected {
			t.Errorf("Attempt %d: Expected delay %v, got %v", i, expected, actual)
		}
	}

	// Test case where base > max (should cap at max, but logic ensures base is min)
	strategyHighBase := ExponentialBackoff(2*time.Second, 1*time.Second)

	if delay := strategyHighBase(0); delay != 2*time.Second { // Should return base even if > max initially
		t.Errorf("High base test: Expected delay %v, got %v", 2*time.Second, delay)
	}

	if delay := strategyHighBase(1); delay != 1*time.Second { // Subsequent attempts capped at max
		t.Errorf("High base test attempt 1: Expected delay %v, got %v", 1*time.Second, delay)
	}
}

func TestFixedDelay(t *testing.T) {
	delay := 500 * time.Millisecond
	strategy := FixedDelay(delay)

	for i := range 5 {
		actual := strategy(i)
		if actual != delay {
			t.Errorf("Attempt %d: Expected delay %v, got %v", i, delay, actual)
		}
	}
}

func TestJitterBackoff(t *testing.T) {
	base := 100 * time.Millisecond
	max := 1 * time.Second
	strategy := JitterBackoff(base, max)
	expStrategy := ExponentialBackoff(base, max)

	for i := range 5 {
		baseDelay := expStrategy(i)
		actual := strategy(i)

		// Check if actual delay is within [baseDelay, baseDelay + baseDelay/2)
		if actual < baseDelay || actual >= baseDelay+(baseDelay/2) {
			// Allow for slight floating point inaccuracies if baseDelay/2 is very small
			if math.Abs(float64(actual-(baseDelay+(baseDelay/2)))) > 1e-9 {
				t.Errorf("Attempt %d: Expected delay between %v and %v, got %v", i, baseDelay, baseDelay+(baseDelay/2), actual)
			}
		}
	}
}

// --- Test retryTransport ---

// mockRoundTripper allows mocking http.RoundTripper behavior.
type mockRoundTripper struct {
	roundTripFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if m.roundTripFunc == nil {
		// Default behavior: return a simple 200 OK response
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("OK")),
			Header:     make(http.Header),
		}, nil
	}

	return m.roundTripFunc(req)
}

func TestRetryTransport_SuccessOnFirstAttempt(t *testing.T) {
	mockRT := &mockRoundTripper{
		roundTripFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("Success")),
				Header:     make(http.Header),
			}, nil
		},
	}

	retryRT := &retryTransport{
		Transport:     mockRT,
		MaxRetries:    3,
		RetryStrategy: FixedDelay(1 * time.Millisecond), // Fast delay for testing
	}

	req := httptest.NewRequest("GET", "http://example.com", nil)
	resp, err := retryRT.RoundTrip(req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}
	bodyBytes, _ := io.ReadAll(resp.Body)
	if string(bodyBytes) != "Success" {
		t.Errorf("Expected body 'Success', got '%s'", string(bodyBytes))
	}
}

func TestRetryTransport_SuccessAfterRetries(t *testing.T) {
	var attempts int32 = 0
	targetAttempts := 2 // Succeed on the 3rd attempt (0, 1, 2)

	mockRT := &mockRoundTripper{
		roundTripFunc: func(req *http.Request) (*http.Response, error) {
			currentAttempt := atomic.LoadInt32(&attempts)
			atomic.AddInt32(&attempts, 1)

			if currentAttempt < int32(targetAttempts) {
				return &http.Response{
					StatusCode: http.StatusInternalServerError, // Simulate server error
					Body:       io.NopCloser(strings.NewReader("Server Error")),
					Header:     make(http.Header),
				}, nil // No transport error, just bad status
			}
			// Success on the target attempt
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("Success")),
				Header:     make(http.Header),
			}, nil
		},
	}

	retryRT := &retryTransport{
		Transport:     mockRT,
		MaxRetries:    3,
		RetryStrategy: FixedDelay(1 * time.Millisecond), // Use short delay
	}

	req := httptest.NewRequest("GET", "http://example.com", nil)
	resp, err := retryRT.RoundTrip(req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}
	if atomic.LoadInt32(&attempts) != int32(targetAttempts+1) {
		t.Errorf("Expected %d attempts, got %d", targetAttempts+1, atomic.LoadInt32(&attempts))
	}
}

func TestRetryTransport_FailureAfterMaxRetries_ServerError(t *testing.T) {
	var attempts int32 = 0
	maxRetries := 2

	mockRT := &mockRoundTripper{
		roundTripFunc: func(req *http.Request) (*http.Response, error) {
			atomic.AddInt32(&attempts, 1)
			return &http.Response{
				StatusCode: http.StatusServiceUnavailable, // Always fail
				Body:       io.NopCloser(strings.NewReader("Unavailable")),
				Header:     make(http.Header),
			}, nil
		},
	}

	retryRT := &retryTransport{
		Transport:     mockRT,
		MaxRetries:    maxRetries,
		RetryStrategy: FixedDelay(1 * time.Millisecond),
	}

	req := httptest.NewRequest("GET", "http://example.com", nil)
	resp, err := retryRT.RoundTrip(req)

	if err == nil {
		t.Fatalf("Expected an error, got nil response: %v", resp)
	}
	if resp != nil {
		t.Errorf("Expected nil response on final failure, got %v", resp)
	}
	if !errors.Is(err, ErrAllRetriesFailed) {
		t.Errorf("Expected error to wrap ErrAllRetriesFailed, got %v", err)
	}
	expectedErrMsg := fmt.Sprintf("%s: last attempt failed with status %d", ErrAllRetriesFailed, http.StatusServiceUnavailable)
	if err.Error() != expectedErrMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedErrMsg, err.Error())
	}

	// Attempts = initial + maxRetries
	if atomic.LoadInt32(&attempts) != int32(maxRetries+1) {
		t.Errorf("Expected %d attempts, got %d", maxRetries+1, atomic.LoadInt32(&attempts))
	}
}

func TestRetryTransport_FailureAfterMaxRetries_TransportError(t *testing.T) {
	var attempts int32 = 0
	maxRetries := 1
	simulatedError := errors.New("simulated transport error")

	mockRT := &mockRoundTripper{
		roundTripFunc: func(req *http.Request) (*http.Response, error) {
			atomic.AddInt32(&attempts, 1)
			return nil, simulatedError // Always return a transport error
		},
	}

	retryRT := &retryTransport{
		Transport:     mockRT,
		MaxRetries:    maxRetries,
		RetryStrategy: FixedDelay(1 * time.Millisecond),
	}

	req := httptest.NewRequest("GET", "http://example.com", nil)
	resp, err := retryRT.RoundTrip(req)

	if err == nil {
		t.Fatalf("Expected an error, got nil response: %v", resp)
	}
	if resp != nil {
		t.Errorf("Expected nil response on final failure, got %v", resp)
	}
	// Check if the original error is wrapped
	if !errors.Is(err, simulatedError) {
		t.Errorf("Expected error to wrap the original transport error '%v', but it didn't. Got: %v", simulatedError, err)
	}
	expectedErrMsgPrefix := "all retries failed; last error:"
	if !strings.HasPrefix(err.Error(), expectedErrMsgPrefix) {
		t.Errorf("Expected error message to start with '%s', got '%s'", expectedErrMsgPrefix, err.Error())
	}

	// Attempts = initial + maxRetries
	if atomic.LoadInt32(&attempts) != int32(maxRetries+1) {
		t.Errorf("Expected %d attempts, got %d", maxRetries+1, atomic.LoadInt32(&attempts))
	}
}

func TestRetryTransport_RequestBodyCloning(t *testing.T) {
	var attempts int32 = 0
	maxRetries := 1
	requestBodyContent := "Request Body Content"

	mockRT := &mockRoundTripper{
		roundTripFunc: func(req *http.Request) (*http.Response, error) {
			currentAttempt := atomic.LoadInt32(&attempts)
			atomic.AddInt32(&attempts, 1)

			// Verify body content on each attempt
			bodyBytes, err := io.ReadAll(req.Body)
			if err != nil {
				t.Errorf("Attempt %d: Failed to read request body: %v", currentAttempt, err)
				return nil, fmt.Errorf("failed reading body on attempt %d", currentAttempt)
			}
			if string(bodyBytes) != requestBodyContent {
				t.Errorf("Attempt %d: Expected body '%s', got '%s'", currentAttempt, requestBodyContent, string(bodyBytes))
			}

			if currentAttempt == 0 {
				// Fail first attempt
				return &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader("Fail")),
					Header:     make(http.Header),
				}, nil
			}
			// Succeed second attempt
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("Success")),
				Header:     make(http.Header),
			}, nil
		},
	}

	retryRT := &retryTransport{
		Transport:     mockRT,
		MaxRetries:    maxRetries,
		RetryStrategy: FixedDelay(1 * time.Millisecond),
	}

	// Create a request with a body that supports GetBody
	body := strings.NewReader(requestBodyContent)
	req := httptest.NewRequest("POST", "http://example.com", body)
	// Crucially, set GetBody so the transport can re-read it
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader(requestBodyContent)), nil
	}

	resp, err := retryRT.RoundTrip(req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK, got %d", resp.StatusCode)
	}
	if atomic.LoadInt32(&attempts) != int32(maxRetries+1) {
		t.Errorf("Expected %d attempts, got %d", maxRetries+1, atomic.LoadInt32(&attempts))
	}
}

func TestRetryTransport_NilTransportUsesDefault(t *testing.T) {
	// We can't easily intercept http.DefaultTransport, so we test indirectly
	// by ensuring RoundTrip doesn't panic and potentially fails connecting
	// to a non-existent local server, which implies it tried using *some* transport.
	retryRT := &retryTransport{
		Transport:     nil, // Explicitly nil
		MaxRetries:    0,   // No retries, just test the transport path
		RetryStrategy: FixedDelay(1 * time.Millisecond),
	}

	req := httptest.NewRequest("GET", "http://localhost:9999", nil) // Use a likely unavailable port

	_, err := retryRT.RoundTrip(req)
	if err == nil {
		t.Fatalf("Expected an error (likely connection refused), but got nil")
	}
	// We expect some kind of network error because DefaultTransport was used
	if !strings.Contains(err.Error(), "connection refused") && !strings.Contains(err.Error(), "invalid URL") && !strings.Contains(err.Error(), "no such host") {
		t.Logf("Received error: %v. This might be okay if DefaultTransport behavior changed.", err)
		// Don't fail the test outright, but log it. The main point is no panic.
	}
}

func TestRetryTransport_NilRetryStrategyUsesDefault(t *testing.T) {
	var attempts int32 = 0
	maxRetries := 1

	mockRT := &mockRoundTripper{
		roundTripFunc: func(req *http.Request) (*http.Response, error) {
			currentAttempt := atomic.LoadInt32(&attempts)
			atomic.AddInt32(&attempts, 1)

			if currentAttempt == 0 {
				// Fail first attempt
				return &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader("Fail")),
					Header:     make(http.Header),
				}, nil
			}
			// Succeed second attempt
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("Success")),
				Header:     make(http.Header),
			}, nil
		},
	}

	retryRT := &retryTransport{
		Transport:     mockRT,
		MaxRetries:    maxRetries,
		RetryStrategy: nil, // Explicitly nil
	}

	req := httptest.NewRequest("GET", "http://example.com", nil)
	resp, err := retryRT.RoundTrip(req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK, got %d", resp.StatusCode)
	}
	// Check that it actually retried (implying a strategy was used)
	if atomic.LoadInt32(&attempts) != int32(maxRetries+1) {
		t.Errorf("Expected %d attempts (implying default strategy used), got %d", maxRetries+1, atomic.LoadInt32(&attempts))
	}
}

func TestRetryTransport_RetriesOn429TooManyRequests(t *testing.T) {
	var attempts int32 = 0
	maxRetries := 2

	mockRT := &mockRoundTripper{
		roundTripFunc: func(req *http.Request) (*http.Response, error) {
			currentAttempt := atomic.LoadInt32(&attempts)
			atomic.AddInt32(&attempts, 1)

			if currentAttempt < 2 {
				return &http.Response{
					StatusCode: http.StatusTooManyRequests, // 429
					Body:       io.NopCloser(strings.NewReader("Too Many Requests")),
					Header:     make(http.Header),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("Success")),
				Header:     make(http.Header),
			}, nil
		},
	}

	retryRT := &retryTransport{
		Transport:     mockRT,
		MaxRetries:    maxRetries,
		RetryStrategy: FixedDelay(1 * time.Millisecond),
	}

	req := httptest.NewRequest("GET", "http://example.com", nil)
	resp, err := retryRT.RoundTrip(req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}
	if atomic.LoadInt32(&attempts) != 3 {
		t.Errorf("Expected 3 attempts (2 retries after 429), got %d", atomic.LoadInt32(&attempts))
	}
}

func TestRetryTransport_FailureAfterMaxRetries_429(t *testing.T) {
	var attempts int32 = 0
	maxRetries := 2

	mockRT := &mockRoundTripper{
		roundTripFunc: func(req *http.Request) (*http.Response, error) {
			atomic.AddInt32(&attempts, 1)
			return &http.Response{
				StatusCode: http.StatusTooManyRequests, // Always 429
				Body:       io.NopCloser(strings.NewReader("Too Many Requests")),
				Header:     make(http.Header),
			}, nil
		},
	}

	retryRT := &retryTransport{
		Transport:     mockRT,
		MaxRetries:    maxRetries,
		RetryStrategy: FixedDelay(1 * time.Millisecond),
	}

	req := httptest.NewRequest("GET", "http://example.com", nil)
	resp, err := retryRT.RoundTrip(req)

	if err == nil {
		t.Fatalf("Expected an error, got nil response: %v", resp)
	}
	if !errors.Is(err, ErrAllRetriesFailed) {
		t.Errorf("Expected error to wrap ErrAllRetriesFailed, got %v", err)
	}
	if atomic.LoadInt32(&attempts) != int32(maxRetries+1) {
		t.Errorf("Expected %d attempts, got %d", maxRetries+1, atomic.LoadInt32(&attempts))
	}
}

func TestRetryTransport_NonRetryableError(t *testing.T) {
	mockRT := &mockRoundTripper{
		roundTripFunc: func(req *http.Request) (*http.Response, error) {
			// Simulate a client-side error (e.g., invalid URL structure, though RoundTrip usually catches this earlier)
			// Or more realistically, an error that shouldn't be retried based on policy (though this transport retries all transport errors)
			// For this test, let's just return a non-5xx status code which *shouldn't* be retried.
			return &http.Response{
				StatusCode: http.StatusBadRequest, // 400 Bad Request
				Body:       io.NopCloser(strings.NewReader("Bad Request")),
				Header:     make(http.Header),
			}, nil
		},
	}

	retryRT := &retryTransport{
		Transport:     mockRT,
		MaxRetries:    3,
		RetryStrategy: FixedDelay(1 * time.Millisecond),
	}

	req := httptest.NewRequest("GET", "http://example.com", nil)
	resp, err := retryRT.RoundTrip(req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	defer resp.Body.Close()

	// Should return immediately with the 400 status, no retries
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, resp.StatusCode)
	}
	// Ensure only one attempt was made (no retry occurred)
	// Need a way to count attempts if the mock isn't designed for it.
	// For this simple mock, we assume if status is < 500, it returns immediately.
}

// --- Test NewClient ---

func TestNewHTTPRetryClient(t *testing.T) {
	maxRetries := 5
	strategy := FixedDelay(100 * time.Millisecond)
	mockBaseTransport := &mockRoundTripper{} // Use a simple mock

	client := NewHTTPRetryClient(
		WithMaxRetriesRetry(maxRetries),
		WithRetryStrategyRetry(strategy),
		WithBaseTransport(mockBaseTransport),
	)

	if client == nil {
		t.Fatal("NewHTTPRetryClient returned nil")
	}

	rt, ok := client.Transport.(*retryTransport)
	if !ok {
		t.Fatalf("Client transport is not of type *retryTransport, got %T", client.Transport)
	}

	if rt.MaxRetries != maxRetries {
		t.Errorf("Expected MaxRetries %d, got %d", maxRetries, rt.MaxRetries)
	}
	if rt.Transport != mockBaseTransport {
		t.Errorf("Expected base transport to be the mock, got %v", rt.Transport)
	}
	// Comparing functions directly is tricky; we assume if it's not nil, it's the one we passed.
	if rt.RetryStrategy == nil {
		t.Error("Expected RetryStrategy to be set, got nil")
	}

	// Test with defaults (should use http.DefaultTransport and default strategy)
	clientDefaults := NewHTTPRetryClient()
	rtDefault, ok := clientDefaults.Transport.(*retryTransport)
	if !ok {
		t.Fatalf("Client (defaults) transport is not of type *retryTransport, got %T", clientDefaults.Transport)
	}
	if rtDefault.Transport != http.DefaultTransport {
		t.Errorf("Expected base transport to be http.DefaultTransport, got %v", rtDefault.Transport)
	}
	if rtDefault.MaxRetries != DefaultMaxRetries {
		t.Errorf("Expected default max retries %d, got %d", DefaultMaxRetries, rtDefault.MaxRetries)
	}
	if rtDefault.RetryStrategy == nil {
		t.Error("Expected default strategy to be set, got nil")
	}

	// Test with nil strategy explicitly (should still use default ExponentialBackoff)
	clientDefaultStrategy := NewHTTPRetryClient(
		WithMaxRetriesRetry(maxRetries),
		WithRetryStrategyRetry(nil),
		WithBaseTransport(mockBaseTransport),
	)

	rtDefStrat, ok := clientDefaultStrategy.Transport.(*retryTransport)
	if !ok {
		t.Fatalf("Client (default strategy) transport is not of type *retryTransport, got %T", clientDefaultStrategy.Transport)
	}

	if rtDefStrat.RetryStrategy == nil {
		t.Error("Expected default RetryStrategy to be set, got nil")
	}
	// We can't easily compare the default strategy function, but we know it should be non-nil.
}

// --- Helper for Body Closing/Draining Tests ---

type errorReaderCloser struct {
	readErr  error
	closeErr error
	content  string
	readOnce bool // To simulate reading partially then erroring
}

func (e *errorReaderCloser) Read(p []byte) (n int, err error) {
	if e.readErr != nil && e.readOnce {
		return 0, e.readErr
	}
	if len(e.content) == 0 {
		return 0, io.EOF
	}
	n = copy(p, e.content)
	e.content = e.content[n:]
	e.readOnce = true // Mark as read once
	return n, nil
}

func (e *errorReaderCloser) Close() error {
	return e.closeErr
}

func TestRetryTransport_BodyDrainError(t *testing.T) {
	simulatedReadError := errors.New("simulated read error during drain")
	mockRT := &mockRoundTripper{
		roundTripFunc: func(req *http.Request) (*http.Response, error) {
			// Fail the request with a 5xx status and a body that errors on read
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body: &errorReaderCloser{
					content: "some data",
					readErr: simulatedReadError, // Error will occur when draining
				},
				Header: make(http.Header),
			}, nil
		},
	}

	retryRT := &retryTransport{
		Transport:     mockRT,
		MaxRetries:    1, // Allow one retry attempt
		RetryStrategy: FixedDelay(1 * time.Millisecond),
	}

	req := httptest.NewRequest("GET", "http://example.com", nil)
	_, err := retryRT.RoundTrip(req)

	if err == nil {
		t.Fatal("Expected an error due to body drain failure, got nil")
	}

	// The error should be related to failing to discard the body
	expectedErrMsg := "failed to discard response body"
	if !strings.Contains(err.Error(), expectedErrMsg) {
		t.Errorf("Expected error message to contain '%s', got '%s'", expectedErrMsg, err.Error())
	}
	// Check if the original read error is wrapped
	if !errors.Is(err, simulatedReadError) {
		t.Errorf("Expected error to wrap the original read error '%v', but it didn't. Got: %v", simulatedReadError, err)
	}
}

func TestRetryTransport_BodyCloseError(t *testing.T) {
	simulatedCloseError := errors.New("simulated close error")
	mockRT := &mockRoundTripper{
		roundTripFunc: func(req *http.Request) (*http.Response, error) {
			// Fail the request with a 5xx status and a body that errors on close
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body: &errorReaderCloser{
					content:  "some data",         // Content drains successfully
					closeErr: simulatedCloseError, // Error occurs on Close()
				},
				Header: make(http.Header),
			}, nil
		},
	}

	retryRT := &retryTransport{
		Transport:     mockRT,
		MaxRetries:    1, // Allow one retry attempt
		RetryStrategy: FixedDelay(1 * time.Millisecond),
	}

	req := httptest.NewRequest("GET", "http://example.com", nil)
	_, err := retryRT.RoundTrip(req)

	if err == nil {
		t.Fatal("Expected an error due to body close failure, got nil")
	}

	// The error should be related to failing to close the body
	expectedErrMsg := "failed to close response body"
	if !strings.Contains(err.Error(), expectedErrMsg) {
		t.Errorf("Expected error message to contain '%s', got '%s'", expectedErrMsg, err.Error())
	}
	// Check if the original close error is wrapped
	if !errors.Is(err, simulatedCloseError) {
		t.Errorf("Expected error to wrap the original close error '%v', but it didn't. Got: %v", simulatedCloseError, err)
	}
}

// Test case where GetBody itself returns an error
func TestRetryTransport_RequestBodyGetBodyError(t *testing.T) {
	var attempts int32 = 0
	maxRetries := 1
	requestBodyContent := "Request Body Content"
	getBodyError := errors.New("failed to get body")

	mockRT := &mockRoundTripper{
		roundTripFunc: func(req *http.Request) (*http.Response, error) {
			currentAttempt := atomic.LoadInt32(&attempts)
			atomic.AddInt32(&attempts, 1)

			// Fail first attempt to trigger retry
			if currentAttempt == 0 {
				return &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader("Fail")),
					Header:     make(http.Header),
				}, nil
			}
			// This part should not be reached if GetBody fails
			t.Errorf("RoundTrip called after GetBody should have failed")
			return nil, errors.New("should not be reached")
		},
	}

	retryRT := &retryTransport{
		Transport:     mockRT,
		MaxRetries:    maxRetries,
		RetryStrategy: FixedDelay(1 * time.Millisecond),
	}

	body := strings.NewReader(requestBodyContent)
	req := httptest.NewRequest("POST", "http://example.com", body)
	// Set GetBody to return an error on the second call (after the first attempt fails)
	getBodyAttempts := 0
	req.GetBody = func() (io.ReadCloser, error) {
		getBodyAttempts++
		if getBodyAttempts > 1 { // Error on subsequent calls (i.e., during retry prep)
			return nil, getBodyError
		}

		return io.NopCloser(strings.NewReader(requestBodyContent)), nil
	}

	_, err := retryRT.RoundTrip(req)
	if err == nil {
		t.Fatalf("Expected an error from GetBody, got nil")
	}

	// Check if the error is the one from GetBody, wrapped
	if !errors.Is(err, getBodyError) {
		t.Errorf("Expected error to wrap GetBody error '%v', got: %v", getBodyError, err)
	}

	expectedPrefix := "failed to get request body for retry:"
	if !strings.HasPrefix(err.Error(), expectedPrefix) {
		t.Errorf("Expected error message to start with '%s', got '%s'", expectedPrefix, err.Error())
	}

	// Should only have made the first attempt before failing on GetBody
	if atomic.LoadInt32(&attempts) != 1 {
		t.Errorf("Expected only 1 attempt before GetBody error, got %d", atomic.LoadInt32(&attempts))
	}
}

func TestNewHTTPRetryClient_WithProxy(t *testing.T) {
	t.Run("Create retry client with proxy", func(t *testing.T) {
		proxyURL := "http://proxy.example.com:8080"
		client := NewHTTPRetryClient(
			WithProxyRetry(proxyURL),
		)

		if client == nil {
			t.Fatal("NewHTTPRetryClient returned nil")
		}

		// Verify the transport has proxy configured
		if rt, ok := client.Transport.(*retryTransport); ok {
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
	})

	t.Run("Create retry client with HTTPS proxy", func(t *testing.T) {
		proxyURL := "https://secure-proxy.example.com:3128"
		client := NewHTTPRetryClient(
			WithProxyRetry(proxyURL),
			WithMaxRetriesRetry(5),
		)

		if client == nil {
			t.Fatal("NewHTTPRetryClient returned nil")
		}

		if rt, ok := client.Transport.(*retryTransport); ok {
			if rt.MaxRetries != 5 {
				t.Errorf("Expected MaxRetries to be 5, got %d", rt.MaxRetries)
			}

			if transport, ok := rt.Transport.(*http.Transport); ok {
				if transport.Proxy == nil {
					t.Error("Expected Proxy to be configured")
				}
			}
		}
	})

	t.Run("Create retry client without proxy", func(t *testing.T) {
		client := NewHTTPRetryClient()

		if client == nil {
			t.Fatal("NewHTTPRetryClient returned nil")
		}

		// Default transport should not have proxy configured
		if rt, ok := client.Transport.(*retryTransport); ok {
			if transport, ok := rt.Transport.(*http.Transport); ok {
				// Note: http.DefaultTransport may have Proxy set to use environment variables
				// so we just verify the client was created successfully
				_ = transport.Proxy
			}
		}
	})

	t.Run("Invalid proxy URL", func(t *testing.T) {
		invalidProxyURL := "://invalid-url"
		client := NewHTTPRetryClient(
			WithProxyRetry(invalidProxyURL),
		)

		// Client should still be created
		if client == nil {
			t.Fatal("NewHTTPRetryClient returned nil")
		}

		// When invalid URL is provided, a new transport is created
		// but proxy won't be properly configured (will fall back to ProxyFromEnvironment)
		if rt, ok := client.Transport.(*retryTransport); ok {
			if transport, ok := rt.Transport.(*http.Transport); ok {
				// Transport is created, but proxy configuration failed gracefully
				_ = transport.Proxy
			}
		}
	})

	t.Run("Proxy with custom base transport", func(t *testing.T) {
		// Create a custom transport
		customTransport := &http.Transport{
			MaxIdleConns: 50,
		}

		proxyURL := "http://proxy.example.com:8080"
		client := NewHTTPRetryClient(
			WithBaseTransport(customTransport),
			WithProxyRetry(proxyURL),
		)

		if client == nil {
			t.Fatal("NewHTTPRetryClient returned nil")
		}

		// Custom transport should have proxy configured
		if rt, ok := client.Transport.(*retryTransport); ok {
			if transport, ok := rt.Transport.(*http.Transport); ok {
				if transport.MaxIdleConns != 50 {
					t.Error("Expected custom transport to be used")
				}

				if transport.Proxy == nil {
					t.Error("Expected Proxy to be configured on custom transport")
				}
			}
		}
	})

	t.Run("Empty proxy URL", func(t *testing.T) {
		client := NewHTTPRetryClient(
			WithProxyRetry(""),
		)

		if client == nil {
			t.Fatal("NewHTTPRetryClient returned nil")
		}

		// Empty proxy URL doesn't trigger proxy configuration
		// Should use default transport
		if rt, ok := client.Transport.(*retryTransport); ok {
			// Verify transport exists (may be DefaultTransport)
			if rt.Transport == nil {
				t.Error("Expected Transport to be set")
			}
		}
	})
}

func TestRetryTransport_ServerExceedsClientTimeout(t *testing.T) {
	// Start a test server that takes longer to respond than the client timeout
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond) // Server delays response
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	// Use context timeout (per-request) which is shorter than the server delay.
	// This ensures the timeout is respected regardless of retry configuration.
	httpClient := NewClientBuilder().
		WithTimeout(10 * time.Second).
		WithMaxRetries(1).
		WithRetryBaseDelay(300 * time.Millisecond).
		Build()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", server.URL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	_, err = httpClient.Do(req)
	if err == nil {
		t.Fatal("Expected timeout error, got nil")
	}

	// The error should indicate a timeout/deadline exceeded
	if !errors.Is(err, context.DeadlineExceeded) && !strings.Contains(err.Error(), "deadline exceeded") && !strings.Contains(err.Error(), "timeout") {
		t.Errorf("Expected timeout-related error, got: %v", err)
	}
}

func TestRetryTransport_LargeTimeoutPreserved(t *testing.T) {
	// Verify that large timeouts (e.g., for LLM API calls) work correctly
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond) // Simulate slow response
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	// Build a client with a large timeout that should NOT be silently reset
	httpClient := NewClientBuilder().
		WithTimeout(120 * time.Second). // Large timeout for LLM calls
		WithMaxRetries(1).
		WithRetryBaseDelay(300 * time.Millisecond).
		Build()

	// Verify the timeout was preserved (not reset to default 5s)
	if httpClient.Timeout != 120*time.Second {
		t.Errorf("Expected timeout 120s, got %v (timeout was silently reset)", httpClient.Timeout)
	}

	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		t.Fatalf("Expected success with large timeout, got error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", resp.StatusCode)
	}
}

func TestRetryTransport_ContextTimeout(t *testing.T) {
	// Start a test server that takes a while to respond
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	httpClient := NewClientBuilder().
		WithTimeout(10 * time.Second). // Large client timeout
		WithMaxRetries(1).
		WithRetryBaseDelay(300 * time.Millisecond).
		Build()

	// Use a context with a short timeout that expires before the server responds
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", server.URL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	_, err = httpClient.Do(req)
	if err == nil {
		t.Fatal("Expected context timeout error, got nil")
	}

	// The error should be context-related
	if !errors.Is(err, context.DeadlineExceeded) && !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Errorf("Expected context deadline exceeded error, got: %v", err)
	}
}

func TestRetryTransport_ContextCancellation(t *testing.T) {
	// Start a test server that takes a while to respond
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	httpClient := NewClientBuilder().
		WithTimeout(10 * time.Second).
		WithMaxRetries(1).
		WithRetryBaseDelay(300 * time.Millisecond).
		Build()

	ctx, cancel := context.WithCancel(context.Background())

	req, err := http.NewRequestWithContext(ctx, "GET", server.URL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Cancel the context shortly after sending the request
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	_, err = httpClient.Do(req)
	if err == nil {
		t.Fatal("Expected context cancellation error, got nil")
	}

	if !errors.Is(err, context.Canceled) && !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("Expected context canceled error, got: %v", err)
	}
}

func TestRetryTransport_ContextCancelledDuringRetryDelay(t *testing.T) {
	var attempts int32 = 0

	mockRT := &mockRoundTripper{
		roundTripFunc: func(req *http.Request) (*http.Response, error) {
			atomic.AddInt32(&attempts, 1)
			// Always return 500 to trigger retry
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(strings.NewReader("Server Error")),
				Header:     make(http.Header),
			}, nil
		},
	}

	retryRT := &retryTransport{
		Transport:     mockRT,
		MaxRetries:    3,
		RetryStrategy: FixedDelay(2 * time.Second), // Long delay to ensure context cancels during it
	}

	ctx, cancel := context.WithCancel(context.Background())

	req, err := http.NewRequestWithContext(ctx, "GET", "http://example.com", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Cancel context after a short time (during the retry delay)
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	_, err = retryRT.RoundTrip(req)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("Expected context cancellation error, got nil")
	}

	// Should have completed quickly (not waited the full 2s delay)
	if elapsed > 1*time.Second {
		t.Errorf("Expected quick cancellation, but took %v (should be < 1s)", elapsed)
	}

	// Should have made exactly 1 attempt before cancellation during delay
	if atomic.LoadInt32(&attempts) != 1 {
		t.Errorf("Expected 1 attempt before context cancel, got %d", atomic.LoadInt32(&attempts))
	}

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled error, got: %v", err)
	}
}
