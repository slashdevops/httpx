package httpx_test

import (
	"fmt"
	"time"

	"github.com/slashdevops/httpx"
)

// ExampleClientBuilder_WithProxy demonstrates how to configure an HTTP client with a proxy.
func ExampleClientBuilder_WithProxy() {
	// Create an HTTP client with proxy configuration
	client := httpx.NewClientBuilder().
		WithProxy("http://proxy.example.com:8080").
		WithTimeout(10 * time.Second).
		WithMaxRetries(3).
		WithRetryStrategy(httpx.ExponentialBackoffStrategy).
		Build()

	// Client is ready to use with proxy
	fmt.Printf("Client configured with proxy\n")
	_ = client

	// Output: Client configured with proxy
}

// ExampleClientBuilder_WithProxy_https demonstrates using an HTTPS proxy.
func ExampleClientBuilder_WithProxy_https() {
	// Create an HTTP client with HTTPS proxy
	client := httpx.NewClientBuilder().
		WithProxy("https://secure-proxy.example.com:3128").
		WithTimeout(15 * time.Second).
		Build()

	fmt.Printf("Client configured with HTTPS proxy\n")
	_ = client

	// Output: Client configured with HTTPS proxy
}

// ExampleClientBuilder_WithProxy_authentication demonstrates using a proxy with authentication.
func ExampleClientBuilder_WithProxy_authentication() {
	// Create an HTTP client with authenticated proxy
	// Proxy credentials are included in the URL
	client := httpx.NewClientBuilder().
		WithProxy("http://username:password@proxy.example.com:8080").
		WithTimeout(10 * time.Second).
		Build()

	fmt.Printf("Client configured with authenticated proxy\n")
	_ = client

	// Output: Client configured with authenticated proxy
}

// ExampleNewGenericClient_withProxy demonstrates using a proxy with the generic client.
func ExampleNewGenericClient_withProxy() {
	type APIResponse struct {
		Message string `json:"message"`
		Status  string `json:"status"`
	}

	// Create a generic client with proxy configuration
	client := httpx.NewGenericClient[APIResponse](
		httpx.WithProxy[APIResponse]("http://proxy.example.com:8080"),
		httpx.WithTimeout[APIResponse](10*time.Second),
		httpx.WithMaxRetries[APIResponse](3),
	)

	fmt.Printf("Generic client configured with proxy\n")
	_ = client

	// Output: Generic client configured with proxy
}

// ExampleNewGenericClient_withProxy_combined demonstrates combining proxy with other options.
func ExampleNewGenericClient_withProxy_combined() {
	type User struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	// Create a fully configured generic client with proxy
	client := httpx.NewGenericClient[User](
		httpx.WithProxy[User]("http://proxy.example.com:8080"),
		httpx.WithTimeout[User](15*time.Second),
		httpx.WithMaxRetries[User](5),
		httpx.WithRetryStrategy[User](httpx.JitterBackoffStrategy),
		httpx.WithRetryBaseDelay[User](500*time.Millisecond),
		httpx.WithRetryMaxDelay[User](30*time.Second),
	)

	fmt.Printf("Generic client with proxy and retry configuration\n")
	_ = client

	// Output: Generic client with proxy and retry configuration
}

// ExampleNewHTTPRetryClient_withProxy demonstrates using a proxy with the retry client.
func ExampleNewHTTPRetryClient_withProxy() {
	// Create a retry client with proxy configuration
	client := httpx.NewHTTPRetryClient(
		httpx.WithProxyRetry("http://proxy.example.com:8080"),
		httpx.WithMaxRetriesRetry(5),
		httpx.WithRetryStrategyRetry(
			httpx.ExponentialBackoff(500*time.Millisecond, 30*time.Second),
		),
	)

	fmt.Printf("Retry client configured with proxy\n")
	_ = client

	// Output: Retry client configured with proxy
}

// ExampleClientBuilder_WithProxy_disabling demonstrates how to disable proxy.
func ExampleClientBuilder_WithProxy_disabling() {
	// Create an HTTP client without proxy (explicit disable)
	client := httpx.NewClientBuilder().
		WithProxy(""). // Empty string disables proxy
		WithTimeout(10 * time.Second).
		Build()

	fmt.Printf("Client configured without proxy\n")
	_ = client

	// Output: Client configured without proxy
}

// ExampleNewGenericClient_withProxy_portVariations demonstrates different proxy port configurations.
func ExampleNewGenericClient_withProxy_portVariations() {
	type Data struct {
		Value string `json:"value"`
	}

	// Example 1: Standard HTTP proxy on port 8080
	client1 := httpx.NewGenericClient[Data](
		httpx.WithProxy[Data]("http://proxy.example.com:8080"),
	)

	// Example 2: HTTPS proxy on port 3128 (common Squid port)
	client2 := httpx.NewGenericClient[Data](
		httpx.WithProxy[Data]("https://proxy.example.com:3128"),
	)

	// Example 3: Custom port
	client3 := httpx.NewGenericClient[Data](
		httpx.WithProxy[Data]("http://proxy.example.com:9090"),
	)

	fmt.Printf("Configured clients with different proxy ports\n")
	_, _, _ = client1, client2, client3

	// Output: Configured clients with different proxy ports
}
