package git_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/output"
	"github.com/quantmind-br/repodocs-go/internal/state"
	gitstrat "github.com/quantmind-br/repodocs-go/internal/strategies/git"
	"github.com/quantmind-br/repodocs-go/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDependencies(t *testing.T, tmpDir string) *gitstrat.StrategyDependencies {
	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		Force:   true,
	})

	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	stateManager := state.NewManager(state.ManagerOptions{
		BaseDir:  tmpDir,
		Disabled: false,
	})

	return &gitstrat.StrategyDependencies{
		Writer:       writer,
		Logger:       logger,
		HTTPClient:   httpClient,
		WriteFunc:    nil,
		StateManager: stateManager,
	}
}

func TestNewStrategy_ValidDeps(t *testing.T) {
	tmpDir := t.TempDir()
	deps := setupTestDependencies(t, tmpDir)

	strategy := gitstrat.NewStrategy(deps)

	assert.NotNil(t, strategy)
	assert.Equal(t, "git", strategy.Name())
}

func TestNewStrategy_NilDeps(t *testing.T) {
	strategy := gitstrat.NewStrategy(nil)

	assert.NotNil(t, strategy)
	assert.Equal(t, "git", strategy.Name())
}

func TestNewStrategy_WithoutLogger(t *testing.T) {
	tmpDir := t.TempDir()
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		Force:   true,
	})

	httpClient := &http.Client{}

	deps := &gitstrat.StrategyDependencies{
		Writer:     writer,
		HTTPClient: httpClient,
	}

	strategy := gitstrat.NewStrategy(deps)

	assert.NotNil(t, strategy)
	assert.Equal(t, "git", strategy.Name())
}

func TestNewStrategy_WithDefaultHTTPClient(t *testing.T) {
	tmpDir := t.TempDir()
	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		Force:   true,
	})

	deps := &gitstrat.StrategyDependencies{
		Writer: writer,
		Logger: logger,
	}

	strategy := gitstrat.NewStrategy(deps)

	assert.NotNil(t, strategy)
	assert.Equal(t, "git", strategy.Name())
}

func TestCanHandle_GitHubURL(t *testing.T) {
	tmpDir := t.TempDir()
	deps := setupTestDependencies(t, tmpDir)
	strategy := gitstrat.NewStrategy(deps)

	tests := []string{
		"https://github.com/user/repo",
		"https://github.com/user/repo.git",
		"http://github.com/user/repo",
	}

	for _, url := range tests {
		t.Run(url, func(t *testing.T) {
			assert.True(t, strategy.CanHandle(url), "Should handle GitHub URL: %s", url)
		})
	}
}

func TestCanHandle_GitLabURL(t *testing.T) {
	tmpDir := t.TempDir()
	deps := setupTestDependencies(t, tmpDir)
	strategy := gitstrat.NewStrategy(deps)

	tests := []string{
		"https://gitlab.com/user/repo",
		"https://gitlab.com/user/repo.git",
		"http://gitlab.com/user/repo",
	}

	for _, url := range tests {
		t.Run(url, func(t *testing.T) {
			assert.True(t, strategy.CanHandle(url), "Should handle GitLab URL: %s", url)
		})
	}
}

func TestCanHandle_BitbucketURL(t *testing.T) {
	tmpDir := t.TempDir()
	deps := setupTestDependencies(t, tmpDir)
	strategy := gitstrat.NewStrategy(deps)

	tests := []string{
		"https://bitbucket.org/user/repo",
		"https://bitbucket.org/user/repo.git",
		"http://bitbucket.org/user/repo",
	}

	for _, url := range tests {
		t.Run(url, func(t *testing.T) {
			assert.True(t, strategy.CanHandle(url), "Should handle Bitbucket URL: %s", url)
		})
	}
}

func TestCanHandle_SSH(t *testing.T) {
	tmpDir := t.TempDir()
	deps := setupTestDependencies(t, tmpDir)
	strategy := gitstrat.NewStrategy(deps)

	tests := []string{
		"git@github.com:user/repo.git",
		"git@gitlab.com:user/repo.git",
		"ssh://git@github.com/user/repo.git",
	}

	for _, url := range tests {
		t.Run(url, func(t *testing.T) {
			assert.True(t, strategy.CanHandle(url), "Should handle SSH URL: %s", url)
		})
	}
}

func TestCanHandle_GitURL(t *testing.T) {
	tmpDir := t.TempDir()
	deps := setupTestDependencies(t, tmpDir)
	strategy := gitstrat.NewStrategy(deps)

	tests := []string{
		"git://github.com/user/repo.git",
		"git://gitlab.com/user/repo.git",
	}

	for _, url := range tests {
		t.Run(url, func(t *testing.T) {
			assert.True(t, strategy.CanHandle(url), "Should handle git:// URL: %s", url)
		})
	}
}

func TestCanHandle_NonGitURL(t *testing.T) {
	tmpDir := t.TempDir()
	deps := setupTestDependencies(t, tmpDir)
	strategy := gitstrat.NewStrategy(deps)

	tests := []string{
		"https://example.com",
		"http://example.com/docs",
		"ftp://example.com/file.txt",
		"https://docs.example.com",
	}

	for _, url := range tests {
		t.Run(url, func(t *testing.T) {
			assert.False(t, strategy.CanHandle(url), "Should not handle non-git URL: %s", url)
		})
	}
}

func TestCanHandle_EmptyURL(t *testing.T) {
	tmpDir := t.TempDir()
	deps := setupTestDependencies(t, tmpDir)
	strategy := gitstrat.NewStrategy(deps)

	assert.False(t, strategy.CanHandle(""))
}

func TestCanHandle_InvalidURL(t *testing.T) {
	tmpDir := t.TempDir()
	deps := setupTestDependencies(t, tmpDir)
	strategy := gitstrat.NewStrategy(deps)

	tests := []string{
		"not a url",
		"://invalid.com",
		"http://",
	}

	for _, url := range tests {
		t.Run(url, func(t *testing.T) {
			assert.False(t, strategy.CanHandle(url), "Should not handle invalid URL: %s", url)
		})
	}
}

func TestCanHandle_DocsSubdomain(t *testing.T) {
	tmpDir := t.TempDir()
	deps := setupTestDependencies(t, tmpDir)
	strategy := gitstrat.NewStrategy(deps)

	tests := []string{
		"https://docs.github.com/user/repo",
		"https://pages.github.io/user/repo",
		"https://user.github.io",
	}

	for _, url := range tests {
		t.Run(url, func(t *testing.T) {
			assert.False(t, strategy.CanHandle(url), "Should not handle docs subdomain URL: %s", url)
		})
	}
}

func TestCanHandle_WikiURL(t *testing.T) {
	tmpDir := t.TempDir()
	deps := setupTestDependencies(t, tmpDir)
	strategy := gitstrat.NewStrategy(deps)

	tests := []string{
		"https://github.com/user/repo/wiki",
		"https://github.com/user/repo.wiki.git",
		"https://gitlab.com/user/repo/-/wikis",
	}

	for _, url := range tests {
		t.Run(url, func(t *testing.T) {
			assert.False(t, strategy.CanHandle(url), "Should not handle wiki URL: %s", url)
		})
	}
}

func TestCanHandle_BlobURL(t *testing.T) {
	tmpDir := t.TempDir()
	deps := setupTestDependencies(t, tmpDir)
	strategy := gitstrat.NewStrategy(deps)

	tests := []string{
		"https://github.com/user/repo/blob/main/file.md",
		"https://gitlab.com/user/repo/-/blob/main/file.md",
	}

	for _, url := range tests {
		t.Run(url, func(t *testing.T) {
			assert.False(t, strategy.CanHandle(url), "Should not handle blob URL: %s", url)
		})
	}
}

