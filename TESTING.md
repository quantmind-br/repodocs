# Testing Guide

This document provides comprehensive information about testing practices in the repodocs-go project.

## Test Coverage Overview

**Current Coverage**: 64.8% overall

### Coverage by Package

| Package | Coverage | Status |
|---------|----------|--------|
| git | 100.0% | ✅ Excellent |
| domain | 97.1% | ✅ Excellent |
| manifest | 97.0% | ✅ Excellent |
| state | 95.5% | ✅ Excellent |
| output | 94.4% | ✅ Excellent |
| llm | 92.9% | ✅ Excellent |
| utils | 90.9% | ✅ Excellent |
| cache | 91.3% | ✅ Excellent |
| config | 91.4% | ✅ Excellent |
| converter | 87.3% | ✅ Good |
| fetcher | 84.1% | ✅ Good |
| strategies/git | 79.5% | ✅ Good |
| app | 58.8% | ⚠️ Needs improvement |
| strategies | 40.0% | ⚠️ Needs improvement |
| renderer | 30.3% | ⚠️ Needs work |

## Running Tests

### Unit Tests (Fast)
```bash
make test
# or
go test ./... -short
```

### Integration Tests
```bash
make test-integration
# or
go test ./tests/integration/... -tags=integration
```

### E2E Tests
```bash
make test-e2e
# or
go test ./tests/e2e/... -tags=e2e
```

### Coverage Report
```bash
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.html
open coverage.html
```

## Test Structure

```
tests/
├── unit/               # Fast unit tests with mocks
│   ├── app/           # Application orchestrator tests
│   ├── cache/         # Cache implementation tests
│   ├── config/        # Configuration tests
│   ├── converter/     # HTML/Markdown conversion tests
│   ├── domain/        # Domain models and interfaces
│   ├── fetcher/       # HTTP client tests
│   ├── git/           # Git operations tests
│   ├── llm/           # LLM provider tests
│   ├── manifest/      # Manifest parsing tests
│   ├── output/        # Output writer tests
│   ├── renderer/      # Browser renderer tests
│   ├── state/         # State manager tests
│   └── strategies/    # Strategy pattern tests
├── integration/       # Integration tests with real services
│   ├── fetcher/
│   ├── llm/
│   ├── renderer/
│   └── strategies/
└── e2e/              # End-to-end tests
```

## Testing Patterns

### 1. Table-Driven Tests

Preferred for testing multiple scenarios:

```go
func TestFunction(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantErr bool
    }{
        {"valid input", "test", false},
        {"invalid input", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := YourFunction(tt.input)
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

### 2. Mock Testing

Using `go.uber.org/mock` for interfaces:

```go
// Generate mock
//go:generate mockgen -source=domain/cache.go -destination=../../mocks/mock_cache.go

func TestWithMock(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    mockCache := mocks.NewMockCache(ctrl)
    mockCache.EXPECT().Get(gomock.Any(), "key").Return([]byte("value"), nil)
}
```

### 3. Test Helpers

Create reusable test helpers:

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

### 4. Fixture Organization

```
tests/
└── fixtures/
    ├── html/           # HTML samples for testing
    │   ├── simple.html
    │   ├── spa.html
    │   └── markdown.html
    └── sitemap/        # Sitemap samples
        ├── small.xml
        └── index.xml
```

## Coverage Targets

- **Minimum**: 80% per package
- **Target**: 90% per package
- **Excellent**: 95%+ per package

Packages below 80% require attention and additional test coverage.

## CI/CD Integration

Tests run automatically on:
- Pull request creation
- Push to main branch
- Scheduled nightly builds

Coverage badges are updated automatically based on test results.

## Best Practices

1. **Write tests first** (TDD when possible)
2. **Keep tests fast** - Use `-short` flag for quick feedback
3. **Use descriptive test names** - `TestFunction_Scenario`
4. **Table-driven for multiple cases** - Better maintainability
5. **Mock external dependencies** - Tests should be deterministic
6. **Clean up resources** - Use `t.Cleanup()` and `defer`
7. **Test error paths** - Not just happy paths
8. **Add benchmarks** - For performance-critical code

## Example: Adding New Tests

```go
// 1. Identify function to test
func MyFunction(input string) (string, error)

// 2. Create test file: tests/unit/mypackage/mypackage_test.go
package mypackage_test

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/quantmind-br/repodocs-go/internal/mypackage"
)

// 3. Write test
func TestMyFunction(t *testing.T) {
    result, err := mypackage.MyFunction("test")
    assert.NoError(t, err)
    assert.Equal(t, "expected", result)
}

// 4. Run test
// go test ./tests/unit/mypackage/ -v
```

## Coverage Reports

Generate detailed coverage reports:

```bash
# Function coverage
go test ./internal/... -coverprofile=coverage.out
go tool cover -func=coverage.out | grep -v "100.0%"

# HTML report
go tool cover -html=coverage.out -o coverage.html

# By package
go test ./internal/... -coverprofile=coverage.out
go tool cover -func=coverage.out | grep "^github.com"
```

## Troubleshooting

### Tests Timing Out
- Use `-short` flag to skip integration tests
- Check for infinite loops in goroutines
- Add explicit timeouts to contexts

### Browser Tests Failing
- Ensure Chrome/Chromium is installed
- Check `renderer.IsAvailable()` before running
- Use `testing.Short()` to skip when browser unavailable

### Cache Issues
- Clear cache: `rm -rf ~/.repodocs-cache`
- Run with cache disabled: `REPODOCS_CACHE=off ./repodocs`

### Mock Generation
```bash
# Install mockgen
go install go.uber.org/mock/mockgen@latest

# Generate mocks
go generate ./...
```

## Additional Resources

- [Effective Go Testing](https://go.dev/doc/effective_go.html#testing)
- [Table-Driven Tests](https://dave.cheney.net/2019/03/04/table-driven-tests-in-go)
- [Test Coverage](https://blog.golang.org/cover/go14)
- [Go Mock](https://github.com/golang/mock)
