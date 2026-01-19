package httpx

import (
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

const (
	ValidMaxIdleConns             = 200
	ValidMinIdleConns             = 1
	ValidMaxIdleConnsPerHost      = 200
	ValidMinIdleConnsPerHost      = 1
	ValidMaxIdleConnTimeout       = 120 * time.Second
	ValidMinIdleConnTimeout       = 1 * time.Second
	ValidMaxTLSHandshakeTimeout   = 15 * time.Second
	ValidMinTLSHandshakeTimeout   = 1 * time.Second
	ValidMaxExpectContinueTimeout = 5 * time.Second
	ValidMinExpectContinueTimeout = 1 * time.Second
	ValidMaxTimeout               = 30 * time.Second
	ValidMinTimeout               = 1 * time.Second
	ValidMaxRetries               = 10
	ValidMinRetries               = 1
	ValidMaxBaseDelay             = 5 * time.Second
	ValidMinBaseDelay             = 300 * time.Millisecond
	ValidMaxMaxDelay              = 120 * time.Second
	ValidMinMaxDelay              = 300 * time.Millisecond

	// DefaultMaxRetries is the default number of retry attempts
	DefaultMaxRetries = 3

	// DefaultBaseDelay is the default base delay for backoff strategies
	DefaultBaseDelay = 500 * time.Millisecond

	// DefaultMaxDelay is the default maximum delay for backoff strategies
	DefaultMaxDelay = 10 * time.Second

	// DefaultMaxIdleConns is the default maximum number of idle connections
	DefaultMaxIdleConns = 100

	// DefaultIdleConnTimeout is the default idle connection timeout
	DefaultIdleConnTimeout = 90 * time.Second

	// DefaultTLSHandshakeTimeout is the default TLS handshake timeout
	DefaultTLSHandshakeTimeout = 10 * time.Second

	// DefaultExpectContinueTimeout is the default expect continue timeout
	DefaultExpectContinueTimeout = 1 * time.Second

	// DefaultDisableKeepAlive is the default disable keep-alive setting
	DefaultDisableKeepAlive = false

	// DefaultMaxIdleConnsPerHost is the default maximum number of idle connections per host
	DefaultMaxIdleConnsPerHost = 100

	// DefaultTimeout is the default timeout for HTTP requests
	DefaultTimeout = 5 * time.Second
)

// ClientError represents an error that occurs during HTTP client operations
type ClientError struct {
	Message string
}

func (e *ClientError) Error() string {
	return e.Message
}

// Strategy defines the type for retry strategies
// It is a string type to allow for easy conversion from string literals
// to the defined types
type Strategy string

const (
	// FixedDelayStrategy represents a fixed delay retry strategy
	// This strategy waits for a constant amount of time between retries
	// regardless of the number of attempts made
	FixedDelayStrategy Strategy = "fixed"

	// JitterBackoffStrategy represents a jitter backoff retry strategy
	// This strategy adds randomness to the backoff delay to prevent
	// synchronized retries across multiple clients
	JitterBackoffStrategy Strategy = "jitter"

	// ExponentialBackoffStrategy represents an exponential backoff retry strategy
	// This strategy increases the delay exponentially with each retry attempt,
	// up to a maximum delay
	ExponentialBackoffStrategy Strategy = "exponential"
)

func (s Strategy) String() string {
	return string(s)
}

func (s Strategy) IsValid() bool {
	switch s {
	case FixedDelayStrategy, JitterBackoffStrategy, ExponentialBackoffStrategy:
		return true
	default:
		return false
	}
}

// Client is a custom HTTP client with configurable settings
// and retry strategies. Works transparently with existing request headers.
// It preserves all headers without requiring explicit configuration.
type Client struct {
	retryStrategyType     Strategy // Store the type, not the function
	maxIdleConns          int
	idleConnTimeout       time.Duration
	tlsHandshakeTimeout   time.Duration
	expectContinueTimeout time.Duration
	maxIdleConnsPerHost   int
	timeout               time.Duration
	maxRetries            int
	retryBaseDelay        time.Duration
	retryMaxDelay         time.Duration
	disableKeepAlive      bool
	proxyURL              string       // Proxy URL (e.g., "http://proxy.example.com:8080")
	logger                *slog.Logger // Optional logger (nil = no logging)
}

// ClientBuilder is a builder for creating a custom HTTP client
type ClientBuilder struct {
	client *Client
}

// NewClientBuilder creates a new ClientBuilder with default settings
// and retry strategy
func NewClientBuilder() *ClientBuilder {
	cb := &ClientBuilder{
		client: &Client{
			maxIdleConns:          DefaultMaxIdleConns,
			idleConnTimeout:       DefaultIdleConnTimeout,
			tlsHandshakeTimeout:   DefaultTLSHandshakeTimeout,
			expectContinueTimeout: DefaultExpectContinueTimeout,
			disableKeepAlive:      DefaultDisableKeepAlive,
			maxIdleConnsPerHost:   DefaultMaxIdleConnsPerHost,
			timeout:               DefaultTimeout,
			maxRetries:            DefaultMaxRetries,
			retryStrategyType:     ExponentialBackoffStrategy,
			retryBaseDelay:        DefaultBaseDelay,
			retryMaxDelay:         DefaultMaxDelay,
		},
	}
	return cb
}

// WithMaxIdleConns sets the maximum number of idle connections
// and returns the ClientBuilder for method chaining
func (b *ClientBuilder) WithMaxIdleConns(maxIdleConns int) *ClientBuilder {
	b.client.maxIdleConns = maxIdleConns

	return b
}

// WithIdleConnTimeout sets the idle connection timeout
// and returns the ClientBuilder for method chaining
func (b *ClientBuilder) WithIdleConnTimeout(idleConnTimeout time.Duration) *ClientBuilder {
	b.client.idleConnTimeout = idleConnTimeout

	return b
}

// WithTLSHandshakeTimeout sets the TLS handshake timeout
// and returns the ClientBuilder for method chaining
func (b *ClientBuilder) WithTLSHandshakeTimeout(tlsHandshakeTimeout time.Duration) *ClientBuilder {
	b.client.tlsHandshakeTimeout = tlsHandshakeTimeout

	return b
}

// WithExpectContinueTimeout sets the expect continue timeout
// and returns the ClientBuilder for method chaining
func (b *ClientBuilder) WithExpectContinueTimeout(expectContinueTimeout time.Duration) *ClientBuilder {
	b.client.expectContinueTimeout = expectContinueTimeout

	return b
}

// WithDisableKeepAlive sets whether to disable keep-alive
// and returns the ClientBuilder for method chaining
func (b *ClientBuilder) WithDisableKeepAlive(disableKeepAlive bool) *ClientBuilder {
	b.client.disableKeepAlive = disableKeepAlive

	return b
}

// WithMaxIdleConnsPerHost sets the maximum number of idle connections per host
// and returns the ClientBuilder for method chaining
func (b *ClientBuilder) WithMaxIdleConnsPerHost(maxIdleConnsPerHost int) *ClientBuilder {
	b.client.maxIdleConnsPerHost = maxIdleConnsPerHost

	return b
}

// WithTimeout sets the timeout for HTTP requests
// and returns the ClientBuilder for method chaining
func (b *ClientBuilder) WithTimeout(timeout time.Duration) *ClientBuilder {
	b.client.timeout = timeout

	return b
}

// WithMaxRetries sets the maximum number of retry attempts
// and returns the ClientBuilder for method chaining
func (b *ClientBuilder) WithMaxRetries(maxRetries int) *ClientBuilder {
	b.client.maxRetries = maxRetries

	return b
}

// WithRetryBaseDelay sets the base delay for retry strategies
// and returns the ClientBuilder for method chaining
func (b *ClientBuilder) WithRetryBaseDelay(baseDelay time.Duration) *ClientBuilder {
	b.client.retryBaseDelay = baseDelay

	return b
}

// WithRetryMaxDelay sets the maximum delay for retry strategies
// and returns the ClientBuilder for method chaining
func (b *ClientBuilder) WithRetryMaxDelay(maxDelay time.Duration) *ClientBuilder {
	b.client.retryMaxDelay = maxDelay

	return b
}

// WithRetryStrategy sets the retry strategy type
// and returns the ClientBuilder for method chaining
func (b *ClientBuilder) WithRetryStrategy(strategy Strategy) *ClientBuilder {
	b.client.retryStrategyType = strategy

	return b
}

// WithRetryStrategyAsString sets the retry strategy type from a string
// and returns the ClientBuilder for method chaining
func (b *ClientBuilder) WithRetryStrategyAsString(strategy string) *ClientBuilder {
	s := Strategy(strategy)

	if !s.IsValid() {
		if b.client.logger != nil {
			b.client.logger.Warn("Invalid retry strategy type, using default (Exponential)", "invalidValue", s, "defaultValue", ExponentialBackoffStrategy)
		}
		s = ExponentialBackoffStrategy
	}

	b.client.retryStrategyType = s

	return b
}

// WithLogger sets the logger for logging HTTP operations (retries, errors, etc.).
// Pass nil to disable logging (default behavior).
// and returns the ClientBuilder for method chaining
func (b *ClientBuilder) WithLogger(logger *slog.Logger) *ClientBuilder {
	b.client.logger = logger

	return b
}

// WithProxy sets the proxy URL for HTTP requests.
// The proxy URL should be in the format "http://proxy.example.com:8080" or "https://proxy.example.com:8080".
// Pass an empty string to disable proxy (default behavior).
// and returns the ClientBuilder for method chaining
func (b *ClientBuilder) WithProxy(proxyURL string) *ClientBuilder {
	b.client.proxyURL = proxyURL

	return b
}

// Build creates and returns a new HTTP client with the specified settings
// and retry strategy. The client works transparently, preserving any existing
// headers in requests without requiring explicit configuration.
func (b *ClientBuilder) Build() *http.Client {
	if b.client.maxIdleConns < ValidMinIdleConns || b.client.maxIdleConns > ValidMaxIdleConns {
		if b.client.logger != nil {
			b.client.logger.Warn("Invalid max idle connections, using default value", "invalidValue", b.client.maxIdleConns, "defaultValue", DefaultMaxIdleConns)
		}

		b.client.maxIdleConns = DefaultMaxIdleConns
	}

	if b.client.idleConnTimeout < ValidMinIdleConnTimeout || b.client.idleConnTimeout > ValidMaxIdleConnTimeout {
		if b.client.logger != nil {
			b.client.logger.Warn("Invalid idle connection timeout, using default value", "invalidValue", b.client.idleConnTimeout, "defaultValue", DefaultIdleConnTimeout)
		}

		b.client.idleConnTimeout = DefaultIdleConnTimeout
	}

	if b.client.tlsHandshakeTimeout < ValidMinTLSHandshakeTimeout || b.client.tlsHandshakeTimeout > ValidMaxTLSHandshakeTimeout {
		if b.client.logger != nil {
			b.client.logger.Warn("Invalid TLS handshake timeout, using default value", "invalidValue", b.client.tlsHandshakeTimeout, "defaultValue", DefaultTLSHandshakeTimeout)
		}

		b.client.tlsHandshakeTimeout = DefaultTLSHandshakeTimeout
	}

	if b.client.expectContinueTimeout < ValidMinExpectContinueTimeout || b.client.expectContinueTimeout > ValidMaxExpectContinueTimeout {
		if b.client.logger != nil {
			b.client.logger.Warn("Invalid expect continue timeout, using default value", "invalidValue", b.client.expectContinueTimeout, "defaultValue", DefaultExpectContinueTimeout)
		}

		b.client.expectContinueTimeout = DefaultExpectContinueTimeout
	}

	if b.client.maxIdleConnsPerHost < ValidMinIdleConnsPerHost || b.client.maxIdleConnsPerHost > ValidMaxIdleConnsPerHost {
		if b.client.logger != nil {
			b.client.logger.Warn("Invalid max idle connections per host, using default value", "invalidValue", b.client.maxIdleConnsPerHost, "defaultValue", DefaultMaxIdleConnsPerHost)
		}

		b.client.maxIdleConnsPerHost = DefaultMaxIdleConnsPerHost
	}

	if b.client.timeout < ValidMinTimeout || b.client.timeout > ValidMaxTimeout {
		if b.client.logger != nil {
			b.client.logger.Warn("Invalid timeout, using default value", "invalidValue", b.client.timeout, "defaultValue", DefaultTimeout)
		}

		b.client.timeout = DefaultTimeout
	}

	if b.client.maxRetries < ValidMinRetries || b.client.maxRetries > ValidMaxRetries {
		if b.client.logger != nil {
			b.client.logger.Warn("Invalid max retries, using default value", "invalidValue", b.client.maxRetries, "defaultValue", DefaultMaxRetries)
		}

		b.client.maxRetries = DefaultMaxRetries
	}

	if b.client.retryBaseDelay < ValidMinBaseDelay || b.client.retryBaseDelay > ValidMaxBaseDelay {
		if b.client.logger != nil {
			b.client.logger.Warn("Invalid retry base delay, using default value", "invalidValue", b.client.retryBaseDelay, "defaultValue", DefaultBaseDelay)
		}

		b.client.retryBaseDelay = DefaultBaseDelay
	}

	if b.client.retryMaxDelay < ValidMinMaxDelay || b.client.retryMaxDelay > ValidMaxMaxDelay {
		if b.client.logger != nil {
			b.client.logger.Warn("Invalid retry max delay, using default value", "invalidValue", b.client.retryMaxDelay, "defaultValue", DefaultMaxDelay)
		}

		b.client.retryMaxDelay = DefaultMaxDelay
	}

	// Determine the final strategy type, defaulting if necessary
	finalStrategyType := b.client.retryStrategyType
	switch finalStrategyType {
	case FixedDelayStrategy, JitterBackoffStrategy, ExponentialBackoffStrategy:
		// Valid type provided
	default:
		if b.client.logger != nil {
			b.client.logger.Warn("No valid retry strategy type set, using default (Exponential)", "currentType", finalStrategyType)
		}

		finalStrategyType = ExponentialBackoffStrategy
	}

	var finalRetryStrategy RetryStrategy
	switch finalStrategyType {
	case FixedDelayStrategy:
		finalRetryStrategy = FixedDelay(b.client.retryBaseDelay)
	case JitterBackoffStrategy:
		finalRetryStrategy = JitterBackoff(b.client.retryBaseDelay, b.client.retryMaxDelay)
	case ExponentialBackoffStrategy:
		finalRetryStrategy = ExponentialBackoff(b.client.retryBaseDelay, b.client.retryMaxDelay)
	default:
		finalRetryStrategy = ExponentialBackoff(b.client.retryBaseDelay, b.client.retryMaxDelay)
	}

	// Create the underlying standard transport
	transport := &http.Transport{
		MaxIdleConns:          b.client.maxIdleConns,
		IdleConnTimeout:       b.client.idleConnTimeout,
		TLSHandshakeTimeout:   b.client.tlsHandshakeTimeout,
		ExpectContinueTimeout: b.client.expectContinueTimeout,
		DisableKeepAlives:     b.client.disableKeepAlive,
		MaxIdleConnsPerHost:   b.client.maxIdleConnsPerHost,
	}

	// Configure proxy if set
	if b.client.proxyURL != "" {
		parsedProxyURL, err := url.Parse(b.client.proxyURL)
		if err != nil {
			if b.client.logger != nil {
				b.client.logger.Warn("Failed to parse proxy URL, proceeding without proxy", "proxyURL", b.client.proxyURL, "error", err)
			}
		} else {
			transport.Proxy = http.ProxyURL(parsedProxyURL)
		}
	}

	// Create retry transport - this is the only layer needed for transparent operation
	// It automatically preserves all existing headers without any explicit auth configuration
	finalTransport := &retryTransport{
		Transport:     transport,
		MaxRetries:    b.client.maxRetries,
		RetryStrategy: finalRetryStrategy,
		logger:        b.client.logger,
	}

	// Create the HTTP client with the specified settings
	return &http.Client{
		Timeout:   b.client.timeout,
		Transport: finalTransport,
	}
}
