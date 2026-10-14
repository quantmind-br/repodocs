---
type: Overview
title: RepoDocs Quickstart
description: RepoDocs is a Go CLI and manifest tool that extracts documentation from websites, Git repositories, sitemaps, package indexes, and wikis into structured Markdown, with optional LLM enrichment and a probe-informed self-healing fallback pipeline.
resource: github.com/quantmind-br/repodocs
tags: [repodocs, quickstart, cli, documentation-extraction]
---

# RepoDocs Quickstart

**RepoDocs** (`github.com/quantmind-br/repodocs`, Go 1.24+) extracts
documentation from many source types into structured Markdown with YAML
frontmatter. It supports reproducible RAG ingestion, local documentation
mirrors, and dataset preparation from a single CLI command or a batch manifest.

Sources it understands out of the box: documentation websites (crawler),
`llms.txt` indexes, `pkg.go.dev`, `docs.rs` (Rust crates), sitemaps,
GitHub/Bitbucket wikis, GitHub Pages sites, and Git repositories. A
[self-healing fallback pipeline](workflows/self-healing.md) automatically
retries with alternative strategies when an extraction produces too little
output.

> This wiki documents the repository, not the published user guide. Start here,
> then follow the links to each section. The canonical user-facing docs are
> `README.md` and `TESTING.md`.

## Install and run

```bash
git clone https://github.com/quantmind-br/repodocs.git
cd repodocs
make deps
make build
./build/repodocs version
```

Or `go install github.com/quantmind-br/repodocs/cmd/repodocs@latest`. Verify
runtime dependencies with `repodocs doctor` (it checks internet access,
Chrome/Chromium for JS rendering, write permissions, and cache/config dirs).

A few common invocations:

```bash
repodocs https://docs.example.com                     # crawl a docs site
repodocs https://github.com/user/repo --max-depth 2   # clone a repo
repodocs https://spa-docs.com --render-js             # force JS rendering
repodocs https://example.com --json-meta --limit 10   # cap pages, emit JSON index
repodocs --manifest sources.yaml                      # batch multiple sources
```

## Core concepts

- **Strategy** — a pluggable extractor for one source type. See
  [Extraction strategies](architecture/strategies.md) for the full list and
  detection order.
- **Orchestrator** — wires strategies to shared dependencies (fetcher, renderer,
  converter, writer, cache, state) and runs single-URL or manifest extraction.
  See [Architecture overview](architecture/overview.md).
- **StrategyResult** — structured telemetry each strategy returns (URLs
  discovered/attempted, docs written/skipped/failed, diagnostics). See
  [Domain data model](domain/data-model.md).
- **Self-healing** — when a strategy yields too few documents, the orchestrator
  validates the outcome and retries up to two alternative strategies chosen by a
  static planner and, if needed, probe-informed refinement. See
  [Self-healing pipeline](workflows/self-healing.md).

## How a URL is handled

RepoDocs [detects the strategy](architecture/strategies.md) from the URL, then runs
it through a [shared extraction pipeline](workflows/extraction-pipeline.md):
fetch (cached, stealth HTTP) → optional JS render → convert to Markdown → write
document → optional LLM metadata enhancement. The detector order is:

```
git@/git+ssh → llms.txt → pkg.go.dev → docs.rs → sitemap.xml → wiki → GitHub Pages → Git → crawler (fallback)
```

Pass `--strategy <name>` to force a strategy, or `--no-fallback` to disable
automatic fallback.

## CLI flags at a glance

