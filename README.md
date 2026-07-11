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
- 🌐 **Proxy Support** - HTTP/HTTPS proxy configuration with authentication (corporate proxies, authenticated proxies, and custom ports)
- 📝 **Optional Logging** - `log/slog` integration for observability (disabled by default)
- 📦 **Zero External Dependencies** - Only the Go standard library, no third-party packages

## Table of Contents

- [Installation](#installation)
- [Upgrade](#upgrade)
- [Quick Start](#quick-start)
- [Architecture](#architecture)
  - [Component Overview](#component-overview)
  - [Request Lifecycle](#request-lifecycle)
  - [Composition Model](#composition-model)
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
- [Thread Safety](#thread-safety)
- [Testing](#testing)
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

// Build and validate a simple GET request
req, err := httpx.NewRequestBuilder("https://api.example.com").
    WithMethodGET().
    WithPath("/users/123").
    WithHeader("Accept", "application/json").
    Build()

if err != nil {
    log.Fatal(err)
}

// Use with the standard net/http client
resp, err := http.DefaultClient.Do(req)
```

### Type-Safe Requests with the Generic Client

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

// Execute a typed request (pass the full URL)
response, err := client.Get("https://api.example.com/users/123")
if err != nil {
    log.Fatal(err)
}

// response.Data is strongly typed as User
fmt.Printf("User: %s (%s)\n", response.Data.Name, response.Data.Email)
```

### Request with Retry Logic

```go
// Build a *http.Client with retry logic
retryClient := httpx.NewClientBuilder().
    WithMaxRetries(3).
    WithRetryStrategy(httpx.ExponentialBackoffStrategy).
    WithRetryBaseDelay(500 * time.Millisecond).
    Build()

// Plug it into the generic client (takes precedence over other options)
client := httpx.NewGenericClient[User](
    httpx.WithHTTPClient[User](retryClient),
)

response, err := client.Get("https://api.example.com/users/123")
```

## Architecture

`httpx` is intentionally small and composable. It layers three independent building
blocks on top of the standard library's `net/http`, so you can adopt as little or as
much as you need.

### Component Overview

```mermaid
graph TD
    subgraph Build["🔨 Build a request"]
        RB["RequestBuilder<br/><i>fluent, validated</i>"]
    end

    subgraph Execute["🎯 Execute with type safety"]
        GC["GenericClient[T]<br/><i>JSON marshal / unmarshal</i>"]
    end

    subgraph Transport["⚙️ Configure transport & retries"]
        CB["ClientBuilder"]
        HRC["NewHTTPRetryClient"]
        RT["retryTransport<br/><i>http.RoundTripper</i>"]
    end

    STD["net/http<br/>http.Client + http.Transport"]

    RB -->|"*http.Request"| GC
    GC -->|"uses"| CB
    CB -->|"*http.Client"| GC
    CB -->|"builds"| RT
    HRC -->|"builds"| RT
    RT -->|"wraps"| STD

    classDef build fill:#e1f5ff,stroke:#0288d1,color:#01579b
    classDef exec fill:#e8f5e9,stroke:#43a047,color:#1b5e20
    classDef transport fill:#fff3e0,stroke:#fb8c00,color:#e65100
    classDef std fill:#f3e5f5,stroke:#8e24aa,color:#4a148c

    class RB build
    class GC exec
    class CB,HRC,RT transport
    class STD std
```

| Component | Purpose | Returns |
|-----------|---------|---------|
| [`RequestBuilder`](#request-builder) | Construct and validate a request fluently | `*http.Request` |
| [`GenericClient[T]`](#generic-http-client) | Execute requests with type-safe JSON (un)marshaling | `*Response[T]` |
| [`ClientBuilder`](#client-builder) | Configure timeouts, pooling, proxy, and retries | `*http.Client` |
| [`NewHTTPRetryClient`](#advanced-direct-retry-client) | Low-level retry transport wrapper | `*http.Client` |

### Request Lifecycle

The sequence below shows a request flowing through every layer, including a transient
`503` that triggers one retry before succeeding.

```mermaid
sequenceDiagram
    participant App as Your Code
    participant RB as RequestBuilder
    participant GC as GenericClient[T]
    participant RT as retryTransport
    participant Srv as Server

    App->>RB: WithMethodGET().WithPath(...).Build()
    RB-->>App: *http.Request (validated)
    App->>GC: Execute(req)
    GC->>RT: Do(req)

    RT->>Srv: RoundTrip (attempt 1)
    Srv-->>RT: 503 Service Unavailable
    Note over RT: 5xx → wait backoff delay
    RT->>Srv: RoundTrip (attempt 2)
    Srv-->>RT: 200 OK + JSON

    RT-->>GC: *http.Response
    GC->>GC: json.Unmarshal into T
    GC-->>App: *Response[T] (typed Data)
```

### Composition Model

Every layer is optional and can be swapped. Use the builder without the client, the
client without retries, or bring your own `HTTPClient` implementation for testing.

```mermaid
flowchart LR
    A["RequestBuilder<br/>(optional)"] -.->|"*http.Request"| B
    B["GenericClient[T]<br/>or plain *http.Client"]
    C["Custom HTTPClient<br/>(mocks, tracing…)"] -.->|"WithHTTPClient"| B
    D["ClientBuilder / NewHTTPRetryClient"] -.->|"*http.Client"| B

    style A stroke-dasharray: 5 5
    style C stroke-dasharray: 5 5
    style D stroke-dasharray: 5 5
```

> The `HTTPClient` interface is just `Do(*http.Request) (*http.Response, error)` — the
> same shape as `*http.Client`. Anything satisfying it (including a mock) can be injected
> with `WithHTTPClient`, which makes the generic client trivial to unit test.

## Features

### Request Builder

The `RequestBuilder` provides a fluent, chainable API for constructing HTTP requests with comprehensive validation.

#### Key Features

- ✅ HTTP methods: GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS, TRACE, CONNECT
- ✅ Convenience methods for every standard HTTP method (`WithMethodGET`, `WithMethodPOST`, `WithMethodPUT`, `WithMethodDELETE`, `WithMethodPATCH`, `WithMethodHEAD`, `WithMethodOPTIONS`, `WithMethodTRACE`, `WithMethodCONNECT`)
- ✅ Query parameters with automatic URL encoding
- ✅ Custom headers with validation
- ✅ Authentication (Basic Auth, Bearer Token)
- ✅ Multiple body formats (JSON, string, bytes, `io.Reader`)
- ✅ Context support for timeouts and cancellation
- ✅ Input validation with error accumulation
- ✅ Comprehensive error messages
- ✅ Automatic `GetBody` wiring for replayable JSON bodies (so retries work)

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

#### Error Accumulation

Unlike most builders that fail fast, `RequestBuilder` **accumulates** validation errors
as you chain calls and reports them all at `Build()` time (or on demand via `HasErrors`
/ `GetErrors`). This lets you surface every problem in a user-supplied request at once.

```mermaid
flowchart TD
    Start(["NewRequestBuilder(baseURL)"]) --> Chain["Chain WithMethod / WithHeader /<br/>WithQueryParam / WithBody …"]
    Chain --> Valid{"Input valid?"}
    Valid -->|"yes"| Store["Apply to builder state"]
    Valid -->|"no"| Acc["Append to errors[]"]
    Store --> More{"More calls?"}
    Acc --> More
    More -->|"yes"| Chain
    More -->|"no"| Build["Build()"]
    Build --> HasErr{"len(errors) > 0<br/>or invalid state?"}
    HasErr -->|"yes"| Err(["return nil, error<br/>(all accumulated)"])
    HasErr -->|"no"| Req(["return *http.Request, nil"])

    classDef ok fill:#e8f5e9,stroke:#43a047,color:#1b5e20
    classDef bad fill:#ffebee,stroke:#e53935,color:#b71c1c
    class Req,Store ok
    class Err,Acc bad
```

```go
builder := httpx.NewRequestBuilder("https://api.example.com")
builder.WithMethod("")                 // Error: empty method
builder.WithHeader("", "value")        // Error: empty header key
builder.WithQueryParam("key=", "val")  // Error: invalid character in key

// Check for errors before building
if builder.HasErrors() {
    for _, err := range builder.GetErrors() {
        log.Printf("Validation error: %v", err)
    }
}

// Or let Build() report all errors at once
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
req1, _ := builder.WithMethodGET().WithPath("/users").Build()

// Reset and reuse for a fresh request
builder.Reset()
req2, _ := builder.WithMethodPOST().WithPath("/posts").Build()
```

> **Note:** A `RequestBuilder` is *not* safe for concurrent use. Create one per goroutine,
> or build requests up front and share the resulting `*http.Request` values.

### Generic HTTP Client

The `GenericClient[T]` provides type-safe HTTP requests with automatic JSON marshaling and unmarshaling using Go generics.

#### Key Features

- 🎯 Type-safe responses with automatic JSON unmarshaling into `T`
- 🔄 Convenience methods: `Get`, `Post`, `Put`, `Delete`, `Patch`
- 🔌 `Execute` / `Do` for use with `RequestBuilder`
- 📦 `ExecuteRaw` for non-JSON responses (images, files, streams)
- ❌ Structured error responses (`*ErrorResponse`) for HTTP status >= 400
- 🔁 Built-in retry logic (configured via options) or bring your own client
- 🧪 Injectable `HTTPClient` for easy mocking in tests

> **URLs:** The convenience methods (`Get`, `Post`, …) take a **full URL**. For a shared
> base URL plus paths, use [`RequestBuilder`](#request-builder) with `Execute`/`Do`.

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

// POST request with a body
newPost := Post{Title: "New Post", Body: "Content", UserID: 1}
postData, _ := json.Marshal(newPost)
created, err := client.Post("https://api.example.com/posts", bytes.NewReader(postData))
```

#### With RequestBuilder

Combine `GenericClient` with `RequestBuilder` for maximum flexibility — the builder wires
up authentication, headers, query params, and a replayable JSON body, and the client
executes it with type safety:

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

// Build a complex request
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

For any response with status code >= 400, the client returns a structured `*ErrorResponse`:

```go
response, err := client.Get("https://api.example.com/users/999999")
if err != nil {
    // Check if it's a structured API error
    if apiErr, ok := err.(*httpx.ErrorResponse); ok {
        fmt.Printf("API Error %d: %s\n", apiErr.StatusCode, apiErr.Message)
        // e.g. StatusCode: 404, Message: "User not found"
    } else {
        // Network error, parsing error, retries exhausted, etc.
        log.Printf("Request failed: %v\n", err)
    }
    return
}
```

#### Non-JSON Responses

Use `ExecuteRaw` when the response isn't JSON (binary downloads, streaming, etc.). It
returns the raw `*http.Response` without unmarshaling — remember to close the body:

```go
client := httpx.NewGenericClient[any]()

req, _ := http.NewRequest(http.MethodGet, "https://example.com/image.png", nil)

resp, err := client.ExecuteRaw(req)
if err != nil {
    log.Fatal(err)
}
defer resp.Body.Close()

fmt.Printf("Content-Type: %s\n", resp.Header.Get("Content-Type"))
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

// Fetch a user, then that user's posts
userResp, _ := userClient.Get("https://api.example.com/users/1")
postsResp, _ := postClient.Get(fmt.Sprintf("https://api.example.com/users/%d/posts", userResp.Data.ID))
```

### Retry Logic

The package provides transparent retry logic with configurable strategies. Retries
preserve all headers and authentication, and replay the request body via `GetBody`.

#### What Gets Retried?

```mermaid
flowchart TD
    Req["Send request (attempt n)"] --> Err{"Transport error?"}
    Err -->|"yes<br/>(connection, timeout)"| Retryable
    Err -->|"no"| Code{"Status code?"}
    Code -->|"5xx (500–599)"| Retryable
    Code -->|"429 Too Many Requests"| Retryable
    Code -->|"2xx / 3xx / other 4xx"| Success(["Return response<br/>✅ no retry"])

    Retryable{"Attempts left?"} -->|"yes"| Wait["Wait backoff delay<br/>(respects context)"]
    Retryable -->|"no"| Fail(["Return last error<br/>❌ retries exhausted"])
    Wait --> Req

    classDef ok fill:#e8f5e9,stroke:#43a047,color:#1b5e20
    classDef bad fill:#ffebee,stroke:#e53935,color:#b71c1c
    classDef warn fill:#fff3e0,stroke:#fb8c00,color:#e65100
    class Success ok
    class Fail bad
    class Wait,Retryable warn
```

**Retried automatically:**

- Network errors (connection failures, timeouts)
- HTTP 5xx server errors (500–599)
- HTTP 429 (Too Many Requests)

**Not retried:**

- HTTP 4xx client errors (except 429)
- HTTP 2xx / 3xx responses
- Requests without a `GetBody` (non-replayable bodies)

> **Context awareness:** If the request's context is cancelled or its deadline expires
> (including when `http.Client.Timeout` fires), retries stop immediately and the original
> error is returned — no misleading "retry cancelled" churn.

#### Retry Strategies

The three built-in strategies differ in how the delay grows between attempts:

```mermaid
xychart-beta
    title "Delay per retry attempt (base=500ms, max=10s)"
    x-axis "Attempt" [1, 2, 3, 4, 5]
    y-axis "Delay (ms)" 0 --> 8500
    line "Exponential" [500, 1000, 2000, 4000, 8000]
    line "Fixed" [500, 500, 500, 500, 500]
    line "Jitter (avg)" [625, 1250, 2500, 5000, 7500]
```

##### Exponential Backoff (Recommended)

Doubles the wait time between retries, capped at `maxDelay`:

```go
client := httpx.NewClientBuilder().
    WithMaxRetries(3).
    WithRetryStrategy(httpx.ExponentialBackoffStrategy).
    WithRetryBaseDelay(500 * time.Millisecond).
    WithRetryMaxDelay(10 * time.Second).
    Build()
```

Wait times: `500ms → 1s → 2s → 4s` (capped at `maxDelay`)

##### Fixed Delay

Waits a constant duration between retries:

```go
client := httpx.NewClientBuilder().
    WithMaxRetries(3).
    WithRetryStrategy(httpx.FixedDelayStrategy).
    WithRetryBaseDelay(1 * time.Second).
    Build()
```

Wait times: `1s → 1s → 1s`

##### Jitter Backoff

Adds randomization on top of exponential backoff to prevent the *thundering herd*
problem when many clients retry in lockstep:

```go
client := httpx.NewClientBuilder().
    WithMaxRetries(3).
    WithRetryStrategy(httpx.JitterBackoffStrategy).
    WithRetryBaseDelay(500 * time.Millisecond).
    WithRetryMaxDelay(10 * time.Second).
    Build()
```

Wait times: exponential base plus a random `0 – base/2` jitter per attempt.

#### Retry with the Generic Client

You can configure retries directly on the generic client, or build a `*http.Client`
and inject it:

```go
// Option A: configure retries directly on the generic client
client := httpx.NewGenericClient[User](
    httpx.WithMaxRetries[User](3),
    httpx.WithRetryStrategy[User](httpx.ExponentialBackoffStrategy),
    httpx.WithRetryBaseDelay[User](500*time.Millisecond),
)

// Option B: build a client and inject it (takes precedence)
retryClient := httpx.NewClientBuilder().
    WithMaxRetries(3).
    WithRetryStrategy(httpx.ExponentialBackoffStrategy).
    Build()

client = httpx.NewGenericClient[User](
    httpx.WithHTTPClient[User](retryClient),
)

response, err := client.Get("https://api.example.com/users/1")
```

#### Advanced: Direct Retry Client

For full control over the transport, `NewHTTPRetryClient` returns a plain `*http.Client`
whose transport applies retries. The `RetryStrategy` here is a function you pass directly:

```go
client := httpx.NewHTTPRetryClient(
    httpx.WithMaxRetriesRetry(5),
    httpx.WithRetryStrategyRetry(
        httpx.ExponentialBackoff(500*time.Millisecond, 30*time.Second),
    ),
    httpx.WithBaseTransport(http.DefaultTransport),
)

resp, err := client.Get("https://api.example.com/data")
```

### Client Builder

The `ClientBuilder` provides fine-grained control over HTTP client configuration and
returns a ready-to-use `*http.Client`.

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

    // Proxy (optional)
    WithProxy("http://proxy.example.com:8080").

    // Logging (optional)
    WithLogger(logger).

    Build()
```

#### Default Values

The builder validates every setting and silently falls back to the default when a value
is out of range (logging a warning if a logger is configured).

| Setting | Default | Valid Range |
|---------|---------|-------------|
| Timeout | 5s | 1s – 600s |
| MaxRetries | 3 | 1 – 10 |
| RetryBaseDelay | 500ms | 300ms – 5s |
| RetryMaxDelay | 10s | 300ms – 120s |
| MaxIdleConns | 100 | 1 – 200 |
| MaxIdleConnsPerHost | 100 | 1 – 200 |
| IdleConnTimeout | 90s | 1s – 120s |
| TLSHandshakeTimeout | 10s | 1s – 15s |
| ExpectContinueTimeout | 1s | 1s – 5s |

### Proxy Configuration

`httpx` provides comprehensive HTTP/HTTPS proxy support across all client types. Route
requests through corporate firewalls, load balancers, or testing proxies.

#### Key Features

- ✅ HTTP and HTTPS proxy support
- 🔐 Proxy authentication (username/password embedded in the URL)
- 🔄 Works with retry logic
- 🎯 Compatible with `ClientBuilder`, `GenericClient`, and `NewHTTPRetryClient`
- 📝 Graceful fallback on invalid URLs (a warning is logged if a logger is set)

#### With ClientBuilder

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

#### With GenericClient

```go
client := httpx.NewGenericClient[User](
    httpx.WithProxy[User]("http://proxy.example.com:8080"),
    httpx.WithTimeout[User](10*time.Second),
    httpx.WithMaxRetries[User](3),
)

response, err := client.Get("https://api.example.com/users/1")
```

#### With the Retry Client

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

**Security note:** For production, source credentials from the environment or a secret
manager rather than hard-coding them:

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

Pass an empty string to disable proxying (overriding any `HTTP_PROXY`/`HTTPS_PROXY`
environment variables):

```go
client := httpx.NewClientBuilder().
    WithProxy("").
    Build()
```

### Logging

`httpx` supports optional logging using Go's standard `log/slog`. **Logging is disabled
by default** to keep HTTP operations clean and silent. Enable it to gain observability
into retries and failures. When no logger is set, the overhead is a single `nil` check.

Loggers are wired in per client type:

| Client | Option |
|--------|--------|
| `ClientBuilder` | `WithLogger(logger)` |
| `GenericClient[T]` | `WithLogger[T](logger)` |
| `NewHTTPRetryClient` | `WithLoggerRetry(logger)` |

#### Quick Start

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

// Wire it into any client type
client := httpx.NewClientBuilder().
    WithMaxRetries(3).
    WithLogger(logger).
    Build()
```

With the generic client:

```go
logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))

client := httpx.NewGenericClient[User](
    httpx.WithMaxRetries[User](3),
    httpx.WithLogger[User](logger),
)
```

#### What Gets Logged

**Retry attempts (WARN level)** — emitted each time a request fails and is retried:

```text
time=2026-01-17T21:00:00.000+00:00 level=WARN msg="HTTP request returned server error, retrying" attempt=1 max_retries=3 delay=500ms status_code=500 url=https://api.example.com/users method=GET
```

**All retries failed (ERROR level)** — emitted when every attempt is exhausted:

```text
time=2026-01-17T21:00:00.500+00:00 level=ERROR msg="All retry attempts failed" attempts=4 status_code=503 url=https://api.example.com/users method=GET
```

The `GenericClient` additionally logs request/response details (method, URL, headers,
body) at **DEBUG** level, which is useful when troubleshooting a specific call.

#### Output Formats

Choose text for development readability, JSON for production log aggregation:

```go
// Text (development)
logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelWarn,
}))

// JSON (production)
logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
    Level: slog.LevelError, // only final failures
}))
```

#### Logging Best Practices

1. **Default to no logging** in production unless actively troubleshooting.
2. **Use JSON** for machine-readable logs that play well with aggregators.
3. **Enable conditionally** via an environment variable when investigating issues:

   ```go
   var logger *slog.Logger
   if os.Getenv("DEBUG_HTTP") != "" {
       logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
           Level: slog.LevelWarn,
       }))
   }

   client := httpx.NewClientBuilder().
       WithMaxRetries(3).
       WithLogger(logger). // nil when not debugging = no logging
       Build()
   ```

4. **Add context** with `slog`'s `With` to tag a client's traffic:

   ```go
   logger := slog.New(slog.NewJSONHandler(os.Stderr, nil)).
       With("service", "api-client", "version", "1.0.0")
   ```

## Examples

Runnable examples live alongside the source as Go [example tests](https://pkg.go.dev/testing#hdr-Examples):

| File | Covers |
|------|--------|
| [example_request_builder_test.go](example_request_builder_test.go) | Building and validating requests |
| [example_generic_client_test.go](example_generic_client_test.go) | Type-safe requests, error handling, multiple clients |
| [example_http_client_test.go](example_http_client_test.go) | `ClientBuilder` configuration |
| [example_http_retrier_test.go](example_http_retrier_test.go) | Retry strategies and the direct retry client |
| [example_proxy_test.go](example_proxy_test.go) | Proxy configuration |
| [example_logger_test.go](example_logger_test.go) | `slog` logging |

Browse them all on [pkg.go.dev](https://pkg.go.dev/github.com/slashdevops/httpx#pkg-examples).

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
    const baseURL = "https://jsonplaceholder.typicode.com"

    // Build a *http.Client with retries and inject it into the typed client
    retryClient := httpx.NewClientBuilder().
        WithMaxRetries(3).
        WithRetryStrategy(httpx.ExponentialBackoffStrategy).
        WithTimeout(10 * time.Second).
        Build()

    client := httpx.NewGenericClient[Todo](
        httpx.WithHTTPClient[Todo](retryClient),
    )

    // GET - Read
    fmt.Println("Fetching todo...")
    todo, err := client.Get(baseURL + "/todos/1")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Todo: %s (completed: %v)\n", todo.Data.Title, todo.Data.Completed)

    // POST - Create (build the request with RequestBuilder)
    fmt.Println("\nCreating new todo...")
    newTodo := Todo{Title: "Learn httpx", Completed: false, UserID: 1}

    req, _ := httpx.NewRequestBuilder(baseURL).
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

    req, _ = httpx.NewRequestBuilder(baseURL).
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
    deleteResp, err := client.Delete(fmt.Sprintf("%s/todos/%d", baseURL, updateTodo.ID))
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
req, err = httpx.NewRequestBuilder("https://api.example.com").
    WithMethodGET().
    WithPath("/protected/resource").
    WithBearerAuth("your-jwt-token").
    Build()
```

### Context and Timeout

```go
// Request with a per-request deadline
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

req, err := httpx.NewRequestBuilder("https://api.example.com").
    WithMethodGET().
    WithPath("/slow-endpoint").
    WithContext(ctx).
    Build()

// Request with manual cancellation
ctx, cancel = context.WithCancel(context.Background())
go func() {
    time.Sleep(2 * time.Second)
    cancel() // cancel after 2 seconds
}()

req, err = httpx.NewRequestBuilder("https://api.example.com").
    WithMethodGET().
    WithPath("/endpoint").
    WithContext(ctx).
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
    WithUserAgent("MyApp/1.0 (Go)").
    Build()
```

## API Reference

Full, always-up-to-date reference: [pkg.go.dev/github.com/slashdevops/httpx](https://pkg.go.dev/github.com/slashdevops/httpx)

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
- `WithMethod(method string) *RequestBuilder` — custom HTTP method with validation

#### URL and Parameters

- `WithPath(path string) *RequestBuilder` — set the URL path
- `WithQueryParam(key, value string) *RequestBuilder` — add a single query parameter
- `WithQueryParams(params map[string]string) *RequestBuilder` — add multiple query parameters

#### Headers

- `WithHeader(key, value string) *RequestBuilder` — set a single header
- `WithHeaders(headers map[string]string) *RequestBuilder` — set multiple headers
- `WithContentType(contentType string) *RequestBuilder` — set the `Content-Type` header
- `WithAccept(accept string) *RequestBuilder` — set the `Accept` header
- `WithUserAgent(userAgent string) *RequestBuilder` — set the `User-Agent` header (validated)

#### Authentication

- `WithBasicAuth(username, password string) *RequestBuilder` — set Basic authentication
- `WithBearerAuth(token string) *RequestBuilder` — set Bearer token authentication

#### Body

- `WithJSONBody(body any) *RequestBuilder` — set a JSON body (auto-marshals, sets `Content-Type`, enables retry replay)
- `WithRawBody(body io.Reader) *RequestBuilder` — set a raw `io.Reader` body
- `WithStringBody(body string) *RequestBuilder` — set a string body
- `WithBytesBody(body []byte) *RequestBuilder` — set a `[]byte` body

#### Other

- `WithContext(ctx context.Context) *RequestBuilder` — set the request context
- `Build() (*http.Request, error)` — build and validate the request

#### Error Handling

- `HasErrors() bool` — whether any validation errors were accumulated
- `GetErrors() []error` — all accumulated validation errors
- `Reset() *RequestBuilder` — reset the builder to a clean state

### GenericClient[T any]

#### Constructor

- `NewGenericClient[T any](options ...GenericClientOption[T]) *GenericClient[T]`

#### Options

- `WithHTTPClient[T any](httpClient HTTPClient) GenericClientOption[T]` — use a pre-configured client (takes precedence over all other options)
- `WithTimeout[T any](timeout time.Duration) GenericClientOption[T]`
- `WithMaxRetries[T any](maxRetries int) GenericClientOption[T]`
- `WithRetryStrategy[T any](strategy Strategy) GenericClientOption[T]`
- `WithRetryStrategyAsString[T any](strategy string) GenericClientOption[T]`
- `WithRetryBaseDelay[T any](baseDelay time.Duration) GenericClientOption[T]`
- `WithRetryMaxDelay[T any](maxDelay time.Duration) GenericClientOption[T]`
- `WithMaxIdleConns[T any](maxIdleConns int) GenericClientOption[T]`
- `WithIdleConnTimeout[T any](idleConnTimeout time.Duration) GenericClientOption[T]`
- `WithTLSHandshakeTimeout[T any](tlsHandshakeTimeout time.Duration) GenericClientOption[T]`
- `WithExpectContinueTimeout[T any](expectContinueTimeout time.Duration) GenericClientOption[T]`
- `WithMaxIdleConnsPerHost[T any](maxIdleConnsPerHost int) GenericClientOption[T]`
- `WithDisableKeepAlive[T any](disableKeepAlive bool) GenericClientOption[T]`
- `WithProxy[T any](proxyURL string) GenericClientOption[T]`
- `WithLogger[T any](logger *slog.Logger) GenericClientOption[T]`

#### Methods

- `Execute(req *http.Request) (*Response[T], error)` — execute a request with type safety
- `ExecuteRaw(req *http.Request) (*http.Response, error)` — execute and return the raw response
- `Do(req *http.Request) (*Response[T], error)` — alias for `Execute`
- `Get(url string) (*Response[T], error)`
- `Post(url string, body io.Reader) (*Response[T], error)`
- `Put(url string, body io.Reader) (*Response[T], error)`
- `Delete(url string) (*Response[T], error)`
- `Patch(url string, body io.Reader) (*Response[T], error)`

### ClientBuilder

#### Constructor

- `NewClientBuilder() *ClientBuilder`

#### Configuration Methods

- `WithTimeout(timeout time.Duration) *ClientBuilder`
- `WithMaxRetries(maxRetries int) *ClientBuilder`
- `WithRetryStrategy(strategy Strategy) *ClientBuilder`
- `WithRetryStrategyAsString(strategy string) *ClientBuilder`
- `WithRetryBaseDelay(baseDelay time.Duration) *ClientBuilder`
- `WithRetryMaxDelay(maxDelay time.Duration) *ClientBuilder`
- `WithMaxIdleConns(maxIdleConns int) *ClientBuilder`
- `WithMaxIdleConnsPerHost(maxIdleConnsPerHost int) *ClientBuilder`
- `WithIdleConnTimeout(idleConnTimeout time.Duration) *ClientBuilder`
- `WithTLSHandshakeTimeout(tlsHandshakeTimeout time.Duration) *ClientBuilder`
- `WithExpectContinueTimeout(expectContinueTimeout time.Duration) *ClientBuilder`
- `WithDisableKeepAlive(disableKeepAlive bool) *ClientBuilder`
- `WithProxy(proxyURL string) *ClientBuilder`
- `WithLogger(logger *slog.Logger) *ClientBuilder`
- `Build() *http.Client` — build the configured client

### Direct Retry Client

- `NewHTTPRetryClient(options ...RetryClientOption) *http.Client`
- `WithMaxRetriesRetry(maxRetries int) RetryClientOption`
- `WithRetryStrategyRetry(strategy RetryStrategy) RetryClientOption`
- `WithBaseTransport(transport http.RoundTripper) RetryClientOption`
- `WithProxyRetry(proxyURL string) RetryClientOption`
- `WithLoggerRetry(logger *slog.Logger) RetryClientOption`

### Retry Strategy Functions

- `ExponentialBackoff(base, maxDelay time.Duration) RetryStrategy`
- `FixedDelay(delay time.Duration) RetryStrategy`
- `JitterBackoff(base, maxDelay time.Duration) RetryStrategy`

### Types

#### Response[T any]

```go
type Response[T any] struct {
    Data       T           // Parsed response data
    Headers    http.Header // Response headers
    RawBody    []byte      // Raw response body
    StatusCode int         // HTTP status code
}
```

#### ErrorResponse

```go
type ErrorResponse struct {
    Message    string `json:"message,omitempty"`
    ErrorMsg   string `json:"error,omitempty"`
    Details    string `json:"details,omitempty"`
    StatusCode int    `json:"statusCode,omitempty"`
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

#### HTTPClient

```go
// Any type with this method can be injected via WithHTTPClient.
type HTTPClient interface {
    Do(req *http.Request) (*http.Response, error)
}
```

## Best Practices

### 1. Always Check Build Errors

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
type User struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}

client := httpx.NewGenericClient[User]()

response, err := client.Get("https://api.example.com/users/1")
// response.Data is a User, not interface{}
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

Clients are safe for concurrent use and pool connections — create once, share widely:

```go
retryClient := httpx.NewClientBuilder().
    WithMaxRetries(3).
    Build()

userClient := httpx.NewGenericClient[User](httpx.WithHTTPClient[User](retryClient))
postClient := httpx.NewGenericClient[Post](httpx.WithHTTPClient[Post](retryClient))
```

### 5. Use Context for Timeouts

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

req, err := httpx.NewRequestBuilder(baseURL).
    WithMethodGET().
    WithPath("/endpoint").
    WithContext(ctx).
    Build()
```

### 6. Validate User-Supplied Input Before Building

```go
builder := httpx.NewRequestBuilder(baseURL).
    WithMethodGET().
    WithPath("/endpoint")

builder.WithHeader(userProvidedKey, userProvidedValue)
builder.WithQueryParam(userProvidedParam, userProvidedValue)

if builder.HasErrors() {
    for _, err := range builder.GetErrors() {
        log.Printf("Validation error: %v", err)
    }
    return
}

req, err := builder.Build()
```

### 7. Handle API Errors by Status Code

```go
response, err := client.Get("https://api.example.com/resource")
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

- ✅ **`GenericClient[T]`** — safe for concurrent use across goroutines.
- ✅ **`*http.Client`** built by `ClientBuilder` / `NewHTTPRetryClient` — safe for concurrent use.
- ✅ **Retry logic** — preserves request immutability; bodies are replayed via `GetBody`.
- ⚠️ **`RequestBuilder`** — **not** safe for concurrent use. Use one per goroutine.

```go
client := httpx.NewGenericClient[User]()

var wg sync.WaitGroup
for i := 1; i <= 10; i++ {
    wg.Add(1)
    go func(id int) {
        defer wg.Done()
        user, err := client.Get(fmt.Sprintf("https://api.example.com/users/%d", id))
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

The package ships with a comprehensive test suite (89%+ statement coverage):

```bash
go test ./...            # run all tests
go test ./... -cover     # with coverage
go test ./... -race      # with the race detector
```

Because the generic client accepts any `HTTPClient`, you can inject a mock to test your
own code without hitting the network:

```go
type mockClient struct{}

func (m *mockClient) Do(req *http.Request) (*http.Response, error) {
    body := io.NopCloser(strings.NewReader(`{"id":1,"name":"Ada"}`))
    return &http.Response{StatusCode: 200, Body: body, Header: http.Header{}}, nil
}

client := httpx.NewGenericClient[User](httpx.WithHTTPClient[User](&mockClient{}))
resp, _ := client.Get("https://api.example.com/users/1")
// resp.Data.Name == "Ada"
```

## Contributing

Contributions are welcome! Before opening a pull request, please ensure:

1. The build passes: `go build ./...`
2. All tests pass: `go test ./...`
3. Code is formatted: `go fmt ./...`
4. Linters pass: `golangci-lint run ./...`
5. New features include tests and documentation updates.

## License

Apache License 2.0. See [LICENSE](LICENSE) for details.

## Credits

Developed by the slashdevops team. Inspired by popular HTTP client libraries and Go best practices.
