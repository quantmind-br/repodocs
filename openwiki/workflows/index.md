# Files

- [Extraction Pipeline](extraction-pipeline.md) - The shared data flow a single document takes through RepoDocs, covering the stealth fetcher with caching and retries, lazy headless rendering, the HTML-to-Markdown converter, the output writer with metadata, and incremental sync state.
- [Manifest batch processing and incremental sync](manifest-and-sync.md) - How RepoDocs processes multiple sources from a YAML/JSON manifest, tracks incremental sync state, writes Markdown with frontmatter, and emits a consolidated metadata index.
- [Self-Healing Fallback Pipeline](self-healing.md) - RepoDocs recovery pipeline that validates strategy outcomes and retries with alternative strategies, using a two-tier planner (static candidates plus probe-informed refinement) bounded to three executions per URL.
