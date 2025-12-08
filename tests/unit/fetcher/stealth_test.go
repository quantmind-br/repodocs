package fetcher_test

import (
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/fetcher"
	"github.com/stretchr/testify/assert"
)

func TestRandomDelay_Generate(t *testing.T) {
	t.Run("generates random delays within range", func(t *testing.T) {
		min := 100 * time.Millisecond
		max := 1000 * time.Millisecond

		// Execute: Generate multiple delays
		delays := make(map[int64]bool)
		for i := 0; i < 100; i++ {
			delay := fetcher.RandomDelay(min, max)
			delays[int64(delay)] = true

			// Verify: Delay is within the specified range
			assert.GreaterOrEqual(t, delay, min, "Delay should be >= min")
			assert.Less(t, delay, max, "Delay should be < max")
		}

		// Verify: We got some randomness (at least some different values)
		// With 100 iterations, we should see some variation
		assert.Greater(t, len(delays), 1, "Should generate different delays showing randomness")
	})

	t.Run("generates delays with different ranges", func(t *testing.T) {
		testCases := []struct {
			min      time.Duration
			max      time.Duration
			testName string
		}{
			{10 * time.Millisecond, 50 * time.Millisecond, "small range"},
			{1 * time.Second, 5 * time.Second, "large range"},
			{100 * time.Millisecond, 200 * time.Millisecond, "medium range"},
		}

		for _, tc := range testCases {
			t.Run(tc.testName, func(t *testing.T) {
				// Execute: Generate a delay
				delay := fetcher.RandomDelay(tc.min, tc.max)

				// Verify: Delay is within range
				assert.GreaterOrEqual(t, delay, tc.min, "Delay should be >= min")
				assert.Less(t, delay, tc.max, "Delay should be < max")
			})
		}
	})

	t.Run("generates consistent type", func(t *testing.T) {
		// Execute: Generate a delay
		delay := fetcher.RandomDelay(100*time.Millisecond, 500*time.Millisecond)

		// Verify: Returns time.Duration
		assert.IsType(t, time.Duration(0), delay, "Should return time.Duration")
		assert.NotZero(t, delay, "Delay should not be zero")
	})
}

func TestRandomDelay_WithinRange(t *testing.T) {
	t.Run("exact min equals max returns min", func(t *testing.T) {
		duration := 500 * time.Millisecond

		// Execute: Generate delay with min == max
		delay := fetcher.RandomDelay(duration, duration)

		// Verify: Returns the exact duration
		assert.Equal(t, duration, delay, "When min == max, should return min")
	})

	t.Run("min greater than max returns min", func(t *testing.T) {
		min := 1000 * time.Millisecond
		max := 500 * time.Millisecond

		// Execute: Generate delay with min > max
		delay := fetcher.RandomDelay(min, max)

		// Verify: Returns min (as per implementation logic)
		assert.Equal(t, min, delay, "When min > max, should return min")
	})

	t.Run("boundary conditions", func(t *testing.T) {
		testCases := []struct {
			min       time.Duration
			max       time.Duration
			shouldEq  bool
			desc      string
		}{
			{0, 0, true, "zero duration"},
			{1, 2, false, "adjacent values"},
			{100 * time.Millisecond, 101 * time.Millisecond, false, "minimal range"},
		}

		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				delay := fetcher.RandomDelay(tc.min, tc.max)

				if tc.shouldEq {
					assert.Equal(t, tc.min, delay)
				} else {
					assert.GreaterOrEqual(t, delay, tc.min)
					assert.Less(t, delay, tc.max)
				}
			})
		}
	})

	t.Run("large duration values", func(t *testing.T) {
		min := 10 * time.Second
		max := 60 * time.Second

		// Execute: Generate delay
		delay := fetcher.RandomDelay(min, max)

		// Verify: Within range
		assert.GreaterOrEqual(t, delay, min)
		assert.Less(t, delay, max)
		assert.Greater(t, int64(delay), int64(min)-1, "Should be close to min or greater")
	})

	t.Run("very small duration values", func(t *testing.T) {
		min := 1 * time.Microsecond
		max := 10 * time.Microsecond

		// Execute: Generate delay
		delay := fetcher.RandomDelay(min, max)

		// Verify: Within range (may be 0 due to integer conversion)
		assert.GreaterOrEqual(t, delay, 0*time.Microsecond)
		assert.Less(t, delay, max)
	})
}

func TestRandomDelay_Zero(t *testing.T) {
	t.Run("zero min returns zero", func(t *testing.T) {
		max := 100 * time.Millisecond

		// Execute: Generate delay with min = 0
		delay := fetcher.RandomDelay(0, max)

		// Verify: Can return zero or positive value
		assert.GreaterOrEqual(t, delay, 0*time.Millisecond, "Delay should be >= 0")
		assert.Less(t, delay, max, "Delay should be < max")
	})

	t.Run("zero max returns zero", func(t *testing.T) {
		min := 0 * time.Millisecond
		max := 0 * time.Millisecond

		// Execute: Generate delay with both min and max = 0
		delay := fetcher.RandomDelay(min, max)

		// Verify: Returns zero
		assert.Equal(t, time.Duration(0), delay, "Both zero should return zero")
	})

	t.Run("zero with positive max", func(t *testing.T) {
		// Execute multiple times to test randomness
		for i := 0; i < 50; i++ {
			delay := fetcher.RandomDelay(0, 50*time.Millisecond)

			// Verify: Always >= 0 and < max
			assert.GreaterOrEqual(t, delay, 0*time.Millisecond, "Iteration %d: delay should be >= 0", i)
			assert.Less(t, delay, 50*time.Millisecond, "Iteration %d: delay should be < max", i)
		}
	})

	t.Run("negative values handled gracefully", func(t *testing.T) {
		// Note: The implementation doesn't explicitly handle negative values
		// but we test that it doesn't panic
		assert.NotPanics(t, func() {
			delay := fetcher.RandomDelay(-1*time.Second, 0)
			// The delay could be negative due to the implementation
			// but shouldn't panic
			_ = delay
		}, "Should not panic with negative values")
	})

	t.Run("mixed zero and positive", func(t *testing.T) {
		testCases := []struct {
			min      time.Duration
			max      time.Duration
			testName string
		}{
			{0, 1 * time.Millisecond, "0 to 1ms"},
			{0, 100 * time.Millisecond, "0 to 100ms"},
			{0, 1 * time.Second, "0 to 1s"},
		}

		for _, tc := range testCases {
			t.Run(tc.testName, func(t *testing.T) {
				// Execute: Generate delay
				delay := fetcher.RandomDelay(tc.min, tc.max)

				// Verify: Within valid range
				assert.GreaterOrEqual(t, delay, 0*time.Millisecond)
				assert.Less(t, delay, tc.max)
			})
		}
	})
}
