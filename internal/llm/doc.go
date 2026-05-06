// Package llm provides LLM provider integrations for metadata enhancement.
//
// It builds providers through a factory and composes resilience wrappers such
// as rate limiting, circuit breakers, and retry handling. Strategies and output
// code use these providers to enrich extracted documentation metadata.
package llm
