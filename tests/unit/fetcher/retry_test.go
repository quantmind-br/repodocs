package fetcher_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/fetcher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRetrier_Success(t *testing.T) {
	t.Run("creates retrier with default options", func(t *testing.T) {
		// Execute: Create retrier with default options
		opts := fetcher.DefaultRetrierOptions()
		retrier := fetcher.NewRetrier(opts)

		// Verify: Retrier created successfully
		assert.NotNil(t, retrier)
	})

	t.Run("creates retrier with custom options", func(t *testing.T) {
		// Execute: Create retrier with custom options
		opts := fetcher.RetrierOptions{
			MaxRetries:      5,
			InitialInterval: 500 * time.Millisecond,
			MaxInterval:     1 * time.Minute,
			Multiplier:      3.0,
		}
		retrier := fetcher.NewRetrier(opts)

		// Verify: Retrier created successfully
		assert.NotNil(t, retrier)
	})

	t.Run("zero max retries defaults to 3", func(t *testing.T) {
		// Execute: Create retrier with zero max retries
		opts := fetcher.RetrierOptions{
			MaxRetries:      0,
			InitialInterval: 1 * time.Second,
			MaxInterval:     30 * time.Second,
			Multiplier:      2.0,
		}
		retrier := fetcher.NewRetrier(opts)

		// Verify: Retrier created with default value
		assert.NotNil(t, retrier)
	})

	t.Run("zero initial interval defaults to 1 second", func(t *testing.T) {
		// Execute: Create retrier with zero initial interval
		opts := fetcher.RetrierOptions{
			MaxRetries:      3,
			InitialInterval: 0,
			MaxInterval:     30 * time.Second,
			Multiplier:      2.0,
		}
		retrier := fetcher.NewRetrier(opts)

		// Verify: Retrier created with default value
		assert.NotNil(t, retrier)
	})

	t.Run("zero max interval defaults to 30 seconds", func(t *testing.T) {
		// Execute: Create retrier with zero max interval
		opts := fetcher.RetrierOptions{
			MaxRetries:      3,
			InitialInterval: 1 * time.Second,
			MaxInterval:     0,
			Multiplier:      2.0,
		}
		retrier := fetcher.NewRetrier(opts)

		// Verify: Retrier created with default value
		assert.NotNil(t, retrier)
	})

	t.Run("zero multiplier defaults to 2.0", func(t *testing.T) {
		// Execute: Create retrier with zero multiplier
		opts := fetcher.RetrierOptions{
			MaxRetries:      3,
			InitialInterval: 1 * time.Second,
			MaxInterval:     30 * time.Second,
			Multiplier:      0,
		}
		retrier := fetcher.NewRetrier(opts)

		// Verify: Retrier created with default value
		assert.NotNil(t, retrier)
	})

	t.Run("negative values are handled", func(t *testing.T) {
		// Execute: Create retrier with negative values
		opts := fetcher.RetrierOptions{
			MaxRetries:      -1,
			InitialInterval: -1 * time.Second,
			MaxInterval:     -1 * time.Second,
			Multiplier:      -1.0,
		}
		retrier := fetcher.NewRetrier(opts)

		// Verify: Retrier created (negative values are replaced with defaults)
		assert.NotNil(t, retrier)
	})
}

func TestDefaultRetrierOptions(t *testing.T) {
	// Execute: Get default options
	opts := fetcher.DefaultRetrierOptions()

	// Verify: All default values are correct
	t.Run("default max retries", func(t *testing.T) {
		assert.Equal(t, 3, opts.MaxRetries)
	})

	t.Run("default initial interval", func(t *testing.T) {
		assert.Equal(t, 1*time.Second, opts.InitialInterval)
	})

	t.Run("default max interval", func(t *testing.T) {
		assert.Equal(t, 30*time.Second, opts.MaxInterval)
	})

	t.Run("default multiplier", func(t *testing.T) {
		assert.Equal(t, 2.0, opts.Multiplier)
	})
}

