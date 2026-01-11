package git_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/quantmind-br/repodocs-go/internal/strategies/git"
)

func TestNewParser(t *testing.T) {
	p := git.NewParser()

	assert.NotNil(t, p)
}

func TestParser_ParseURL_GitHub(t *testing.T) {
	p := git.NewParser()

	tests := []struct {
		name string
		url  string
		want *git.RepoInfo
	}{
		{
			name: "HTTPS URL",
			url:  "https://github.com/owner/repo",
			want: &git.RepoInfo{
				Platform: git.PlatformGitHub,
				Owner:    "owner",
				Repo:     "repo",
				URL:      "https://github.com/owner/repo",
			},
		},
		{
			name: "HTTPS URL with .git",
			url:  "https://github.com/owner/repo.git",
			want: &git.RepoInfo{
				Platform: git.PlatformGitHub,
				Owner:    "owner",
				Repo:     "repo",
				URL:      "https://github.com/owner/repo.git",
			},
		},
		{
			name: "SSH URL",
			url:  "git@github.com:owner/repo.git",
			want: &git.RepoInfo{
				Platform: git.PlatformGitHub,
				Owner:    "owner",
				Repo:     "repo",
				URL:      "git@github.com:owner/repo.git",
			},
		},
		{
			name: "with trailing slash",
			url:  "https://github.com/owner/repo/",
			want: &git.RepoInfo{
				Platform: git.PlatformGitHub,
				Owner:    "owner",
				Repo:     "repo",
				URL:      "https://github.com/owner/repo/",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := p.ParseURL(tt.url)

			assert.NoError(t, err)
			assert.Equal(t, tt.want.Platform, got.Platform)
			assert.Equal(t, tt.want.Owner, got.Owner)
			assert.Equal(t, tt.want.Repo, got.Repo)
			assert.Equal(t, tt.want.URL, got.URL)
		})
	}
}

func TestParser_ParseURL_GitLab(t *testing.T) {
	p := git.NewParser()

	tests := []struct {
		name string
		url  string
		want *git.RepoInfo
	}{
		{
			name: "HTTPS URL",
			url:  "https://gitlab.com/owner/repo",
			want: &git.RepoInfo{
				Platform: git.PlatformGitLab,
				Owner:    "owner",
				Repo:     "repo",
				URL:      "https://gitlab.com/owner/repo",
			},
		},
		{
			name: "HTTPS URL with .git",
			url:  "https://gitlab.com/owner/repo.git",
			want: &git.RepoInfo{
				Platform: git.PlatformGitLab,
				Owner:    "owner",
				Repo:     "repo",
				URL:      "https://gitlab.com/owner/repo.git",
			},
		},
		{
			name: "SSH URL",
			url:  "git@gitlab.com:owner/repo.git",
			want: &git.RepoInfo{
				Platform: git.PlatformGitLab,
				Owner:    "owner",
				Repo:     "repo",
				URL:      "git@gitlab.com:owner/repo.git",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := p.ParseURL(tt.url)

			assert.NoError(t, err)
			assert.Equal(t, tt.want.Platform, got.Platform)
			assert.Equal(t, tt.want.Owner, got.Owner)
			assert.Equal(t, tt.want.Repo, got.Repo)
			assert.Equal(t, tt.want.URL, got.URL)
		})
	}
}

func TestParser_ParseURL_Bitbucket(t *testing.T) {
	p := git.NewParser()

	tests := []struct {
		name string
		url  string
		want *git.RepoInfo
	}{
		{
			name: "HTTPS URL",
			url:  "https://bitbucket.org/owner/repo",
			want: &git.RepoInfo{
				Platform: git.PlatformBitbucket,
				Owner:    "owner",
				Repo:     "repo",
				URL:      "https://bitbucket.org/owner/repo",
			},
		},
		{
			name: "HTTPS URL with .git",
			url:  "https://bitbucket.org/owner/repo.git",
			want: &git.RepoInfo{
				Platform: git.PlatformBitbucket,
				Owner:    "owner",
				Repo:     "repo",
				URL:      "https://bitbucket.org/owner/repo.git",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := p.ParseURL(tt.url)

			assert.NoError(t, err)
			assert.Equal(t, tt.want.Platform, got.Platform)
			assert.Equal(t, tt.want.Owner, got.Owner)
			assert.Equal(t, tt.want.Repo, got.Repo)
			assert.Equal(t, tt.want.URL, got.URL)
		})
	}
}

func TestParser_ParseURL_Invalid(t *testing.T) {
	p := git.NewParser()

	tests := []struct {
		name string
		url  string
	}{
		{
			name: "no git platform",
			url:  "https://example.com/repo",
		},
		{
			name: "empty string",
			url:  "",
		},
		{
			name: "malformed URL",
			url:  "not-a-url",
		},
		{
			name: "missing owner",
			url:  "https://github.com/repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := p.ParseURL(tt.url)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), "unsupported git URL format")
		})
	}
}