func TestExecute_InvalidURL(t *testing.T) {
	tmpDir := t.TempDir()
	deps := setupTestDependencies(t, tmpDir)
	strategy := gitstrat.NewStrategy(deps)

	ctx := context.Background()
	opts := gitstrat.ExecuteOptions{
		Output: tmpDir,
	}

	err := strategy.Execute(ctx, "not-a-valid-url", opts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse git URL")
}

func TestExecute_ParseError(t *testing.T) {
	tmpDir := t.TempDir()
	deps := setupTestDependencies(t, tmpDir)
	strategy := gitstrat.NewStrategy(deps)

	ctx := context.Background()
	opts := gitstrat.ExecuteOptions{
		Output: tmpDir,
	}

	err := strategy.Execute(ctx, "https://github.com/invalid", opts)
	assert.Error(t, err)
}

func TestExecute_ContextCancelled(t *testing.T) {
	tmpDir := t.TempDir()
	deps := setupTestDependencies(t, tmpDir)
	strategy := gitstrat.NewStrategy(deps)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	opts := gitstrat.ExecuteOptions{
		Output: tmpDir,
	}

	err := strategy.Execute(ctx, "https://github.com/user/repo", opts)
	assert.Error(t, err)
}

func TestExecute_TempDirError(t *testing.T) {
	oldPerm := os.FileMode(0755)
	tmpDir, err := os.MkdirTemp("", "repodocs-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	err = os.Chmod(tmpDir, 0444)
	require.NoError(t, err)
	defer os.Chmod(tmpDir, oldPerm)

	deps := setupTestDependencies(t, tmpDir)
	strategy := gitstrat.NewStrategy(deps)

	ctx := context.Background()
	opts := gitstrat.ExecuteOptions{
		Output: tmpDir,
	}

	err = strategy.Execute(ctx, "https://github.com/user/repo", opts)
	assert.Error(t, err)
}

func TestTryArchiveDownload_SSHURL(t *testing.T) {
	tmpDir := t.TempDir()
	deps := setupTestDependencies(t, tmpDir)
	strategy := gitstrat.NewStrategy(deps)

	ctx := context.Background()
	_, _, err := strategy.TryArchiveDownload(ctx, "git@github.com:user/repo.git", tmpDir)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "SSH URLs not supported for archive download")
}

func TestTryArchiveDownload_InvalidURL(t *testing.T) {
	tmpDir := t.TempDir()
	deps := setupTestDependencies(t, tmpDir)
	strategy := gitstrat.NewStrategy(deps)

	ctx := context.Background()
	_, _, err := strategy.TryArchiveDownload(ctx, "not-a-url", tmpDir)

	assert.Error(t, err)
}

func TestCloneRepository_InvalidURL(t *testing.T) {
	tmpDir := t.TempDir()
	deps := setupTestDependencies(t, tmpDir)
	strategy := gitstrat.NewStrategy(deps)

	ctx := context.Background()
	_, err := strategy.CloneRepository(ctx, "not-a-url", tmpDir)

	assert.Error(t, err)
}

func TestCloneRepository_ContextCancelled(t *testing.T) {
	tmpDir := t.TempDir()
	deps := setupTestDependencies(t, tmpDir)
	strategy := gitstrat.NewStrategy(deps)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := strategy.CloneRepository(ctx, "https://github.com/user/repo", tmpDir)

	assert.Error(t, err)
}

func TestCloneRepository_DirectoryError(t *testing.T) {
	tmpDir := t.TempDir()
	deps := setupTestDependencies(t, tmpDir)
	strategy := gitstrat.NewStrategy(deps)

	ctx := context.Background()

	filePath := tmpDir + "/file.txt"
	err := os.WriteFile(filePath, []byte("test"), 0644)
	require.NoError(t, err)

	_, err = strategy.CloneRepository(ctx, "https://github.com/user/repo", filePath)

	assert.Error(t, err)
}

func TestStrategy_Name(t *testing.T) {
	strategy := gitstrat.NewStrategy(nil)
	assert.Equal(t, "git", strategy.Name())
}

func TestStrategy_NilDepsCanHandle(t *testing.T) {
	strategy := gitstrat.NewStrategy(nil)

	assert.True(t, strategy.CanHandle("https://github.com/user/repo"))
	assert.False(t, strategy.CanHandle("https://example.com"))
}

func TestNewStrategy_CustomHTTPClient(t *testing.T) {
	tmpDir := t.TempDir()
	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		Force:   true,
	})

	customClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	deps := &gitstrat.StrategyDependencies{
		Writer:     writer,
		Logger:     logger,
		HTTPClient: customClient,
	}

	strategy := gitstrat.NewStrategy(deps)

	assert.NotNil(t, strategy)
	assert.Equal(t, "git", strategy.Name())
}

func TestExecute_WithMockServer(t *testing.T) {
	tmpDir := t.TempDir()
	deps := setupTestDependencies(t, tmpDir)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	deps.HTTPClient = server.Client()
	strategy := gitstrat.NewStrategy(deps)

	ctx := context.Background()
	opts := gitstrat.ExecuteOptions{
		Output: tmpDir,
	}

	err := strategy.Execute(ctx, server.URL, opts)
	assert.Error(t, err)
}

func TestCanHandle_URLCases(t *testing.T) {
	tmpDir := t.TempDir()
	deps := setupTestDependencies(t, tmpDir)
	strategy := gitstrat.NewStrategy(deps)

	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{"GitHub HTTPS", "https://github.com/user/repo", true},
		{"GitHub HTTPS with .git", "https://github.com/user/repo.git", true},
		{"GitHub SSH", "git@github.com:user/repo.git", true},
		{"GitLab HTTPS", "https://gitlab.com/user/repo", true},
		{"GitLab SSH", "git@gitlab.com:user/repo.git", true},
		{"Bitbucket HTTPS", "https://bitbucket.org/user/repo", true},
		{"Bitbucket with .git", "https://bitbucket.org/user/repo.git", true},
		{"git:// protocol", "git://github.com/user/repo.git", true},
		{"Not git URL", "https://example.com", false},
		{"Docs subdomain", "https://docs.github.com", false},
		{"Pages", "https://pages.github.io", false},
		{"Wiki URL", "https://github.com/user/repo/wiki", false},
		{"Wiki with .git", "https://github.com/user/repo.wiki.git", false},
		{"Blob URL", "https://github.com/user/repo/blob/main/file", false},
		{"Empty URL", "", false},
		{"Invalid", "not-a-url", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := strategy.CanHandle(tc.url)
			assert.Equal(t, tc.expected, result, "URL: %s", tc.url)
		})
	}
}

func TestExecuteOptions_DefaultValues(t *testing.T) {
	opts := gitstrat.ExecuteOptions{
		Output:      "/tmp/output",
		Concurrency: 5,
		Limit:       10,
		DryRun:      true,
		FilterURL:   "docs/",
	}

	assert.Equal(t, "/tmp/output", opts.Output)
	assert.Equal(t, 5, opts.Concurrency)
	assert.Equal(t, 10, opts.Limit)
	assert.True(t, opts.DryRun)
	assert.Equal(t, "docs/", opts.FilterURL)
}

func TestExecuteOptions_ZeroDefaults(t *testing.T) {
	opts := gitstrat.ExecuteOptions{}

	assert.Equal(t, "", opts.Output)
	assert.Equal(t, 0, opts.Concurrency)
	assert.Equal(t, 0, opts.Limit)
	assert.False(t, opts.DryRun)
	assert.Equal(t, "", opts.FilterURL)
}

func TestExecute_WithWriteFunc(t *testing.T) {
	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})

	writeCalled := false
	writeFunc := func(ctx context.Context, doc *domain.Document) error {
		writeCalled = true
		return nil
	}

	deps := &gitstrat.StrategyDependencies{
		Logger:    logger,
		WriteFunc: writeFunc,
	}

	strategy := gitstrat.NewStrategy(deps)
	assert.NotNil(t, strategy)
	assert.False(t, writeCalled)
}

func BenchmarkStrategy_CanHandle(b *testing.B) {
	tmpDir := b.TempDir()
	deps := setupTestDependencies(&testing.T{}, tmpDir)
	strategy := gitstrat.NewStrategy(deps)

	urls := []string{
		"https://github.com/user/repo",
		"https://gitlab.com/user/repo",
		"https://example.com",
		"git@github.com:user/repo.git",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, url := range urls {
			strategy.CanHandle(url)
		}
	}
}

func BenchmarkStrategy_NewStrategy(b *testing.B) {
	tmpDir := b.TempDir()
	deps := setupTestDependencies(&testing.T{}, tmpDir)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gitstrat.NewStrategy(deps)
	}
}

func TestMain(m *testing.M) {
	fmt.Println("Running git strategy tests...")
	os.Exit(m.Run())
}

func TestNewArchiveFetcher_ValidOptions(t *testing.T) {
	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})
	httpClient := &http.Client{Timeout: 10 * time.Second}

	fetcher := gitstrat.NewArchiveFetcher(gitstrat.ArchiveFetcherOptions{
		HTTPClient: httpClient,
		Logger:     logger,
	})

	assert.NotNil(t, fetcher)
	assert.Equal(t, "archive", fetcher.Name())
}

