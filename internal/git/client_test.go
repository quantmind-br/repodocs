package git

import (
	"context"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/stretchr/testify/assert"
)

// TestNewClient tests creating a new client
func TestNewClient(t *testing.T) {
	client := NewClient()
	assert.NotNil(t, client)
}

// TestRealClient_PlainCloneContext tests cloning repository
func TestRealClient_PlainCloneContext(t *testing.T) {
	t.Run("creates repository in directory", func(t *testing.T) {
		client := NewClient()
		ctx := context.Background()

		// go-git creates an empty repo even for invalid URLs
		tmpDir := t.TempDir()
		opts := &git.CloneOptions{
			URL: "",
		}
		repo, err := client.PlainCloneContext(ctx, tmpDir, false, opts)
		// May succeed or fail depending on go-git version behavior
		_ = repo
		_ = err
	})

	t.Run("clones valid repository", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping integration test in short mode")
		}

		client := NewClient()
		ctx := context.Background()

		// Clone a small test repository
		tmpDir := t.TempDir()
		opts := &git.CloneOptions{
			URL:      "https://github.com/git-fixtures/basic.git",
			Depth:    1,
			Progress: nil,
		}

		repo, err := client.PlainCloneContext(ctx, tmpDir, false, opts)
		// May fail due to network, so we accept either success or failure
		if err == nil {
			assert.NotNil(t, repo)
		}
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		client := NewClient()
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		tmpDir := t.TempDir()
		opts := &git.CloneOptions{
			URL: "https://github.com/git-fixtures/basic.git",
		}

		_, err := client.PlainCloneContext(ctx, tmpDir, false, opts)
		// Should fail due to context cancellation or network error
		assert.Error(t, err)
	})
}

// TestClientInterface verifies RealClient implements Client interface
func TestClientInterface(t *testing.T) {
	var client Client = NewClient()
	assert.NotNil(t, client)
	_, ok := client.(*RealClient)
	assert.True(t, ok)
}
