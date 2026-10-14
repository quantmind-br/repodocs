---
type: Operations
title: Configuration
description: RepoDocs configuration model and sources (CLI flags, REPODOCS_ environment variables, config file), the proxy model with its fetcher-vs-renderer asymmetry, defaults, and the interactive config TUI.
resource: https://github.com/quantmind-br/repodocs
tags: [config, proxy, environment, tui]
---

# Configuration

RepoDocs resolves configuration from three layers, last-wins: a config file,
`REPODOCS_*` environment variables, and CLI flags (some flags are bound to
config keys via Viper).

## Configuration sources

- **Config file** — default `~/.repodocs/config.yaml`. `config.Load` uses the
  global Viper; `LoadWithViper` returns an isolated instance. Search paths are
  `.` (project-local) then `~/.repodocs`; a missing file is non-fatal. A template
  ships at `configs/config.yaml.template` and is seeded by `make install`.
- **Environment variables** — prefix `REPODOCS_`, `.` -> `_` (e.g.
  `REPODOCS_RENDERING_CDP_ENDPOINT`). `AutomaticEnv` is enabled after every key
  is registered with a default in `setDefaults`.
- **CLI flags** — `cmd/repodocs/main.go` registers persistent flags; a subset
  (`output`, `concurrency`, `max-depth`, `timeout`, `nofolders`, `force`,
  `no-cache`, `cache-ttl`, `render-js`, `cdp-endpoint`, `json-meta`,
  `user-agent`) are bound to config keys. Flags not bound are read directly.

After unmarshal, `Config.Validate()` fills invalid/missing values with defaults
and validates LLM rate-limit/circuit-breaker ranges. Use `repodocs config` for
the interactive TUI, or `config show` / `config init` / `config path`.

## Config structure

`Config` (`internal/config/config.go`) has these sections:

| Section | Key fields | Defaults |
|---------|-----------|----------|
| `output` | `directory`, `flat`, `json_metadata`, `overwrite` | `./docs`, off, off, off |
| `concurrency` | `workers`, `timeout`, `max_depth` | `5`, `90s`, `3` |
| `cache` | `enabled`, `ttl`, `directory` | on, `24h`, `~/.repodocs/cache` |
| `rendering` | `force_js`, `js_timeout`, `scroll_to_end`, `cdp_endpoint` | off, `60s`, on |
| `stealth` | `user_agent`, `random_delay_min/max` | empty (rotates), small delays |
| `proxy` | `enabled`, `url`, `type`, `host`, `port`, `username`, `password` | disabled |
| `exclude` | regex patterns | `.pdf`, login/logout/admin/sign-in/up paths |
| `logging` | `level`, `format` | `info`, `pretty` |
| `llm` | provider, api_key, model, max_tokens, temperature, timeout, retries, rate-limit, circuit-breaker | unset / conservative |
| `git` | `max_file_size` | `10MB` |

Named defaults live in `internal/config/defaults.go` (`Default()` returns a
fully populated `*Config` with proxy disabled). Path helpers: `ConfigDir()` ->
`~/.repodocs`, `CacheDir()` -> `~/.repodocs/cache`.

## Proxy model

Proxy supports `http`, `https`, `socks5`, `socks5h` and can be configured two
ways (`ProxyConfig.Resolve()` in `config.go`):

1. **URL form** (takes precedence): a single `proxy.url` such as
   `socks5://user:pass@host:1080`.
2. **Structured form:** `proxy.type` + `proxy.host` + `proxy.port` + optional
   `proxy.username`/`proxy.password`. `socks` and empty type normalize to
   `socks5`; SOCKS5 requires an explicit port.

`Resolve()` returns a single URL consumed by both the fetcher and the renderer.
`resolve` deliberately never echoes credentials — errors use generic messages and
`u.Redacted()`. Proxy is validated lazily at point of use (in
`applyProxyFlag`/`NewOrchestrator`), so a broken config-file proxy does not block
a `--proxy` CLI override.

```mermaid
flowchart LR
    P[ProxyConfig.Resolve] -->|single URL| F[Fetcher: tls-client]
    P -->|single URL| R[Renderer: Chrome --proxy-server]
    F -. full SOCKS5 auth .- F
    R -. HTTP HTTPS CDP auth only SOCKS5 creds ignored .- R
```

**Important asymmetry:** the fetcher (tls-client) supports full SOCKS5 auth. The
renderer (Chrome `--proxy-server`) cannot authenticate SOCKS5 proxies, so SOCKS5
credentials are silently ignored on the render path — full SOCKS5 auth is handled
only by the fetcher. `socks5h` is mapped to `socks5` for Chrome to avoid an IP
leak. See [Extraction pipeline](../workflows/extraction-pipeline.md) for both paths.

## LLM rate limiting and circuit breaking

The `llm` section configures optional resilience around provider calls:

- **Rate limit:** `requests_per_minute` (60), `burst_size` (10), plus retry
  backoff (`initial_delay`, `max_delay`, `multiplier`, `jitter_factor`).
- **Circuit breaker:** `failure_threshold` (5), `success_threshold_half_open` (1),
  `reset_timeout` (30s).

These are consumed by the `RateLimitedProvider` wrapper; see
[LLM integration](../integrations/llm.md) for execution order.
