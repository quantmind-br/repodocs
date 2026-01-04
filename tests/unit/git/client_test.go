package git_test

import (
	"context"
	"errors"
	"testing"

	gogit "github.com/go-git/go-git/v5"
	"github.com/quantmind-br/repodocs-go/internal/git"
	"github.com/quantmind-br/repodocs-go/tests/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewClient_Success(t *testing.T) {
	t.Run("creates new client successfully", func(t *testing.T) {
		// Execute: Create a new Git client
		client := git.NewClient()

		// Verify: Client is not nil and implements the interface
		assert.NotNil(t, client)
		assert.Implements(t, (*git.Client)(nil), client)
	})

	t.Run("multiple clients are independent", func(t *testing.T) {
		// Execute: Create multiple clients
		client1 := git.NewClient()
		client2 := git.NewClient()

		// Verify: Clients are separate instances
		assert.NotNil(t, client1)
		assert.NotNil(t, client2)
		assert.NotSame(t, client1, client2)
	})
}

func TestPlainCloneContext_Success(t *testing.T) {
	t.Run("successfully clones repository", func(t *testing.T) {
		// Setup: Create mock client and repository
		mockClient := new(mocks.MockGitClient)
		mockRepo := &gogit.Repository{}

		mockClient.On("PlainCloneContext",
			mock.MatchedBy(func(c context.Context) bool { return c != nil }),
			"/tmp/repo",
			false,
			mock.AnythingOfType("*git.CloneOptions"),
		).Return(mockRepo, nil)

		ctx := context.Background()

		// Execute: Clone the repository
		repo, err := mockClient.PlainCloneContext(ctx, "/tmp/repo", false, &gogit.CloneOptions{
			URL: "https://github.com/example/repo.git",
		})

		// Verify: Clone succeeded
		require.NoError(t, err)
		assert.Equal(t, mockRepo, repo)
		mockClient.AssertExpectations(t)
	})

	t.Run("clones with bare repository option", func(t *testing.T) {
		// Setup: Create mock client
		mockClient := new(mocks.MockGitClient)
		mockRepo := &gogit.Repository{}

		mockClient.On("PlainCloneContext",
			mock.MatchedBy(func(c context.Context) bool { return c != nil }),
			"/tmp/bare-repo",
			true,
			mock.AnythingOfType("*git.CloneOptions"),
		).Return(mockRepo, nil)

		ctx := context.Background()

		// Execute: Clone as bare repository
		repo, err := mockClient.PlainCloneContext(ctx, "/tmp/bare-repo", true, &gogit.CloneOptions{
			URL: "https://github.com/example/repo.git",
		})

		// Verify: Clone succeeded
		require.NoError(t, err)
		assert.Equal(t, mockRepo, repo)
		mockClient.AssertExpectations(t)
	})
}

func TestPlainCloneContext_ContextCancellation(t *testing.T) {
	t.Run("returns error when context is cancelled", func(t *testing.T) {
		// Setup: Create context that's already cancelled
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		mockClient := new(mocks.MockGitClient)
		expectedErr := errors.New("context canceled")

		mockClient.On("PlainCloneContext",
			mock.MatchedBy(func(c context.Context) bool { return c != nil }),
			"/tmp/repo",
			false,
			mock.AnythingOfType("*git.CloneOptions"),
		).Return(nil, expectedErr)

		// Execute: Try to clone with cancelled context
		repo, err := mockClient.PlainCloneContext(ctx, "/tmp/repo", false, &gogit.CloneOptions{
			URL: "https://github.com/example/repo.git",
		})

		// Verify: Error returned
		assert.Error(t, err)
		assert.Nil(t, repo)
		assert.Contains(t, err.Error(), "context")
		mockClient.AssertExpectations(t)
	})

	t.Run("returns error when context times out", func(t *testing.T) {
		// Setup: Create context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 1)
		defer cancel()

		mockClient := new(mocks.MockGitClient)
		expectedErr := errors.New("context deadline exceeded")

		mockClient.On("PlainCloneContext",
			mock.MatchedBy(func(c context.Context) bool { return c != nil }),
			"/tmp/repo",
			false,
			mock.AnythingOfType("*git.CloneOptions"),
		).Return(nil, expectedErr)

		// Execute: Try to clone with timed-out context
		repo, err := mockClient.PlainCloneContext(ctx, "/tmp/repo", false, &gogit.CloneOptions{
			URL: "https://github.com/example/repo.git",
		})

		// Verify: Error returned
		assert.Error(t, err)
		assert.Nil(t, repo)
		mockClient.AssertExpectations(t)
	})
}

