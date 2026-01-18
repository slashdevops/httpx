package httpx

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// TestRetryTransport_WithLogger verifies that logging works correctly when enabled
func TestRetryTransport_WithLogger(t *testing.T) {
	attempts := atomic.Int32{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt := attempts.Add(1)
		if attempt < 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	}))

	client := NewHTTPRetryClient(
		WithMaxRetriesRetry(2),
		WithRetryStrategyRetry(FixedDelay(10*time.Millisecond)),
		WithLoggerRetry(logger),
	)

	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Verify log output contains retry information
	logOutput := logBuf.String()
	if !strings.Contains(logOutput, "HTTP request returned server error, retrying") {
		t.Errorf("Expected retry log message, got: %s", logOutput)
	}

	if !strings.Contains(logOutput, "attempt=1") {
		t.Errorf("Expected attempt number in log, got: %s", logOutput)
	}

	if !strings.Contains(logOutput, "status_code=500") {
		t.Errorf("Expected status code in log, got: %s", logOutput)
	}
}

// TestRetryTransport_WithoutLogger verifies that no logging occurs when logger is nil
func TestRetryTransport_WithoutLogger(t *testing.T) {
	attempts := atomic.Int32{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt := attempts.Add(1)
		if attempt < 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// No logger provided (nil)
	client := NewHTTPRetryClient(
		WithMaxRetriesRetry(2),
		WithRetryStrategyRetry(FixedDelay(10*time.Millisecond)),
	)

	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Test passes if no panic occurred - logging is safely disabled
}

// TestRetryTransport_LoggerAllRetriesFailed verifies error logging when all retries fail
func TestRetryTransport_LoggerAllRetriesFailed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	client := NewHTTPRetryClient(
		WithMaxRetriesRetry(2),
		WithRetryStrategyRetry(FixedDelay(10*time.Millisecond)),
		WithLoggerRetry(logger),
	)

	_, err := client.Get(server.URL)
	if err == nil {
		t.Fatal("Expected error when all retries fail")
	}

	// Verify error log output
	logOutput := logBuf.String()
	if !strings.Contains(logOutput, "All retry attempts failed") {
		t.Errorf("Expected final error log message, got: %s", logOutput)
	}

	if !strings.Contains(logOutput, "attempts=3") { // 1 initial + 2 retries
		t.Errorf("Expected attempt count in log, got: %s", logOutput)
	}
}

// TestClientBuilder_WithLogger verifies logger integration with ClientBuilder
func TestClientBuilder_WithLogger(t *testing.T) {
	attempts := atomic.Int32{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt := attempts.Add(1)
		if attempt < 2 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	}))

	client := NewClientBuilder().
		WithMaxRetries(2).
		WithRetryBaseDelay(500 * time.Millisecond).
		WithRetryMaxDelay(10 * time.Second).
		WithLogger(logger).
		Build()

	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Verify logging occurred
	logOutput := logBuf.String()
	if !strings.Contains(logOutput, "retrying") {
		t.Errorf("Expected retry log from ClientBuilder, got: %s", logOutput)
	}
}

// TestGenericClient_WithLogger verifies logger integration with GenericClient
func TestGenericClient_WithLogger(t *testing.T) {
	type Response struct {
		Message string `json:"message"`
	}

	attempts := atomic.Int32{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt := attempts.Add(1)
		if attempt < 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"success"}`))
	}))
	defer server.Close()

	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	}))

	client := NewGenericClient[Response](
		WithMaxRetries[Response](2),
		WithRetryBaseDelay[Response](500*time.Millisecond),
		WithRetryMaxDelay[Response](10*time.Second),
		WithLogger[Response](logger),
	)

	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.Data.Message != "success" {
		t.Errorf("Expected message 'success', got '%s'", resp.Data.Message)
	}

	// Verify logging occurred
	logOutput := logBuf.String()
	if !strings.Contains(logOutput, "retrying") {
		t.Errorf("Expected retry log from GenericClient, got: %s", logOutput)
	}
}
