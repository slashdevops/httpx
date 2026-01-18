package httpx_test

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/slashdevops/httpx"
)

// ExampleNewClientBuilder demonstrates creating a basic HTTP client with default settings
func ExampleNewClientBuilder() {
	// Create client with default settings
	client := httpx.NewClientBuilder().Build()

	// Use the client for HTTP requests
	resp, err := client.Get("https://api.example.com/health")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	fmt.Printf("Status: %s\n", resp.Status)
	// Output would show: Status: 200 OK
}

// ExampleNewClientBuilder_withTimeout demonstrates configuring a client with custom timeout
func ExampleNewClientBuilder_withTimeout() {
	// Create client with custom timeout
	client := httpx.NewClientBuilder().
		WithTimeout(10 * time.Second).
		Build()

	resp, err := client.Get("https://api.example.com/users")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Response length: %d bytes\n", len(body))
}

// ExampleNewClientBuilder_withRetryStrategy demonstrates configuring retry behavior
func ExampleNewClientBuilder_withRetryStrategy() {
	// Configure client with exponential backoff retry strategy
	client := httpx.NewClientBuilder().
		WithMaxRetries(5).
		WithRetryStrategy(httpx.ExponentialBackoffStrategy).
		WithRetryBaseDelay(500 * time.Millisecond).
		WithRetryMaxDelay(30 * time.Second).
		Build()

	// The client will automatically retry on transient failures
	resp, err := client.Get("https://api.example.com/data")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	fmt.Printf("Status: %d\n", resp.StatusCode)
}

// ExampleNewClientBuilder_fixedDelayStrategy demonstrates using fixed delay retry strategy
func ExampleNewClientBuilder_fixedDelayStrategy() {
	// Configure client with fixed delay between retries
	client := httpx.NewClientBuilder().
		WithMaxRetries(3).
		WithRetryStrategyAsString("fixed").
		WithRetryBaseDelay(1 * time.Second).
		Build()

	// Each retry will wait exactly 1 second
	resp, err := client.Get("https://api.example.com/endpoint")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	fmt.Printf("Success: %v\n", resp.StatusCode == 200)
}

// ExampleNewClientBuilder_jitterBackoffStrategy demonstrates using jitter backoff for retry
func ExampleNewClientBuilder_jitterBackoffStrategy() {
	// Configure client with jitter backoff to prevent thundering herd
	client := httpx.NewClientBuilder().
		WithMaxRetries(4).
		WithRetryStrategy(httpx.JitterBackoffStrategy).
		WithRetryBaseDelay(500 * time.Millisecond).
		WithRetryMaxDelay(10 * time.Second).
		Build()

	// Retries will have randomized delays
	resp, err := client.Get("https://api.example.com/resource")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	fmt.Printf("Completed with status: %d\n", resp.StatusCode)
}

// ExampleNewClientBuilder_connectionPooling demonstrates configuring connection pooling
func ExampleNewClientBuilder_connectionPooling() {
	// Configure connection pooling for high-throughput scenarios
	client := httpx.NewClientBuilder().
		WithMaxIdleConns(200).
		WithMaxIdleConnsPerHost(20).
		WithIdleConnTimeout(90 * time.Second).
		Build()

	// Reuse connections efficiently across multiple requests
	for i := 0; i < 10; i++ {
		resp, err := client.Get(fmt.Sprintf("https://api.example.com/items/%d", i))
		if err != nil {
			log.Printf("Request %d failed: %v", i, err)
			continue
		}
		resp.Body.Close()
	}

	fmt.Println("Completed batch requests")
}

// ExampleNewClientBuilder_tlsConfiguration demonstrates TLS timeout settings
func ExampleNewClientBuilder_tlsConfiguration() {
	// Configure TLS handshake timeout for secure connections
	client := httpx.NewClientBuilder().
		WithTLSHandshakeTimeout(10 * time.Second).
		WithExpectContinueTimeout(2 * time.Second).
		Build()

	resp, err := client.Get("https://secure-api.example.com/data")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	fmt.Printf("Secure request completed: %s\n", resp.Status)
}

