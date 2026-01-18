package httpx_test

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/slashdevops/httpx"
)

// TestExample_withLogger_retryLogging demonstrates how to enable logging for HTTP retries.
// By default, logging is disabled. Pass a *slog.Logger to see retry attempts.
func TestExample_withLogger_retryLogging(t *testing.T) {
	t.Skip("This is a demonstration of logger usage, not an automated test")
	attempts := atomic.Int32{}

	// Create a test server that fails twice, then succeeds
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt := attempts.Add(1)
		if attempt <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "Success")
	}))
	defer server.Close()

	// Create a logger with a custom level (Info level to see retry warnings)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Create client with logging enabled using ClientBuilder
	client := httpx.NewClientBuilder().
		WithMaxRetries(3).
		WithRetryBaseDelay(500 * time.Millisecond). // Use valid delay (>= 300ms)
		WithRetryMaxDelay(10 * time.Second).        // Use valid delay (>= 300ms)
		WithRetryStrategy(httpx.ExponentialBackoffStrategy).
		WithLogger(logger).
		Build()

	fmt.Println("Making request with retry logging enabled...")
	resp, err := client.Get(server.URL)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Response: %s\n", body)

	// Logs will show retry warnings (omitted from output due to timestamps)
	fmt.Println("Request succeeded after retries")
}

// TestExample_withLogger_genericClient demonstrates logging with the GenericClient.
func TestExample_withLogger_genericClient(t *testing.T) {
	t.Skip("This is a demonstration of logger usage, not an automated test")
	type Response struct {
		Message string `json:"message"`
	}

	attempts := atomic.Int32{}

	// Create a test server that fails once, then succeeds
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt := attempts.Add(1)
		if attempt == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"message":"Success"}`)
	}))
	defer server.Close()

	// Create a logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	}))

	// Create generic client with logging
	client := httpx.NewGenericClient[Response](
		httpx.WithMaxRetries[Response](2),
		httpx.WithRetryBaseDelay[Response](500*time.Millisecond), // Use valid delay
		httpx.WithRetryMaxDelay[Response](10*time.Second),        // Use valid delay
		httpx.WithRetryStrategy[Response](httpx.ExponentialBackoffStrategy),
		httpx.WithLogger[Response](logger),
	)

	fmt.Println("Making typed request with retry logging...")
	resp, err := client.Get(server.URL)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Response: %s\n", resp.Data.Message)
}

// Example_withLogger_disabled demonstrates the default behavior with no logging.
// This example shows that by default, logging is disabled for clean, silent operation.
func Example_withLogger_disabled() {
	attempts := atomic.Int32{}

	// Create a test server that fails twice, then succeeds
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt := attempts.Add(1)
		if attempt <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "Success")
	}))
	defer server.Close()

	// Create client WITHOUT logging (default behavior)
	// No logger means silent retries - clean operation without log noise
	client := httpx.NewClientBuilder().
		WithMaxRetries(3).
		WithRetryBaseDelay(500 * time.Millisecond). // Use valid delay
		WithRetryMaxDelay(10 * time.Second).        // Use valid delay
		Build()                                     // No WithLogger call = no logging

	fmt.Println("Making request with logging disabled (default)...")
	resp, err := client.Get(server.URL)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Response: %s\n", body)
	fmt.Println("No retry logs appear - silent operation")

	// Output:
	// Making request with logging disabled (default)...
	// Response: Success
	// No retry logs appear - silent operation
}

// TestExample_newHTTPRetryClient_withLogger demonstrates using NewHTTPRetryClient with logging.
func TestExample_newHTTPRetryClient_withLogger(t *testing.T) {
	t.Skip("This is a demonstration of logger usage, not an automated test")
	attempts := atomic.Int32{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt := attempts.Add(1)
		if attempt == 1 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "Success")
	}))
	defer server.Close()

	// Create a logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	}))

	// Create retry client with logger
	client := httpx.NewHTTPRetryClient(
		httpx.WithMaxRetriesRetry(3),
		httpx.WithRetryStrategyRetry(httpx.ExponentialBackoff(500*time.Millisecond, 10*time.Second)),
		httpx.WithLoggerRetry(logger),
	)

	fmt.Println("Making request with NewHTTPRetryClient and logging...")
	resp, err := client.Get(server.URL)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Response: %s\n", body)
}
