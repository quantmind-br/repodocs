package git

import (
	"context"

	"github.com/go-git/go-git/v5"
)

// Client defines the interface for Git operations
type Client interface {
	PlainCloneContext(ctx context.Context, path string, isBare bool, o *git.CloneOptions) (*git.Repository, error)
}
