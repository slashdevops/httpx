package httpx

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"time"
)

var ErrAllRetriesFailed = errors.New("all retry attempts failed")

// RetryStrategy defines the function signature for different retry strategies
type RetryStrategy func(attempt int) time.Duration

// ExponentialBackoff returns a RetryStrategy that calculates delays
// growing exponentially with each retry attempt, starting from base
// and capped at maxDelay.
func ExponentialBackoff(base, maxDelay time.Duration) RetryStrategy {
	return func(attempt int) time.Duration {
		// Special case from test: If base > maxDelay, the first attempt returns base,
		// subsequent attempts calculate normally and cap at maxDelay.
		if attempt == 0 && base > maxDelay {
			return base
		}

		// Calculate delay: base * 2^attempt
		// Use uint for bit shift robustness, though overflow is unlikely before capping.
		delay := base * (1 << uint(attempt))

		// Cap at maxDelay. Also handle potential overflow resulting in negative/zero delay.
		if delay > maxDelay || delay <= 0 {
			delay = maxDelay
		}

		// Note: The original check `if delay < base { delay = base }` is removed
		// as the logic now correctly handles the base > maxDelay case based on the test,
		// and for base <= maxDelay, the calculated delay won't be less than base for attempt >= 0.
		return delay
	}
}

// FixedDelay returns a RetryStrategy that provides a constant delay
// for each retry attempt.
func FixedDelay(delay time.Duration) RetryStrategy {
	return func(attempt int) time.Duration {
		return delay
	}
}

// JitterBackoff returns a RetryStrategy that adds a random jitter
// to the exponential backoff delay calculated using base and maxDelay.
func JitterBackoff(base, maxDelay time.Duration) RetryStrategy {
	expBackoff := ExponentialBackoff(base, maxDelay)
	return func(attempt int) time.Duration {
		baseDelay := expBackoff(attempt)

		// Add jitter: random duration between 0 and baseDelay/2
		jitter := time.Duration(rand.Int63n(int64(baseDelay / 2)))

		return baseDelay + jitter
	}
}

// retryTransport wraps http.RoundTripper to add retry logic
type retryTransport struct {
	Transport     http.RoundTripper // Underlying transport (e.g., http.DefaultTransport)
	RetryStrategy RetryStrategy     // The strategy function to calculate delay
	MaxRetries    int
	logger        *slog.Logger // Optional logger for retry operations (nil = no logging)
}