func TestRetrier_Retry_Success(t *testing.T) {
	ctx := context.Background()

	t.Run("operation succeeds on first attempt", func(t *testing.T) {
		// Setup: Create retrier
		opts := fetcher.DefaultRetrierOptions()
		retrier := fetcher.NewRetrier(opts)

		attempts := 0
		operation := func() error {
			attempts++
			return nil
		}

		// Execute: Retry operation
		err := retrier.Retry(ctx, operation)

		// Verify: Operation succeeded on first attempt
		require.NoError(t, err)
		assert.Equal(t, 1, attempts)
	})

	t.Run("operation succeeds after retry", func(t *testing.T) {
		// Setup: Create retrier
		opts := fetcher.RetrierOptions{
			MaxRetries:      3,
			InitialInterval: 10 * time.Millisecond,
			MaxInterval:     100 * time.Millisecond,
			Multiplier:      2.0,
		}
		retrier := fetcher.NewRetrier(opts)

		attempts := 0
		operation := func() error {
			attempts++
			if attempts < 2 {
				return &domain.RetryableError{
					Err: errors.New("temporary error"),
				}
			}
			return nil
		}

		// Execute: Retry operation
		err := retrier.Retry(ctx, operation)

		// Verify: Operation succeeded after retry
		require.NoError(t, err)
		assert.Equal(t, 2, attempts)
	})

	t.Run("operation succeeds after multiple retries", func(t *testing.T) {
		// Setup: Create retrier
		opts := fetcher.RetrierOptions{
			MaxRetries:      5,
			InitialInterval: 10 * time.Millisecond,
			MaxInterval:     100 * time.Millisecond,
			Multiplier:      2.0,
		}
		retrier := fetcher.NewRetrier(opts)

		attempts := 0
		operation := func() error {
			attempts++
			if attempts < 3 {
				return &domain.RetryableError{
					Err: errors.New("temporary error"),
				}
			}
			return nil
		}

		// Execute: Retry operation
		err := retrier.Retry(ctx, operation)

		// Verify: Operation succeeded after retries
		require.NoError(t, err)
		assert.Equal(t, 3, attempts)
	})
}

func TestRetrier_Retry_Error(t *testing.T) {
	ctx := context.Background()

	t.Run("non-retryable error fails immediately", func(t *testing.T) {
		// Setup: Create retrier
		opts := fetcher.DefaultRetrierOptions()
		retrier := fetcher.NewRetrier(opts)

		attempts := 0
		operation := func() error {
			attempts++
			return errors.New("permanent error")
		}

		// Execute: Retry operation
		err := retrier.Retry(ctx, operation)

		// Verify: Operation failed immediately (no retries)
		assert.Error(t, err)
		assert.Equal(t, 1, attempts)
	})

	t.Run("retryable error exhausts retries", func(t *testing.T) {
		// Setup: Create retrier
		opts := fetcher.RetrierOptions{
			MaxRetries:      2,
			InitialInterval: 10 * time.Millisecond,
			MaxInterval:     100 * time.Millisecond,
			Multiplier:      2.0,
		}
		retrier := fetcher.NewRetrier(opts)

		attempts := 0
		operation := func() error {
			attempts++
			return &domain.RetryableError{
				Err: errors.New("temporary error"),
			}
		}

		// Execute: Retry operation
		err := retrier.Retry(ctx, operation)

		// Verify: Operation failed after exhausting retries
		assert.Error(t, err)
		assert.Equal(t, 3, attempts) // Initial + 2 retries
	})

	t.Run("context cancellation stops retry", func(t *testing.T) {
		// Setup: Create retrier
		opts := fetcher.RetrierOptions{
			MaxRetries:      5,
			InitialInterval: 100 * time.Millisecond,
			MaxInterval:     1 * time.Second,
			Multiplier:      2.0,
		}
		retrier := fetcher.NewRetrier(opts)

		attempts := 0
		operation := func() error {
			attempts++
			return &domain.RetryableError{
				Err: errors.New("temporary error"),
			}
		}

		// Setup: Create context with timeout
		ctx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
		defer cancel()

		// Execute: Retry operation
		err := retrier.Retry(ctx, operation)

		// Verify: Operation failed due to context cancellation
		assert.Error(t, err)
		// Should have made at least one attempt before timeout
		assert.GreaterOrEqual(t, attempts, 1)
	})

	t.Run("permanent error does not retry", func(t *testing.T) {
		// Setup: Create retrier
		opts := fetcher.DefaultRetrierOptions()
		retrier := fetcher.NewRetrier(opts)

		attempts := 0
		operation := func() error {
			attempts++
			return fmt.Errorf("permanent error")
		}

		// Execute: Retry operation
		err := retrier.Retry(ctx, operation)

		// Verify: Operation failed immediately (no retries for non-retryable errors)
		assert.Error(t, err)
		assert.Equal(t, 1, attempts)
	})
}