func TestParser_ParseURLWithPath(t *testing.T) {
	p := git.NewParser()

	tests := []struct {
		name    string
		url     string
		want    *git.GitURLInfo
		wantErr bool
	}{
		{
			name: "GitHub with path",
			url:  "https://github.com/owner/repo/tree/main/docs",
			want: &git.GitURLInfo{
				Platform: git.PlatformGitHub,
				RepoURL:  "https://github.com/owner/repo",
				Owner:    "owner",
				Repo:     "repo",
				Branch:   "main",
				SubPath:  "docs",
			},
			wantErr: false,
		},
		{
			name: "GitLab with path",
			url:  "https://gitlab.com/owner/repo/-/tree/develop/guides",
			want: &git.GitURLInfo{
				Platform: git.PlatformGitLab,
				RepoURL:  "https://gitlab.com/owner/repo",
				Owner:    "owner",
				Repo:     "repo",
				Branch:   "develop",
				SubPath:  "guides",
			},
			wantErr: false,
		},
		{
			name: "Bitbucket with path",
			url:  "https://bitbucket.org/owner/repo/src/feature/docs",
			want: &git.GitURLInfo{
				Platform: git.PlatformBitbucket,
				RepoURL:  "https://bitbucket.org/owner/repo",
				Owner:    "owner",
				Repo:     "repo",
				Branch:   "feature",
				SubPath:  "docs",
			},
			wantErr: false,
		},
		{
			name: "without path",
			url:  "https://github.com/owner/repo",
			want: &git.GitURLInfo{
				Platform: git.PlatformGitHub,
				RepoURL:  "https://github.com/owner/repo",
				Owner:    "owner",
				Repo:     "repo",
				Branch:   "",
				SubPath:  "",
			},
			wantErr: false,
		},
		{
			name: "with trailing slash",
			url:  "https://github.com/owner/repo/tree/main/docs/",
			want: &git.GitURLInfo{
				Platform: git.PlatformGitHub,
				RepoURL:  "https://github.com/owner/repo",
				Owner:    "owner",
				Repo:     "repo",
				Branch:   "main",
				SubPath:  "docs",
			},
			wantErr: false,
		},
		{
			name: "deep nested path",
			url:  "https://github.com/owner/repo/tree/main/docs/api/v2",
			want: &git.GitURLInfo{
				Platform: git.PlatformGitHub,
				RepoURL:  "https://github.com/owner/repo",
				Owner:    "owner",
				Repo:     "repo",
				Branch:   "main",
				SubPath:  "docs/api/v2",
			},
			wantErr: false,
		},
		{
			name: "generic HTTP URL",
			url:  "https://example.com/repo.git",
			want: &git.GitURLInfo{
				Platform: git.PlatformGeneric,
				RepoURL:  "https://example.com/repo.git",
			},
			wantErr: false,
		},
		{
			name:    "invalid URL",
			url:     "not-a-url",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := p.ParseURLWithPath(tt.url)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want.Platform, got.Platform)
				assert.Equal(t, tt.want.RepoURL, got.RepoURL)
				assert.Equal(t, tt.want.Owner, got.Owner)
				assert.Equal(t, tt.want.Repo, got.Repo)
				assert.Equal(t, tt.want.Branch, got.Branch)
				assert.Equal(t, tt.want.SubPath, got.SubPath)
			}
		})
	}
}

func TestNormalizeFilterPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "empty path",
			path:     "",
			expected: "",
		},
		{
			name:     "simple path",
			path:     "docs",
			expected: "docs",
		},
		{
			name:     "trailing slash",
			path:     "docs/",
			expected: "docs",
		},
		{
			name:     "nested path",
			path:     "docs/api/v1",
			expected: "docs/api/v1",
		},
		{
			name:     "URL encoded",
			path:     "docs%2Fapi",
			expected: "docs/api",
		},
		{
			name:     "backslashes to forward slashes",
			path:     "docs\\api\\v1",
			expected: "docs/api/v1",
		},
		{
			name:     "leading and trailing slashes",
			path:     "/docs/api/",
			expected: "docs/api",
		},
		{
			name:     "with dots",
			path:     "docs/../api",
			expected: "api",
		},
		{
			name:     "with current dir dot",
			path:     "./docs/api",
			expected: "docs/api",
		},
		{
			name:     "tree URL",
			path:     "https://github.com/owner/repo/tree/main/docs",
			expected: "docs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := git.NormalizeFilterPath(tt.path)

			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestExtractPathFromTreeURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "GitHub tree URL",
			url:      "https://github.com/owner/repo/tree/main/docs",
			expected: "docs",
		},
		{
			name:     "GitHub blob URL",
			url:      "https://github.com/owner/repo/blob/main/docs/api.md",
			expected: "docs/api.md",
		},
		{
			name:     "GitLab tree URL",
			url:      "https://gitlab.com/owner/repo/-/tree/develop/guides",
			expected: "guides",
		},
		{
			name:     "GitLab blob URL",
			url:      "https://gitlab.com/owner/repo/-/blob/develop/guides/install.md",
			expected: "guides/install.md",
		},
		{
			name:     "Bitbucket src URL",
			url:      "https://bitbucket.org/owner/repo/src/feature/docs",
			expected: "docs",
		},
		{
			name:     "deep nested path",
			url:      "https://github.com/owner/repo/tree/main/docs/api/v2/guides",
			expected: "docs/api/v2/guides",
		},
		{
			name:     "no path component",
			url:      "https://github.com/owner/repo/tree/main",
			expected: "https://github.com/owner/repo/tree/main",
		},
		{
			name:     "non-tree URL",
			url:      "https://github.com/owner/repo",
			expected: "https://github.com/owner/repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := git.ExtractPathFromTreeURL(tt.url)

			assert.Equal(t, tt.expected, got)
		})
	}
}
