package git

import (
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
)

type platformPattern struct {
	platform    Platform
	repoPattern *regexp.Regexp
	treePattern *regexp.Regexp
}

type Parser struct {
	patterns []platformPattern
}

func NewParser() *Parser {
	return &Parser{
		patterns: []platformPattern{
			{
				platform:    PlatformGitHub,
				repoPattern: regexp.MustCompile(`^(https?://github\.com/([^/]+)/([^/]+?))(\.git)?(/|$)`),
				treePattern: regexp.MustCompile(`/tree/([^/]+)(?:/(.+))?$`),
			},
			{
				platform:    PlatformGitLab,
				repoPattern: regexp.MustCompile(`^(https?://gitlab\.com/([^/]+)/([^/]+?))(\.git)?(/|$)`),
				treePattern: regexp.MustCompile(`/-/tree/([^/]+)(?:/(.+))?$`),
			},
			{
				platform:    PlatformBitbucket,
				repoPattern: regexp.MustCompile(`^(https?://bitbucket\.org/([^/]+)/([^/]+?))(\.git)?(/|$)`),
				treePattern: regexp.MustCompile(`/src/([^/]+)(?:/(.+))?$`),
			},
		},
	}
}

func (p *Parser) ParseURL(rawURL string) (*RepoInfo, error) {
	patterns := []struct {
		platform Platform
		regex    *regexp.Regexp
	}{
		{PlatformGitHub, regexp.MustCompile(`github\.com[:/]([^/]+)/([^/.]+)`)},
		{PlatformGitLab, regexp.MustCompile(`gitlab\.com[:/]([^/]+)/([^/.]+)`)},
		{PlatformBitbucket, regexp.MustCompile(`bitbucket\.org[:/]([^/]+)/([^/.]+)`)},
	}

	for _, pat := range patterns {
		if matches := pat.regex.FindStringSubmatch(rawURL); len(matches) == 3 {
			return &RepoInfo{
				Platform: pat.platform,
				Owner:    matches[1],
				Repo:     strings.TrimSuffix(matches[2], ".git"),
				URL:      rawURL,
			}, nil
		}
	}

	return nil, fmt.Errorf("unsupported git URL format: %s", rawURL)
}

func (p *Parser) ParseURLWithPath(rawURL string) (*GitURLInfo, error) {
	info := &GitURLInfo{}
	lower := strings.ToLower(rawURL)

	for _, pat := range p.patterns {
		if !strings.Contains(lower, string(pat.platform)) {
			continue
		}

		repoMatches := pat.repoPattern.FindStringSubmatch(rawURL)
		if len(repoMatches) < 4 {
			continue
		}

		info.Platform = pat.platform
		info.RepoURL = repoMatches[1]
		info.Owner = repoMatches[2]
		info.Repo = strings.TrimSuffix(repoMatches[3], ".git")

		treeMatches := pat.treePattern.FindStringSubmatch(rawURL)
		if len(treeMatches) >= 2 {
			info.Branch = treeMatches[1]
			if len(treeMatches) >= 3 && treeMatches[2] != "" {
				info.SubPath = NormalizeFilterPath(treeMatches[2])
			}
		}

		return info, nil
	}

	if strings.HasPrefix(rawURL, "http://") || strings.HasPrefix(rawURL, "https://") {
		info.Platform = PlatformGeneric
		info.RepoURL = rawURL
		return info, nil
	}

	return nil, fmt.Errorf("unsupported git URL format: %s", rawURL)
}

func NormalizeFilterPath(path string) string {
	if path == "" {
		return ""
	}

	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		path = ExtractPathFromTreeURL(path)
	}

	decoded, err := url.PathUnescape(path)
	if err == nil {
		path = decoded
	}

	path = strings.ReplaceAll(path, "\\", "/")
	path = strings.Trim(path, "/")
	path = filepath.Clean(path)

	return path
}

func ExtractPathFromTreeURL(rawURL string) string {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`github\.com/[^/]+/[^/]+/(?:tree|blob)/[^/]+/(.+)$`),
		regexp.MustCompile(`gitlab\.com/[^/]+/[^/]+/-/(?:tree|blob)/[^/]+/(.+)$`),
		regexp.MustCompile(`bitbucket\.org/[^/]+/[^/]+/src/[^/]+/(.+)$`),
	}

	for _, pat := range patterns {
		if matches := pat.FindStringSubmatch(rawURL); len(matches) >= 2 {
			return matches[1]
		}
	}

	return rawURL
}
