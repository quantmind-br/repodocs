package git

import (
	"context"

	"github.com/go-git/go-git/v5"
)

// RealClient implements Client using go-git
type RealClient struct{}

// NewClient creates a new RealClient
func NewClient() *RealClient {
	return &RealClient{}
}

// PlainCloneContext calls git.PlainCloneContext
func (c *RealClient) PlainCloneContext(ctx context.Context, path string, isBare bool, o *git.CloneOptions) (*git.Repository, error) {
	return git.PlainCloneContext(ctx, path, isBare, o)
}
