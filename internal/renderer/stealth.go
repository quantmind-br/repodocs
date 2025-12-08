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

	// Additional JavaScript to run on every page to hide automation
	js := `
		// Override navigator.webdriver
		Object.defineProperty(navigator, 'webdriver', {
			get: () => undefined
		});

		// Override navigator.plugins
		Object.defineProperty(navigator, 'plugins', {
			get: () => [
				{
					0: {type: "application/x-google-chrome-pdf", suffixes: "pdf", description: "Portable Document Format"},
					description: "Portable Document Format",
					filename: "internal-pdf-viewer",
					length: 1,
					name: "Chrome PDF Plugin"
				},
				{
					0: {type: "application/pdf", suffixes: "pdf", description: "Portable Document Format"},
					description: "Portable Document Format",
					filename: "mhjfbmdgcfjbbpaeojofohoefgiehjai",
					length: 1,
					name: "Chrome PDF Viewer"
				},
				{
					0: {type: "application/x-nacl", suffixes: "", description: "Native Client Executable"},
					1: {type: "application/x-pnacl", suffixes: "", description: "Portable Native Client Executable"},
					description: "",
					filename: "internal-nacl-plugin",
					length: 2,
					name: "Native Client"
				}
			]
		});

		// Override navigator.languages
		Object.defineProperty(navigator, 'languages', {
			get: () => ['en-US', 'en']
		});

		// Override WebGL vendor
		const getParameter = WebGLRenderingContext.prototype.getParameter;
		WebGLRenderingContext.prototype.getParameter = function(parameter) {
			if (parameter === 37445) {
				return 'Intel Inc.';
			}
			if (parameter === 37446) {
				return 'Intel Iris OpenGL Engine';
			}
			return getParameter.apply(this, arguments);
		};
	`

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