func TestNewArchiveFetcher_NilOptions(t *testing.T) {
	fetcher := gitstrat.NewArchiveFetcher(gitstrat.ArchiveFetcherOptions{})

	assert.NotNil(t, fetcher)
	assert.Equal(t, "archive", fetcher.Name())
}

func TestArchiveFetcher_BuildArchiveURL_GitHub(t *testing.T) {
	fetcher := gitstrat.NewArchiveFetcher(gitstrat.ArchiveFetcherOptions{})

	info := &gitstrat.RepoInfo{
		Platform: gitstrat.PlatformGitHub,
		Owner:    "user",
		Repo:     "repo",
	}

	url := fetcher.BuildArchiveURL(info, "main")
	assert.Equal(t, "https://github.com/user/repo/archive/refs/heads/main.tar.gz", url)
}

func TestArchiveFetcher_BuildArchiveURL_GitLab(t *testing.T) {
	fetcher := gitstrat.NewArchiveFetcher(gitstrat.ArchiveFetcherOptions{})

	info := &gitstrat.RepoInfo{
		Platform: gitstrat.PlatformGitLab,
		Owner:    "user",
		Repo:     "repo",
	}

	url := fetcher.BuildArchiveURL(info, "main")
	assert.Equal(t, "https://gitlab.com/user/repo/-/archive/main/repo-main.tar.gz", url)
}

func TestArchiveFetcher_BuildArchiveURL_Bitbucket(t *testing.T) {
	fetcher := gitstrat.NewArchiveFetcher(gitstrat.ArchiveFetcherOptions{})

	info := &gitstrat.RepoInfo{
		Platform: gitstrat.PlatformBitbucket,
		Owner:    "user",
		Repo:     "repo",
	}

	url := fetcher.BuildArchiveURL(info, "main")
	assert.Equal(t, "https://bitbucket.org/user/repo/get/main.tar.gz", url)
}

func TestArchiveFetcher_BuildArchiveURL_Generic(t *testing.T) {
	fetcher := gitstrat.NewArchiveFetcher(gitstrat.ArchiveFetcherOptions{})

	info := &gitstrat.RepoInfo{
		Platform: gitstrat.PlatformGeneric,
		Owner:    "user",
		Repo:     "repo",
	}

	url := fetcher.BuildArchiveURL(info, "main")
	assert.Equal(t, "https://github.com/user/repo/archive/refs/heads/main.tar.gz", url)
}

func TestArchiveFetcher_DownloadAndExtract_404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	fetcher := gitstrat.NewArchiveFetcher(gitstrat.ArchiveFetcherOptions{
		HTTPClient: server.Client(),
	})

	tmpDir := t.TempDir()
	err := fetcher.DownloadAndExtract(context.Background(), server.URL+"/archive.tar.gz", tmpDir)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "archive not found")
}

func TestArchiveFetcher_DownloadAndExtract_401(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	fetcher := gitstrat.NewArchiveFetcher(gitstrat.ArchiveFetcherOptions{
		HTTPClient: server.Client(),
	})

	tmpDir := t.TempDir()
	err := fetcher.DownloadAndExtract(context.Background(), server.URL+"/archive.tar.gz", tmpDir)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "authentication required")
}

func TestArchiveFetcher_DownloadAndExtract_500(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	fetcher := gitstrat.NewArchiveFetcher(gitstrat.ArchiveFetcherOptions{
		HTTPClient: server.Client(),
	})

	tmpDir := t.TempDir()
	err := fetcher.DownloadAndExtract(context.Background(), server.URL+"/archive.tar.gz", tmpDir)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "download failed with status")
}

func TestArchiveFetcher_DownloadAndExtract_InvalidTarGz(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not a valid tar.gz file"))
	}))
	defer server.Close()

	fetcher := gitstrat.NewArchiveFetcher(gitstrat.ArchiveFetcherOptions{
		HTTPClient: server.Client(),
	})

	tmpDir := t.TempDir()
	err := fetcher.DownloadAndExtract(context.Background(), server.URL+"/archive.tar.gz", tmpDir)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "gzip reader failed")
}

func TestArchiveFetcher_DownloadAndExtract_ContextCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	fetcher := gitstrat.NewArchiveFetcher(gitstrat.ArchiveFetcherOptions{
		HTTPClient: server.Client(),
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	tmpDir := t.TempDir()
	err := fetcher.DownloadAndExtract(ctx, server.URL+"/archive.tar.gz", tmpDir)

	assert.Error(t, err)
}

func TestArchiveFetcher_ExtractTarGz_ValidArchive(t *testing.T) {
	fetcher := gitstrat.NewArchiveFetcher(gitstrat.ArchiveFetcherOptions{})

	tarGz := createTestTarGz(t, map[string]string{
		"repo-main/README.md":     "# Test",
		"repo-main/docs/guide.md": "# Guide",
	})

	tmpDir := t.TempDir()
	err := fetcher.ExtractTarGz(tarGz, tmpDir)
	require.NoError(t, err)

	readmeContent, err := os.ReadFile(filepath.Join(tmpDir, "README.md"))
	require.NoError(t, err)
	assert.Equal(t, "# Test", string(readmeContent))

	guideContent, err := os.ReadFile(filepath.Join(tmpDir, "docs", "guide.md"))
	require.NoError(t, err)
	assert.Equal(t, "# Guide", string(guideContent))
}

func TestArchiveFetcher_DownloadAndExtract_Success(t *testing.T) {
	tarGz := createTestTarGz(t, map[string]string{
		"repo-main/README.md": "# Test",
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(tarGz.Bytes())
	}))
	defer server.Close()

	fetcher := gitstrat.NewArchiveFetcher(gitstrat.ArchiveFetcherOptions{
		HTTPClient: server.Client(),
	})

	tmpDir := t.TempDir()
	err := fetcher.DownloadAndExtract(context.Background(), server.URL+"/archive.tar.gz", tmpDir)
	require.NoError(t, err)

	readmeContent, err := os.ReadFile(filepath.Join(tmpDir, "README.md"))
	require.NoError(t, err)
	assert.Equal(t, "# Test", string(readmeContent))
}

func createTestTarGz(t *testing.T, files map[string]string) *bytes.Buffer {
	t.Helper()

	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	for name, content := range files {
		hdr := &tar.Header{
			Name: name,
			Mode: 0644,
			Size: int64(len(content)),
		}
		err := tw.WriteHeader(hdr)
		require.NoError(t, err)
		_, err = tw.Write([]byte(content))
		require.NoError(t, err)
	}

	err := tw.Close()
	require.NoError(t, err)
	err = gzw.Close()
	require.NoError(t, err)

	return &buf
}

func TestNewCloneFetcher_ValidOptions(t *testing.T) {
	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})

	fetcher := gitstrat.NewCloneFetcher(gitstrat.CloneFetcherOptions{
		Logger: logger,
	})

	assert.NotNil(t, fetcher)
	assert.Equal(t, "clone", fetcher.Name())
}

func TestNewCloneFetcher_NilOptions(t *testing.T) {
	fetcher := gitstrat.NewCloneFetcher(gitstrat.CloneFetcherOptions{})

	assert.NotNil(t, fetcher)
	assert.Equal(t, "clone", fetcher.Name())
}

func TestNewProcessor_ValidOptions(t *testing.T) {
	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})

	processor := gitstrat.NewProcessor(gitstrat.ProcessorOptions{
		Logger: logger,
	})

	assert.NotNil(t, processor)
}

func TestNewProcessor_NilOptions(t *testing.T) {
	processor := gitstrat.NewProcessor(gitstrat.ProcessorOptions{})
	assert.NotNil(t, processor)
}

func TestProcessor_FindDocumentationFiles_EmptyDir(t *testing.T) {
	processor := gitstrat.NewProcessor(gitstrat.ProcessorOptions{})

	tmpDir := t.TempDir()

	files, err := processor.FindDocumentationFiles(tmpDir, "")
	require.NoError(t, err)
	assert.Empty(t, files)
}

func TestProcessor_FindDocumentationFiles_MarkdownOnly(t *testing.T) {
	processor := gitstrat.NewProcessor(gitstrat.ProcessorOptions{})

	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("# Test"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "guide.mdx"), []byte("# Guide"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "style.css"), []byte("body{}"), 0644))

	files, err := processor.FindDocumentationFiles(tmpDir, "")
	require.NoError(t, err)
	assert.Len(t, files, 2)
}

