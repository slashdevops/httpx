# httpx

[![main branch](https://github.com/slashdevops/httpx/actions/workflows/main.yml/badge.svg)](https://github.com/slashdevops/httpx/actions/workflows/main.yml)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/slashdevops/httpx?style=plastic)
[![Go Reference](https://pkg.go.dev/badge/github.com/slashdevops/httpx.svg)](https://pkg.go.dev/github.com/slashdevops/httpx)
[![Go Report Card](https://goreportcard.com/badge/github.com/slashdevops/httpx)](https://goreportcard.com/report/github.com/slashdevops/httpx)
[![license](https://img.shields.io/github/license/slashdevops/httpx.svg)](https://github.com/slashdevops/httpx/blob/main/LICENSE)
[![Release](https://github.com/slashdevops/httpx/actions/workflows/release.yml/badge.svg)](https://github.com/slashdevops/httpx/actions/workflows/release.yml)

A comprehensive Go package for building and executing HTTP requests with advanced features.

**🚀 Zero Dependencies** - Built entirely using the Go standard library for maximum reliability, security, and minimal maintenance overhead. See [go.mod](go.mod)

## Key Features

- 🔨 **Fluent Request Builder** - Chainable API for constructing HTTP requests
- 🔄 **Automatic Retry Logic** - Configurable retry strategies with exponential backoff
- 🎯 **Type-Safe Generic Client** - Go generics for type-safe HTTP responses
- ✅ **Input Validation** - Comprehensive validation with error accumulation
- 🔐 **Authentication Support** - Built-in Basic and Bearer token authentication
- 🌐 **Proxy Support** - HTTP/HTTPS proxy configuration with authentication (supports corporate proxies, authenticated proxies, and custom ports)
- 📝 **Optional Logging** - slog integration for observability (disabled by default)
- 📦 **Zero External Dependencies** - Only Go standard library, no third-party packages

## Table of Contents

- [Installation](#installation)
- [Upgrade](#upgrade)
- [Quick Start](#quick-start)
- [Features](#features)
  - [Request Builder](#request-builder)
  - [Generic HTTP Client](#generic-http-client)
  - [Retry Logic](#retry-logic)
  - [Client Builder](#client-builder)
  - [Proxy Configuration](#proxy-configuration)
  - [Logging](#logging)
- [Examples](#examples)
- [API Reference](#api-reference)
- [Best Practices](#best-practices)
- [Contributing](#contributing)

## Installation

**Requirements:** Go 1.22 or higher

```bash
go get github.com/slashdevops/httpx
```

## Upgrade

To upgrade to the latest version, run:

```bash
go get -u github.com/slashdevops/httpx
```

## Quick Start

### Simple GET Request

```go
import "github.com/slashdevops/httpx"

// Build and execute a simple GET request
req, err := httpx.NewRequestBuilder("https://api.example.com").
    WithMethodGET().
    WithPath("/users/123").
    WithHeader("Accept", "application/json").
    Build()

if err != nil {
    log.Fatal(err)
}

// Use with standard http.Client
resp, err := http.DefaultClient.Do(req)
```

### Type-Safe Requests with Generic Client

```go
type User struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

// Create a typed client with configuration
client := httpx.NewGenericClient[User](
    httpx.WithTimeout[User](10 * time.Second),
    httpx.WithMaxRetries[User](3),
    httpx.WithRetryStrategy[User](httpx.ExponentialBackoffStrategy),
)

// Execute typed request
response, err := client.Get("https://api.example.com/users/123")
if err != nil {
    log.Fatal(err)
}

// response.Data is strongly typed as User
fmt.Printf("User: %s (%s)\n", response.Data.Name, response.Data.Email)
```

### Request with Retry Logic

```go
// Create client with retry logic
retryClient := httpx.NewClientBuilder().
    WithMaxRetries(3).
    WithRetryStrategy(httpx.ExponentialBackoffStrategy).
    WithRetryBaseDelay(500 * time.Millisecond).
    Build()

// Use with generic client
client := httpx.NewGenericClient[User](
    httpx.WithHTTPClient[User](retryClient),
    httpx.)

response, err := client.Get("/users/123")
```

## Features

### Request Builder

The `RequestBuilder` provides a fluent, chainable API for constructing HTTP requests with comprehensive validation.

#### Key Features

- ✅ HTTP methods: GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS, TRACE, CONNECT
- ✅ Convenience methods for all standard HTTP methods (WithMethodGET, WithMethodPOST, WithMethodPUT, WithMethodDELETE, WithMethodPATCH, WithMethodHEAD, WithMethodOPTIONS, WithMethodTRACE, WithMethodCONNECT)
- ✅ Query parameters with automatic URL encoding
- ✅ Custom headers with validation
- ✅ Authentication (Basic Auth, Bearer Token)
- ✅ Multiple body formats (JSON, string, bytes, io.Reader)
- ✅ Context support for timeouts and cancellation
- ✅ Input validation with error accumulation
- ✅ Comprehensive error messages

#### Usage Example

```go
req, err := httpx.NewRequestBuilder("https://api.example.com").
    WithMethodPOST().
    WithPath("/users").
    WithQueryParam("notify", "true").
    WithHeader("Content-Type", "application/json").
    WithHeader("X-Request-ID", "unique-id-123").
    WithBearerAuth("your-token-here").
    WithJSONBody(map[string]string{
        "name":  "John Doe",
        "email": "john@example.com",
    }).
    Build()

if err != nil {
    // Handle validation errors
    log.Fatal(err)
}
```

#### Validation Features

The RequestBuilder validates inputs and accumulates errors:

```go
builder := httpx.NewRequestBuilder("https://api.example.com")
builder.HTTPMethod("")           // Error: empty method
builder.WithHeader("", "value")      // Error: empty header key
builder.WithQueryParam("key=", "val") // Error: invalid character in key

// Check for errors before building
if builder.HasErrors() {
    for _, err := range builder.GetErrors() {
        log.Printf("Validation error: %v", err)
    }
}

// Or let Build() report all errors
req, err := builder.Build()
if err != nil {
    // err contains all accumulated validation errors
    log.Fatal(err)
}
```

#### Reset and Reuse

```go
builder := httpx.NewRequestBuilder("https://api.example.com")

// Use builder
req1, _ := builder.WithWithMethodGET().WithPath("/users").Build()

// Reset and reuse
builder.Reset()
req2, _ := builder.WithWithMethodPOST().WithPath("/posts").Build()
```

### Generic HTTP Client

The `GenericClient` provides type-safe HTTP requests with automatic JSON marshaling and unmarshaling using Go generics.

#### Key Features

- 🎯 Type-safe responses with automatic JSON unmarshaling
- 🔄 Convenience methods: Get, Post, Put, Delete, Patch
- 🔌 Execute method for use with RequestBuilder
- 📦 ExecuteRaw for non-JSON responses
- 🌐 Base URL resolution for relative paths
- 📋 Default headers applied to all requests
- ❌ Structured error responses
- 🔁 Full integration with retry logic

#### Basic Usage

```go
type Post struct {
    ID     int    `json:"id"`
    Title  string `json:"title"`
    Body   string `json:"body"`
    UserID int    `json:"userId"`
}

client := httpx.NewGenericClient[Post](
    httpx.WithTimeout[Post](10 * time.Second),
    httpx.WithMaxRetries[Post](3),
    httpx.WithRetryStrategy[Post](httpx.ExponentialBackoffStrategy),
)

// GET request
response, err := client.Get("https://api.example.com/posts/1")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Title: %s\n", response.Data.Title)

// POST request
newPost := Post{Title: "New Post", Body: "Content", UserID: 1}
postData, _ := json.Marshal(newPost)
response, err = client.Post("https://api.example.com/posts", bytes.NewReader(postData))
```

#### With RequestBuilder

Combine GenericClient with RequestBuilder for maximum flexibility:

```go
type User struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

client := httpx.NewGenericClient[User](
    httpx.WithTimeout[User](15 * time.Second),
    httpx.WithMaxRetries[User](3),
)

// Build complex request
req, err := httpx.NewRequestBuilder("https://api.example.com").
    WithMethodPOST().
    WithPath("/users").
    WithContentType("application/json").
    WithHeader("X-Request-ID", "unique-123").
    WithJSONBody(User{Name: "Jane", Email: "jane@example.com"}).
    Build()

if err != nil {
    log.Fatal(err)
}

// Execute with type safety
response, err := client.Execute(req)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Created user ID: %d\n", response.Data.ID)
```

#### Error Handling

The generic client returns structured errors:

```go
response, err := client.Get("/users/999999")
if err != nil {
    // Check if it's an API error
    if apiErr, ok := err.(*httpx.ErrorResponse); ok {
        fmt.Printf("API Error %d: %s\n", apiErr.StatusCode, apiErr.Message)
        // StatusCode: 404
        // Message: "User not found"
    } else {
        // Network error, parsing error, etc.
        log.Printf("Request failed: %v\n", err)
    }
    return
}
```

#### Multiple Typed Clients

Use different clients for different response types:

```go
type User struct { /* ... */ }
type Post struct { /* ... */ }

userClient := httpx.NewGenericClient[User](
    httpx.WithTimeout[User](10 * time.Second),
)

postClient := httpx.NewGenericClient[Post](
    httpx.WithTimeout[Post](10 * time.Second),
)

// Fetch user
userResp, _ := userClient.Get("/users/1")

// Fetch user's posts
postsResp, _ := postClient.Get(fmt.Sprintf("/users/%d/posts", userResp.Data.ID))
```

### Retry Logic

The package provides transparent retry logic with configurable strategies.

#### Retry Strategies

##### Exponential Backoff (Recommended)

Doubles the wait time between retries:

```go
client := httpx.NewClientBuilder().
    WithMaxRetries(3).
    WithRetryStrategy(httpx.ExponentialBackoffStrategy).
    WithRetryBaseDelay(500 * time.Millisecond).
    WithRetryMaxDelay(10 * time.Second).
    Build()
```

Wait times: 500ms → 1s → 2s → 4s (capped at maxDelay)

##### Fixed Delay

Waits a constant duration between retries:

```go
client := httpx.NewClientBuilder().
    WithMaxRetries(3).
    WithRetryStrategy(httpx.FixedDelayStrategy).
    WithRetryBaseDelay(1 * time.Second).
    Build()
```

Wait times: 1s → 1s → 1s

##### Jitter Backoff

Adds randomization to exponential backoff to prevent thundering herd:

```go
client := httpx.NewClientBuilder().
    WithMaxRetries(3).
    WithRetryStrategy(httpx.JitterBackoffStrategy).
    WithRetryBaseDelay(500 * time.Millisecond).
    WithRetryMaxDelay(10 * time.Second).
    Build()
```

Wait times: Random between 0-500ms → 0-1s → 0-2s

#### What Gets Retried?

The retry logic automatically retries:

- Network errors (connection failures, timeouts)
- HTTP 5xx server errors (500-599)
- HTTP 429 (Too Many Requests)

Does NOT retry:

- HTTP 4xx client errors (except 429)
- HTTP 2xx/3xx successful responses
- Requests without GetBody (non-replayable requests)

#### Retry with Generic Client

```go
// Create retry client
retryClient := httpx.NewClientBuilder().
    WithMaxRetries(3).
    WithRetryStrategy(httpx.ExponentialBackoffStrategy).
    Build()

// Use with generic client
client := httpx.NewGenericClient[User](
    httpx.WithHTTPClient[User](retryClient),
    httpx.)

// Requests automatically retry on failure
response, err := client.Get("/users/1")
```

### Client Builder

The `ClientBuilder` provides fine-grained control over HTTP client configuration.

#### Configuration Options

```go
client := httpx.NewClientBuilder().
    // Timeouts
    WithTimeout(30 * time.Second).
    WithIdleConnTimeout(90 * time.Second).
    WithTLSHandshakeTimeout(10 * time.Second).
    WithExpectContinueTimeout(1 * time.Second).

    // Connection pooling
    WithMaxIdleConns(100).
    WithMaxIdleConnsPerHost(10).
    WithDisableKeepAlive(false).

    // Retry configuration
    WithMaxRetries(3).
    WithRetryStrategy(httpx.ExponentialBackoffStrategy).
    WithRetryBaseDelay(500 * time.Millisecond).
    WithRetryMaxDelay(10 * time.Second).

    Build()
```

#### Default Values

| Setting | Default | Valid Range |
|---------|---------|-------------|
| Timeout | 5s | 1s - 30s |
| MaxRetries | 3 | 1 - 10 |
| RetryBaseDelay | 500ms | 300ms - 5s |
| RetryMaxDelay | 10s | 300ms - 120s |
| MaxIdleConns | 100 | 1 - 200 |
| IdleConnTimeout | 90s | 1s - 120s |
| TLSHandshakeTimeout | 10s | 1s - 15s |

The builder validates all settings and uses defaults for out-of-range values.

### Proxy Configuration

The httpx package provides comprehensive HTTP/HTTPS proxy support across all client types. Configure proxies to route your requests through corporate firewalls, load balancers, or testing proxies.

#### Key Features

- ✅ HTTP and HTTPS proxy support
- 🔐 Proxy authentication (username/password)
- 🔄 Works with retry logic
- 🎯 Compatible with all client types
- 🌐 Full URL or host:port formats
- 📝 Graceful fallback on invalid URLs

#### Basic Usage

##### With ClientBuilder

```go
// HTTP proxy
client := httpx.NewClientBuilder().
    WithProxy("http://proxy.example.com:8080").
    WithTimeout(10 * time.Second).
    Build()

// HTTPS proxy
client := httpx.NewClientBuilder().
    WithProxy("https://secure-proxy.example.com:3128").
    Build()
```

##### With GenericClient

```go
type User struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

client := httpx.NewGenericClient[User](
    httpx.WithProxy[User]("http://proxy.example.com:8080"),
    httpx.WithTimeout[User](10*time.Second),
    httpx.WithMaxRetries[User](3),
)

response, err := client.Get("https://api.example.com/users/1")
```

##### With Retry Client

```go
client := httpx.NewHTTPRetryClient(
    httpx.WithProxyRetry("http://proxy.example.com:8080"),
    httpx.WithMaxRetriesRetry(5),
    httpx.WithRetryStrategyRetry(
        httpx.ExponentialBackoff(500*time.Millisecond, 30*time.Second),
    ),
)
```

#### Proxy Authentication

Include credentials directly in the proxy URL:

```go
client := httpx.NewClientBuilder().
    WithProxy("http://username:password@proxy.example.com:8080").
    Build()
```

**Security Note:** For production, consider using environment variables or secret management:

```go
proxyURL := fmt.Sprintf("http://%s:%s@%s:%s",
    os.Getenv("PROXY_USER"),
    os.Getenv("PROXY_PASS"),
    os.Getenv("PROXY_HOST"),
    os.Getenv("PROXY_PORT"),
)

client := httpx.NewClientBuilder().
    WithProxy(proxyURL).
    Build()
```

#### Common Proxy Ports

- **HTTP Proxy**: 8080, 3128, 8888
- **HTTPS Proxy**: 3128, 8443
- **Squid**: 3128 (most common)
- **Corporate Proxies**: 8080, 80

#### Disable Proxy

Override environment proxy settings by passing an empty string:

```go
// Disable proxy (ignore HTTP_PROXY environment variable)
client := httpx.NewClientBuilder().
    WithProxy("").
    Build()
```

#### Complete Example

```go
package main

import (
    "fmt"
    "log"
    "time"

    "github.com/slashdevops/httpx"
)

type APIResponse struct {
    Message string `json:"message"`
    Status  string `json:"status"`
}

func main() {
    // Configure client with proxy and full options
    client := httpx.NewGenericClient[APIResponse](
        httpx.WithProxy[APIResponse]("http://proxy.example.com:8080"),
        httpx.WithTimeout[APIResponse](15*time.Second),
        httpx.WithMaxRetries[APIResponse](5),
        httpx.WithRetryStrategy[APIResponse](httpx.JitterBackoffStrategy),
        httpx.WithRetryBaseDelay[APIResponse](500*time.Millisecond),
        httpx.WithRetryMaxDelay[APIResponse](30*time.Second),
    )

    // Build request with authentication
    req, err := httpx.NewRequestBuilder("https://api.example.com").
        WithMethodGET().
        WithPath("/data").
        WithBearerAuth("your-token-here").
        WithHeader("Accept", "application/json").
        Build()

    if err != nil {
        log.Fatal(err)
    }

    // Execute through proxy
    response, err := client.Do(req)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Response: %s\n", response.Data.Message)
}
```

#### Error Handling

The library gracefully handles proxy configuration errors:

```go
client := httpx.NewClientBuilder().
    WithProxy("://invalid-url").  // Invalid URL
    WithLogger(logger).            // Optional: log warnings
    Build()

// Client builds successfully, but proxy is not configured
// Warning logged if logger is provided
```

### Logging

The httpx package supports optional logging using Go's standard `log/slog` package. **Logging is disabled by default** to maintain clean, silent HTTP operations. Enable it when you need observability into retries, errors, and other HTTP client operations.

#### Quick Start

##### Basic Usage

```go
import (
    "log/slog"
    "os"

    "github.com/slashdevops/httpx"
)

// Create a logger
logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelWarn,
}))

// Use with ClientBuilder
client := httpx.NewClientBuilder().
    WithMaxRetries(3).
    WithLogger(logger).  // Enable logging
    Build()
```

##### With Generic Client

```go
type User struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}

logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))

client := httpx.NewGenericClient[User](
    httpx.WithMaxRetries[User](3),
    httpx.WithLogger[User](logger),
)
```

##### With NewHTTPRetryClient

```go
logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

client := httpx.NewHTTPRetryClient(
    httpx.WithMaxRetriesRetry(3),
    httpx.WithRetryStrategyRetry(httpx.ExponentialBackoff(500*time.Millisecond, 10*time.Second)),
    httpx.WithLoggerRetry(logger),
)
```

#### What Gets Logged

##### Retry Attempts (Warn Level)

When a request fails and is being retried:

```
time=2026-01-17T21:00:00.000+00:00 level=WARN msg="HTTP request returned server error, retrying" attempt=1 max_retries=3 delay=500ms status_code=500 url=https://api.example.com/users method=GET
```

Attributes logged:

- `attempt`: Current retry attempt number (1-indexed)
- `max_retries`: Maximum number of retries configured
- `delay`: How long the client will wait before retrying
- `status_code`: HTTP status code (for server errors) OR
- `error`: Error message (for network/connection errors)
- `url`: Full request URL
- `method`: HTTP method (GET, POST, etc.)

##### All Retries Failed (Error Level)

When all retry attempts are exhausted:

```
time=2026-01-17T21:00:00.500+00:00 level=ERROR msg="All retry attempts failed" attempts=4 status_code=503 url=https://api.example.com/users method=GET
```

Attributes logged:

- `attempts`: Total number of attempts made (including initial request)
- `status_code` OR `error`: Final failure reason
- `url`: Full request URL
- `method`: HTTP method

#### Logger Configuration

##### Log Levels

Choose the appropriate log level based on your needs:

```go
// Only log final failures (recommended for production)
logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
    Level: slog.LevelError,
}))

// Log all retry attempts (useful for debugging)
logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelWarn,
}))

// Log everything including debug info from other packages
logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelDebug,
}))
```

##### Output Formats

###### Text Format (Development)

Best for human readability during development:

```go
logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelWarn,
}))
```

Output:

```
time=2026-01-17T21:00:00.000+00:00 level=WARN msg="HTTP request returned server error, retrying" attempt=1 max_retries=3 delay=500ms status_code=500
```

###### JSON Format (Production)

Best for structured logging and log aggregation:

```go
logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
    Level: slog.LevelError,
}))
```

Output:

```json
{"time":"2026-01-17T21:00:00.000Z","level":"ERROR","msg":"All retry attempts failed","attempts":4,"status_code":503,"url":"https://api.example.com/users","method":"GET"}
```

##### Writing to Files

```go
logFile, err := os.OpenFile("http.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
if err != nil {
    log.Fatal(err)
}
defer logFile.Close()

logger := slog.New(slog.NewJSONHandler(logFile, &slog.HandlerOptions{
    Level: slog.LevelWarn,
}))
```

#### Logging Best Practices

1. **Default to No Logging**: Keep logging disabled in production unless actively troubleshooting:

   ```go
   // Production - no logging (default)
   client := httpx.NewClientBuilder().
       WithMaxRetries(3).
       Build()  // No WithLogger() call = no logging
   ```

2. **Use Structured Logging in Production**: JSON format is machine-readable and works well with log aggregators:

   ```go
   logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
       Level: slog.LevelError,  // Only final failures
   }))
   ```

3. **Enable for Specific Troubleshooting**: Turn on logging temporarily when investigating issues:

   ```go
   // Temporarily enable for debugging
   var logger *slog.Logger
   if os.Getenv("DEBUG_HTTP") != "" {
       logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
           Level: slog.LevelWarn,
       }))
   }

   client := httpx.NewClientBuilder().
       WithMaxRetries(3).
       WithLogger(logger).  // Will be nil if not debugging
       Build()
   ```

4. **Add Context with Attributes**: Enhance logs with additional context:

   ```go
   // Create logger with service context
   logger := slog.New(slog.NewJSONHandler(os.Stderr, nil)).
       With("service", "api-client").
       With("version", "1.0.0")

   client := httpx.NewClientBuilder().
       WithLogger(logger).
       Build()
   ```

5. **Different Loggers for Different Clients**: Use separate loggers for different clients to distinguish traffic:

   ```go
   // User service client
   userLogger := slog.New(slog.NewJSONHandler(os.Stderr, nil)).
       With("client", "user-service")
   userClient := httpx.NewClientBuilder().
       WithLogger(userLogger).
       Build()

   // Payment service client
   paymentLogger := slog.New(slog.NewJSONHandler(os.Stderr, nil)).
       With("client", "payment-service")
   paymentClient := httpx.NewClientBuilder().
       WithLogger(paymentLogger).
       Build()
   ```

#### Performance Considerations

- **Minimal Overhead**: When logging is disabled (logger is `nil`), the overhead is just a simple nil check
- **No Allocations**: Log statements use slog's efficient attribute system
- **Deferred Work**: The logger only formats messages if the log level is enabled

#### Disabling Logging

Simply pass `nil` or omit the logger:

```go
// Explicitly pass nil
client := httpx.NewClientBuilder().
    WithLogger(nil).  // No logging
    Build()

// Or just don't call WithLogger
client := httpx.NewClientBuilder().
    WithMaxRetries(3).
    Build()  // No logging (default)
```

#### Migration Guide

If you have existing code without logging, no changes are needed. The feature is fully backward compatible:

```go
// Old code - still works, no logging
client := httpx.NewClientBuilder().
    WithMaxRetries(3).
    Build()

// New code - add logging when needed
logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
client := httpx.NewClientBuilder().
    WithMaxRetries(3).
    WithLogger(logger).  // Just add this line
    Build()
```

#### Logging Examples

##### Example 1: Development Debugging

```go
package main

import (
    "log/slog"
    "os"
    "time"

    "github.com/slashdevops/httpx"
)

func main() {
    // Text output with warn level for debugging
    logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
        Level: slog.LevelWarn,
    }))

    client := httpx.NewClientBuilder().
        WithMaxRetries(3).
        WithRetryBaseDelay(500 * time.Millisecond).
        WithLogger(logger).
        Build()

    // You'll see retry attempts in the console
    resp, err := client.Get("https://api.example.com/flaky-endpoint")
    if err != nil {
        log.Fatal(err)
    }
    defer resp.Body.Close()
}
```

##### Example 2: Production Monitoring

```go
package main

import (
    "log/slog"
    "os"

    "github.com/slashdevops/httpx"
)

func main() {
    // JSON output, only errors, to stderr
    logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
        Level: slog.LevelError,
    })).With(
        "service", "payment-processor",
        "environment", "production",
    )

    client := httpx.NewClientBuilder().
        WithMaxRetries(3).
        WithLogger(logger).
        Build()

    // Only final failures will be logged
    resp, err := client.Get("https://payment-api.example.com/status")
    // ...
}
```

##### Example 3: Conditional Logging

```go
package main

import (
    "log/slog"
    "os"

    "github.com/slashdevops/httpx"
)

func createClient() *http.Client {
    var logger *slog.Logger

    // Only enable logging if DEBUG environment variable is set
    if os.Getenv("DEBUG") != "" {
        logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
            Level: slog.LevelDebug,
        }))
    }

    return httpx.NewClientBuilder().
        WithMaxRetries(3).
        WithLogger(logger).  // Will be nil in production
        Build()
}
```

#### Troubleshooting

##### Not Seeing Any Logs?

1. **Check logger level**: Make sure the level is set to at least `LevelWarn`:

   ```go
   logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
       Level: slog.LevelWarn,  // Not Info or Debug
   }))
   ```

2. **Verify logger is passed**: Make sure you called `WithLogger()`:

   ```go
   client := httpx.NewClientBuilder().
       WithLogger(logger).  // Don't forget this!
       Build()
   ```

3. **Check if retries are happening**: Logs only appear when requests fail and retry. Successful first attempts don't log.

##### Too Many Logs?

1. **Increase log level** to `LevelError` to only see final failures
2. **Disable logging** in production environments where retry behavior is well understood
3. **Use sampling** if your log aggregation system supports it

#### Logging Summary

The logging feature in httpx provides:

- ✅ **Optional** - Disabled by default, zero overhead when not in use
- ✅ **Standard** - Uses Go's `log/slog` package
- ✅ **Flexible** - Configurable output format, level, and destination
- ✅ **Informative** - Rich attributes for debugging and monitoring
- ✅ **Backward Compatible** - Existing code works without changes

Enable it when you need visibility, keep it off for clean, silent operations.

## Examples

### Complete Example: CRUD Operations

```go
package main

import (
    "fmt"
    "log"
    "time"

    "github.com/slashdevops/httpx"
)

type Todo struct {
    ID        int    `json:"id"`
    Title     string `json:"title"`
    Completed bool   `json:"completed"`
    UserID    int    `json:"userId"`
}

func main() {
    // Create retry client
    retryClient := httpx.NewClientBuilder().
        WithMaxRetries(3).
        WithRetryStrategy(httpx.ExponentialBackoffStrategy).
        WithTimeout(10 * time.Second).
        Build()

    // Create typed client
    client := httpx.NewGenericClient[Todo](
        httpx.WithHTTPClient[Todo](retryClient),
        httpx.        httpx.    )

    // GET - Read
    fmt.Println("Fetching todo...")
    todo, err := client.Get("/todos/1")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Todo: %s (completed: %v)\n", todo.Data.Title, todo.Data.Completed)

    // POST - Create
    fmt.Println("\nCreating new todo...")
    newTodo := Todo{
        Title:     "Learn httputils",
        Completed: false,
        UserID:    1,
    }

    req, _ := httpx.NewRequestBuilder("https://jsonplaceholder.typicode.com").
        WithMethodPOST().
        WithPath("/todos").
        WithContentType("application/json").
        WithJSONBody(newTodo).
        Build()

    created, err := client.Execute(req)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Created todo ID: %d\n", created.Data.ID)

    // PUT - Update
    fmt.Println("\nUpdating todo...")
    updateTodo := created.Data
    updateTodo.Completed = true

    req, _ = httpx.NewRequestBuilder("https://jsonplaceholder.typicode.com").
        WithMethodPUT().
        WithPath(fmt.Sprintf("/todos/%d", updateTodo.ID)).
        WithContentType("application/json").
        WithJSONBody(updateTodo).
        Build()

    updated, err := client.Execute(req)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Updated: completed = %v\n", updated.Data.Completed)

    // DELETE
    fmt.Println("\nDeleting todo...")
    deleteResp, err := client.Delete(fmt.Sprintf("/todos/%d", updateTodo.ID))
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Deleted (status: %d)\n", deleteResp.StatusCode)
}
```

### Authentication Example

```go
// Basic Authentication
req, err := httpx.NewRequestBuilder("https://api.example.com").
    WithMethodGET().
    WithPath("/protected/resource").
    WithBasicAuth("username", "password").
    Build()

// Bearer Token Authentication
req, err := httpx.NewRequestBuilder("https://api.example.com").
    WithMethodGET().
    WithPath("/protected/resource").
    WithBearerAuth("your-jwt-token").
    Build()

// With Generic Client
client := httpx.NewGenericClient[Resource](
    httpx.    httpx.)
```

### Context and Timeout

```go
// Request with timeout
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

req, err := httpx.NewRequestBuilder("https://api.example.com").
    WithMethodGET().
    WithPath("/slow-endpoint").
    Context(ctx).
    Build()

// Request with cancellation
ctx, cancel := context.WithCancel(context.Background())

go func() {
    time.Sleep(2 * time.Second)
    cancel() // Cancel after 2 seconds
}()

req, err := httpx.NewRequestBuilder("https://api.example.com").
    WithMethodGET().
    WithPath("/endpoint").
    Context(ctx).
    Build()
```

### Custom Headers and Query Parameters

```go
req, err := httpx.NewRequestBuilder("https://api.example.com").
    WithMethodGET().
    WithPath("/search").
    WithQueryParam("q", "golang").
    WithQueryParam("sort", "relevance").
    WithQueryParam("limit", "10").
    WithHeader("Accept", "application/json").
    WithHeader("Accept-Language", "en-US").
    WithHeader("X-Request-ID", generateRequestID()).
    WithHeader("X-Correlation-ID", getCorrelationID()).
    WithUserAgent("MyApp/1.0 (Go)").
    Build()
```

## API Reference

### RequestBuilder

#### Constructor

- `NewRequestBuilder(baseURL string) *RequestBuilder`

#### HTTP Methods

- `WithMethodGET() *RequestBuilder`
- `WithMethodPOST() *RequestBuilder`
- `WithMethodPUT() *RequestBuilder`
- `WithMethodDELETE() *RequestBuilder`
- `WithMethodPATCH() *RequestBuilder`
- `WithMethodHEAD() *RequestBuilder`
- `WithMethodOPTIONS() *RequestBuilder`
- `WithMethodTRACE() *RequestBuilder`
- `WithMethodCONNECT() *RequestBuilder`
- `WithMethod(method string) *RequestBuilder` - Custom HTTP method with validation

#### URL and Parameters

- `WithPath(path string) *RequestBuilder` - Set URL path
- `WithQueryParam(key, value string) *RequestBuilder` - Add single query parameter
- `QueryParams(params map[string]string) *RequestBuilder` - Add multiple query parameters

#### Headers

- `WithHeader(key, value string) *RequestBuilder` - Set single header
- `Headers(headers map[string]string) *RequestBuilder` - Set multiple headers
- `WithContentType(contentType string) *RequestBuilder` - Set Content-Type header
- `WithAccept(accept string) *RequestBuilder` - Set Accept header
- `WithUserAgent(userAgent string) *RequestBuilder` - Set User-Agent header

#### Authentication

- `WithBasicAuth(username, password string) *RequestBuilder` - Set Basic authentication
- `WithBearerAuth(token string) *RequestBuilder` - Set Bearer token authentication

#### Body

- `WithJSONBody(body any) *RequestBuilder` - Set JSON body (auto-marshals)
- `RawBody(body io.Reader) *RequestBuilder` - Set raw body
- `WithStringBody(body string) *RequestBuilder` - Set string body
- `BytesBody(body []byte) *RequestBuilder` - Set bytes body

#### Other

- `Context(ctx context.Context) *RequestBuilder` - Set request context
- `Build() (*http.Request, error)` - Build and validate request

#### Error Handling

- `HasErrors() bool` - Check if there are validation errors
- `GetErrors() []error` - Get all validation errors
- `Reset() *RequestBuilder` - Reset builder state

### GenericClient[T any]

#### Constructor

- `NewGenericClient[T any](options ...GenericClientOption[T]) *GenericClient[T]`

#### Options

- `WithHTTPClient[T any](httpClient HTTPClient) GenericClientOption[T]` - Use a pre-configured HTTP client (takes precedence)
- `WithTimeout[T any](timeout time.Duration) GenericClientOption[T]` - Set request timeout
- `WithMaxRetries[T any](maxRetries int) GenericClientOption[T]` - Set maximum retry attempts
- `WithRetryStrategy[T any](strategy Strategy) GenericClientOption[T]` - Set retry strategy (fixed, jitter, exponential)
- `WithRetryStrategyAsString[T any](strategy string) GenericClientOption[T]` - Set retry strategy from string
- `WithRetryBaseDelay[T any](baseDelay time.Duration) GenericClientOption[T]` - Set base delay for retry strategies
- `WithRetryMaxDelay[T any](maxDelay time.Duration) GenericClientOption[T]` - Set maximum delay for retry strategies
- `WithMaxIdleConns[T any](maxIdleConns int) GenericClientOption[T]` - Set maximum idle connections
- `WithIdleConnTimeout[T any](idleConnTimeout time.Duration) GenericClientOption[T]` - Set idle connection timeout
- `WithTLSHandshakeTimeout[T any](tlsHandshakeTimeout time.Duration) GenericClientOption[T]` - Set TLS handshake timeout
- `WithExpectContinueTimeout[T any](expectContinueTimeout time.Duration) GenericClientOption[T]` - Set expect continue timeout
- `WithMaxIdleConnsPerHost[T any](maxIdleConnsPerHost int) GenericClientOption[T]` - Set maximum idle connections per host
- `WithDisableKeepAlive[T any](disableKeepAlive bool) GenericClientOption[T]` - Disable HTTP keep-alive

#### Methods

- `Execute(req *http.Request) (*Response[T], error)` - Execute request with type safety
- `ExecuteRaw(req *http.Request) (*http.Response, error)` - Execute and return raw response
- `Do(req *http.Request) (*Response[T], error)` - Alias for Execute
- `Get(url string) (*Response[T], error)` - Execute GET request
- `Post(url string, body io.Reader) (*Response[T], error)` - Execute POST request
- `Put(url string, body io.Reader) (*Response[T], error)` - Execute PUT request
- `Delete(url string) (*Response[T], error)` - Execute DELETE request
- `Patch(url string, body io.Reader) (*Response[T], error)` - Execute PATCH request
- `GetBaseURL() string` - Get configured base URL
- `GetDefaultHeaders() map[string]string` - Get configured headers

### ClientBuilder

#### Constructor

- `NewClientBuilder() *ClientBuilder`

#### Configuration Methods

- `WithTimeout(timeout time.Duration) *ClientBuilder`
- `WithMaxRetries(maxRetries int) *ClientBuilder`
- `WithRetryStrategy(strategy Strategy) *ClientBuilder`
- `WithRetryBaseDelay(baseDelay time.Duration) *ClientBuilder`
- `WithRetryMaxDelay(maxDelay time.Duration) *ClientBuilder`
- `WithMaxIdleConns(maxIdleConns int) *ClientBuilder`
- `WithMaxIdleConnsPerHost(maxIdleConnsPerHost int) *ClientBuilder`
- `WithIdleConnTimeout(idleConnTimeout time.Duration) *ClientBuilder`
- `WithTLSHandshakeTimeout(tlsHandshakeTimeout time.Duration) *ClientBuilder`
- `WithExpectContinueTimeout(expectContinueTimeout time.Duration) *ClientBuilder`
- `WithDisableKeepAlive(disableKeepAlive bool) *ClientBuilder`
- `Build() *http.Client` - Build configured client

### Retry Strategies

- `ExponentialBackoff(base, maxDelay time.Duration) RetryStrategy`
- `FixedDelay(delay time.Duration) RetryStrategy`
- `JitterBackoff(base, maxDelay time.Duration) RetryStrategy`

### Types

#### Response[T any]

```go
type Response[T any] struct {
    Data       T           // Parsed response data
    StatusCode int         // HTTP status code
    Headers    http.Header // Response headers
    RawBody    []byte      // Raw response body
}
```

#### ErrorResponse

```go
type ErrorResponse struct {
    Message    string `json:"message,omitempty"`
    StatusCode int    `json:"statusCode,omitempty"`
    ErrorMsg   string `json:"error,omitempty"`
    Details    string `json:"details,omitempty"`
}
```

#### Strategy

```go
const (
    FixedDelayStrategy         Strategy = "fixed"
    JitterBackoffStrategy      Strategy = "jitter"
    ExponentialBackoffStrategy Strategy = "exponential"
)
```

## Best Practices

### 1. Always Check for Errors

```go
req, err := httpx.NewRequestBuilder(baseURL).
    WithMethodGET().
    WithPath("/endpoint").
    Build()

if err != nil {
    log.Printf("Request building failed: %v", err)
    return
}
```

### 2. Use Type-Safe Clients for JSON APIs

```go
// Define your model
type User struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}

// Create typed client
client := httpx.NewGenericClient[User](
    httpx.)

// Enjoy type safety
response, err := client.Get("/users/1")
// response.Data is User, not interface{}
```

### 3. Configure Retry Logic for Production

```go
client := httpx.NewClientBuilder().
    WithMaxRetries(3).
    WithRetryStrategy(httpx.ExponentialBackoffStrategy).
    WithRetryBaseDelay(500 * time.Millisecond).
    WithRetryMaxDelay(10 * time.Second).
    WithTimeout(30 * time.Second).
    Build()
```

### 4. Reuse HTTP Clients

```go
// Create once, reuse many times
retryClient := httpx.NewClientBuilder().
    WithMaxRetries(3).
    Build()

userClient := httpx.NewGenericClient[User](
    httpx.WithHTTPClient[User](retryClient),
)

postClient := httpx.NewGenericClient[Post](
    httpx.WithHTTPClient[Post](retryClient),
)
```

### 5. Use Context for Timeouts

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

req, err := httpx.NewRequestBuilder(baseURL).
    WithMethodGET().
    WithPath("/endpoint").
    Context(ctx).
    Build()
```

### 6. Validate Before Building

```go
builder := httpx.NewRequestBuilder(baseURL).
    WithMethodGET().
    WithPath("/endpoint")

// Add potentially invalid inputs
builder.WithHeader(userProvidedKey, userProvidedValue)
builder.WithQueryParam(userProvidedParam, userProvidedValue)

// Check for errors before building
if builder.HasErrors() {
    for _, err := range builder.GetErrors() {
        log.Printf("Validation error: %v", err)
    }
    return
}

req, err := builder.Build()
```

### 7. Handle API Errors Properly

```go
response, err := client.Get("/resource")
if err != nil {
    if apiErr, ok := err.(*httpx.ErrorResponse); ok {
        switch apiErr.StatusCode {
        case 404:
            log.Printf("Resource not found: %s", apiErr.Message)
        case 401:
            log.Printf("Authentication failed: %s", apiErr.Message)
        case 429:
            log.Printf("Rate limit exceeded: %s", apiErr.Message)
        default:
            log.Printf("API error %d: %s", apiErr.StatusCode, apiErr.Message)
        }
    } else {
        log.Printf("Network error: %v", err)
    }

    return
}
```

## Thread Safety

All utilities in this package are safe for concurrent use:

```go
client := httpx.NewGenericClient[User](
    httpx.)

// Safe to use from multiple goroutines
var wg sync.WaitGroup
for i := 1; i <= 10; i++ {
    wg.Add(1)
    go func(id int) {
        defer wg.Done()
        user, err := client.Get(fmt.Sprintf("/users/%d", id))
        if err != nil {
            log.Printf("Error fetching user %d: %v", id, err)
            return
        }
        log.Printf("Fetched user: %s", user.Data.Name)
    }(i)
}

wg.Wait()
```

## Testing

The package has comprehensive test coverage (88%+):

```bash
go test ./... -v
go test ./... -cover
```

## Contributing

Contributions are welcome! Please ensure:

1. Build passes: `go build ./...`
2. All tests pass: `go test ./...`
3. Code is formatted: `go fmt ./...`
4. Linters pass: `golangci-lint run ./...`
5. Add tests for new features
6. Update documentation

## License

Apache License 2.0. See [LICENSE](LICENSE) for details.

## Credits

Developed by the slashdevops team using Agentic Development. Inspired by popular HTTP client libraries and Go best practices.
