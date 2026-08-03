# Testing Guide

This document describes how RepoDocs is tested, how to run the suites, and the coverage rules enforced in CI.

## Test Layout

The project keeps its tests in the `tests/` directory alongside the main module:

```
tests/
├── unit/          # Fast unit tests with mocks (default `go test ./...` scope)
├── integration/   # Tests against real services (httptest mocks, live dependencies)
├── e2e/           # End-to-end pipeline tests
├── benchmark/     # Benchmarks for performance-critical code
├── mocks/         # Generated mocks
├── testutil/      # Shared test helpers and factories
└── fixtures/      # HTML, sitemap, docsrs, git, llms sample data
```

In-package tests also live next to the source they cover (e.g. `internal/llm/`, `internal/config/`).

## Running the Suites

All test entry points are wired through the Makefile:

```bash
make test        # unit tests:  go test -v -race -short ./...
make test-all    # full suite:  go test -v -race ./...
make coverage    # coverage report + HTML in ./coverage/
```

Individual suites can be run directly with `go test`:

```bash
# Unit tests
go test ./tests/unit/...

# Integration tests
go test ./tests/integration/...

# End-to-end tests
go test ./tests/e2e/...

# Benchmarks
go test ./tests/benchmark/... -bench=.
```

The `-short` flag used by `make test` skips the slower suites (renderer/browser-heavy tests, long-running integration cases) so the quick feedback loop stays fast. Use `make test-all` for the complete run.

## Coverage

Coverage is measured per package and enforced in CI with **package-specific thresholds**, not a single global minimum.

### Thresholds enforced in CI

| Package | Required | Package | Required |
|---------|----------|---------|----------|
| `internal/domain` | 85% | `internal/app` | 48% |
| `internal/converter` | 75% | `internal/config` | 54% |
| `internal/output` | 80% | `internal/cache` | 75% |
| `internal/git` | 80% | `internal/fetcher` | 70% |
| `internal/llm` | 70% | `internal/renderer` | 28% |
| `internal/strategies` | 34% | `cmd/repodocs` | 48% |

Thresholds live in `.github/workflows/ci.yml`. When you raise a package's coverage, update the threshold there if you want CI to enforce the new bar.

### Generating a local report

```bash
make coverage
# HTML report: ./coverage/coverage.html
```

## CI Behavior

The CI workflow (`.github/workflows/ci.yml`) runs on every push to `main`/`master` and on pull requests:

1. **`test`** (Linux) — runs `go test -short ./...` with coverage, then checks every package against its threshold. A package below its threshold fails the job.
2. **`test-windows`** — runs the same suite on Windows to catch platform-specific failures.
3. **`comment-pr`** — posts the coverage summary on pull requests.

## Writing Tests

### Conventions

- **Table-driven tests** for multiple cases: `TestFunction_Scenario`.
- **Mock external dependencies** (network, LLM providers, git) so tests are deterministic. Mocks live in `tests/mocks/`; shared helpers in `tests/testutil/`.
- **Use `t.Cleanup()`** for resource cleanup (servers, temp dirs).
- **Test error paths**, not just happy paths.
- **Respect the `-short` flag**: anything slow (browser launch, long retries, live network) should be skipped in short mode or placed in `tests/integration/` / `tests/e2e/`.
- **Avoid `_ = err` patterns** in production code; assert real error values.

### Example: mock-based unit test

```go
// internal/cache/cache_test.go
func TestCache_Get_Miss(t *testing.T) {
    // use a testutil helper or a real in-memory implementation
    c, err := cache.New(cache.Options{Dir: t.TempDir()})
    if err != nil {
        t.Fatalf("New: %v", err)
    }
    got, err := c.Get(context.Background(), "missing")
    if err != nil {
        t.Fatalf("Get on missing key: %v", err)
    }
    if len(got) != 0 {
        t.Fatalf("expected empty result, got %q", got)
    }
}
```

### Example: HTTP server helper

```go
// tests/testutil/testutil.go
func SetupTestServer(t *testing.T) *httptest.Server {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("OK"))
    }))
    t.Cleanup(server.Close)
    return server
}
```

### Adding a benchmark

```go
func BenchmarkExtract(b *testing.B) {
    for i := 0; i < b.N; i++ {
        // exercise the path under test
    }
}
```

Benchmarks live in `tests/benchmark/` or next to the code they measure.

## Best Practices

1. Write the test that captures the regression **before** fixing the bug (reproduce → fix → confirm).
2. Keep the quick suite fast — move slow cases behind `-short` or into integration/e2e.
3. Name tests descriptively (`TestFunction_Scenario`) so failures are self-explanatory.
4. One behavior per test; assert observable contracts, not implementation details.
5. When a bug fix changes behavior, add a regression test that fails on the old behavior.