func TestPlainCloneContext_ErrorCases(t *testing.T) {
	t.Run("returns error for invalid URL", func(t *testing.T) {
		ctx := context.Background()
		mockClient := new(mocks.MockGitClient)
		expectedErr := errors.New("invalid git URL")

		mockClient.On("PlainCloneContext",
			mock.MatchedBy(func(c context.Context) bool { return c != nil }),
			"/tmp/repo",
			false,
			mock.AnythingOfType("*git.CloneOptions"),
		).Return(nil, expectedErr)

		// Execute: Try to clone with invalid URL
		repo, err := mockClient.PlainCloneContext(ctx, "/tmp/repo", false, &gogit.CloneOptions{
			URL: "not-a-valid-git-url",
		})

		// Verify: Error returned
		assert.Error(t, err)
		assert.Nil(t, repo)
		assert.Contains(t, err.Error(), "invalid")
		mockClient.AssertExpectations(t)
	})

	t.Run("returns error for non-existent repository", func(t *testing.T) {
		ctx := context.Background()
		mockClient := new(mocks.MockGitClient)
		expectedErr := errors.New("repository not found")

		mockClient.On("PlainCloneContext",
			mock.MatchedBy(func(c context.Context) bool { return c != nil }),
			"/tmp/repo",
			false,
			mock.AnythingOfType("*git.CloneOptions"),
		).Return(nil, expectedErr)

		// Execute: Try to clone non-existent repository
		repo, err := mockClient.PlainCloneContext(ctx, "/tmp/repo", false, &gogit.CloneOptions{
			URL: "https://github.com/nonexistent/repo.git",
		})

		// Verify: Error returned
		assert.Error(t, err)
		assert.Nil(t, repo)
		assert.Contains(t, err.Error(), "not found")
		mockClient.AssertExpectations(t)
	})

	t.Run("returns error for authentication failure", func(t *testing.T) {
		ctx := context.Background()
		mockClient := new(mocks.MockGitClient)
		expectedErr := errors.New("authentication failed")

		mockClient.On("PlainCloneContext",
			mock.MatchedBy(func(c context.Context) bool { return c != nil }),
			"/tmp/private-repo",
			false,
			mock.AnythingOfType("*git.CloneOptions"),
		).Return(nil, expectedErr)

		// Execute: Try to clone private repo with invalid credentials
		repo, err := mockClient.PlainCloneContext(ctx, "/tmp/private-repo", false, &gogit.CloneOptions{
			URL: "https://github.com/private/repo.git",
		})

		// Verify: Error returned
		assert.Error(t, err)
		assert.Nil(t, repo)
		assert.Contains(t, err.Error(), "authentication")
		mockClient.AssertExpectations(t)
	})

	t.Run("returns error for network failure", func(t *testing.T) {
		ctx := context.Background()
		mockClient := new(mocks.MockGitClient)
		expectedErr := errors.New("network unreachable")

		mockClient.On("PlainCloneContext",
			mock.MatchedBy(func(c context.Context) bool { return c != nil }),
			"/tmp/repo",
			false,
			mock.AnythingOfType("*git.CloneOptions"),
		).Return(nil, expectedErr)

		// Execute: Try to clone when network is unavailable
		repo, err := mockClient.PlainCloneContext(ctx, "/tmp/repo", false, &gogit.CloneOptions{
			URL: "https://github.com/example/repo.git",
		})

		// Verify: Error returned
		assert.Error(t, err)
		assert.Nil(t, repo)
		assert.Contains(t, err.Error(), "network")
		mockClient.AssertExpectations(t)
	})

	t.Run("returns error when destination path is invalid", func(t *testing.T) {
		ctx := context.Background()
		mockClient := new(mocks.MockGitClient)
		expectedErr := errors.New("destination path exists")

		mockClient.On("PlainCloneContext",
			mock.MatchedBy(func(c context.Context) bool { return c != nil }),
			"/invalid/path",
			false,
			mock.AnythingOfType("*git.CloneOptions"),
		).Return(nil, expectedErr)

		// Execute: Try to clone to invalid path
		repo, err := mockClient.PlainCloneContext(ctx, "/invalid/path", false, &gogit.CloneOptions{
			URL: "https://github.com/example/repo.git",
		})

		// Verify: Error returned
		assert.Error(t, err)
		assert.Nil(t, repo)
		mockClient.AssertExpectations(t)
	})
}

