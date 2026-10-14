---
type: Architecture
title: Extraction strategies and detection
description: The eight RepoDocs extraction strategies, the Strategy interface and Dependencies, the URL detection cascade, and the --strategy/--no-fallback CLI controls.
resource: https://github.com/quantmind-br/repodocs
tags: [strategies, detection, crawler, git, sitemap]
---

# Extraction strategies and detection

RepoDocs selects one of eight strategies per URL, each implementing the same
`Strategy` interface and producing a `domain.StrategyResult`. Detection is a
first-match cascade; the crawler is the universal fallback.

## The `Strategy` interface

```go
type Strategy interface {
    Name() string
    CanHandle(url string) bool
    Execute(ctx context.Context, url string, opts Options) (*domain.StrategyResult, error)
}
```

- `Options` (`internal/strategies/strategy.go`) embeds `domain.CommonOptions`
  (`Verbose, DryRun, Force, RenderJS, Limit, Sync, FullSync, Prune`) plus
  `Output, Concurrency, MaxDepth, Exclude, NoFolders, Split, IncludeAssets,
  ContentSelector, ExcludeSelector, CacheTTL, FilterURL`.
- `Dependencies` is the shared service bundle injected into every strategy:
  `Fetcher, Renderer, Cache, Converter, Writer, Logger, LLMProvider,
  MetadataEnhancer, Collector, HTTPClient, StateManager`. It is built once in
  `NewDependencies`; the renderer is lazy via `GetRenderer()` (`sync.Once`).
- Every `Execute` creates a result with `domain.NewStrategyResult(name, url)`,
  increments counters as it works, and calls `result.Finish()`. Errors still
  return a populated result so the recovery layer can reason about partial work.

## Detection cascade

`DetectStrategy(rawURL)` in `internal/app/detector.go` checks, in order, and
returns the first match:

| # | Condition | Result |
|---|-----------|--------|
| 1 | empty / whitespace | `unknown` |
| 2 | `git@` or `git+ssh://` prefix | `git` |
| 3 | unparseable but `http(s)://` | `crawler` |
| 4 | `git://` scheme | `unknown` (unsupported) |
| 5 | path ends with `llms.txt` | `llms` |
| 6 | URL contains `pkg.go.dev` | `pkggo` |
| 7 | contains `docs.rs` (no `/src/`, `/source/`) | `docsrs` |
| 8 | path ends `sitemap.xml`[.gz] or `*sitemap*.xml` | `sitemap` |
| 9 | `IsWikiURL` (GitHub/Bitbucket `/wiki`, `.wiki.git`) | `wiki` |
| 10 | host ends `.github.io` | `github_pages` |
| 11 | `.git` suffix, or `github.com`/`gitlab.com`/`bitbucket.org` (not blob, not docs subdomain) | `git` |
| 12 | `http(s)` scheme | `crawler` |
| 13 | otherwise | `unknown` |

Notable precedence: wiki and GitHub Pages are checked **before** generic git so
`github.io` and `/wiki` URLs never fall into the git branch; GitHub/GitLab
`/blob/` and `/-/blob/` URLs are excluded from git and fall through to crawler;
`docs.rs/.../src/...` source views fall back to crawler.

`GetAllStrategies` returns the same logical order as a slice;
`FindMatchingStrategy` iterates it and returns the first instance whose
`CanHandle` is true — a runtime counterpart to the static `DetectStrategy`.

In the orchestrator, a crawler is additionally auto-promoted to a sitemap when
`strategies.DiscoverSitemap` finds one, and a `.xml` URL with sitemap body
content is also promoted (see [Architecture overview](overview.md)).

## Strategy catalog

| Strategy | File | Extracts | `CanHandle` |
|----------|------|----------|-------------|
| `llms` | `internal/strategies/llms.go` | `llms.txt` index; fetches each linked page as markdown/plain/HTML | path ends `llms.txt` |
| `pkggo` | `internal/strategies/pkggo.go` | Go package docs from `pkg.go.dev`; `--split` breaks into sections | contains `pkg.go.dev` |
| `docsrs` | `internal/strategies/docsrs.go` | Rust crate docs from **rustdoc JSON** (rendered by `docsrs_renderer.go`) | host `docs.rs`, not a source view |
| `sitemap` | `internal/strategies/sitemap.go` | Sitemap XML; handles index files and `.gz`; sorts by `lastmod` | ends `sitemap.xml`[.gz] or contains `sitemap` |
| `wiki` | `internal/strategies/wiki.go` | GitHub/Bitbucket **wiki git repos** via shallow clone (`wiki_parser.go`) | `IsWikiURL` |
| `github_pages` | `internal/strategies/github_pages.go` | `*.github.io` sites with HTTP-first, browser-fallback two-phase flow | host ends `.github.io` |
| `git` | `internal/strategies/git/` | Documentation files from a git repo; archive download first, clone fallback | git hosts (non-blob, non-docs) |
| `crawler` | `internal/strategies/crawler.go` | Arbitrary sites via `colly`; on-demand JS rendering; 403 renderer fallback | any `http(s)` URL (catch-all) |

### Strategy behaviors to know

- **docsrs** parses docs.rs URLs into crate/version/module components
  (`docsrs_types.go`), fetches rustdoc JSON, validates its `FormatVersion`, and
  renders items through a `RustdocRenderer`. It is not an HTML scraper.
- **sitemap** handles nested sitemap index files in batches
  (`processSitemapIndex`) and emits `DiagFilterZeroed` / `DiagSitemapShallow`
  diagnostics that the recovery planner uses.
- **github_pages** uses SPA-shell heuristics (min HTML/body-text lengths) to
  decide when Chromium is needed, and caps browser concurrency at 5.
- **git** adapter (`internal/strategies/git_compat.go`) wraps
  `internal/strategies/git` and maps common `Options` to `git.ExecuteOptions`;
  it tries a GitHub archive download before cloning and uses
  `DetectDefaultBranch` unless an external HTTP client is supplied.
- **crawler** enforces same-domain scoping, the `FilterURL` base, exclude
  regexps, the limit, and a `sync.Map` visited set. It renders JS when
  `opts.RenderJS` or `renderer.NeedsJSRendering(html)` is true, and exposes a
  `makeRendererFallback` for the fetcher's HTTP-403 path.

## Forcing and suppressing strategies

The CLI (`cmd/repodocs/main.go`) wires two controls into `OrchestratorOptions`:

- `--strategy <name>` — bypasses `DetectStrategy`, validates the name with
  `IsValidStrategy`, and **also disables automatic fallback** (forcing implies
  you want exactly this strategy).
- `--no-fallback` — disables the self-healing retry independently of detection.
- `--min-docs <n>` — sets the minimum documents for a successful extraction;
  the recovery layer retries below this (default effective `1`).

Both controls are read per source in manifest mode. See
[Self-healing fallback](../workflows/self-healing.md) for how the verdict model
consumes strategy outcomes.
