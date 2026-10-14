---
type: Runbook
title: Operations Runbook
description: Operational guidance for running RepoDocs, covering installation and the doctor command, configuration file and environment variables, LLM provider setup with rate limiting and circuit breaking, proxy and CDP sidecar setup for stealth rendering, and common failure scenarios.
resource: github.com/quantmind-br/repodocs
tags: [repodocs, operations, runbook, config, proxy, cdp, llm, troubleshooting]
---

# Operations Runbook

Practical guidance for installing, configuring, and operating RepoDocs. For the
pipeline internals referenced here, see
[Extraction pipeline](../workflows/extraction-pipeline.md) and
[Architecture overview](../architecture/overview.md).

## Installation and prerequisites

- **Go 1.24+** is required to build from source.
- **Chrome or Chromium** is required only for the `--render-js` / JS-heavy-site
  path; it is not needed for static HTML, Markdown, sitemap, or git sources.

```bash
git clone https://github.com/quantmind-br/repodocs.git
cd repodocs
make deps && make build        # → ./build/repodocs
make install                   # install to ~/.local/bin + seed ~/.repodocs/config.yaml
```

Or install a prebuilt binary via the Release workflow, or
`go install github.com/quantmind-br/repodocs/cmd/repodocs@latest`.

Verify the environment with the doctor command, which checks internet
connectivity, Chrome/Chromium presence, write permissions, config, and cache
directory:

```bash
repodocs doctor
```

## Configuration

RepoDocs is configured in layers, later layers overriding earlier ones: defaults
→ config file → environment variables → CLI flags. The config file defaults to
`~/.repodocs/config.yaml`; the full annotated template is
`configs/config.yaml.template` (shipped in archives and `.deb`/`.rpm`).

Manage it interactively or via commands:

```bash
repodocs config            # interactive TUI editor (Bubble Tea + huh)
repodocs config edit       # same as above
repodocs config show       # print current config as YAML
repodocs config init       # write default config to ~/.repodocs/config.yaml
repodocs config path       # print the config file path
ACCESSIBLE=1 repodocs config   # or: repodocs config --accessible (screen-reader mode)
```

Every config key has an environment-variable override with the `REPODOCS_` prefix
and `_` for nested keys (e.g. `REPODOCS_LLM_API_KEY`,
`REPODOCS_CACHE_ENABLED`). Never store secrets in the config file — use env vars
or `repodocs config` for credential entry. For the full config structure, all
sections and defaults, and the proxy resolution model, see
[Configuration](configuration.md).

## LLM metadata enhancement

Optional AI enrichment (summary, tags, category) is off by default. Set
`llm.enhance_metadata: true` (or pass `--json-meta` which implies collection).
Providers are configured under `llm:` in `configs/config.yaml.template`:

| Provider | `llm.provider` | Notes |
| :--- | :--- | :--- |
| OpenAI-compatible (OpenAI, Ollama, vLLM, Groq, Together AI, LM Studio) | `openai` | OpenAI chat-completions wire format; Ollama/LMStudio omit auth |
| Anthropic | `anthropic` | `system` + `messages`, API version `2023-06-01` |
| Google Gemini | `google` | `contents`/`systemInstruction`/`generationConfig` |

API keys are validated on startup (skipped for local Ollama/LMStudio). The
provider is wrapped in a resilience stack (see
[LLM integration](../integrations/llm.md)): **circuit breaker** check →
**retrier** → per-attempt **rate-limit token** → `provider.Complete`. Defaults:
rate limit 60 rpm / burst 10 / 3 retries; circuit breaker 5 failures / 30s
reset. `metadata.go` truncates content to 8000 chars and retries twice with a
simplified prompt.

## Proxy setup

Proxying routes HTTP fetching (and JS rendering where possible) through a proxy.
Supported schemes: `http`, `https`, `socks5`, `socks5h`.

```bash
repodocs https://docs.example.com --proxy socks5://user:pass@host:1080
```

Config-file form (`configs/config.yaml.template`):

