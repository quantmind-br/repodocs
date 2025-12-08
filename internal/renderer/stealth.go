package renderer

import (
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/stealth"
)

// StealthPage creates a new stealth page that's harder to detect as automated
func StealthPage(browser *rod.Browser) (*rod.Page, error) {
	page, err := stealth.Page(browser)
	if err != nil {
		return nil, err
	}
	return page, nil
}

// ApplyStealthMode applies stealth mode configurations to a page
// This includes removing webdriver flags and emulating real browser behavior
func ApplyStealthMode(page *rod.Page) error {
	// The stealth package already handles most of this, but we can add extra measures

	// Set a realistic viewport using proto.EmulationSetDeviceMetricsOverride
	err := page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
		Width:  1920,
		Height: 1080,
	})
	if err != nil {
		return err
	}

	// Simple stealth: just hide webdriver flag
	// Rod expects arrow function format: () => expression
	js := `() => { Object.defineProperty(navigator, 'webdriver', { get: () => undefined }); return true; }`
	_, err = page.Eval(js)
	return err
}

// StealthOptions contains options for stealth mode
type StealthOptions struct {
	// HideWebdriver hides the webdriver property
	HideWebdriver bool
	// EmulatePlugins emulates real browser plugins
	EmulatePlugins bool
	// RandomizeViewport randomizes the viewport size
	RandomizeViewport bool
	// DisableAutomationFlags disables Chrome automation flags
	DisableAutomationFlags bool
}

// DefaultStealthOptions returns default stealth options
func DefaultStealthOptions() StealthOptions {
	return StealthOptions{
		HideWebdriver:          true,
		EmulatePlugins:         true,
		RandomizeViewport:      false,
		DisableAutomationFlags: true,
	}
}