func TestRetryWithValue_Success(t *testing.T) {
	ctx := context.Background()

	t.Run("operation returns value on first attempt", func(t *testing.T) {
		// Setup: Create retrier
		opts := fetcher.DefaultRetrierOptions()
		retrier := fetcher.NewRetrier(opts)

		attempts := 0
		operation := func() (string, error) {
			attempts++
			return "success", nil
		}

		// Execute: RetryWithValue operation
		result, err := fetcher.RetryWithValue(ctx, retrier, operation)

		// Verify: Operation succeeded on first attempt
		require.NoError(t, err)
		assert.Equal(t, "success", result)
		assert.Equal(t, 1, attempts)
	})

	t.Run("operation returns value after retry", func(t *testing.T) {
		// Setup: Create retrier
		opts := fetcher.RetrierOptions{
			MaxRetries:      3,
			InitialInterval: 10 * time.Millisecond,
			MaxInterval:     100 * time.Millisecond,
			Multiplier:      2.0,
		}
		retrier := fetcher.NewRetrier(opts)

		attempts := 0
		operation := func() (string, error) {
			attempts++
			if attempts < 2 {
				return "", &domain.RetryableError{
					Err: errors.New("temporary error"),
				}
			}
			return "success", nil
		}

		// Execute: RetryWithValue operation
		result, err := fetcher.RetryWithValue(ctx, retrier, operation)

		// Verify: Operation succeeded after retry
		require.NoError(t, err)
		assert.Equal(t, "success", result)
		assert.Equal(t, 2, attempts)
	})

	t.Run("operation returns complex value after retry", func(t *testing.T) {
		// Setup: Create retrier
		opts := fetcher.RetrierOptions{
			MaxRetries:      3,
			InitialInterval: 10 * time.Millisecond,
			MaxInterval:     100 * time.Millisecond,
			Multiplier:      2.0,
		}
		retrier := fetcher.NewRetrier(opts)

		type Result struct {
			ID    int
			Name  string
			Value float64
		}

		attempts := 0
		operation := func() (Result, error) {
			attempts++
			if attempts < 2 {
				return Result{}, &domain.RetryableError{
					Err: errors.New("temporary error"),
				}
			}
			return Result{ID: 123, Name: "test", Value: 45.67}, nil
		}

		// Execute: RetryWithValue operation
		result, err := fetcher.RetryWithValue(ctx, retrier, operation)

		// Verify: Operation succeeded after retry
		require.NoError(t, err)
		assert.Equal(t, Result{ID: 123, Name: "test", Value: 45.67}, result)
		assert.Equal(t, 2, attempts)
	})
}

func TestRetryWithValue_Error(t *testing.T) {
	ctx := context.Background()

	t.Run("non-retryable error fails immediately", func(t *testing.T) {
		// Setup: Create retrier
		opts := fetcher.DefaultRetrierOptions()
		retrier := fetcher.NewRetrier(opts)

		attempts := 0
		operation := func() (string, error) {
			attempts++
			return "", errors.New("permanent error")
		}

		// Execute: RetryWithValue operation
		result, err := fetcher.RetryWithValue(ctx, retrier, operation)

		// Verify: Operation failed immediately
		assert.Error(t, err)
		assert.Empty(t, result)
		assert.Equal(t, 1, attempts)
	})

	t.Run("retryable error exhausts retries", func(t *testing.T) {
		// Setup: Create retrier
		opts := fetcher.RetrierOptions{
			MaxRetries:      2,
			InitialInterval: 10 * time.Millisecond,
			MaxInterval:     100 * time.Millisecond,
			Multiplier:      2.0,
		}
		retrier := fetcher.NewRetrier(opts)

		attempts := 0
		operation := func() (string, error) {
			attempts++
			return "", &domain.RetryableError{
				Err: errors.New("temporary error"),
			}
		}

		// Execute: RetryWithValue operation
		result, err := fetcher.RetryWithValue(ctx, retrier, operation)

		// Verify: Operation failed after exhausting retries
		assert.Error(t, err)
		assert.Empty(t, result)
		assert.Equal(t, 3, attempts) // Initial + 2 retries
	})

	t.Run("context cancellation stops retry", func(t *testing.T) {
		// Setup: Create retrier
		opts := fetcher.RetrierOptions{
			MaxRetries:      5,
			InitialInterval: 100 * time.Millisecond,
			MaxInterval:     1 * time.Second,
			Multiplier:      2.0,
		}
		retrier := fetcher.NewRetrier(opts)

		attempts := 0
		operation := func() (string, error) {
			attempts++
			return "", &domain.RetryableError{
				Err: errors.New("temporary error"),
			}
		}

		// Setup: Create context with timeout
		ctx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
		defer cancel()

		// Execute: RetryWithValue operation
		result, err := fetcher.RetryWithValue(ctx, retrier, operation)

		// Verify: Operation failed due to context cancellation
		assert.Error(t, err)
		assert.Empty(t, result)
		assert.GreaterOrEqual(t, attempts, 1)
	})
}

