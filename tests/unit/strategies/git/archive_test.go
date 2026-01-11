package git_test

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/quantmind-br/repodocs-go/internal/strategies/git"
)

func TestNewArchiveFetcher(t *testing.T) {
	f := git.NewArchiveFetcher(git.ArchiveFetcherOptions{})

	assert.NotNil(t, f)
	assert.Equal(t, "archive", f.Name())
}

func TestArchiveFetcher_Name(t *testing.T) {
	f := git.NewArchiveFetcher(git.ArchiveFetcherOptions{})

	assert.Equal(t, "archive", f.Name())
}

func TestBuildArchiveURL(t *testing.T) {
	tests := []struct {
		name    string
		info    *git.RepoInfo
		branch  string
		wantURL string
	}{
		{
			name: "GitHub default branch",
			info: &git.RepoInfo{
				Platform: git.PlatformGitHub,
				Owner:    "owner",
				Repo:     "repo",
			},
			branch:  "main",
			wantURL: "https://github.com/owner/repo/archive/refs/heads/main.tar.gz",
		},
		{
			name: "GitHub custom branch",
			info: &git.RepoInfo{
				Platform: git.PlatformGitHub,
				Owner:    "owner",
				Repo:     "repo",
			},
			branch:  "develop",
			wantURL: "https://github.com/owner/repo/archive/refs/heads/develop.tar.gz",
		},
		{
			name: "GitHub special characters in branch",
			info: &git.RepoInfo{
				Platform: git.PlatformGitHub,
				Owner:    "owner",
				Repo:     "repo",
			},
			branch:  "feature/v2-release",
			wantURL: "https://github.com/owner/repo/archive/refs/heads/feature/v2-release.tar.gz",
		},
		{
			name: "GitLab",
			info: &git.RepoInfo{
				Platform: git.PlatformGitLab,
				Owner:    "owner",
				Repo:     "repo",
			},
			branch:  "main",
			wantURL: "https://gitlab.com/owner/repo/-/archive/main/repo-main.tar.gz",
		},
		{
			name: "Bitbucket",
			info: &git.RepoInfo{
				Platform: git.PlatformBitbucket,
				Owner:    "owner",
				Repo:     "repo",
			},
			branch:  "master",
			wantURL: "https://bitbucket.org/owner/repo/get/master.tar.gz",
		},
		{
			name: "generic platform defaults to GitHub",
			info: &git.RepoInfo{
				Platform: git.PlatformGeneric,
				Owner:    "owner",
				Repo:     "repo",
			},
			branch:  "main",
			wantURL: "https://github.com/owner/repo/archive/refs/heads/main.tar.gz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := git.NewArchiveFetcher(git.ArchiveFetcherOptions{})
			got := f.BuildArchiveURL(tt.info, tt.branch)

			assert.Equal(t, tt.wantURL, got)
		})
	}
}

func TestDownloadAndExtract_404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	f := git.NewArchiveFetcher(git.ArchiveFetcherOptions{
		HTTPClient: server.Client(),
	})
	tmpDir := t.TempDir()

	err := f.DownloadAndExtract(context.Background(), server.URL+"/archive.tar.gz", tmpDir)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "archive not found (404)")
}

func TestDownloadAndExtract_401(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	f := git.NewArchiveFetcher(git.ArchiveFetcherOptions{
		HTTPClient: server.Client(),
	})
	tmpDir := t.TempDir()

	err := f.DownloadAndExtract(context.Background(), server.URL+"/archive.tar.gz", tmpDir)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "authentication required (401)")
}

func TestDownloadAndExtract_500(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	f := git.NewArchiveFetcher(git.ArchiveFetcherOptions{
		HTTPClient: server.Client(),
	})
	tmpDir := t.TempDir()

	err := f.DownloadAndExtract(context.Background(), server.URL+"/archive.tar.gz", tmpDir)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "download failed with status: 500")
}

func TestDownloadAndExtract_Success(t *testing.T) {
	archiveContent := createTestArchive(t, map[string]string{
		"README.md":   "Hello World",
		"docs/api.md": "API Documentation",
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(archiveContent)
	}))
	defer server.Close()

	f := git.NewArchiveFetcher(git.ArchiveFetcherOptions{
		HTTPClient: server.Client(),
	})
	tmpDir := t.TempDir()

	err := f.DownloadAndExtract(context.Background(), server.URL+"/archive.tar.gz", tmpDir)

	assert.NoError(t, err)

	readmePath := filepath.Join(tmpDir, "README.md")
	apiPath := filepath.Join(tmpDir, "docs/api.md")

	content, err := os.ReadFile(readmePath)
	assert.NoError(t, err)
	assert.Equal(t, "Hello World", string(content))

	content, err = os.ReadFile(apiPath)
	assert.NoError(t, err)
	assert.Equal(t, "API Documentation", string(content))
}

func TestExtractTarGz_InvalidGzip(t *testing.T) {
	f := git.NewArchiveFetcher(git.ArchiveFetcherOptions{})
	tmpDir := t.TempDir()

	err := f.ExtractTarGz(strings.NewReader("not a gzip file"), tmpDir)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "gzip reader failed")
}

