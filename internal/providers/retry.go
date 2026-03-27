package providers

import (
	"math"
	"net/http"
	"strconv"
	"time"
)

const (
	maxRetries = 3
	baseDelay  = 500 * time.Millisecond
	maxDelay   = 30 * time.Second
)

// IsRetryable returns true for status codes that warrant a retry.
func IsRetryable(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests || statusCode == http.StatusServiceUnavailable
}

// RetryDelay returns the backoff duration for the given attempt (0-indexed).
// Uses exponential backoff: 500ms, 1s, 2s, capped at 30s.
// Reads Retry-After header if present.
func RetryDelay(attempt int, resp *http.Response) time.Duration {
	if resp != nil {
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			if secs, err := strconv.Atoi(ra); err == nil && secs > 0 {
				d := time.Duration(secs) * time.Second
				if d > maxDelay {
					return maxDelay
				}
				return d
			}
		}
	}
	d := time.Duration(float64(baseDelay) * math.Pow(2, float64(attempt)))
	if d > maxDelay {
		return maxDelay
	}
	return d
}

// MaxRetries returns the number of retries to attempt.
func MaxRetries() int {
	return maxRetries
}
