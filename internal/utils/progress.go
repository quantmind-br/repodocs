package utils

import "github.com/schollz/progressbar/v3"

// Standard progress bar descriptions
const (
	DescCrawling    = "Crawling"
	DescDownloading = "Downloading"
	DescProcessing  = "Processing"
	DescExtracting  = "Extracting"
)

// NewProgressBar creates a consistently styled progress bar.
//
// Parameters:
//   - total: Total number of items. Use -1 for unknown totals (indeterminate/spinner mode).
//   - description: Text description to show before the progress bar (e.g., DescCrawling, DescDownloading).
//
// Behavior:
//   - For unknown totals (total < 0): Uses spinner type 14 with blank state rendering.
//   - For known totals (total >= 0): Shows count and iterations/second (its).
//   - All progress bars show count.
//
// Example:
//
//	bar := utils.NewProgressBar(len(items), utils.DescDownloading)
//	defer bar.Finish()
//
//	for _, item := range items {
//	    // Process item
//	    bar.Add(1)
//	}
func NewProgressBar(total int, description string) *progressbar.ProgressBar {
	// Build common options
	opts := []progressbar.Option{
		progressbar.OptionSetDescription(description),
		progressbar.OptionShowCount(),
	}

	// Add options based on whether total is known
	if total < 0 {
		// Unknown total: use spinner mode
		opts = append(opts,
			progressbar.OptionSpinnerType(14),
			progressbar.OptionSetRenderBlankState(true),
		)
	} else {
		// Known total: show iterations/second
		opts = append(opts,
			progressbar.OptionShowIts(),
		)
	}

	return progressbar.NewOptions(total, opts...)
}