```yaml
proxy:
  enabled: true
  url: "socks5://user:pass@p.example.io:1080"   # Option A: full URL (precedence)
  # Option B: structured fields (used when url is empty)
  # type: socks5   # http|https|socks5|socks5h
  # host: p.example.io
  # port: 1080
  # username: ""
  # password: ""
```

The `--proxy` CLI flag overrides the config file and implicitly enables the
proxy. `ProxyConfig.Resolve()` (`internal/config/config.go`) is the single source
of truth that builds the effective proxy URL, and it never echoes raw URLs or
wraps `url.Error` (both can embed passwords) — errors use `u.Redacted()`. SOCKS5
requires an explicit port (`requireProxyPort`) to avoid opaque `net.Dial` errors.

> SOCKS5 username/password auth is fully supported by the HTTP fetcher, but
> headless Chrome cannot authenticate SOCKS5 proxies. For JS rendering behind a
> proxy, use an `http`/`https` proxy or an IP-allowlisted SOCKS5 endpoint.

## CDP sidecar rendering

For stealth or persistent-browser setups, point RepoDocs at an external CDP
browser (CloakBrowser, Camoufox, or a manually launched Chrome) instead of
launching one:

```bash
repodocs https://spa-docs.com --render-js --cdp-endpoint http://127.0.0.1:9222
```

Config:

```yaml
rendering:
  cdp_endpoint: "http://127.0.0.1:9222"   # host:port, http://, or ws(s)://
```

When `cdp_endpoint` is set, the renderer sets `ownsBrowser=false`, so `Close()`
leaves the sidecar running and proxy/stealth flags are delegated to it. The
`proxy` setting still applies to the HTTP fetcher. `cdp_endpoint` does **not**
enable rendering on its own — combine it with `rendering.force_js` or `--render-js`.
`launcher.ResolveURL` normalizes `host:port`, `http://`, and `ws://` forms.

## Caching and incremental sync

- **Cache** (`~/.repodocs/cache`, BadgerDB): minimizes network load. Disable with
  `--no-cache`, force refresh with `--refresh-cache`, change TTL with
  `--cache-ttl` (default `24h`).
- **Incremental sync** (`--sync`): skips unchanged pages by content hash, tracked
  in `.repodocs-state.json`. `--full-sync` ignores state and re-processes
  everything; `--prune` removes files for pages that disappeared upstream. See
  [Extraction pipeline: sync state](../workflows/extraction-pipeline.md#sync-state).

## Common failure scenarios

| Symptom | Likely cause | Action |
| :--- | :--- | :--- |
| `browser not found` / `--render-js` fails | Chrome/Chromium not installed or not on PATH | Install Chrome; run `repodocs doctor` |
| HTTP 403 / Cloudflare block on a site | Stealth blocked at the HTTP layer | Use `--render-js` (renderer fallback on 403), or `--cdp-endpoint` with a stealth sidecar |
| `extraction outcome unsatisfactory` with `no_documents` | Strategy produced no output | Check diagnostics in the error; allow fallback (no `--no-fallback`); try `--strategy crawler` or `--min-docs 0` |
| `filter_zeroed` diagnostic | `--filter` excluded all URLs | Pass the filtered URL as the entry point, or check filter spelling/trailing slash |
| LLM `rate limited` or circuit open | API quota or sustained failures | Tune `llm.rate_limit` (rpm, burst) and `circuit_breaker.reset_timeout`; reduce `--concurrency` |
| SOCKS5 auth ignored for rendering | Chrome limitation | Use an http(s) proxy or IP-allowlisted SOCKS5 (see Proxy setup) |
| Slow / hanging on infinite-scroll pages | Lazy-load not settling | `scroll_to_end` is bounded (10 iterations, 300ms, 2 stable heights); reduce `--timeout` if a site never settles |

The `OutcomeError` printed on a hard-fail includes the strategy, entry URL,
discovered/attempted/written/skipped/failed counts, diagnostics, and
suggestions — start there. See [Self-healing pipeline](../workflows/self-healing.md#outcome-errors).