func TestProcessor_FindDocumentationFiles_WithSubdirectories(t *testing.T) {
	processor := gitstrat.NewProcessor(gitstrat.ProcessorOptions{})

	tmpDir := t.TempDir()
	docsDir := filepath.Join(tmpDir, "docs")
	require.NoError(t, os.MkdirAll(docsDir, 0755))

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("# Root"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(docsDir, "guide.md"), []byte("# Guide"), 0644))

	files, err := processor.FindDocumentationFiles(tmpDir, "")
	require.NoError(t, err)
	assert.Len(t, files, 2)
}

func TestProcessor_FindDocumentationFiles_ExcludedDirs(t *testing.T) {
	processor := gitstrat.NewProcessor(gitstrat.ProcessorOptions{})

	tmpDir := t.TempDir()
	nodeModules := filepath.Join(tmpDir, "node_modules")
	gitDir := filepath.Join(tmpDir, ".git")
	require.NoError(t, os.MkdirAll(nodeModules, 0755))
	require.NoError(t, os.MkdirAll(gitDir, 0755))

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("# Root"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(nodeModules, "package.md"), []byte("# Package"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(gitDir, "hooks.md"), []byte("# Hooks"), 0644))

	files, err := processor.FindDocumentationFiles(tmpDir, "")
	require.NoError(t, err)
	assert.Len(t, files, 1)
}

func TestProcessor_FindDocumentationFiles_WithFilterPath(t *testing.T) {
	processor := gitstrat.NewProcessor(gitstrat.ProcessorOptions{})

	tmpDir := t.TempDir()
	docsDir := filepath.Join(tmpDir, "docs")
	apiDir := filepath.Join(tmpDir, "api")
	require.NoError(t, os.MkdirAll(docsDir, 0755))
	require.NoError(t, os.MkdirAll(apiDir, 0755))

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("# Root"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(docsDir, "guide.md"), []byte("# Guide"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(apiDir, "reference.md"), []byte("# API"), 0644))

	files, err := processor.FindDocumentationFiles(tmpDir, "docs")
	require.NoError(t, err)
	assert.Len(t, files, 1)
}

func TestProcessor_FindDocumentationFiles_FilterPathNotExists(t *testing.T) {
	processor := gitstrat.NewProcessor(gitstrat.ProcessorOptions{})

	tmpDir := t.TempDir()

	_, err := processor.FindDocumentationFiles(tmpDir, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "filter path does not exist")
}

func TestProcessor_FindDocumentationFiles_FilterPathNotDir(t *testing.T) {
	processor := gitstrat.NewProcessor(gitstrat.ProcessorOptions{})

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "file.md")
	require.NoError(t, os.WriteFile(filePath, []byte("# Test"), 0644))

	_, err := processor.FindDocumentationFiles(tmpDir, "file.md")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "filter path is not a directory")
}

func TestExtractTitleFromPath_SimpleFile(t *testing.T) {
	title := gitstrat.ExtractTitleFromPath("readme.md")
	assert.Equal(t, "Readme", title)
}

func TestExtractTitleFromPath_WithDashes(t *testing.T) {
	title := gitstrat.ExtractTitleFromPath("getting-started.md")
	assert.Equal(t, "Getting started", title)
}

func TestExtractTitleFromPath_WithUnderscores(t *testing.T) {
	title := gitstrat.ExtractTitleFromPath("api_reference.md")
	assert.Equal(t, "Api reference", title)
}

func TestExtractTitleFromPath_NestedPath(t *testing.T) {
	title := gitstrat.ExtractTitleFromPath("docs/guides/installation.md")
	assert.Equal(t, "Installation", title)
}

func TestExtractTitleFromPath_EmptyString(t *testing.T) {
	title := gitstrat.ExtractTitleFromPath("")
	assert.Equal(t, "", title)
}

func TestNewParser(t *testing.T) {
	parser := gitstrat.NewParser()
	assert.NotNil(t, parser)
}

func TestParser_ParseURL_GitHub(t *testing.T) {
	parser := gitstrat.NewParser()

	tests := []struct {
		url      string
		owner    string
		repo     string
		platform gitstrat.Platform
	}{
		{"https://github.com/user/repo", "user", "repo", gitstrat.PlatformGitHub},
		{"https://github.com/user/repo.git", "user", "repo", gitstrat.PlatformGitHub},
		{"git@github.com:user/repo.git", "user", "repo", gitstrat.PlatformGitHub},
	}

	for _, tc := range tests {
		t.Run(tc.url, func(t *testing.T) {
			info, err := parser.ParseURL(tc.url)
			require.NoError(t, err)
			assert.Equal(t, tc.owner, info.Owner)
			assert.Equal(t, tc.repo, info.Repo)
			assert.Equal(t, tc.platform, info.Platform)
		})
	}
}

func TestParser_ParseURL_GitLab(t *testing.T) {
	parser := gitstrat.NewParser()

	tests := []struct {
		url      string
		owner    string
		repo     string
		platform gitstrat.Platform
	}{
		{"https://gitlab.com/user/repo", "user", "repo", gitstrat.PlatformGitLab},
		{"https://gitlab.com/user/repo.git", "user", "repo", gitstrat.PlatformGitLab},
		{"git@gitlab.com:user/repo.git", "user", "repo", gitstrat.PlatformGitLab},
	}

	for _, tc := range tests {
		t.Run(tc.url, func(t *testing.T) {
			info, err := parser.ParseURL(tc.url)
			require.NoError(t, err)
			assert.Equal(t, tc.owner, info.Owner)
			assert.Equal(t, tc.repo, info.Repo)
			assert.Equal(t, tc.platform, info.Platform)
		})
	}
}

func TestParser_ParseURL_Bitbucket(t *testing.T) {
	parser := gitstrat.NewParser()

	tests := []struct {
		url      string
		owner    string
		repo     string
		platform gitstrat.Platform
	}{
		{"https://bitbucket.org/user/repo", "user", "repo", gitstrat.PlatformBitbucket},
		{"https://bitbucket.org/user/repo.git", "user", "repo", gitstrat.PlatformBitbucket},
		{"git@bitbucket.org:user/repo.git", "user", "repo", gitstrat.PlatformBitbucket},
	}

	for _, tc := range tests {
		t.Run(tc.url, func(t *testing.T) {
			info, err := parser.ParseURL(tc.url)
			require.NoError(t, err)
			assert.Equal(t, tc.owner, info.Owner)
			assert.Equal(t, tc.repo, info.Repo)
			assert.Equal(t, tc.platform, info.Platform)
		})
	}
}

func TestParser_ParseURL_Invalid(t *testing.T) {
	parser := gitstrat.NewParser()

	tests := []string{
		"https://example.com/user/repo",
		"not-a-url",
		"",
	}

	for _, url := range tests {
		t.Run(url, func(t *testing.T) {
			_, err := parser.ParseURL(url)
			assert.Error(t, err)
		})
	}
}

func TestParser_ParseURLWithPath_GitHub(t *testing.T) {
	parser := gitstrat.NewParser()

	tests := []struct {
		name     string
		url      string
		repoURL  string
		branch   string
		subPath  string
		platform gitstrat.Platform
	}{
		{
			name:     "simple repo",
			url:      "https://github.com/user/repo",
			repoURL:  "https://github.com/user/repo",
			branch:   "",
			subPath:  "",
			platform: gitstrat.PlatformGitHub,
		},
		{
			name:     "repo with .git",
			url:      "https://github.com/user/repo.git",
			repoURL:  "https://github.com/user/repo",
			branch:   "",
			subPath:  "",
			platform: gitstrat.PlatformGitHub,
		},
		{
			name:     "repo with tree/branch",
			url:      "https://github.com/user/repo/tree/main",
			repoURL:  "https://github.com/user/repo",
			branch:   "main",
			subPath:  "",
			platform: gitstrat.PlatformGitHub,
		},
		{
			name:     "repo with tree/branch/path",
			url:      "https://github.com/user/repo/tree/main/docs",
			repoURL:  "https://github.com/user/repo",
			branch:   "main",
			subPath:  "docs",
			platform: gitstrat.PlatformGitHub,
		},
		{
			name:     "repo with tree/branch/nested/path",
			url:      "https://github.com/user/repo/tree/develop/docs/api",
			repoURL:  "https://github.com/user/repo",
			branch:   "develop",
			subPath:  "docs/api",
			platform: gitstrat.PlatformGitHub,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			info, err := parser.ParseURLWithPath(tc.url)
			require.NoError(t, err)
			assert.Equal(t, tc.repoURL, info.RepoURL)
			assert.Equal(t, tc.branch, info.Branch)
			assert.Equal(t, tc.subPath, info.SubPath)
			assert.Equal(t, tc.platform, info.Platform)
		})
	}
}