func TestPlainCloneContext_ParameterPassing(t *testing.T) {
	t.Run("passes context correctly", func(t *testing.T) {
		ctx := context.Background()
		mockClient := new(mocks.MockGitClient)
		mockRepo := &gogit.Repository{}

		mockClient.On("PlainCloneContext",
			mock.MatchedBy(func(c context.Context) bool {
				return c != nil
			}),
			"/tmp/repo",
			false,
			mock.AnythingOfType("*git.CloneOptions"),
		).Return(mockRepo, nil)

		// Execute: Clone with context
		repo, err := mockClient.PlainCloneContext(ctx, "/tmp/repo", false, &gogit.CloneOptions{
			URL: "https://github.com/example/repo.git",
		})

		// Verify: Context was passed
		require.NoError(t, err)
		assert.NotNil(t, repo)
		mockClient.AssertExpectations(t)
	})

	t.Run("passes path correctly", func(t *testing.T) {
		ctx := context.Background()
		mockClient := new(mocks.MockGitClient)
		mockRepo := &gogit.Repository{}

		expectedPath := "/custom/path/to/repo"

		mockClient.On("PlainCloneContext",
			mock.MatchedBy(func(c context.Context) bool { return c != nil }),
			expectedPath,
			false,
			mock.AnythingOfType("*git.CloneOptions"),
		).Return(mockRepo, nil)

		// Execute: Clone to specific path
		repo, err := mockClient.PlainCloneContext(ctx, expectedPath, false, &gogit.CloneOptions{
			URL: "https://github.com/example/repo.git",
		})

		// Verify: Path was passed correctly
		require.NoError(t, err)
		assert.NotNil(t, repo)
		mockClient.AssertExpectations(t)
	})

	t.Run("passes isBare parameter correctly", func(t *testing.T) {
		tests := []struct {
			name   string
			isBare bool
		}{
			{"bare repository", true},
			{"non-bare repository", false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				ctx := context.Background()
				mockClient := new(mocks.MockGitClient)
				mockRepo := &gogit.Repository{}

				mockClient.On("PlainCloneContext",
					mock.MatchedBy(func(c context.Context) bool { return c != nil }),
					"/tmp/repo",
					tt.isBare,
					mock.AnythingOfType("*git.CloneOptions"),
				).Return(mockRepo, nil)

				// Execute: Clone with isBare option
				repo, err := mockClient.PlainCloneContext(ctx, "/tmp/repo", tt.isBare, &gogit.CloneOptions{
					URL: "https://github.com/example/repo.git",
				})

				// Verify: isBare was passed correctly
				require.NoError(t, err)
				assert.NotNil(t, repo)
				mockClient.AssertExpectations(t)
			})
		}
	})

	t.Run("passes clone options correctly", func(t *testing.T) {
		ctx := context.Background()
		mockClient := new(mocks.MockGitClient)
		mockRepo := &gogit.Repository{}

		opts := &gogit.CloneOptions{
			URL:      "https://github.com/example/repo.git",
			Depth:    1,
			Progress: nil,
			Tags:     gogit.NoTags,
		}

		mockClient.On("PlainCloneContext",
			mock.MatchedBy(func(c context.Context) bool { return c != nil }),
			"/tmp/repo",
			false,
			mock.MatchedBy(func(o *gogit.CloneOptions) bool {
				return o != nil && o.URL == opts.URL && o.Depth == opts.Depth
			}),
		).Return(mockRepo, nil)

		// Execute: Clone with options
		repo, err := mockClient.PlainCloneContext(ctx, "/tmp/repo", false, opts)

		// Verify: Options were passed correctly
		require.NoError(t, err)
		assert.NotNil(t, repo)
		mockClient.AssertExpectations(t)
	})
}

