package httpx_test

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/slashdevops/httpx"
)

// User represents a user in the API
type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// Post represents a blog post
type Post struct {
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	UserID int    `json:"userId"`
}

// ExampleNewGenericClient demonstrates basic usage of the generic client
func ExampleNewGenericClient() {
	// Create a typed client for User responses with configuration
	client := httpx.NewGenericClient[User](
		httpx.WithTimeout[User](10*time.Second),
		httpx.WithMaxRetries[User](3),
		httpx.WithRetryStrategy[User](httpx.ExponentialBackoffStrategy),
	)

	// Make a GET request with full URL
	response, err := client.Get("https://api.example.com/users/1")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("User: %s (%s)\n", response.Data.Name, response.Data.Email)
}

// ExampleNewGenericClient_allOptions demonstrates using all configuration options
func ExampleNewGenericClient_allOptions() {
	// Create a fully configured client
	client := httpx.NewGenericClient[User](
		// Timeout configuration
		httpx.WithTimeout[User](15*time.Second),

		// Retry configuration
		httpx.WithMaxRetries[User](5),
		httpx.WithRetryStrategy[User](httpx.JitterBackoffStrategy),
		httpx.WithRetryBaseDelay[User](500*time.Millisecond),
		httpx.WithRetryMaxDelay[User](10*time.Second),

		// Connection pooling
		httpx.WithMaxIdleConns[User](100),
		httpx.WithMaxIdleConnsPerHost[User](10),
		httpx.WithIdleConnTimeout[User](90*time.Second),

		// TLS and handshake timeouts
		httpx.WithTLSHandshakeTimeout[User](10*time.Second),
		httpx.WithExpectContinueTimeout[User](1*time.Second),

		// Keep-alive
		httpx.WithDisableKeepAlive[User](false),
	)

	response, err := client.Get("https://api.example.com/users/1")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("User: %s\n", response.Data.Name)
}

// ExampleNewGenericClient_withPreConfiguredClient demonstrates using WithHTTPClient
func ExampleNewGenericClient_withPreConfiguredClient() {
	// Build a custom HTTP client using ClientBuilder
	httpClient := httpx.NewClientBuilder().
		WithMaxRetries(3).
		WithRetryStrategy(httpx.ExponentialBackoffStrategy).
		WithRetryBaseDelay(500 * time.Millisecond).
		WithTimeout(30 * time.Second).
		WithMaxIdleConns(50).
		Build()

	// Use it with GenericClient (takes precedence over other options)
	client := httpx.NewGenericClient[User](
		httpx.WithHTTPClient[User](httpClient),
	)

	response, err := client.Get("https://api.example.com/users/1")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("User: %s\n", response.Data.Name)
}

// ExampleGenericClient_Execute demonstrates using Execute with RequestBuilder
func ExampleGenericClient_Execute() {
	// Create a typed client for Post responses
	client := httpx.NewGenericClient[Post]()

	// Build a request with RequestBuilder
	req, err := httpx.NewRequestBuilder("https://jsonplaceholder.typicode.com").
		WithMethodGET().
		WithPath("/posts/1").
		WithAccept("application/json").
		Build()
	if err != nil {
		log.Fatal(err)
	}

	// Execute the request
	response, err := client.Execute(req)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Post: %s\n", response.Data.Title)
}

// ExampleGenericClient_Post demonstrates making a POST request
func ExampleGenericClient_Post() {
	client := httpx.NewGenericClient[Post]()

	// Create a new post using RequestBuilder
	newPost := Post{
		Title:  "My New Post",
		Body:   "This is the post content",
		UserID: 1,
	}

	req, err := httpx.NewRequestBuilder("https://jsonplaceholder.typicode.com").
		WithMethodPOST().
		WithPath("/posts").
		WithContentType("application/json").
		WithJSONBody(newPost).
		Build()
	if err != nil {
		log.Fatal(err)
	}

	response, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Created post with ID: %d\n", response.Data.ID)
}

// ExampleGenericClient_withRetry demonstrates using generic client with retry logic
func ExampleGenericClient_withRetry() {
	// Option 1: Use ClientBuilder and pass to GenericClient
	retryClient := httpx.NewClientBuilder().
		WithMaxRetries(3).
		WithRetryStrategy(httpx.ExponentialBackoffStrategy).
		WithRetryBaseDelay(500 * time.Millisecond).
		WithTimeout(30 * time.Second).
		Build()

	client := httpx.NewGenericClient[User](
		httpx.WithHTTPClient[User](retryClient),
	)

	// Option 2: Configure retry directly in GenericClient
	client2 := httpx.NewGenericClient[User](
		httpx.WithMaxRetries[User](3),
		httpx.WithRetryStrategy[User](httpx.ExponentialBackoffStrategy),
		httpx.WithRetryBaseDelay[User](500*time.Millisecond),
		httpx.WithTimeout[User](30*time.Second),
	)

	// Make requests - they will automatically retry on failures
	response, err := client.Get("https://api.example.com/users/1")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("User: %s\n", response.Data.Name)

	// Using second client
	response2, err := client2.Get("https://api.example.com/users/2")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("User: %s\n", response2.Data.Name)
}

