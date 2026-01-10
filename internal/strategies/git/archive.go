package git

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/quantmind-br/repodocs-go/internal/utils"
)

type ArchiveFetcher struct {
	httpClient *http.Client
	logger     *utils.Logger
}

type ArchiveFetcherOptions struct {
	HTTPClient *http.Client
	Logger     *utils.Logger
}

func NewArchiveFetcher(opts ArchiveFetcherOptions) *ArchiveFetcher {
	return &ArchiveFetcher{
		httpClient: opts.HTTPClient,
		logger:     opts.Logger,
	}
}

func (f *ArchiveFetcher) Name() string {
	return "archive"
}

func (f *ArchiveFetcher) Fetch(ctx context.Context, info *RepoInfo, branch, destDir string) (*FetchResult, error) {
	archiveURL := f.BuildArchiveURL(info, branch)
	if f.logger != nil {
		f.logger.Debug().Str("archive_url", archiveURL).Msg("Downloading archive")
	}

	if err := f.DownloadAndExtract(ctx, archiveURL, destDir); err != nil {
		return nil, err
	}

	return &FetchResult{
		LocalPath: destDir,
		Branch:    branch,
		Method:    "archive",
	}, nil
}

func (f *ArchiveFetcher) BuildArchiveURL(info *RepoInfo, branch string) string {
	switch info.Platform {
	case PlatformGitHub:
		return fmt.Sprintf("https://github.com/%s/%s/archive/refs/heads/%s.tar.gz",
			info.Owner, info.Repo, branch)
	case PlatformGitLab:
		return fmt.Sprintf("https://gitlab.com/%s/%s/-/archive/%s/%s-%s.tar.gz",
			info.Owner, info.Repo, branch, info.Repo, branch)
	case PlatformBitbucket:
		return fmt.Sprintf("https://bitbucket.org/%s/%s/get/%s.tar.gz",
			info.Owner, info.Repo, branch)
	default:
		return fmt.Sprintf("https://github.com/%s/%s/archive/refs/heads/%s.tar.gz",
			info.Owner, info.Repo, branch)
	}
}

func (f *ArchiveFetcher) DownloadAndExtract(ctx context.Context, archiveURL, destDir string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", archiveURL, nil)
	if err != nil {
		return err
	}

	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "token "+token)
	}

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("download request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("archive not found (404)")
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("authentication required (401)")
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	return f.ExtractTarGz(resp.Body, destDir)
}

func (f *ArchiveFetcher) ExtractTarGz(r io.Reader, destDir string) error {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("gzip reader failed: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar read failed: %w", err)
		}

		parts := strings.SplitN(header.Name, "/", 2)
		if len(parts) < 2 || parts[1] == "" {
			continue
		}
		relativePath := parts[1]

		targetPath := filepath.Join(destDir, relativePath)

		if !strings.HasPrefix(filepath.Clean(targetPath), filepath.Clean(destDir)) {
			continue
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return fmt.Errorf("mkdir failed: %w", err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("mkdir failed: %w", err)
			}

			file, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("create file failed: %w", err)
			}

			if _, err := io.Copy(file, tr); err != nil {
				file.Close()
				return fmt.Errorf("copy failed: %w", err)
			}
			file.Close()
		}
	}

	return nil
}
