---
type: Source Map
title: Source Map
description: Repository navigation map for RepoDocs, listing the top-level directories, internal packages, key files in each, and what they own, with links into the documented workflows and concepts.
resource: github.com/quantmind-br/repodocs
tags: [repodocs, source-map, navigation]
---

# Source Map

A navigation map of the repository. For behavior and rationale, follow the links
into the [workflows](#), [domain](#), and [operations](#) pages.

## Top-level layout

| Path | Purpose |
| :--- | :--- |
| `cmd/repodocs/` | CLI entrypoint (Cobra): flags, subcommands (`doctor`, `version`, `config`), manifest/single-URL wiring |
| `internal/` | All application packages (the bulk of the code) |
| `pkg/version/` | Version ldflags injected at build time |
| `configs/config.yaml.template` | Annotated config template shipped in releases |
| `examples/manifests/` | Sample manifests: `simple`, `multi-source`, `full-options`, `error-tolerant` |
| `tests/` | Centralized test tree (`unit`, `integration`, `e2e`, `benchmark`, `mocks`, `testutil`, `fixtures`, `testdata`) |
| `scripts/` | `release.sh` (interactive tagging), `build.ps1` (Windows build/test) |
| `.github/workflows/` | `ci.yml` (test + coverage), `release.yml` (goreleaser on tags) |
| `Makefile` | Build, test, coverage, lint, release, install targets |
| `.goreleaser.yaml` | Cross-compile matrix (linux/darwin/windows × amd64/arm64), archives, nfpm (.deb/.rpm), changelog |
| `.golangci.yml` | Linter configuration |

## Internal packages

| Package | Key files | Owns / links to |
| :--- | :--- | :--- |
| `internal/app/` | `main.go` is in `cmd/`; here: `orchestrator.go`, `detector.go`, `fallback.go`, `doc.go` | [Architecture overview](architecture/overview.md); [Self-healing pipeline](workflows/self-healing.md); [Extraction strategies](architecture/strategies.md) |
| `internal/strategies/` | `strategy.go`, `crawler.go`, `sitemap.go`, `llms.go`, `pkggo.go`, `docsrs.go` (+`docsrs_*.go`), `wiki.go` (+`wiki_parser.go`), `github_pages.go` (+`_discovery.go`), `git_compat.go` | [Extraction strategies](architecture/strategies.md) |
| `internal/strategies/git/` | `strategy.go`, `processor.go`, `fetcher.go`, `clone.go`, `archive.go`, `parser.go`, `types.go` | Two-layer git strategy (inner strategy + compat wrapper) |
| `internal/fetcher/` | `client.go`, `retry.go`, `stealth.go`, `transport.go` | [Extraction pipeline: fetcher](workflows/extraction-pipeline.md#fetcher) |
| `internal/renderer/` | `rod.go`, `pool.go`, `detector.go`, `stealth.go` | [Extraction pipeline: renderer](workflows/extraction-pipeline.md#renderer) |
| `internal/converter/` | `pipeline.go`, `markdown.go`, `readability.go`, `sanitizer.go`, `rst.go`, `code_blocks.go`, `content_type.go`, `encoding.go` | [Extraction pipeline: converter](workflows/extraction-pipeline.md#converter) |
| `internal/cache/` | `badger.go`, `interface.go`, `keys.go` | [Extraction pipeline: cache](workflows/extraction-pipeline.md#cache) |
| `internal/output/` | `collector.go`, `writer.go` | [Extraction pipeline: output writer](workflows/extraction-pipeline.md#output-writer) |
| `internal/state/` | `manager.go`, `models.go`, `errors.go` | [Extraction pipeline: sync state](workflows/extraction-pipeline.md#sync-state); [Domain data model: SyncState](domain/data-model.md#syncstate) |
| `internal/recovery/` | `validator.go`, `planner.go`, `probes.go`, `probe_cache.go`, `errors.go` | [Self-healing pipeline](workflows/self-healing.md) |
| `internal/llm/` | `provider.go`, `provider_wrapper.go`, `{anthropic,google,openai,ollama,lmstudio}.go`, `retry.go`, `circuit_breaker.go`, `ratelimit.go`, `metadata.go` | [LLM integration](integrations/llm.md) |
| `internal/config/` | `config.go`, `defaults.go`, `loader.go` (+ `proxy_test.go`) | [Operations runbook: configuration](operations/runbook.md#configuration) |
| `internal/converter/` | (see above) | — |
| `internal/manifest/` | `loader.go`, `types.go`, `errors.go` | Batch manifest loading |
| `internal/git/` | `client.go`, `interface.go` | Thin injectable `go-git` wrapper |
| `internal/tui/` | `app.go`, `forms.go`, `categories.go`, `config_adapter.go`, `styles.go`, `validation.go` | Interactive config editor |
| `internal/utils/` | `url.go`, `fs.go`, `logger.go`, `progress.go`, `workerpool.go` | Leaf-level helpers (URL normalization, path generation, zerolog logger, progress bar, generic worker pool) |
| `internal/domain/` | `interfaces.go`, `models.go`, `strategy_result.go`, `errors.go`, `options.go` | [Domain data model](domain/data-model.md) |

## Key files to start with

- **Understand a run**: `cmd/repodocs/main.go` → `internal/app/orchestrator.go`
  (`Run`) → `internal/app/detector.go` (`DetectStrategy`).
- **Add or change a strategy**: `internal/strategies/strategy.go` (interface +
  `Dependencies`), then the relevant strategy file; register detection in
  `internal/app/detector.go`.
- **Change recovery behavior**: `internal/recovery/{validator,planner,probes}.go`
  and `internal/app/fallback.go`.
- **Change output format**: `internal/output/writer.go` and
  `internal/converter/{pipeline,markdown}.go`.
- **Add an LLM provider**: `internal/llm/provider.go` (factory) and a new
  provider file; see [LLM integration](integrations/llm.md).