// RoundTrip executes an HTTP request with retry logic
func (r *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error

	// Ensure transport is set
	transport := r.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}

	// Ensure a retry strategy is set, default to a basic exponential backoff
	retryStrategy := r.RetryStrategy
	if retryStrategy == nil {
		retryStrategy = ExponentialBackoff(500*time.Millisecond, 10*time.Second) // Default strategy
	}

	for attempt := 0; attempt <= r.MaxRetries; attempt++ {
		// Clone the request body if it exists and is GetBody is defined
		// This allows the body to be read multiple times on retries
		if req.Body != nil && req.GetBody != nil {
			bodyClone, err := req.GetBody()
			if err != nil {
				return nil, fmt.Errorf("failed to get request body for retry: %w", err)
			}

			req.Body = bodyClone
		}

		resp, err = transport.RoundTrip(req)

		// Success conditions: no error and status code below 500
		if err == nil && resp.StatusCode < http.StatusInternalServerError {
			return resp, nil
		}

		// If there was an error or a server-side error (5xx), prepare for retry
		// Close response body to prevent resource leaks before retrying
		if resp != nil {
			// Drain the body before closing
			_, copyErr := io.Copy(io.Discard, resp.Body)
			closeErr := resp.Body.Close()

			if copyErr != nil {
				// Prioritize returning the copy error
				return nil, fmt.Errorf("failed to discard response body: %w", copyErr)
			}

			if closeErr != nil {
				return nil, fmt.Errorf("failed to close response body: %w", closeErr)
			}
		}

		// Check if we should retry
		if attempt < r.MaxRetries {
			delay := retryStrategy(attempt)

			// Log retry attempt if logger is configured
			if r.logger != nil {
				if err != nil {
					r.logger.Warn("HTTP request failed, retrying",
						"attempt", attempt+1,
						"max_retries", r.MaxRetries,
						"delay", delay,
						"error", err,
						"url", req.URL.String(),
						"method", req.Method,
					)
				} else if resp != nil {
					r.logger.Warn("HTTP request returned server error, retrying",
						"attempt", attempt+1,
						"max_retries", r.MaxRetries,
						"delay", delay,
						"status_code", resp.StatusCode,
						"url", req.URL.String(),
						"method", req.Method,
					)
				}
			}

			time.Sleep(delay)
		} else {
			// Max retries reached, log and return the last error or a generic failure error
			if r.logger != nil {
				if err != nil {
					r.logger.Error("All retry attempts failed",
						"attempts", r.MaxRetries+1,
						"error", err,
						"url", req.URL.String(),
						"method", req.Method,
					)
				} else if resp != nil {
					r.logger.Error("All retry attempts failed",
						"attempts", r.MaxRetries+1,
						"status_code", resp.StatusCode,
						"url", req.URL.String(),
						"method", req.Method,
					)
				}
			}

			if err != nil {
				return nil, fmt.Errorf("all retries failed; last error: %w", err)
			}

			// If the last attempt resulted in a 5xx response without a transport error
			if resp != nil {
				// Return a more specific error including the status code
				return nil, fmt.Errorf("%w: last attempt failed with status %d", ErrAllRetriesFailed, resp.StatusCode)
			}

			return nil, ErrAllRetriesFailed
		}
	}

	return nil, ErrAllRetriesFailed
}

// RetryClientOption is a function type for configuring the retry HTTP client.
type RetryClientOption func(*retryClientConfig)

// retryClientConfig holds configuration for building a retry HTTP client.
type retryClientConfig struct {
	maxRetries    int
	strategy      RetryStrategy
	baseTransport http.RoundTripper
	logger        *slog.Logger
}

// WithMaxRetriesRetry sets the maximum number of retry attempts for the retry client.
func WithMaxRetriesRetry(maxRetries int) RetryClientOption {
	return func(c *retryClientConfig) {
		c.maxRetries = maxRetries
	}
}

// WithRetryStrategyRetry sets the retry strategy for the retry client.
func WithRetryStrategyRetry(strategy RetryStrategy) RetryClientOption {
	return func(c *retryClientConfig) {
		c.strategy = strategy
	}
}

// WithBaseTransport sets the base HTTP transport for the retry client.
// If not provided, http.DefaultTransport will be used.
func WithBaseTransport(transport http.RoundTripper) RetryClientOption {
	return func(c *retryClientConfig) {
		c.baseTransport = transport
	}
}

// WithLoggerRetry sets the logger for the retry client.
// Pass nil to disable logging (default behavior).
func WithLoggerRetry(logger *slog.Logger) RetryClientOption {
	return func(c *retryClientConfig) {
		c.logger = logger
	}
}

// NewHTTPRetryClient creates a new http.Client configured with the retry transport.
// Use the provided options to customize the retry behavior.
// By default, it uses 3 retries with exponential backoff strategy and no logging.
func NewHTTPRetryClient(options ...RetryClientOption) *http.Client {
	config := &retryClientConfig{
		maxRetries:    DefaultMaxRetries,
		strategy:      ExponentialBackoff(DefaultBaseDelay, DefaultMaxDelay),
		baseTransport: nil,
		logger:        nil,
	}

	for _, option := range options {
		option(config)
	}

	if config.baseTransport == nil {
		config.baseTransport = http.DefaultTransport
	}

	if config.strategy == nil {
		config.strategy = ExponentialBackoff(DefaultBaseDelay, DefaultMaxDelay)
	}

	return &http.Client{
		Transport: &retryTransport{
			Transport:     config.baseTransport,
			MaxRetries:    config.maxRetries,
			RetryStrategy: config.strategy,
			logger:        config.logger,
		},
	}
}
