# Mocks Directory

This directory contains generated mocks for testing using `go.uber.org/mock/gomock`.

## Generated Mocks

### domain.go
Contains gomock mocks for all domain interfaces:
- `MockStrategy` - mocks domain.Strategy
- `MockFetcher` - mocks domain.Fetcher
- `MockRenderer` - mocks domain.Renderer
- `MockCache` - mocks domain.Cache
- `MockConverter` - mocks domain.Converter
- `MockWriter` - mocks domain.Writer
- `MockLLMProvider` - mocks domain.LLMProvider

Also includes simple test helpers:
- `SimpleMockCache` - in-memory cache for basic testing
- `SimpleMockFetcher` - simple fetcher for basic testing
- `MultiResponseMockFetcher` - fetcher with multiple URL responses

### git.go
Contains gomock mock for Git client:
- `MockClient` - mocks git.Client interface

### git_mock.go (Legacy)
Contains testify/mock for Git client (kept for backward compatibility):
- `MockGitClient` - mocks git.Client interface using testify

## Usage

### Using gomock mocks

```go
import (
    "testing"
    "go.uber.org/mock/gomock"
    "github.com/quantmind-br/repodocs-go/tests/mocks"
)

func TestSomething(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    mockFetcher := mocks.NewMockFetcher(ctrl)
    mockFetcher.EXPECT().Get(gomock.Any(), "https://example.com").
        Return(&domain.Response{StatusCode: 200, Body: []byte("test")}, nil)

    // Use mockFetcher in tests
}
```

### Using simple test helpers

```go
import (
    "github.com/quantmind-br/repodocs-go/tests/mocks"
)

func TestSomething(t *testing.T) {
    mockCache := mocks.NewSimpleMockCache()
    defer mockCache.Close()

    // Use mockCache - no controller needed
}
```

## Regenerating Mocks

To regenerate mocks after interface changes:

```bash
# Regenerate domain mocks
go run go.uber.org/mock/mockgen -source=internal/domain/interfaces.go -destination=tests/mocks/domain.go -package mocks

# Regenerate git mock
go run go.uber.org/mock/mockgen -source=internal/git/interface.go -destination=tests/mocks/git.go -package mocks
```

## Notes

- All gomock mocks follow the standard pattern: `NewMock{Interface}(ctrl)`
- Simple helpers are provided for common test scenarios
- Legacy testify mocks are kept for backward compatibility during migration
- Domain interfaces (Fetcher, Renderer) don't have separate mock files as they're included in domain.go
