// Package fetcher provides a stealth HTTP client with caching, retries, and
// transport customization.
//
// It implements the domain Fetcher interface, applies browser-like headers and
// stealth behavior, reuses cached responses when available, and retries
// transient failures through configurable HTTP transports.
package fetcher
