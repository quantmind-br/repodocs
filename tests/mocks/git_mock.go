package mocks

import (
	"context"

	"github.com/go-git/go-git/v5"
	"github.com/stretchr/testify/mock"
)

// MockGitClient mocks the GitClient interface
type MockGitClient struct {
	mock.Mock
}

// PlainCloneContext mocks the git clone operation
func (m *MockGitClient) PlainCloneContext(ctx context.Context, path string, isBare bool, o *git.CloneOptions) (*git.Repository, error) {
	args := m.Called(ctx, path, isBare, o)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*git.Repository), args.Error(1)
}
