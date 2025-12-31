package strategies

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/output"
	"github.com/quantmind-br/repodocs-go/internal/utils"
	"github.com/schollz/progressbar/v3"
)

// WikiStrategy extracts documentation from GitHub wiki repositories
type WikiStrategy struct {
	writer *output.Writer
	logger *utils.Logger
}

// NewWikiStrategy creates a new wiki strategy
func NewWikiStrategy(deps *Dependencies) *WikiStrategy {
	return &WikiStrategy{
		writer: deps.Writer,
		logger: deps.Logger,
	}
}

// Name returns the strategy name
func (s *WikiStrategy) Name() string {
	return "wiki"
}

// CanHandle returns true if this strategy can handle the given URL
func (s *WikiStrategy) CanHandle(url string) bool {
	return IsWikiURL(url)
}

// IsWikiURL checks if a URL points to a GitHub wiki
func IsWikiURL(url string) bool {
	lower := strings.ToLower(url)

	// Pattern 1: github.com/{owner}/{repo}/wiki
	if strings.Contains(lower, "github.com") && strings.Contains(lower, "/wiki") {
		return true
	}

	// Pattern 2: {repo}.wiki.git
	if strings.HasSuffix(lower, ".wiki.git") {
		return true
	}

	return false
}

// Execute runs the wiki extraction strategy
func (s *WikiStrategy) Execute(ctx context.Context, url string, opts Options) error {
	s.logger.Info().Str("url", url).Msg("Starting wiki extraction")

	// Step 1: Parse wiki URL
	wikiInfo, err := ParseWikiURL(url)
	if err != nil {
		return fmt.Errorf("failed to parse wiki URL: %w", err)
	}

	s.logger.Debug().
		Str("owner", wikiInfo.Owner).
		Str("repo", wikiInfo.Repo).
		Str("clone_url", wikiInfo.CloneURL).
		Msg("Parsed wiki URL")

	// Step 2: Create temporary directory
	tmpDir, err := os.MkdirTemp("", "repodocs-wiki-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Step 3: Clone wiki repository
	if err := s.cloneWiki(ctx, wikiInfo.CloneURL, tmpDir); err != nil {
		return fmt.Errorf("failed to clone wiki: %w", err)
	}

	// Step 4: Parse wiki structure
	structure, err := s.parseWikiStructure(tmpDir)
	if err != nil {
		return fmt.Errorf("failed to parse wiki structure: %w", err)
	}

	s.logger.Info().
		Int("pages", len(structure.Pages)).
		Int("sections", len(structure.Sections)).
		Bool("has_sidebar", structure.HasSidebar).
		Msg("Parsed wiki structure")

	// Step 5: Process and write documents
	return s.processPages(ctx, structure, wikiInfo, opts)
}

// cloneWiki clones the wiki repository
func (s *WikiStrategy) cloneWiki(ctx context.Context, cloneURL, destDir string) error {
	s.logger.Info().Str("url", cloneURL).Msg("Cloning wiki repository")

	cloneOpts := &git.CloneOptions{
		URL:      cloneURL,
		Depth:    1, // Shallow clone for speed
		Progress: nil,
	}

	// Use HTTPS auth if available
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		cloneOpts.Auth = &githttp.BasicAuth{
			Username: "token",
			Password: token,
		}
	}

	_, err := git.PlainCloneContext(ctx, destDir, false, cloneOpts)
	if err != nil {
		// Check if wiki doesn't exist
		if strings.Contains(err.Error(), "not found") ||
			strings.Contains(err.Error(), "404") ||
			strings.Contains(err.Error(), "repository not found") {
			return fmt.Errorf("wiki not found or not enabled for this repository")
		}
		return err
	}

	return nil
}

// parseWikiStructure parses the wiki file structure and sidebar
func (s *WikiStrategy) parseWikiStructure(dir string) (*WikiStructure, error) {
	structure := &WikiStructure{
		Pages:    make(map[string]*WikiPage),
		Sections: []WikiSection{},
	}

	// Read all markdown files
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		ext := strings.ToLower(filepath.Ext(name))

		// Only process markdown files
		if ext != ".md" && ext != ".markdown" {
			continue
		}

		content, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			s.logger.Warn().Err(err).Str("file", name).Msg("Failed to read file")
			continue
		}

		page := &WikiPage{
			Filename:  name,
			Title:     FilenameToTitle(name),
			Content:   string(content),
			IsHome:    strings.EqualFold(name, "Home.md"),
			IsSpecial: strings.HasPrefix(name, "_"),
		}

		structure.Pages[name] = page
	}

	if sidebarPage, exists := structure.Pages["_Sidebar.md"]; exists {
		structure.HasSidebar = true
		structure.Sections = ParseSidebarContent(sidebarPage.Content, structure.Pages)
	} else {
		structure.Sections = CreateDefaultStructure(structure.Pages)
	}

	return structure, nil
}

// processPages processes all wiki pages and writes them to output
func (s *WikiStrategy) processPages(
	ctx context.Context,
	structure *WikiStructure,
	wikiInfo *WikiInfo,
	opts Options,
) error {
	// Count processable pages (exclude special files)
	var processablePages []*WikiPage
	for _, page := range structure.Pages {
		if !page.IsSpecial {
			processablePages = append(processablePages, page)
		}
	}

	if len(processablePages) == 0 {
		s.logger.Warn().Msg("No processable pages found in wiki")
		return nil
	}

	// Apply limit
	if opts.Limit > 0 && len(processablePages) > opts.Limit {
		processablePages = processablePages[:opts.Limit]
	}

	// Create progress bar
	bar := progressbar.NewOptions(len(processablePages),
		progressbar.OptionSetDescription("Processing wiki pages"),
		progressbar.OptionShowCount(),
	)

	// Build base wiki URL for references
	baseWikiURL := fmt.Sprintf("https://github.com/%s/%s/wiki", wikiInfo.Owner, wikiInfo.Repo)

	// Process each page
	for _, page := range processablePages {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := s.processPage(ctx, page, structure, baseWikiURL, opts); err != nil {
			s.logger.Warn().Err(err).Str("page", page.Filename).Msg("Failed to process page")
		}
		bar.Add(1)
	}

	s.logger.Info().
		Int("processed", len(processablePages)).
		Msg("Wiki extraction completed")

	return nil
}

// processPage processes a single wiki page
func (s *WikiStrategy) processPage(
	ctx context.Context,
	page *WikiPage,
	structure *WikiStructure,
	baseWikiURL string,
	opts Options,
) error {
	content := ConvertWikiLinks(page.Content, structure.Pages)

	pageName := strings.TrimSuffix(page.Filename, filepath.Ext(page.Filename))
	pageURL := baseWikiURL
	if !page.IsHome {
		pageURL = fmt.Sprintf("%s/%s", baseWikiURL, pageName)
	}

	relativePath := BuildRelativePath(page, structure, opts.NoFolders)

	// Create document
	doc := &domain.Document{
		URL:            pageURL,
		Title:          page.Title,
		Content:        content,
		FetchedAt:      time.Now(),
		WordCount:      len(strings.Fields(content)),
		CharCount:      len(content),
		SourceStrategy: s.Name(),
		RelativePath:   relativePath,
	}

	// Write document
	if !opts.DryRun {
		return s.writer.Write(ctx, doc)
	}

	return nil
}