func TestParser_ParseURLWithPath_GitLab(t *testing.T) {
	parser := gitstrat.NewParser()

	tests := []struct {
		name     string
		url      string
		repoURL  string
		branch   string
		subPath  string
		platform gitstrat.Platform
	}{
		{
			name:     "simple repo",
			url:      "https://gitlab.com/user/repo",
			repoURL:  "https://gitlab.com/user/repo",
			branch:   "",
			subPath:  "",
			platform: gitstrat.PlatformGitLab,
		},
		{
			name:     "repo with tree",
			url:      "https://gitlab.com/user/repo/-/tree/main",
			repoURL:  "https://gitlab.com/user/repo",
			branch:   "main",
			subPath:  "",
			platform: gitstrat.PlatformGitLab,
		},
		{
			name:     "repo with tree/path",
			url:      "https://gitlab.com/user/repo/-/tree/main/docs",
			repoURL:  "https://gitlab.com/user/repo",
			branch:   "main",
			subPath:  "docs",
			platform: gitstrat.PlatformGitLab,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			info, err := parser.ParseURLWithPath(tc.url)
			require.NoError(t, err)
			assert.Equal(t, tc.repoURL, info.RepoURL)
			assert.Equal(t, tc.branch, info.Branch)
			assert.Equal(t, tc.subPath, info.SubPath)
			assert.Equal(t, tc.platform, info.Platform)
		})
	}
}

func TestParser_ParseURLWithPath_Bitbucket(t *testing.T) {
	parser := gitstrat.NewParser()

	tests := []struct {
		name     string
		url      string
		repoURL  string
		branch   string
		subPath  string
		platform gitstrat.Platform
	}{
		{
			name:     "simple repo",
			url:      "https://bitbucket.org/user/repo",
			repoURL:  "https://bitbucket.org/user/repo",
			branch:   "",
			subPath:  "",
			platform: gitstrat.PlatformBitbucket,
		},
		{
			name:     "repo with src",
			url:      "https://bitbucket.org/user/repo/src/main",
			repoURL:  "https://bitbucket.org/user/repo",
			branch:   "main",
			subPath:  "",
			platform: gitstrat.PlatformBitbucket,
		},
		{
			name:     "repo with src/path",
			url:      "https://bitbucket.org/user/repo/src/main/docs",
			repoURL:  "https://bitbucket.org/user/repo",
			branch:   "main",
			subPath:  "docs",
			platform: gitstrat.PlatformBitbucket,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			info, err := parser.ParseURLWithPath(tc.url)
			require.NoError(t, err)
			assert.Equal(t, tc.repoURL, info.RepoURL)
			assert.Equal(t, tc.branch, info.Branch)
			assert.Equal(t, tc.subPath, info.SubPath)
			assert.Equal(t, tc.platform, info.Platform)
		})
	}
}

func TestParser_ParseURLWithPath_Generic(t *testing.T) {
	parser := gitstrat.NewParser()

	info, err := parser.ParseURLWithPath("https://example.com/user/repo")
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/user/repo", info.RepoURL)
	assert.Equal(t, gitstrat.PlatformGeneric, info.Platform)
}

func TestParser_ParseURLWithPath_Invalid(t *testing.T) {
	parser := gitstrat.NewParser()

	tests := []string{
		"not-a-url",
		"ftp://example.com/repo",
	}

	for _, url := range tests {
		t.Run(url, func(t *testing.T) {
			_, err := parser.ParseURLWithPath(url)
			assert.Error(t, err)
		})
	}
}

