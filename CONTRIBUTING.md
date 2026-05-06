# Contributing to RepoDocs

Thank you for helping improve RepoDocs. This guide covers the basics for setting up the project, running checks, and submitting changes.

## Getting Started

```bash
git clone https://github.com/quantmind-br/repodocs.git
cd repodocs
make deps
make build
make test
```

The build output is written to `./build/repodocs`. Use `./build/repodocs doctor` to check local runtime dependencies.

## Development Setup

- Install Go 1.24 or newer.
- Install Chrome or Chromium if you need to use or test JavaScript rendering (`--render-js`).
- Run `make deps` after changing `go.mod` or `go.sum`.
- Review [TESTING.md](TESTING.md), [Makefile](Makefile), and [.github/workflows/ci.yml](.github/workflows/ci.yml) before larger changes.

## Code Style

- Use standard Go formatting: `gofmt` / `go fmt`.
- Run `go vet ./...` for static checks.
- Run `make lint` before opening a pull request. It runs `gofmt -s -w .` and `golangci-lint run ./...`.
- Keep imports grouped as standard library, external dependencies, then internal packages.
- Prefer small interfaces in `internal/domain` and implementations in package-specific directories.

## Testing

Use the Makefile and test guide as the source of truth.

```bash
make test       # short race-enabled test suite
make test-all   # full race-enabled test suite
make coverage   # coverage report in ./coverage/
```

Additional suites may be run directly:

```bash
go test ./tests/integration/...
go test ./tests/e2e/...
```

Aim for the coverage targets documented in [TESTING.md](TESTING.md) and enforced in [.github/workflows/ci.yml](.github/workflows/ci.yml). Add or update tests for behavior changes and bug fixes.

## Pull Request Process

1. Open an issue first for major features or behavior changes.
2. Keep pull requests focused and reasonably small.
3. Update documentation, examples, and tests when behavior changes.
4. Run `make test` and relevant integration or e2e tests before requesting review.
5. Ensure CI passes before merge.

## Commit Message Convention

Use concise, imperative commit messages:

```text
fix crawler redirect handling
add docs.rs renderer tests
update manifest validation errors
```

Prefer prefixes such as `add`, `fix`, `update`, `remove`, `refactor`, `test`, or `docs` when they clarify intent.

## Reporting Issues

When reporting a bug, include:

- RepoDocs version or commit.
- Operating system and Go version.
- Command, manifest, or configuration used.
- Expected behavior and actual behavior.
- Minimal reproduction steps and relevant logs.

For security issues, do not open a public issue. Follow [SECURITY.md](SECURITY.md).