// ExampleNewClientBuilder_disableKeepAlive demonstrates disabling connection reuse
func ExampleNewClientBuilder_disableKeepAlive() {
	// Disable keep-alive for scenarios requiring fresh connections
	client := httpx.NewClientBuilder().
		WithDisableKeepAlive(true).
		Build()

	// Each request will use a new connection
	resp, err := client.Get("https://api.example.com/status")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	fmt.Printf("Response: %s\n", resp.Status)
}

// ExampleNewClientBuilder_productionConfig demonstrates a production-ready configuration
func ExampleNewClientBuilder_productionConfig() {
	// Production-ready client with optimal settings
	client := httpx.NewClientBuilder().
		// Timeouts
		WithTimeout(30 * time.Second).
		WithTLSHandshakeTimeout(10 * time.Second).
		WithExpectContinueTimeout(1 * time.Second).

		// Connection pooling
		WithMaxIdleConns(100).
		WithMaxIdleConnsPerHost(10).
		WithIdleConnTimeout(90 * time.Second).

		// Retry configuration
		WithMaxRetries(5).
		WithRetryStrategy(httpx.ExponentialBackoffStrategy).
		WithRetryBaseDelay(500 * time.Millisecond).
		WithRetryMaxDelay(30 * time.Second).
		Build()

	// Create request with proper headers
	req, err := http.NewRequest("GET", "https://api.example.com/v1/users", nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer prod-token-xyz")
	req.Header.Set("User-Agent", "MyApp/2.0")

	// Execute with automatic retries and connection pooling
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Retrieved %d bytes with status %d\n", len(body), resp.StatusCode)
}

// ExampleNewClientBuilder_allOptions demonstrates using all available configuration options
func ExampleNewClientBuilder_allOptions() {
	// Comprehensive configuration showing all available options
	client := httpx.NewClientBuilder().
		// HTTP client timeout
		WithTimeout(20 * time.Second).

		// Connection pool settings
		WithMaxIdleConns(150).
		WithMaxIdleConnsPerHost(15).
		WithIdleConnTimeout(60 * time.Second).

		// TLS and protocol settings
		WithTLSHandshakeTimeout(8 * time.Second).
		WithExpectContinueTimeout(2 * time.Second).
		WithDisableKeepAlive(false).

		// Retry configuration
		WithMaxRetries(4).
		WithRetryStrategy(httpx.JitterBackoffStrategy).
		WithRetryBaseDelay(300 * time.Millisecond).
		WithRetryMaxDelay(20 * time.Second).
		Build()

	// Use the fully configured client
	resp, err := client.Get("https://api.example.com/comprehensive")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	fmt.Printf("Request completed with all options configured\n")
}

// ExampleNewClientBuilder_multipleClients demonstrates creating clients with different configurations
func ExampleNewClientBuilder_multipleClients() {
	// Fast client for health checks
	healthClient := httpx.NewClientBuilder().
		WithTimeout(2 * time.Second).
		WithMaxRetries(1).
		Build()

	// Standard client for API calls
	apiClient := httpx.NewClientBuilder().
		WithTimeout(15 * time.Second).
		WithMaxRetries(3).
		WithRetryStrategy(httpx.ExponentialBackoffStrategy).
		Build()

	// Heavy client for bulk operations
	bulkClient := httpx.NewClientBuilder().
		WithTimeout(60 * time.Second).
		WithMaxRetries(5).
		WithRetryStrategy(httpx.JitterBackoffStrategy).
		WithMaxIdleConns(200).
		WithMaxIdleConnsPerHost(20).
		Build()

	// Use different clients for different purposes
	_, _ = healthClient.Get("https://api.example.com/health")
	_, _ = apiClient.Get("https://api.example.com/users/123")
	_, _ = bulkClient.Post("https://api.example.com/bulk-import", "application/json", nil)

	fmt.Println("Multiple clients with different configurations")
}
