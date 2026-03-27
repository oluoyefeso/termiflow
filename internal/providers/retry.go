package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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

// RateLimitError is returned when the API responds with 429.
// RetryAfter contains the server-specified wait duration.
type RateLimitError struct {
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string {
	if e.RetryAfter >= time.Minute {
		minutes := int(math.Ceil(e.RetryAfter.Minutes()))
		return fmt.Sprintf("Rate limited. Try again in %dm.", minutes)
	}
	if e.RetryAfter > 0 {
		return fmt.Sprintf("Rate limited. Try again in %ds.", int(math.Ceil(e.RetryAfter.Seconds())))
	}
	return "Rate limited. Try again later."
}

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

// DoWithRetry executes fn with retry logic for transient errors.
// Returns immediately with *RateLimitError on 429 (no retry).
// Retries 503 up to maxRetries times with exponential backoff.
// The caller is responsible for closing the returned response body.
func DoWithRetry(ctx context.Context, fn func() (*http.Response, error)) (*http.Response, error) { //nolint:bodyclose // caller closes on success; 429/503 paths close explicitly
	var resp *http.Response
	var err error

	for attempt := 0; ; attempt++ {
		resp, err = fn()
		if err != nil {
			return nil, err
		}

		// 429: parse rate limit info and return immediately (no retry)
		if resp.StatusCode == http.StatusTooManyRequests {
			retryAfter := ParseRateLimitResponse(resp)
			resp.Body.Close()
			return nil, &RateLimitError{RetryAfter: retryAfter}
		}

		// 503: retry with backoff
		if resp.StatusCode == http.StatusServiceUnavailable && attempt < maxRetries {
			resp.Body.Close()
			delay := RetryDelay(attempt, resp)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
			continue
		}

		// Success or non-retryable error: return as-is
		return resp, nil
	}
}

// ParseRateLimitResponse extracts the retry duration from a 429 response.
// Tries JSON body field "retry_after_seconds" first, then Retry-After header.
func ParseRateLimitResponse(resp *http.Response) time.Duration {
	// Try JSON body first
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024))
	if err == nil && len(body) > 0 {
		var parsed struct {
			RetryAfterSeconds int `json:"retry_after_seconds"`
		}
		if json.Unmarshal(body, &parsed) == nil && parsed.RetryAfterSeconds > 0 {
			return time.Duration(parsed.RetryAfterSeconds) * time.Second
		}
	}

	// Fall back to Retry-After header
	if ra := resp.Header.Get("Retry-After"); ra != "" {
		if secs, err := strconv.Atoi(ra); err == nil && secs > 0 {
			return time.Duration(secs) * time.Second
		}
	}

	return 0
}
