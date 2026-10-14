---
type: Guide
title: Testing Guide
description: How to run and write tests for RepoDocs, covering the test layout (unit, integration, e2e, benchmark), Make targets and coverage thresholds, generated mocks and shared test utilities, and conventions for table-driven, mock-based, short-flag-respecting tests.
resource: github.com/quantmind-br/repodocs
tags: [repodocs, testing, ci, coverage, mocks]
---

# Testing Guide

RepoDocs uses a centralized `tests/` tree plus in-package tests colocated with
source. The authoritative conventions are in `TESTING.md`; this page summarizes
them and links the key locations.

## Running tests

All entry points are Make targets (`Makefile`); on Windows use
`scripts/build.ps1 <target>`:

```bash
make test          # go test -v -race -short ./...   (unit; -short skips slow suites)
make test-all      # go test -v -race ./...          (unit + integration + e2e)
make coverage      # coverage report → ./coverage/coverage.{out,html}
make lint          # gofmt -s -w . && golangci-lint run ./...
make build         # CGO_ENABLED=0 go build → ./build/repodocs
```

Run a single suite directly:

```bash
go test ./tests/unit/...
go test ./tests/integration/...
go test ./tests/e2e/...
go test ./tests/benchmark/... -bench=.
```

The `-short` flag (used by `make test`) skips browser-launch, long-retry, and
live-network cases to keep the quick loop fast.

## Test layout

| Subdir | Purpose |
| :--- | :--- |
| `tests/unit/` | Fast mock-based tests; default `go test ./...` scope. Mixed flat files and package subdirs (`app/`, `recovery/`, `strategies/`, `llm/`, …). |
| `tests/integration/` | Real-service or `httptest` tests: orchestrator, converter pipeline, cache flow, fetcher, renderer, strategies. |
| `tests/e2e/` | End-to-end pipelines: `full_pipeline_test.go`, `crawl_test.go`, `sitemap_test.go`, `self_healing_test.go`, `config_test.go`. |
| `tests/benchmark/` | Performance tests (`git_clone_benchmark_test.go`). |
| `tests/mocks/` | Generated mocks (see below). |
| `tests/testutil/` | Shared helpers and factories. |
| `tests/helpers/` | `fixtures.go`, `http.go`. |
| `tests/fixtures/`, `tests/testdata/` | Sample data and golden files. |

In-package tests live alongside source (e.g. `internal/llm/*_test.go`,
`internal/recovery/*_test.go`).

## CI and coverage

`.github/workflows/ci.yml` runs on push/PR to `main`/`master` with three jobs:
`test` (Ubuntu), `test-windows`, and `comment-pr` (posts a coverage comment on
PRs). Go is pinned via `go-version-file: 'go.mod'`; both jobs run
`go test -short ./...` with coverage.

Coverage thresholds are **package-specific** (a `declare -A thresholds` map in
`ci.yml` splits `coverage.out` per package with `awk`):

| Package | Min | Package | Min |
| :--- | :--- | :--- | :--- |
| `internal/domain` | 85% | `internal/app` | 48% |
| `internal/output` | 80% | `internal/git` | 80% |
| `internal/cache` | 75% | `internal/converter` | 75% |
| `internal/llm` | 70% | `internal/fetcher` | 70% |
| `internal/config` | 54% | `cmd/repodocs` | 48% |
| `internal/strategies` | 34% | `internal/renderer` | 28% |

Releases are built by `.github/workflows/release.yml` (goreleaser) on `v*` tags.
See the [Operations runbook](operations/runbook.md) for release commands.

## Mocks and utilities

Mocks are generated with `go.uber.org/mock/gomock` (with some legacy
`testify/mock` retained), in `tests/mocks/`:

- `domain.go` — gomock mocks for all domain interfaces: `MockStrategy`,
  `MockFetcher`, `MockRenderer`, `MockCache`, `MockConverter`, `MockWriter`,
  `MockLLMProvider`; plus `SimpleMockCache`, `SimpleMockFetcher`,
  `MultiResponseMockFetcher`.
- `git.go` — gomock `MockClient` for `git.Client`; `git_mock.go` is the legacy
  testify variant.

Regenerate (from `tests/mocks/README.md`):

```bash
go run go.uber.org/mock/mockgen -source=internal/domain/interfaces.go -destination=tests/mocks/domain.go -package mocks
go run go.uber.org/mock/mockgen -source=internal/git/interface.go     -destination=tests/mocks/git.go     -package mocks
```

Shared helpers in `tests/testutil/` (per `tests/testutil/README.md`): in-memory
cache + `VerifyCacheEntry` (`cache.go`), `TempDir`/`TempOutputDir` (`temp.go`),
`NewTestServer` (`http.go`), test loggers, document factories, and assertions
(`AssertFileExists`, `AssertFileContains`, `AssertDocumentContent`).

## Conventions

From `TESTING.md`:

- **Table-driven** tests, named `TestFunction_Scenario`.
- **Mock external deps** (network, LLM, git) for determinism.
- Use `t.Cleanup()` for servers and temp dirs.
- **Test error paths**, not just happy paths; avoid `_ = err` — assert real
  errors.
- **Respect `-short`**: slow tests go behind `-short` or into `integration/`/
  `e2e/`.
- **Regression first**: write the failing test before fixing a bug.
- One behavior per test; assert observable contracts, not implementation details.

### Testing the self-healing pipeline

The recovery and fallback code has dedicated suites (recently hardened in
commits `f62b6e0`/`852d83c`):

- `tests/e2e/self_healing_test.go` — end-to-end fallback scenarios.
- `internal/recovery/{planner,probes,probe_cache}_test.go` — planner candidates
  and probe behavior.
- `internal/app/fallback_test.go` — the fallback loop and dedup/budget bounds.
- `tests/unit/recovery/{validator,errors}_test.go` — verdict classification and
  outcome errors.

See [Self-healing pipeline](workflows/self-healing.md) for what these cover.
