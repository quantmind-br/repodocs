package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProgressBar(t *testing.T) {
	t.Run("determinate progress bar with known total", func(t *testing.T) {
		total := 100
		description := DescDownloading

		bar := NewProgressBar(total, description)

		require.NotNil(t, bar)
		assert.NotNil(t, bar)
	})

	t.Run("indeterminate progress bar with unknown total", func(t *testing.T) {
		total := -1
		description := DescCrawling

		bar := NewProgressBar(total, description)

		require.NotNil(t, bar)
		assert.NotNil(t, bar)
	})

	t.Run("zero total", func(t *testing.T) {
		total := 0
		description := DescProcessing

		bar := NewProgressBar(total, description)

		require.NotNil(t, bar)
		assert.NotNil(t, bar)
	})

	t.Run("large total", func(t *testing.T) {
		total := 1000000
		description := DescExtracting

		bar := NewProgressBar(total, description)

		require.NotNil(t, bar)
		assert.NotNil(t, bar)
	})
}

func TestProgressBarDescriptions(t *testing.T) {
	t.Run("DescCrawling constant", func(t *testing.T) {
		assert.Equal(t, "Crawling", DescCrawling)
	})

	t.Run("DescDownloading constant", func(t *testing.T) {
		assert.Equal(t, "Downloading", DescDownloading)
	})

	t.Run("DescProcessing constant", func(t *testing.T) {
		assert.Equal(t, "Processing", DescProcessing)
	})

	t.Run("DescExtracting constant", func(t *testing.T) {
		assert.Equal(t, "Extracting", DescExtracting)
	})
}

func TestProgressBarWithStandardDescriptions(t *testing.T) {
	tests := []struct {
		name        string
		total       int
		description string
	}{
		{
			name:        "crawling with unknown total",
			total:       -1,
			description: DescCrawling,
		},
		{
			name:        "downloading with known total",
			total:       50,
			description: DescDownloading,
		},
		{
			name:        "processing with known total",
			total:       200,
			description: DescProcessing,
		},
		{
			name:        "extracting with known total",
			total:       75,
			description: DescExtracting,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bar := NewProgressBar(tt.total, tt.description)
			require.NotNil(t, bar)
		})
	}
}

func TestProgressBarOptionConsistency(t *testing.T) {
	t.Run("determinate bars are created consistently", func(t *testing.T) {
		total := 100
		description := DescProcessing

		bar1 := NewProgressBar(total, description)
		bar2 := NewProgressBar(total, description)

		// Both bars should be non-nil and created successfully
		require.NotNil(t, bar1)
		require.NotNil(t, bar2)
		assert.NotNil(t, bar1)
		assert.NotNil(t, bar2)
	})

	t.Run("indeterminate bars are created consistently", func(t *testing.T) {
		total := -1
		description := DescCrawling

		bar1 := NewProgressBar(total, description)
		bar2 := NewProgressBar(total, description)

		// Both bars should be non-nil and created successfully
		require.NotNil(t, bar1)
		require.NotNil(t, bar2)
		assert.NotNil(t, bar1)
		assert.NotNil(t, bar2)
	})
}

func TestProgressBarCustomDescription(t *testing.T) {
	t.Run("custom description with known total", func(t *testing.T) {
		customDesc := "Custom Task"
		total := 10

		bar := NewProgressBar(total, customDesc)

		require.NotNil(t, bar)
		assert.NotNil(t, bar)
	})

	t.Run("custom description with unknown total", func(t *testing.T) {
		customDesc := "Custom Indeterminate"
		total := -1

		bar := NewProgressBar(total, customDesc)

		require.NotNil(t, bar)
		assert.NotNil(t, bar)
	})

	t.Run("empty description", func(t *testing.T) {
		emptyDesc := ""
		total := 50

		bar := NewProgressBar(total, emptyDesc)

		require.NotNil(t, bar)
		assert.NotNil(t, bar)
	})
}

func TestProgressBarOperations(t *testing.T) {
	t.Run("add to determinate bar", func(t *testing.T) {
		total := 10
		bar := NewProgressBar(total, DescProcessing)

		require.NotNil(t, bar)

		// Adding should not panic
		assert.NotPanics(t, func() {
			bar.Add(1)
			bar.Add(5)
		})
	})

	t.Run("finish determinate bar", func(t *testing.T) {
		total := 10
		bar := NewProgressBar(total, DescDownloading)

		require.NotNil(t, bar)

		// Finish should not panic
		assert.NotPanics(t, func() {
			bar.Finish()
		})
	})

	t.Run("add to indeterminate bar", func(t *testing.T) {
		total := -1
		bar := NewProgressBar(total, DescCrawling)

		require.NotNil(t, bar)

		// Adding should not panic
		assert.NotPanics(t, func() {
			bar.Add(1)
			bar.Add(5)
		})
	})

	t.Run("finish indeterminate bar", func(t *testing.T) {
		total := -1
		bar := NewProgressBar(total, DescCrawling)

		require.NotNil(t, bar)

		// Finish should not panic
		assert.NotPanics(t, func() {
			bar.Finish()
		})
	})
}

func TestProgressBarEdgeCases(t *testing.T) {
	t.Run("negative total other than -1", func(t *testing.T) {
		total := -100
		bar := NewProgressBar(total, DescProcessing)

		require.NotNil(t, bar)
		assert.NotNil(t, bar)
	})

	t.Run("very small total", func(t *testing.T) {
		total := 1
		bar := NewProgressBar(total, DescDownloading)

		require.NotNil(t, bar)
		assert.NotNil(t, bar)
	})

	t.Run("description with special characters", func(t *testing.T) {
		description := "Loading... (100%)"
		total := 100
		bar := NewProgressBar(total, description)

		require.NotNil(t, bar)
		assert.NotNil(t, bar)
	})

	t.Run("very long description", func(t *testing.T) {
		description := "This is a very long description that might be used for a progress bar "
		bar := NewProgressBar(50, description)

		require.NotNil(t, bar)
		assert.NotNil(t, bar)
	})
}
