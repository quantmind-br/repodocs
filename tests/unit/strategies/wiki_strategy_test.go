package strategies_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/quantmind-br/repodocs-go/internal/output"
	"github.com/quantmind-br/repodocs-go/internal/strategies"
	"github.com/quantmind-br/repodocs-go/internal/utils"
	"github.com/quantmind-br/repodocs-go/tests/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestWikiStrategy_Execute(t *testing.T) {
	// Setup output directory
	outputDir := t.TempDir()

	// Setup logger and writer
	logger := utils.NewLogger(utils.LoggerOptions{Level: "disabled"})
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: outputDir,
		Force:   true,
	})

	// Dependencies
	deps := &strategies.Dependencies{
		Logger: logger,
		Writer: writer,
	}

	// Create strategy
	strategy := strategies.NewWikiStrategy(deps)

	mockGit := new(mocks.MockGitClient)

	// Setup behavior: When PlainCloneContext is called, create fake wiki files in the provided path
	mockGit.On("PlainCloneContext", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			path := args.Get(1).(string)
			_ = os.WriteFile(filepath.Join(path, "Home.md"), []byte("Welcome to the wiki"), 0644)
			_ = os.WriteFile(filepath.Join(path, "_Sidebar.md"), []byte("* [[Home]]\n* [[Setup]]"), 0644)
			_ = os.WriteFile(filepath.Join(path, "Setup.md"), []byte("Setup instructions"), 0644)
		}).
		Return(&git.Repository{}, nil)

	strategy.SetGitClient(mockGit)

	ctx := context.Background()
	opts := strategies.DefaultOptions()
	opts.Output = outputDir

	url := "https://github.com/owner/repo/wiki"

	err := strategy.Execute(ctx, url, opts)
	require.NoError(t, err)

	assert.FileExists(t, filepath.Join(outputDir, "index.md"))
	assert.FileExists(t, filepath.Join(outputDir, "general", "setup.md"))
}