func TestNormalizeFilterPath_Simple(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"docs", "docs"},
		{"docs/", "docs"},
		{"/docs", "docs"},
		{"/docs/", "docs"},
		{"docs/api", "docs/api"},
		{"docs\\api", "docs/api"},
		{"", ""},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := gitstrat.NormalizeFilterPath(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestNormalizeFilterPath_URLDecoding(t *testing.T) {
	result := gitstrat.NormalizeFilterPath("docs%2Fapi")
	assert.Equal(t, "docs/api", result)
}

func TestNormalizeFilterPath_FromTreeURL(t *testing.T) {
	result := gitstrat.NormalizeFilterPath("https://github.com/user/repo/tree/main/docs/api")
	assert.Equal(t, "docs/api", result)
}

func TestExtractPathFromTreeURL_GitHub(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"https://github.com/user/repo/tree/main/docs", "docs"},
		{"https://github.com/user/repo/tree/main/docs/api", "docs/api"},
		{"https://github.com/user/repo/blob/main/README.md", "README.md"},
	}

	for _, tc := range tests {
		t.Run(tc.url, func(t *testing.T) {
			result := gitstrat.ExtractPathFromTreeURL(tc.url)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestExtractPathFromTreeURL_GitLab(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"https://gitlab.com/user/repo/-/tree/main/docs", "docs"},
		{"https://gitlab.com/user/repo/-/blob/main/README.md", "README.md"},
	}

	for _, tc := range tests {
		t.Run(tc.url, func(t *testing.T) {
			result := gitstrat.ExtractPathFromTreeURL(tc.url)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestExtractPathFromTreeURL_Bitbucket(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"https://bitbucket.org/user/repo/src/main/docs", "docs"},
	}

	for _, tc := range tests {
		t.Run(tc.url, func(t *testing.T) {
			result := gitstrat.ExtractPathFromTreeURL(tc.url)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestExtractPathFromTreeURL_NoMatch(t *testing.T) {
	result := gitstrat.ExtractPathFromTreeURL("https://example.com/path")
	assert.Equal(t, "https://example.com/path", result)
}

func TestProcessor_ProcessFile_ValidMarkdown(t *testing.T) {
	processor := gitstrat.NewProcessor(gitstrat.ProcessorOptions{})

	tmpDir := t.TempDir()
	mdPath := filepath.Join(tmpDir, "test.md")
	require.NoError(t, os.WriteFile(mdPath, []byte("# Test Content\n\nThis is a test."), 0644))

	var capturedDoc *domain.Document
	writeFunc := func(ctx context.Context, doc *domain.Document) error {
		capturedDoc = doc
		return nil
	}

	opts := gitstrat.ProcessOptions{
		RepoURL:   "https://github.com/user/repo",
		Branch:    "main",
		WriteFunc: writeFunc,
	}

	err := processor.ProcessFile(context.Background(), mdPath, tmpDir, opts)
	require.NoError(t, err)
	require.NotNil(t, capturedDoc)
	assert.Contains(t, capturedDoc.Content, "# Test Content")
	assert.Equal(t, "Test", capturedDoc.Title)
	assert.Equal(t, "git", capturedDoc.SourceStrategy)
}

func TestProcessor_ProcessFile_NonMarkdown(t *testing.T) {
	processor := gitstrat.NewProcessor(gitstrat.ProcessorOptions{})

	tmpDir := t.TempDir()
	goPath := filepath.Join(tmpDir, "main.go")
	require.NoError(t, os.WriteFile(goPath, []byte("package main\n\nfunc main() {}"), 0644))

	var capturedDoc *domain.Document
	writeFunc := func(ctx context.Context, doc *domain.Document) error {
		capturedDoc = doc
		return nil
	}

	opts := gitstrat.ProcessOptions{
		RepoURL:   "https://github.com/user/repo",
		Branch:    "main",
		WriteFunc: writeFunc,
	}

	err := processor.ProcessFile(context.Background(), goPath, tmpDir, opts)
	require.NoError(t, err)
	require.NotNil(t, capturedDoc)
	assert.Contains(t, capturedDoc.Content, "```")
}

func TestProcessor_ProcessFile_DryRun(t *testing.T) {
	processor := gitstrat.NewProcessor(gitstrat.ProcessorOptions{})

	tmpDir := t.TempDir()
	mdPath := filepath.Join(tmpDir, "test.md")
	require.NoError(t, os.WriteFile(mdPath, []byte("# Test"), 0644))

	writeCalled := false
	writeFunc := func(ctx context.Context, doc *domain.Document) error {
		writeCalled = true
		return nil
	}

	opts := gitstrat.ProcessOptions{
		RepoURL:   "https://github.com/user/repo",
		Branch:    "main",
		WriteFunc: writeFunc,
		DryRun:    true,
	}

	err := processor.ProcessFile(context.Background(), mdPath, tmpDir, opts)
	require.NoError(t, err)
	assert.False(t, writeCalled)
}

func TestProcessor_ProcessFile_NilWriteFunc(t *testing.T) {
	processor := gitstrat.NewProcessor(gitstrat.ProcessorOptions{})

	tmpDir := t.TempDir()
	mdPath := filepath.Join(tmpDir, "test.md")
	require.NoError(t, os.WriteFile(mdPath, []byte("# Test"), 0644))

	opts := gitstrat.ProcessOptions{
		RepoURL:   "https://github.com/user/repo",
		Branch:    "main",
		WriteFunc: nil,
	}

	err := processor.ProcessFile(context.Background(), mdPath, tmpDir, opts)
	require.NoError(t, err)
}

func TestProcessor_ProcessFile_ReadError(t *testing.T) {
	processor := gitstrat.NewProcessor(gitstrat.ProcessorOptions{})

	tmpDir := t.TempDir()

	opts := gitstrat.ProcessOptions{
		RepoURL: "https://github.com/user/repo",
		Branch:  "main",
	}

	err := processor.ProcessFile(context.Background(), filepath.Join(tmpDir, "nonexistent.md"), tmpDir, opts)
	assert.Error(t, err)
}

func TestProcessor_ProcessFile_LargeFile(t *testing.T) {
	processor := gitstrat.NewProcessor(gitstrat.ProcessorOptions{})

	tmpDir := t.TempDir()
	mdPath := filepath.Join(tmpDir, "large.md")
	largeContent := make([]byte, 11*1024*1024)
	require.NoError(t, os.WriteFile(mdPath, largeContent, 0644))

	writeCalled := false
	writeFunc := func(ctx context.Context, doc *domain.Document) error {
		writeCalled = true
		return nil
	}

	opts := gitstrat.ProcessOptions{
		RepoURL:   "https://github.com/user/repo",
		Branch:    "main",
		WriteFunc: writeFunc,
	}

	err := processor.ProcessFile(context.Background(), mdPath, tmpDir, opts)
	require.NoError(t, err)
	assert.False(t, writeCalled)
}

func TestProcessor_ProcessFile_WithStateManager(t *testing.T) {
	processor := gitstrat.NewProcessor(gitstrat.ProcessorOptions{})

	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, "state")
	contentDir := filepath.Join(tmpDir, "content")
	require.NoError(t, os.MkdirAll(stateDir, 0755))
	require.NoError(t, os.MkdirAll(contentDir, 0755))

	mdPath := filepath.Join(contentDir, "test.md")
	require.NoError(t, os.WriteFile(mdPath, []byte("# Test"), 0644))

	stateManager := state.NewManager(state.ManagerOptions{
		BaseDir:  stateDir,
		Disabled: false,
	})

	var writeCalls int
	writeFunc := func(ctx context.Context, doc *domain.Document) error {
		writeCalls++
		stateManager.Update(doc.URL, state.PageState{
			ContentHash: doc.ContentHash,
			FetchedAt:   doc.FetchedAt,
			FilePath:    doc.RelativePath,
		})
		return nil
	}

	opts := gitstrat.ProcessOptions{
		RepoURL:      "https://github.com/user/repo",
		Branch:       "main",
		WriteFunc:    writeFunc,
		StateManager: stateManager,
	}

	err := processor.ProcessFile(context.Background(), mdPath, contentDir, opts)
	require.NoError(t, err)
	assert.Equal(t, 1, writeCalls)

	err = processor.ProcessFile(context.Background(), mdPath, contentDir, opts)
	require.NoError(t, err)
	assert.Equal(t, 1, writeCalls)
}

func TestProcessor_ProcessFiles_Empty(t *testing.T) {
	processor := gitstrat.NewProcessor(gitstrat.ProcessorOptions{})

	tmpDir := t.TempDir()

	opts := gitstrat.ProcessOptions{
		RepoURL:     "https://github.com/user/repo",
		Branch:      "main",
		Concurrency: 1,
	}

	err := processor.ProcessFiles(context.Background(), []string{}, tmpDir, opts)
	require.NoError(t, err)
}

func TestProcessor_ProcessFiles_Multiple(t *testing.T) {
	processor := gitstrat.NewProcessor(gitstrat.ProcessorOptions{})

	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "file1.md")
	file2 := filepath.Join(tmpDir, "file2.md")
	require.NoError(t, os.WriteFile(file1, []byte("# File 1"), 0644))
	require.NoError(t, os.WriteFile(file2, []byte("# File 2"), 0644))

	var processedFiles []string
	var mu sync.Mutex
	writeFunc := func(ctx context.Context, doc *domain.Document) error {
		mu.Lock()
		processedFiles = append(processedFiles, doc.Title)
		mu.Unlock()
		return nil
	}

	opts := gitstrat.ProcessOptions{
		RepoURL:     "https://github.com/user/repo",
		Branch:      "main",
		WriteFunc:   writeFunc,
		Concurrency: 2,
	}

	err := processor.ProcessFiles(context.Background(), []string{file1, file2}, tmpDir, opts)
	require.NoError(t, err)
	assert.Len(t, processedFiles, 2)
}

func BenchmarkParser_ParseURL(b *testing.B) {
	parser := gitstrat.NewParser()
	urls := []string{
		"https://github.com/user/repo",
		"https://gitlab.com/user/repo",
		"https://bitbucket.org/user/repo",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, url := range urls {
			parser.ParseURL(url)
		}
	}
}

func BenchmarkParser_ParseURLWithPath(b *testing.B) {
	parser := gitstrat.NewParser()
	urls := []string{
		"https://github.com/user/repo/tree/main/docs",
		"https://gitlab.com/user/repo/-/tree/main/docs",
		"https://bitbucket.org/user/repo/src/main/docs",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, url := range urls {
			parser.ParseURLWithPath(url)
		}
	}
}

func BenchmarkNormalizeFilterPath(b *testing.B) {
	paths := []string{
		"docs",
		"/docs/api/",
		"https://github.com/user/repo/tree/main/docs",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, path := range paths {
			gitstrat.NormalizeFilterPath(path)
		}
	}
}

func BenchmarkExtractTitleFromPath(b *testing.B) {
	paths := []string{
		"readme.md",
		"getting-started.md",
		"docs/guides/installation.md",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, path := range paths {
			gitstrat.ExtractTitleFromPath(path)
		}
	}
}

func TestArchiveFetcher_ExtractTarGz_DirectoryEntry(t *testing.T) {
	fetcher := gitstrat.NewArchiveFetcher(gitstrat.ArchiveFetcherOptions{})

	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	dirHdr := &tar.Header{
		Name:     "repo-main/subdir/",
		Typeflag: tar.TypeDir,
		Mode:     0755,
	}
	require.NoError(t, tw.WriteHeader(dirHdr))

	fileHdr := &tar.Header{
		Name: "repo-main/subdir/file.md",
		Mode: 0644,
		Size: 6,
	}
	require.NoError(t, tw.WriteHeader(fileHdr))
	_, err := tw.Write([]byte("# Test"))
	require.NoError(t, err)

	require.NoError(t, tw.Close())
	require.NoError(t, gzw.Close())

	tmpDir := t.TempDir()
	err = fetcher.ExtractTarGz(&buf, tmpDir)
	require.NoError(t, err)

	info, err := os.Stat(filepath.Join(tmpDir, "subdir"))
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	content, err := os.ReadFile(filepath.Join(tmpDir, "subdir", "file.md"))
	require.NoError(t, err)
	assert.Equal(t, "# Test", string(content))
}

func TestArchiveFetcher_ExtractTarGz_SkipRootOnly(t *testing.T) {
	fetcher := gitstrat.NewArchiveFetcher(gitstrat.ArchiveFetcherOptions{})

	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	rootHdr := &tar.Header{
		Name:     "repo-main/",
		Typeflag: tar.TypeDir,
		Mode:     0755,
	}
	require.NoError(t, tw.WriteHeader(rootHdr))

	require.NoError(t, tw.Close())
	require.NoError(t, gzw.Close())

	tmpDir := t.TempDir()
	err := fetcher.ExtractTarGz(&buf, tmpDir)
	require.NoError(t, err)
}

func TestArchiveFetcher_ExtractTarGz_PathTraversal(t *testing.T) {
	fetcher := gitstrat.NewArchiveFetcher(gitstrat.ArchiveFetcherOptions{})

	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	badHdr := &tar.Header{
		Name: "repo-main/../../../etc/passwd",
		Mode: 0644,
		Size: 4,
	}
	require.NoError(t, tw.WriteHeader(badHdr))
	_, err := tw.Write([]byte("evil"))
	require.NoError(t, err)

	require.NoError(t, tw.Close())
	require.NoError(t, gzw.Close())

	tmpDir := t.TempDir()
	err = fetcher.ExtractTarGz(&buf, tmpDir)
	require.NoError(t, err)
}

func TestArchiveFetcher_Fetch_WithLogger(t *testing.T) {
	logger := utils.NewLogger(utils.LoggerOptions{Level: "debug"})
	tarGz := createTestTarGz(t, map[string]string{
		"repo-main/README.md": "# Test",
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(tarGz.Bytes())
	}))
	defer server.Close()

	fetcher := gitstrat.NewArchiveFetcher(gitstrat.ArchiveFetcherOptions{
		HTTPClient: server.Client(),
		Logger:     logger,
	})

	tmpDir := t.TempDir()
	err := fetcher.DownloadAndExtract(context.Background(), server.URL+"/archive.tar.gz", tmpDir)
	require.NoError(t, err)
}

func TestExecute_WithFilterPath(t *testing.T) {
	tmpDir := t.TempDir()
	processor := gitstrat.NewProcessor(gitstrat.ProcessorOptions{})

	docsDir := filepath.Join(tmpDir, "docs")
	require.NoError(t, os.MkdirAll(docsDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("# Root"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(docsDir, "guide.md"), []byte("# Guide"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(docsDir, "api.md"), []byte("# API"), 0644))

	files, err := processor.FindDocumentationFiles(tmpDir, "docs")
	require.NoError(t, err)
	assert.Len(t, files, 2)
}

func TestExecute_WithLimit(t *testing.T) {
	tmpDir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "doc1.md"), []byte("# Doc 1"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "doc2.md"), []byte("# Doc 2"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "doc3.md"), []byte("# Doc 3"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "doc4.md"), []byte("# Doc 4"), 0644))

	processor := gitstrat.NewProcessor(gitstrat.ProcessorOptions{})
	files, err := processor.FindDocumentationFiles(tmpDir, "")
	require.NoError(t, err)
	assert.Len(t, files, 4)

	limitedFiles := files[:2]
	assert.Len(t, limitedFiles, 2)
}

func TestExecute_NoDocFiles_WithFilter(t *testing.T) {
	tarGz := createTestTarGz(t, map[string]string{
		"repo-main/src/main.go": "package main",
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(tarGz.Bytes())
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		Force:   true,
	})

	deps := &gitstrat.StrategyDependencies{
		Writer:     writer,
		Logger:     logger,
		HTTPClient: server.Client(),
	}

	strategy := gitstrat.NewStrategy(deps)

	ctx := context.Background()
	opts := gitstrat.ExecuteOptions{
		Output:    tmpDir,
		FilterURL: "nonexistent",
	}

	err := strategy.Execute(ctx, server.URL+"/user/repo", opts)
	assert.Error(t, err)
}

func TestExecute_ArchiveFailsFallbackToClone(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		Force:   true,
	})

	deps := &gitstrat.StrategyDependencies{
		Writer:     writer,
		Logger:     logger,
		HTTPClient: server.Client(),
	}

	strategy := gitstrat.NewStrategy(deps)

	ctx := context.Background()
	opts := gitstrat.ExecuteOptions{
		Output: tmpDir,
	}

	err := strategy.Execute(ctx, "https://github.com/nonexistent/repo", opts)
	assert.Error(t, err)
}

func TestTryArchiveDownload_ValidParsing(t *testing.T) {
	parser := gitstrat.NewParser()

	info, err := parser.ParseURL("https://github.com/user/repo")
	require.NoError(t, err)
	assert.Equal(t, "user", info.Owner)
	assert.Equal(t, "repo", info.Repo)
	assert.Equal(t, gitstrat.PlatformGitHub, info.Platform)

	fetcher := gitstrat.NewArchiveFetcher(gitstrat.ArchiveFetcherOptions{})
	url := fetcher.BuildArchiveURL(info, "main")
	assert.Equal(t, "https://github.com/user/repo/archive/refs/heads/main.tar.gz", url)

	masterURL := fetcher.BuildArchiveURL(info, "master")
	assert.Equal(t, "https://github.com/user/repo/archive/refs/heads/master.tar.gz", masterURL)
}

func TestTryArchiveDownload_BothMainAndMasterFail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})

	deps := &gitstrat.StrategyDependencies{
		Logger:     logger,
		HTTPClient: server.Client(),
	}

	strategy := gitstrat.NewStrategy(deps)

	ctx := context.Background()
	_, _, err := strategy.TryArchiveDownload(ctx, server.URL+"/user/repo", tmpDir)

	assert.Error(t, err)
}

