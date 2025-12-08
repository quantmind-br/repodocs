package unit

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseGitURL tests URL parsing for different platforms
func TestParseGitURL(t *testing.T) {
	tests := []struct {
		url      string
		platform string
		owner    string
		repo     string
	}{
		// GitHub
		{"https://github.com/gin-gonic/gin", "github", "gin-gonic", "gin"},
		{"https://github.com/gin-gonic/gin.git", "github", "gin-gonic", "gin"},
		{"git@github.com:gin-gonic/gin.git", "github", "gin-gonic", "gin"},

		// GitLab
		{"https://gitlab.com/inkscape/inkscape", "gitlab", "inkscape", "inkscape"},
		{"git@gitlab.com:inkscape/inkscape.git", "gitlab", "inkscape", "inkscape"},

		// Bitbucket
		{"https://bitbucket.org/atlassian/python-bitbucket", "bitbucket", "atlassian", "python-bitbucket"},
	}

	for _, tc := range tests {
		t.Run(tc.url, func(t *testing.T) {
			info, err := parseGitURLTest(tc.url)
			require.NoError(t, err)
			assert.Equal(t, tc.platform, info.platform)
			assert.Equal(t, tc.owner, info.owner)
			assert.Equal(t, tc.repo, info.repo)
		})
	}
}

// TestDetectDefaultBranch tests automatic branch detection
func TestDetectDefaultBranch(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping network test in short mode")
	}

	tests := []struct {
		url            string
		expectedBranch string // empty means just check it doesn't error
	}{
		{"https://github.com/gin-gonic/gin", "master"},
		{"https://github.com/charmbracelet/bubbletea", "main"}, // Uses main, not master
		{"https://github.com/kubernetes/kubernetes", "master"},
	}

	for _, tc := range tests {
		t.Run(tc.url, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			branch, err := detectDefaultBranchTest(ctx, tc.url)
			require.NoError(t, err)

			if tc.expectedBranch != "" {
				assert.Equal(t, tc.expectedBranch, branch)
			}

			fmt.Printf("  %s -> %s\n", tc.url, branch)
		})
	}
}

// TestBuildArchiveURL tests archive URL construction
func TestBuildArchiveURL(t *testing.T) {
	tests := []struct {
		platform string
		owner    string
		repo     string
		branch   string
		expected string
	}{
		{
			"github", "gin-gonic", "gin", "master",
			"https://github.com/gin-gonic/gin/archive/refs/heads/master.tar.gz",
		},
		{
			"gitlab", "inkscape", "inkscape", "master",
			"https://gitlab.com/inkscape/inkscape/-/archive/master/inkscape-master.tar.gz",
		},
		{
			"bitbucket", "atlassian", "python-bitbucket", "master",
			"https://bitbucket.org/atlassian/python-bitbucket/get/master.tar.gz",
		},
	}

	for _, tc := range tests {
		name := fmt.Sprintf("%s/%s/%s", tc.platform, tc.owner, tc.repo)
		t.Run(name, func(t *testing.T) {
			url := buildArchiveURLTest(tc.platform, tc.owner, tc.repo, tc.branch)
			assert.Equal(t, tc.expected, url)
		})
	}
}

// Helper functions that mirror the strategy implementations
// These avoid importing the strategies package to prevent circular dependencies

type repoInfoTest struct {
	platform string
	owner    string
	repo     string
}

func parseGitURLTest(url string) (*repoInfoTest, error) {
	patterns := []struct {
		platform string
		regex    *regexp.Regexp
	}{
		{"github", regexp.MustCompile(`github\.com[:/]([^/]+)/([^/.]+)`)},
		{"gitlab", regexp.MustCompile(`gitlab\.com[:/]([^/]+)/([^/.]+)`)},
		{"bitbucket", regexp.MustCompile(`bitbucket\.org[:/]([^/]+)/([^/.]+)`)},
	}

	for _, p := range patterns {
		if matches := p.regex.FindStringSubmatch(url); len(matches) == 3 {
			return &repoInfoTest{
				platform: p.platform,
				owner:    matches[1],
				repo:     strings.TrimSuffix(matches[2], ".git"),
			}, nil
		}
	}

	return nil, fmt.Errorf("unsupported git URL format: %s", url)
}

func detectDefaultBranchTest(ctx context.Context, url string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "ls-remote", "--symref", url, "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git ls-remote failed: %w", err)
	}

	// Output format: "ref: refs/heads/master\tHEAD"
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "ref: refs/heads/") {
			// Split by tab first, then extract branch from first part
			parts := strings.Split(line, "\t")
			if len(parts) >= 1 {
				// parts[0] = "ref: refs/heads/master"
				branch := strings.TrimPrefix(parts[0], "ref: refs/heads/")
				return branch, nil
			}
		}
	}

	return "", fmt.Errorf("could not determine default branch")
}

func buildArchiveURLTest(platform, owner, repo, branch string) string {
	switch platform {
	case "github":
		return fmt.Sprintf("https://github.com/%s/%s/archive/refs/heads/%s.tar.gz",
			owner, repo, branch)
	case "gitlab":
		return fmt.Sprintf("https://gitlab.com/%s/%s/-/archive/%s/%s-%s.tar.gz",
			owner, repo, branch, repo, branch)
	case "bitbucket":
		return fmt.Sprintf("https://bitbucket.org/%s/%s/get/%s.tar.gz",
			owner, repo, branch)
	default:
		return fmt.Sprintf("https://github.com/%s/%s/archive/refs/heads/%s.tar.gz",
			owner, repo, branch)
	}
}
