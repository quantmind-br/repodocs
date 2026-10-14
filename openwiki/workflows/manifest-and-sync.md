---
type: Workflow
title: Manifest batch processing and incremental sync
description: How RepoDocs processes multiple sources from a YAML/JSON manifest, tracks incremental sync state, writes Markdown with frontmatter, and emits a consolidated metadata index.
resource: https://github.com/quantmind-br/repodocs
tags: [manifest, sync, output, metadata]
---

# Manifest batch processing and incremental sync

RepoDocs runs a single URL through the orchestrator, or a whole batch of sources
from a manifest. Output is Markdown with YAML frontmatter plus an optional
metadata index, and incremental runs skip unchanged pages.

## Manifest batch mode

A manifest is a YAML or JSON file with a `sources` list and global `options`
(`internal/manifest/types.go`). `Loader.Load`/`LoadFromBytes`
(`internal/manifest/loader.go`) parse by extension (`.yaml`/`.yml`/`.json`),
apply defaults, and `Validate` (requires at least one source with a non-empty
URL). Run with `repodocs --manifest sources.yaml`.

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

### Source and options fields

| Source field | Type | Notes |
|--------------|------|-------|
| `url` | string | Required. |
| `strategy` | string | Optional force (`crawler`, `git`, `sitemap`, ...). |
| `content_selector` / `exclude_selector` | string | CSS selectors for main content / elements to drop. |
| `exclude` / `include` | array | URL/path patterns to skip; `include` scopes the git strategy. |
| `max_depth` / `limit` | int | Crawl depth / page cap. |
| `render_js` | bool | Force JavaScript rendering for this source. |

| Options field | Default | Notes |
|---------------|---------|-------|
| `continue_on_error` | `false` | Continue on source failure; exit non-zero if any failed. |
| `output` | `./docs` | Output directory for all sources. |
| `concurrency` | `5` | Concurrent workers. |
| `cache_ttl` | `24h` | Cache TTL. |

### How manifest runs work

`Orchestrator.RunManifest` (`internal/app/orchestrator.go`) iterates sources
sequentially. Each source is merged onto a copy of the base `OrchestratorOptions`
(the same struct the CLI assembles), so per-source `strategy`, `content_selector`,
`render_js`, etc. override the global flags. Each source then runs through the
same detection + self-healing path as a single URL. `ManifestResult` records each
source's outcome, error, and duration. Example manifests live in
`examples/manifests/` (`simple.yaml`, `multi-source.yaml`, `error-tolerant.yaml`,
`full-options.yaml`).

## Incremental sync

`state.Manager` (`internal/state/`) persists per-page state to a
`.repodocs-state.json` sidecar in the output directory. Enable incremental mode
with `--sync`; use `--full-sync` to force re-processing everything and ignore
existing state.

- **Schema:** `SyncState{Version, SourceURL, Strategy, LastSync, Pages}` where
  each `PageState{ContentHash, FetchedAt, FilePath}`. `StateVersion = 1`.
- **Predicate:** `ShouldProcess(url, contentHash)` returns true for a new URL or
  when the stored hash differs — the core skip-unchanged check.
- **Deleted pages:** `MarkSeen(url)` records visited pages into a `sync.Map`
  during a run; `GetDeletedPages()` returns known pages not seen; with `--prune`
  the orchestrator removes their files via `deps.PruneDeletedFiles`.
- **Resilience:** a missing sidecar returns `ErrStateNotFound`; bad JSON returns
  `ErrStateCorrupted`; a version mismatch warns and returns
  `ErrVersionMismatch` (forces a rebuild). The manager is disabled in full-sync
  mode and when constructed with `Disabled`, in which case every page is processed.

The orchestrator calls `deps.SaveState(ctx)` at the end of a successful run.

## Output and metadata

`output.Writer` (`internal/output/writer.go`) persists documents under the base
directory (default `./docs`):

- Markdown files are prefixed with YAML frontmatter
  (`converter.AddFrontmatter`); raw documents are written verbatim.
- **No-overwrite default:** if `!force` and the target file exists, `Write`
  returns nil (skips). Use `--force` to overwrite, or `--dry-run` to simulate
  without writing.
- Paths come from `GeneratePath`/`GeneratePathFromRelative` (markdown) or
  `GenerateRawPathFromRelative` (raw); `--nofolders` flattens the structure.

`output.MetadataCollector` (`internal/output/collector.go`) builds a consolidated
index enabled by `--json-meta`:

- `Add` records each document's `SimpleDocumentMetadata` (path relative to the
  base dir, with the run's strategy as `Source`).
- `Flush` writes `<baseDir>/<filename>` (default `metadata.json`) as a
  `SimpleMetadataIndex{GeneratedAt, SourceURL, Strategy, TotalDocuments,
  Documents}` — the LLM-evaluation-friendly format. It only writes when enabled
  and non-empty.

LLM enrichment, when configured, augments each document's `Summary`, `Tags`, and
`Category` before writing; see [LLM integration](../integrations/llm.md).