func TestProcessor_ProcessFiles_WithError(t *testing.T) {
	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})
	processor := gitstrat.NewProcessor(gitstrat.ProcessorOptions{
		Logger: logger,
	})

	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "file1.md")
	require.NoError(t, os.WriteFile(file1, []byte("# File 1"), 0644))

	writeFunc := func(ctx context.Context, doc *domain.Document) error {
		return fmt.Errorf("write error")
	}

	opts := gitstrat.ProcessOptions{
		RepoURL:     "https://github.com/user/repo",
		Branch:      "main",
		WriteFunc:   writeFunc,
		Concurrency: 1,
	}

	err := processor.ProcessFiles(context.Background(), []string{file1}, tmpDir, opts)
	require.NoError(t, err)
}

func TestProcessor_FindDocumentationFiles_WalkError(t *testing.T) {
	processor := gitstrat.NewProcessor(gitstrat.ProcessorOptions{})

	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subdir")
	require.NoError(t, os.MkdirAll(subDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(subDir, "test.md"), []byte("# Test"), 0644))

	require.NoError(t, os.Chmod(subDir, 0000))
	defer os.Chmod(subDir, 0755)

	_, err := processor.FindDocumentationFiles(tmpDir, "")
	assert.Error(t, err)
}

func TestProcessor_FindDocumentationFiles_WithLogger(t *testing.T) {
	logger := utils.NewLogger(utils.LoggerOptions{Level: "debug"})
	processor := gitstrat.NewProcessor(gitstrat.ProcessorOptions{
		Logger: logger,
	})

	tmpDir := t.TempDir()
	docsDir := filepath.Join(tmpDir, "docs")
	require.NoError(t, os.MkdirAll(docsDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(docsDir, "test.md"), []byte("# Test"), 0644))

	files, err := processor.FindDocumentationFiles(tmpDir, "docs")
	require.NoError(t, err)
	assert.Len(t, files, 1)
}

func TestExecute_SuccessfulArchiveDownload(t *testing.T) {
	tarGz := createTestTarGz(t, map[string]string{
		"repo-main/README.md": "# Hello World",
	})

	fetcher := gitstrat.NewArchiveFetcher(gitstrat.ArchiveFetcherOptions{})

	tmpDir := t.TempDir()
	err := fetcher.ExtractTarGz(tarGz, tmpDir)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(tmpDir, "README.md"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "Hello World")
}

func TestCloneRepository_Success(t *testing.T) {
	tmpDir := t.TempDir()
	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		Force:   true,
	})

	deps := &gitstrat.StrategyDependencies{
		Writer: writer,
		Logger: logger,
	}

	strategy := gitstrat.NewStrategy(deps)

	ctx := context.Background()
	_, err := strategy.CloneRepository(ctx, "https://invalid.example.com/nonexistent/repo", tmpDir)
	assert.Error(t, err)
}

func TestStrategy_ParseURLWithBranch(t *testing.T) {
	parser := gitstrat.NewParser()

	info, err := parser.ParseURLWithPath("https://github.com/user/repo/tree/develop/docs")
	require.NoError(t, err)
	assert.Equal(t, "develop", info.Branch)
	assert.Equal(t, "docs", info.SubPath)
}

func TestArchiveFetcher_ExtractTarGz_EmptyArchive(t *testing.T) {
	fetcher := gitstrat.NewArchiveFetcher(gitstrat.ArchiveFetcherOptions{})

	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)
	require.NoError(t, tw.Close())
	require.NoError(t, gzw.Close())

	tmpDir := t.TempDir()
	err := fetcher.ExtractTarGz(&buf, tmpDir)
	require.NoError(t, err)
}

func TestArchiveFetcher_ExtractTarGz_InvalidTar(t *testing.T) {
	fetcher := gitstrat.NewArchiveFetcher(gitstrat.ArchiveFetcherOptions{})

	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	gzw.Write([]byte("invalid tar content"))
	gzw.Close()

	tmpDir := t.TempDir()
	err := fetcher.ExtractTarGz(&buf, tmpDir)
	assert.Error(t, err)
}

