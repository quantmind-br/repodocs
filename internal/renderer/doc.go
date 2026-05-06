// Package renderer provides headless browser rendering for JavaScript-heavy
// sites.
//
// It uses Rod to drive Chromium, manages a tab pool for concurrent rendering,
// applies stealth features, and detects single-page application behavior before
// returning rendered HTML to extraction strategies.
package renderer