| Flag | Default | Purpose |
| :--- | :--- | :--- |
| `--manifest` | | Path to YAML/JSON manifest for batch processing |
| `-o, --output` | `./docs` | Output directory |
| `-j, --concurrency` | `5` | Concurrent workers |
| `-d, --max-depth` | `4` | Maximum crawl depth |
| `-l, --limit` | `0` (unlimited) | Maximum pages |
| `--strategy` | auto | Force: `llms`, `pkggo`, `docsrs`, `sitemap`, `wiki`, `github_pages`, `git`, `crawler` |
| `--render-js` | `false` | Force JavaScript rendering |
| `--cdp-endpoint` | | External CDP browser URL |
| `--proxy` | | Proxy URL (`http`, `https`, `socks5`, `socks5h`) |
| `--content-selector` | | CSS selector for main content |
| `--exclude-selector` | | CSS selector for elements to remove |
| `--exclude` | | Regex URL/path patterns to skip |
| `--filter` | | Web: base URL scope; git: subdirectory |
| `--json-meta` | `false` | Generate a JSON metadata index |
| `--sync` / `--full-sync` / `--prune` | `false` | Incremental sync options |
| `--min-docs` | `1` | Minimum documents for success; below triggers fallback |
| `--no-fallback` | `false` | Disable strategy fallback |
| `--no-cache` / `--refresh-cache` / `--cache-ttl` | | Cache control (`--cache-ttl` default `24h`) |
| `--timeout` | `1m30s` | Request timeout |
| `--dry-run` / `--force` / `--nofolders` / `--split` / `--include-assets` | | Output control |
| `--verbose` (`-v`) | `false` | Debug logging |

The full flag table is in `README.md`; configuration beyond flags lives in a
YAML config file and env vars — see
[Operations runbook: configuration](operations/runbook.md#configuration).

## Batch processing with manifests

A manifest declares multiple sources and shared options:

```yaml
sources:
  - url: https://docs.example.com
    strategy: crawler
    content_selector: "article.main"
    max_depth: 3
  - url: https://github.com/org/repo
    strategy: git
    include:
      - "docs/**/*.md"
      - "README.md"
options:
  output: ./knowledge-base
  continue_on_error: true
```

Run with `repodocs --manifest sources.yaml`. Without `continue_on_error`, the
first failing source stops the run. Manifests are loaded by
`internal/manifest/`; the orchestrator runs sources in a bounded worker pool
(concurrency capped at 3 for batch runs). See `/examples/manifests/` for
ready-made examples (`simple.yaml`, `multi-source.yaml`, `error-tolerant.yaml`,
`full-options.yaml`). For the full manifest schema, source/options fields, and
run mechanics, see [Manifest and sync](workflows/manifest-and-sync.md).
Incremental sync behavior is documented in
[Extraction pipeline: sync state](workflows/extraction-pipeline.md#sync-state).

## Where to go next

- [Architecture overview](architecture/overview.md) — layers, orchestrator, dependencies, single-URL flow.
- [Extraction strategies](architecture/strategies.md) — all eight strategies and detection rules.
- [Extraction pipeline](workflows/extraction-pipeline.md) — fetcher, renderer, converter, cache, output, sync.
- [Self-healing pipeline](workflows/self-healing.md) — validation verdicts, planner tiers, probes.
- [Domain data model](domain/data-model.md) — `Document`, `StrategyResult`, diagnostics, interfaces, errors.
- [Operations runbook](operations/runbook.md) — installation, config, LLM setup, proxy/CDP, troubleshooting.
- [LLM integration](integrations/llm.md) — providers, resilience stack, metadata enrichment.
- [Testing guide](testing.md) — layout, make targets, CI coverage thresholds, mocks.
- [Source map](source-map.md) — directory and file index.

## Backlog

Areas noted but not yet given a dedicated page; each is covered briefly where it
naturally belongs.

- **TUI config editor internals** (`/internal/tui/`) — covered as a user-facing feature in [Operations runbook: configuration](operations/runbook.md#configuration), not a dedicated internals page.
- **goreleaser / release workflow** (`/.goreleaser.yaml`, `/scripts/release.sh`, `/.github/workflows/release.yml`) — documented as commands in the [Source map](source-map.md) and [Testing guide](testing.md#ci-and-coverage); a dedicated release-engineering page is deferred until the process grows.
- **utils/ shared helpers** (`/internal/utils/`) — URL normalization, path generation, zerolog logger, progress bar, generic worker pool `Pool[T]`; summarized in the [Source map](source-map.md) rather than a standalone page.