func TestArchiveFetcher_DownloadAndExtract_WithToken(t *testing.T) {
	originalToken := os.Getenv("GITHUB_TOKEN")
	os.Setenv("GITHUB_TOKEN", "test-token")
	defer func() {
		if originalToken != "" {
			os.Setenv("GITHUB_TOKEN", originalToken)
		} else {
			os.Unsetenv("GITHUB_TOKEN")
		}
	}()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "token test-token", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	fetcher := gitstrat.NewArchiveFetcher(gitstrat.ArchiveFetcherOptions{
		HTTPClient: server.Client(),
	})

	tmpDir := t.TempDir()
	err := fetcher.DownloadAndExtract(context.Background(), server.URL+"/archive.tar.gz", tmpDir)
	assert.Error(t, err)
}

func TestProcessor_ProcessFile_WriteError(t *testing.T) {
	processor := gitstrat.NewProcessor(gitstrat.ProcessorOptions{})

	tmpDir := t.TempDir()
	mdPath := filepath.Join(tmpDir, "test.md")
	require.NoError(t, os.WriteFile(mdPath, []byte("# Test"), 0644))

	writeFunc := func(ctx context.Context, doc *domain.Document) error {
		return fmt.Errorf("write error")
	}

	opts := gitstrat.ProcessOptions{
		RepoURL:   "https://github.com/user/repo",
		Branch:    "main",
		WriteFunc: writeFunc,
	}

	err := processor.ProcessFile(context.Background(), mdPath, tmpDir, opts)
	assert.Error(t, err)
}

func TestCanHandle_TreeURL(t *testing.T) {
	tmpDir := t.TempDir()
	deps := setupTestDependencies(t, tmpDir)
	strategy := gitstrat.NewStrategy(deps)

	tests := []struct {
		url      string
		expected bool
	}{
		{"https://github.com/user/repo/tree/main/docs", true},
		{"https://gitlab.com/user/repo/-/tree/main/docs", true},
		{"https://bitbucket.org/user/repo/src/main/docs", true},
	}

	for _, tc := range tests {
		t.Run(tc.url, func(t *testing.T) {
			result := strategy.CanHandle(tc.url)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestRepoInfo_Fields(t *testing.T) {
	info := &gitstrat.RepoInfo{
		Platform: gitstrat.PlatformGitHub,
		Owner:    "testowner",
		Repo:     "testrepo",
		URL:      "https://github.com/testowner/testrepo",
	}

	assert.Equal(t, gitstrat.PlatformGitHub, info.Platform)
	assert.Equal(t, "testowner", info.Owner)
	assert.Equal(t, "testrepo", info.Repo)
	assert.Equal(t, "https://github.com/testowner/testrepo", info.URL)
}

func TestGitURLInfo_Fields(t *testing.T) {
	info := &gitstrat.GitURLInfo{
		RepoURL:  "https://github.com/user/repo",
		Platform: gitstrat.PlatformGitHub,
		Owner:    "user",
		Repo:     "repo",
		Branch:   "main",
		SubPath:  "docs",
	}

	assert.Equal(t, "https://github.com/user/repo", info.RepoURL)
	assert.Equal(t, gitstrat.PlatformGitHub, info.Platform)
	assert.Equal(t, "user", info.Owner)
	assert.Equal(t, "repo", info.Repo)
	assert.Equal(t, "main", info.Branch)
	assert.Equal(t, "docs", info.SubPath)
}

func TestFetchResult_Fields(t *testing.T) {
	result := &gitstrat.FetchResult{
		LocalPath: "/tmp/repo",
		Branch:    "main",
		Method:    "archive",
	}

	assert.Equal(t, "/tmp/repo", result.LocalPath)
	assert.Equal(t, "main", result.Branch)
	assert.Equal(t, "archive", result.Method)
}

func TestDocumentExtensions(t *testing.T) {
	assert.True(t, gitstrat.DocumentExtensions[".md"])
	assert.True(t, gitstrat.DocumentExtensions[".mdx"])
	assert.False(t, gitstrat.DocumentExtensions[".txt"])
	assert.False(t, gitstrat.DocumentExtensions[".go"])
}

func TestIgnoreDirs(t *testing.T) {
	assert.True(t, gitstrat.IgnoreDirs[".git"])
	assert.True(t, gitstrat.IgnoreDirs["node_modules"])
	assert.True(t, gitstrat.IgnoreDirs["vendor"])
	assert.True(t, gitstrat.IgnoreDirs["__pycache__"])
	assert.True(t, gitstrat.IgnoreDirs[".venv"])
	assert.True(t, gitstrat.IgnoreDirs["venv"])
	assert.True(t, gitstrat.IgnoreDirs["dist"])
	assert.True(t, gitstrat.IgnoreDirs["build"])
	assert.True(t, gitstrat.IgnoreDirs[".next"])
	assert.True(t, gitstrat.IgnoreDirs[".nuxt"])
	assert.False(t, gitstrat.IgnoreDirs["src"])
	assert.False(t, gitstrat.IgnoreDirs["docs"])
}

func TestPlatformConstants(t *testing.T) {
	assert.Equal(t, gitstrat.Platform("github"), gitstrat.PlatformGitHub)
	assert.Equal(t, gitstrat.Platform("gitlab"), gitstrat.PlatformGitLab)
	assert.Equal(t, gitstrat.Platform("bitbucket"), gitstrat.PlatformBitbucket)
	assert.Equal(t, gitstrat.Platform("generic"), gitstrat.PlatformGeneric)
}

func TestProcessor_ProcessFile_MdxExtension(t *testing.T) {
	processor := gitstrat.NewProcessor(gitstrat.ProcessorOptions{})

	tmpDir := t.TempDir()
	mdxPath := filepath.Join(tmpDir, "component.mdx")
	require.NoError(t, os.WriteFile(mdxPath, []byte("# MDX Content\n\nimport Component from './Component'"), 0644))

	var capturedDoc *domain.Document
	writeFunc := func(ctx context.Context, doc *domain.Document) error {
		capturedDoc = doc
		return nil
	}

	opts := gitstrat.ProcessOptions{
		RepoURL:   "https://github.com/user/repo",
		Branch:    "main",
		WriteFunc: writeFunc,
	}

	err := processor.ProcessFile(context.Background(), mdxPath, tmpDir, opts)
	require.NoError(t, err)
	require.NotNil(t, capturedDoc)
	assert.Contains(t, capturedDoc.Content, "# MDX Content")
}

func TestExecuteOptions_Fields(t *testing.T) {
	opts := gitstrat.ExecuteOptions{
		Output:      "/output",
		Concurrency: 10,
		Limit:       100,
		DryRun:      true,
		FilterURL:   "docs/",
	}

	assert.Equal(t, "/output", opts.Output)
	assert.Equal(t, 10, opts.Concurrency)
	assert.Equal(t, 100, opts.Limit)
	assert.True(t, opts.DryRun)
	assert.Equal(t, "docs/", opts.FilterURL)
}

func TestProcessOptions_Fields(t *testing.T) {
	opts := gitstrat.ProcessOptions{
		RepoURL:     "https://github.com/user/repo",
		Branch:      "main",
		FilterPath:  "docs",
		Concurrency: 5,
		Limit:       50,
		DryRun:      false,
	}

	assert.Equal(t, "https://github.com/user/repo", opts.RepoURL)
	assert.Equal(t, "main", opts.Branch)
	assert.Equal(t, "docs", opts.FilterPath)
	assert.Equal(t, 5, opts.Concurrency)
	assert.Equal(t, 50, opts.Limit)
	assert.False(t, opts.DryRun)
}

func TestArchiveFetcherOptions_Fields(t *testing.T) {
	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})
	httpClient := &http.Client{}

	opts := gitstrat.ArchiveFetcherOptions{
		HTTPClient: httpClient,
		Logger:     logger,
	}

	assert.Equal(t, httpClient, opts.HTTPClient)
	assert.Equal(t, logger, opts.Logger)
}

func TestCloneFetcherOptions_Fields(t *testing.T) {
	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})

	opts := gitstrat.CloneFetcherOptions{
		Logger: logger,
	}

	assert.Equal(t, logger, opts.Logger)
}

func TestProcessorOptions_Fields(t *testing.T) {
	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})

	opts := gitstrat.ProcessorOptions{
		Logger: logger,
	}

	assert.Equal(t, logger, opts.Logger)
}
