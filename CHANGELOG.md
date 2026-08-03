# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Probe-informed self-healing fallback (Phase 3)
- 1-level self-healing strategy fallback (Phase 2)
- SOCKS5 proxy and external CDP endpoint support
- `--strategy` flag to force extraction method
- Self-healing pipeline (Phase 0+1)
- Bare URL and description support in `llms.txt`

### Changed

- Repository relicensed from CC BY-NC 4.0 to MIT
- Repository renamed from `repodocs-go` to `repodocs`

### Removed

- Agent-tooling artifacts and internal analysis documents from repository history

## [1.0.2] - 2026

### Fixed

- Cross-platform test assertion and goreleaser repo name
- Disable homebrew tap upload

## [1.0.0] - 2026

### Added

- Initial public release of the RepoDocs CLI and library
- Multi-source extraction: web crawler, Git/GitHub, sitemaps, `llms.txt`, `pkg.go.dev`, `docs.rs`, GitHub Wiki, GitHub Pages
- HTML → Markdown conversion pipeline (encoding → readability → sanitizer → markdown)
- JavaScript rendering via headless Chromium (`go-rod`) with tab pooling
- Stealth HTTP client with user-agent rotation and TLS fingerprinting
- BadgerDB persistent caching with configurable TTL
- Exponential backoff with circuit breaker and rate limiting
- Multi-provider LLM integration (OpenAI, Anthropic, Google, Ollama, LM Studio)
- Interactive configuration TUI (Bubble Tea/Huh)
- Batch processing with YAML/JSON manifests
- Incremental synchronization state tracking
- Consolidated `repodocs.json` metadata output
- GitHub Actions CI with per-package coverage thresholds
- Multi-platform release builds (deb, rpm) via GoReleaser

## [0.1.0] - 2025

### Added

- Project foundation: detector → strategy → fetcher/renderer → converter → writer pipeline
- Sitemap auto-discovery and detection
- Git strategy with archive download and subdirectory filtering
- Markdown content detection to avoid HTML parser node limits
- Initial configuration model and CLI entrypoint

[Unreleased]: https://github.com/quantmind-br/repodocs/compare/v1.0.2...HEAD
[1.0.2]: https://github.com/quantmind-br/repodocs/compare/v1.0.0...v1.0.2
[1.0.0]: https://github.com/quantmind-br/repodocs/compare/v0.1.0...v1.0.0
[0.1.0]: https://github.com/quantmind-br/repodocs/releases/tag/v0.1.0