func TestExtractTarGz_EmptyArchive(t *testing.T) {
	var buf strings.Builder
	gz := gzip.NewWriter(&buf)
	gz.Close()

	f := git.NewArchiveFetcher(git.ArchiveFetcherOptions{})
	tmpDir := t.TempDir()

	err := f.ExtractTarGz(strings.NewReader(buf.String()), tmpDir)

	assert.NoError(t, err)
}

func TestExtractTarGz_WithDirectories(t *testing.T) {
	archiveContent := createTestArchive(t, map[string]string{
		"docs/README.md":        "Docs README",
		"docs/api/README.md":    "API README",
		"docs/guide/install.md": "Installation Guide",
	})

	f := git.NewArchiveFetcher(git.ArchiveFetcherOptions{})
	tmpDir := t.TempDir()

	err := f.ExtractTarGz(strings.NewReader(string(archiveContent)), tmpDir)

	assert.NoError(t, err)

	docsReadme := filepath.Join(tmpDir, "docs", "README.md")
	apiReadme := filepath.Join(tmpDir, "docs", "api", "README.md")
	install := filepath.Join(tmpDir, "docs", "guide", "install.md")

	content, err := os.ReadFile(docsReadme)
	assert.NoError(t, err)
	assert.Equal(t, "Docs README", string(content))

	content, err = os.ReadFile(apiReadme)
	assert.NoError(t, err)
	assert.Equal(t, "API README", string(content))

	content, err = os.ReadFile(install)
	assert.NoError(t, err)
	assert.Equal(t, "Installation Guide", string(content))
}

func TestExtractTarGz_SecurityPathTraversal(t *testing.T) {
	buf := new(strings.Builder)
	gz := gzip.NewWriter(buf)
	tr := tar.NewWriter(gz)

	header := &tar.Header{
		Name:     "../evil.txt",
		Mode:     0600,
		Typeflag: tar.TypeReg,
		Size:     int64(len("evil")),
	}
	tr.WriteHeader(header)
	tr.Write([]byte("evil"))

	tr.Close()
	gz.Close()

	f := git.NewArchiveFetcher(git.ArchiveFetcherOptions{})
	tmpDir := t.TempDir()

	err := f.ExtractTarGz(strings.NewReader(buf.String()), tmpDir)

	assert.NoError(t, err)

	evilPath := filepath.Join(tmpDir, "..", "evil.txt")
	_, err = os.Stat(evilPath)
	assert.Error(t, err)
	assert.True(t, os.IsNotExist(err))
}

func TestFetch_Success(t *testing.T) {
	archiveContent := createTestArchive(t, map[string]string{
		"README.md": "Hello",
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(archiveContent)
	}))
	defer server.Close()

	f := git.NewArchiveFetcher(git.ArchiveFetcherOptions{
		HTTPClient: server.Client(),
	})

	tmpDir := t.TempDir()

	result, err := f.Fetch(context.Background(), &git.RepoInfo{
		Platform: git.PlatformGitHub,
		Owner:    "owner",
		Repo:     "repo",
		URL:      server.URL + "/archive.tar.gz",
	}, "main", tmpDir)

	assert.NoError(t, err)
	assert.Equal(t, tmpDir, result.LocalPath)
	assert.Equal(t, "main", result.Branch)
	assert.Equal(t, "archive", result.Method)

	readmePath := filepath.Join(tmpDir, "README.md")
	content, err := os.ReadFile(readmePath)
	assert.NoError(t, err)
	assert.Equal(t, "Hello", string(content))
}

func TestFetch_NetworkError(t *testing.T) {
	f := git.NewArchiveFetcher(git.ArchiveFetcherOptions{
		HTTPClient: &http.Client{Timeout: 1},
	})
	info := &git.RepoInfo{
		Platform: git.PlatformGitHub,
		Owner:    "owner",
		Repo:     "repo",
	}
	tmpDir := t.TempDir()

	_, err := f.Fetch(context.Background(), info, "main", tmpDir)

	assert.Error(t, err)
}

func TestFetch_404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	f := git.NewArchiveFetcher(git.ArchiveFetcherOptions{
		HTTPClient: server.Client(),
	})
	info := &git.RepoInfo{
		Platform: git.PlatformGitHub,
		Owner:    "owner",
		Repo:     "repo",
	}
	tmpDir := t.TempDir()

	_, err := f.Fetch(context.Background(), info, "main", tmpDir)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "archive not found (404)")
}

func createTestArchive(t *testing.T, files map[string]string) []byte {
	buf := new(strings.Builder)
	gz := gzip.NewWriter(buf)
	tr := tar.NewWriter(gz)

	for path, content := range files {
		header := &tar.Header{
			Name:     "repo-main/" + path,
			Mode:     0600,
			Typeflag: tar.TypeReg,
			Size:     int64(len(content)),
		}
		tr.WriteHeader(header)
		tr.Write([]byte(content))
	}

	tr.Close()
	gz.Close()

	return []byte(buf.String())
}