func TestShouldRetryStatus(t *testing.T) {
	testCases := []struct {
		statusCode  int
		shouldRetry bool
		desc        string
	}{
		{429, true, "429 Too Many Requests"},
		{502, true, "502 Bad Gateway"},
		{503, true, "503 Service Unavailable"},
		{504, true, "504 Gateway Timeout"},
		{520, true, "520 Cloudflare Error"},
		{521, true, "521 Cloudflare Error"},
		{522, true, "522 Cloudflare Error"},
		{530, true, "530 Cloudflare Error"},
		{400, false, "400 Bad Request"},
		{401, false, "401 Unauthorized"},
		{403, false, "403 Forbidden"},
		{404, false, "404 Not Found"},
		{500, false, "500 Internal Server Error"},
		{501, false, "501 Not Implemented"},
		{200, false, "200 OK"},
		{301, false, "301 Moved Permanently"},
		{302, false, "302 Found"},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			// Execute: Check if status code should retry
			shouldRetry := fetcher.ShouldRetryStatus(tc.statusCode)

			// Verify: Result matches expected
			assert.Equal(t, tc.shouldRetry, shouldRetry)
		})
	}

	t.Run("all Cloudflare errors (520-530) are retryable", func(t *testing.T) {
		for code := 520; code <= 530; code++ {
			// Execute: Check if status code should retry
			shouldRetry := fetcher.ShouldRetryStatus(code)

			// Verify: All Cloudflare errors are retryable
			assert.True(t, shouldRetry, "Status code %d should be retryable", code)
		}
	})

	t.Run("boundary status codes", func(t *testing.T) {
		testCases := []struct {
			statusCode  int
			shouldRetry bool
			desc        string
		}{
			{519, false, "519 (just below Cloudflare range)"},
			{531, false, "531 (just above Cloudflare range)"},
			{428, false, "428 (just below 429)"},
			{430, false, "430 (just above 429)"},
		}

		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				shouldRetry := fetcher.ShouldRetryStatus(tc.statusCode)
				assert.Equal(t, tc.shouldRetry, shouldRetry)
			})
		}
	})
}

func TestParseRetryAfter(t *testing.T) {
	t.Run("empty string returns zero", func(t *testing.T) {
		// Execute: Parse empty retry-after header
		duration := fetcher.ParseRetryAfter("")

		// Verify: Returns zero duration
		assert.Equal(t, time.Duration(0), duration)
	})

	t.Run("parses seconds as integer", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected time.Duration
			desc     string
		}{
			{"5", 5 * time.Second, "5 seconds"},
			{"10", 10 * time.Second, "10 seconds"},
			{"60", 60 * time.Second, "60 seconds"},
			{"120", 120 * time.Second, "120 seconds"},
			{"0", 0 * time.Second, "0 seconds"},
		}

		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				// Execute: Parse retry-after header
				duration := fetcher.ParseRetryAfter(tc.input)

				// Verify: Duration matches expected
				assert.Equal(t, tc.expected, duration)
			})
		}
	})

	t.Run("ignores non-numeric content", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected time.Duration
			desc     string
		}{
			{"abc", 0 * time.Second, "alphabetic characters"},
			{"5abc", 5 * time.Second, "mixed alphanumeric (parses until non-digit)"},
			{"-5", 0 * time.Second, "negative number"},
			{"3.14", 3 * time.Second, "decimal number (parses integer part)"},
		}

		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				// Execute: Parse retry-after header
				duration := fetcher.ParseRetryAfter(tc.input)

				// Verify: Duration matches expected
				assert.Equal(t, tc.expected, duration)
			})
		}
	})

	t.Run("large values", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected time.Duration
			desc     string
		}{
			{"3600", 3600 * time.Second, "1 hour"},
			{"86400", 86400 * time.Second, "1 day"},
		}

		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				// Execute: Parse retry-after header
				duration := fetcher.ParseRetryAfter(tc.input)

				// Verify: Duration matches expected
				assert.Equal(t, tc.expected, duration)
			})
		}
	})
}