// ExampleGenericClient_multipleClients demonstrates using multiple typed clients
func ExampleGenericClient_multipleClients() {
	baseURL := "https://api.example.com"

	// Create a client for User responses with specific configuration
	userClient := httpx.NewGenericClient[User](
		httpx.WithTimeout[User](10*time.Second),
		httpx.WithMaxRetries[User](3),
	)

	// Create a client for Post responses with different configuration
	postClient := httpx.NewGenericClient[Post](
		httpx.WithTimeout[Post](15*time.Second),
		httpx.WithMaxRetries[Post](5),
		httpx.WithRetryStrategy[Post](httpx.JitterBackoffStrategy),
	)

	// Fetch user
	userResp, err := userClient.Get(baseURL + "/users/1")
	if err != nil {
		log.Fatal(err)
	}

	// Fetch posts by that user
	postResp, err := postClient.Get(fmt.Sprintf("%s/users/%d/posts", baseURL, userResp.Data.ID))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("User %s has %d posts\n", userResp.Data.Name, len(postResp.Data.Title))
}

// ExampleGenericClient_ExecuteRaw demonstrates using ExecuteRaw for non-JSON responses
func ExampleGenericClient_ExecuteRaw() {
	client := httpx.NewGenericClient[User]()

	req, err := http.NewRequest(http.MethodGet, "https://example.com/image.png", nil)
	if err != nil {
		log.Fatal(err)
	}

	// Get raw response for binary data
	response, err := client.ExecuteRaw(req)
	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()

	fmt.Printf("Content-Type: %s\n", response.Header.Get("Content-Type"))
	fmt.Printf("Status: %d\n", response.StatusCode)
}

// ExampleGenericClient_errorHandling demonstrates error handling
func ExampleGenericClient_errorHandling() {
	client := httpx.NewGenericClient[User]()

	response, err := client.Get("https://api.example.com/users/999999")
	if err != nil {
		// Check if it's an ErrorResponse
		if apiErr, ok := err.(*httpx.ErrorResponse); ok {
			fmt.Printf("API Error: Status %d - %s\n", apiErr.StatusCode, apiErr.Message)
			return
		}
		// Other errors (network, parsing, etc.)
		log.Fatal(err)
	}

	fmt.Printf("User: %s\n", response.Data.Name)
}

// ExampleGenericClient_withCustomHeaders demonstrates adding custom headers per request
func ExampleGenericClient_withCustomHeaders() {
	client := httpx.NewGenericClient[User]()

	// Build request with headers using RequestBuilder
	req, err := httpx.NewRequestBuilder("https://api.example.com").
		WithMethodGET().
		WithPath("/users/1").
		WithHeader("Authorization", "Bearer token").
		WithHeader("X-Request-ID", "unique-id-123").
		WithHeader("X-Trace-ID", "trace-456").
		Build()
	if err != nil {
		log.Fatal(err)
	}

	// Execute the request
	response, err := client.Execute(req)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("User: %s\n", response.Data.Name)
}

// ExampleGenericClient_productionConfiguration demonstrates a production-ready configuration
func ExampleGenericClient_productionConfiguration() {
	// Configure client with production-ready settings
	client := httpx.NewGenericClient[User](
		// Aggressive retry for resilience
		httpx.WithMaxRetries[User](5),
		httpx.WithRetryStrategy[User](httpx.ExponentialBackoffStrategy),
		httpx.WithRetryBaseDelay[User](500*time.Millisecond),
		httpx.WithRetryMaxDelay[User](30*time.Second),

		// Reasonable timeout
		httpx.WithTimeout[User](30*time.Second),

		// Connection pooling for performance
		httpx.WithMaxIdleConns[User](100),
		httpx.WithMaxIdleConnsPerHost[User](10),
		httpx.WithIdleConnTimeout[User](90*time.Second),

		// TLS optimization
		httpx.WithTLSHandshakeTimeout[User](10*time.Second),
	)

	// Build request with headers
	req, err := httpx.NewRequestBuilder("https://api.example.com").
		WithMethodGET().
		WithPath("/users/1").
		WithHeader("Authorization", "Bearer prod-token").
		WithHeader("X-Request-ID", "req-123").
		WithUserAgent("MyApp/1.0.0").
		Build()
	if err != nil {
		log.Fatal(err)
	}

	// Execute with automatic retries and error handling
	response, err := client.Execute(req)
	if err != nil {
		if apiErr, ok := err.(*httpx.ErrorResponse); ok {
			log.Printf("API Error: %d - %s", apiErr.StatusCode, apiErr.Message)
			return
		}
		log.Fatal(err)
	}

	fmt.Printf("User: %s (%s)\n", response.Data.Name, response.Data.Email)
}

// ExampleGenericClient_retryStrategies demonstrates different retry strategies
func ExampleGenericClient_retryStrategies() {
	// Fixed delay - predictable retry timing
	fixedClient := httpx.NewGenericClient[User](
		httpx.WithRetryStrategyAsString[User]("fixed"),
		httpx.WithMaxRetries[User](3),
		httpx.WithRetryBaseDelay[User](1*time.Second),
	)

	// Exponential backoff - doubles delay each time
	exponentialClient := httpx.NewGenericClient[User](
		httpx.WithRetryStrategy[User](httpx.ExponentialBackoffStrategy),
		httpx.WithMaxRetries[User](5),
		httpx.WithRetryBaseDelay[User](500*time.Millisecond),
		httpx.WithRetryMaxDelay[User](10*time.Second),
	)

	// Jitter backoff - random delays to prevent thundering herd
	jitterClient := httpx.NewGenericClient[User](
		httpx.WithRetryStrategy[User](httpx.JitterBackoffStrategy),
		httpx.WithMaxRetries[User](3),
		httpx.WithRetryBaseDelay[User](500*time.Millisecond),
		httpx.WithRetryMaxDelay[User](5*time.Second),
	)

	// Use different clients based on use case
	_, _ = fixedClient.Get("https://api.example.com/users/1")
	_, _ = exponentialClient.Get("https://api.example.com/users/2")
	_, _ = jitterClient.Get("https://api.example.com/users/3")
}
