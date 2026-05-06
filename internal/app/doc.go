// Package app provides top-level orchestration and URL detection for
// documentation extraction.
//
// Orchestrator coordinates strategy selection, dependency wiring, and execution
// for single URLs and manifest runs. Detector identifies the right extraction
// strategy for a URL before control passes into the strategies package.
package app