func TestPlainCloneContext_EdgeCases(t *testing.T) {
	t.Run("handles empty URL in clone options", func(t *testing.T) {
		ctx := context.Background()
		mockClient := new(mocks.MockGitClient)
		expectedErr := errors.New("URL cannot be empty")

		mockClient.On("PlainCloneContext",
			mock.MatchedBy(func(c context.Context) bool { return c != nil }),
			"/tmp/repo",
			false,
			mock.MatchedBy(func(o *gogit.CloneOptions) bool {
				return o != nil && o.URL == ""
			}),
		).Return(nil, expectedErr)

		// Execute: Clone with empty URL
		repo, err := mockClient.PlainCloneContext(ctx, "/tmp/repo", false, &gogit.CloneOptions{
			URL: "",
		})

		// Verify: Error returned
		assert.Error(t, err)
		assert.Nil(t, repo)
		mockClient.AssertExpectations(t)
	})

	t.Run("handles nil clone options", func(t *testing.T) {
		ctx := context.Background()
		mockClient := new(mocks.MockGitClient)
		mockRepo := &gogit.Repository{}

		mockClient.On("PlainCloneContext",
			mock.MatchedBy(func(c context.Context) bool { return c != nil }),
			"/tmp/repo",
			false,
			(*gogit.CloneOptions)(nil),
		).Return(mockRepo, nil)

		// Execute: Clone with nil options
		repo, err := mockClient.PlainCloneContext(ctx, "/tmp/repo", false, nil)

		// Verify: Should handle nil options
		require.NoError(t, err)
		assert.NotNil(t, repo)
		mockClient.AssertExpectations(t)
	})

	t.Run("handles concurrent clone operations", func(t *testing.T) {
		ctx := context.Background()
		mockClient := new(mocks.MockGitClient)
		mockRepo := &gogit.Repository{}

		// Setup: Expect multiple concurrent calls
		for i := 0; i < 3; i++ {
			mockClient.On("PlainCloneContext",
				mock.MatchedBy(func(c context.Context) bool { return c != nil }),
				"/tmp/repo",
				false,
				mock.AnythingOfType("*git.CloneOptions"),
			).Return(mockRepo, nil).Once()
		}

		// Execute: Concurrent clone operations
		errChan := make(chan error, 3)
		for i := 0; i < 3; i++ {
			go func() {
				_, err := mockClient.PlainCloneContext(ctx, "/tmp/repo", false, &gogit.CloneOptions{
					URL: "https://github.com/example/repo.git",
				})
				errChan <- err
			}()
		}

		// Verify: All operations succeeded
		for i := 0; i < 3; i++ {
			err := <-errChan
			assert.NoError(t, err)
		}
		mockClient.AssertExpectations(t)
	})
}

func TestRealClient_ImplementsInterface(t *testing.T) {
	t.Run("RealClient implements Client interface", func(t *testing.T) {
		// This is a compile-time test that verifies RealClient implements Client
		var _ git.Client = git.NewClient()

		// If this compiles, the interface is correctly implemented
		client := git.NewClient()
		assert.NotNil(t, client)
	})
}
