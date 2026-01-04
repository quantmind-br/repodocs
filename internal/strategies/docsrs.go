package strategies

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/output"
	"github.com/quantmind-br/repodocs-go/internal/utils"
)

type DocsRSURL struct {
	CrateName    string
	Version      string
	ModulePath   string
	IsCratePage  bool
	IsSourceView bool
}

type DocsRSStrategy struct {
	deps     *Dependencies
	fetcher  domain.Fetcher
	writer   *output.Writer
	logger   *utils.Logger
	baseHost string
}

func NewDocsRSStrategy(deps *Dependencies) *DocsRSStrategy {
	if deps == nil {
		return &DocsRSStrategy{baseHost: "docs.rs"}
	}
	return &DocsRSStrategy{
		deps:     deps,
		fetcher:  deps.Fetcher,
		writer:   deps.Writer,
		logger:   deps.Logger,
		baseHost: "docs.rs",
	}
}

func (s *DocsRSStrategy) Name() string {
	return "docsrs"
}

func (s *DocsRSStrategy) SetFetcher(f domain.Fetcher) {
	s.fetcher = f
}

func (s *DocsRSStrategy) SetBaseHost(host string) {
	s.baseHost = host
}

func (s *DocsRSStrategy) parseURL(rawURL string) (*DocsRSURL, error) {
	return parseDocsRSPathWithHost(rawURL, s.baseHost)
}

func (s *DocsRSStrategy) CanHandle(rawURL string) bool {
	parsed, err := parseDocsRSPath(rawURL)
	if err != nil {
		return false
	}

	if parsed.IsSourceView {
		return false
	}

	return parsed.CrateName != ""
}

func parseDocsRSPath(rawURL string) (*DocsRSURL, error) {
	return parseDocsRSPathWithHost(rawURL, "docs.rs")
}

func parseDocsRSPathWithHost(rawURL, expectedHost string) (*DocsRSURL, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	if !strings.Contains(u.Host, expectedHost) {
		return nil, fmt.Errorf("not a docs.rs URL")
	}

	u.Fragment = ""
	u.RawQuery = ""

	segments := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(segments) == 0 || segments[0] == "" {
		return nil, fmt.Errorf("empty path")
	}

	result := &DocsRSURL{}

	if segments[0] == "crate" {
		result.IsCratePage = true
		if len(segments) >= 2 {
			result.CrateName = segments[1]
		}
		if len(segments) >= 3 {
			result.Version = segments[2]
		} else {
			result.Version = "latest"
		}
		if len(segments) >= 4 && (segments[3] == "source" || segments[3] == "src") {
			result.IsSourceView = true
		}
		return result, nil
	}

	for _, seg := range segments {
		if seg == "src" || seg == "source" {
			result.IsSourceView = true
		}
	}

	result.CrateName = segments[0]

	if len(segments) >= 2 {
		result.Version = segments[1]
	} else {
		result.Version = "latest"
	}

	if len(segments) >= 4 {
		result.ModulePath = strings.Join(segments[3:], "/")
	}

	return result, nil
}

func (s *DocsRSStrategy) Execute(ctx context.Context, rawURL string, opts Options) error {
	s.logger.Info().Str("url", rawURL).Msg("Starting docs.rs JSON extraction")

	if s.fetcher == nil {
		return fmt.Errorf("docsrs strategy fetcher is nil")
	}
	if s.writer == nil {
		return fmt.Errorf("docsrs strategy writer is nil")
	}

	baseInfo, err := s.parseURL(rawURL)
	if err != nil {
		return fmt.Errorf("invalid docs.rs URL: %w", err)
	}

	s.logger.Info().
		Str("crate", baseInfo.CrateName).
		Str("version", baseInfo.Version).
		Msg("Parsed docs.rs URL")

	index, err := s.fetchRustdocJSON(ctx, baseInfo.CrateName, baseInfo.Version)
	if err != nil {
		return fmt.Errorf("failed to fetch rustdoc JSON: %w", err)
	}

	if err := s.checkFormatVersion(index.FormatVersion); err != nil {
		return err
	}

	renderer := NewRustdocRenderer(index, baseInfo.CrateName, baseInfo.Version)

	items := s.collectItems(index, opts)
	s.logger.Info().Int("count", len(items)).Msg("Collected items to process")

	if opts.Limit > 0 && len(items) > opts.Limit {
		items = items[:opts.Limit]
		s.logger.Info().Int("limit", opts.Limit).Msg("Applied item limit")
	}

	bar := utils.NewProgressBar(len(items), utils.DescExtracting)

	errors := utils.ParallelForEach(ctx, items, opts.Concurrency, func(ctx context.Context, item *RustdocItem) error {
		defer bar.Add(1)
		return s.processItem(ctx, item, renderer, baseInfo, opts)
	})

	if err := utils.FirstError(errors); err != nil {
		return err
	}

	s.logger.Info().Int("items", len(items)).Msg("docs.rs JSON extraction completed")
	return nil
}

func (s *DocsRSStrategy) processItem(ctx context.Context, item *RustdocItem, renderer *RustdocRenderer, baseInfo *DocsRSURL, opts Options) error {
	itemURL := s.buildItemURL(item, baseInfo)

	if !opts.Force && s.writer.Exists(itemURL) {
		return nil
	}

	markdown := renderer.RenderItem(item)
	if markdown == "" {
		return nil
	}

	doc := &domain.Document{
		URL:            itemURL,
		Title:          s.buildItemTitle(item),
		Content:        markdown,
		Description:    s.buildItemDescription(item, baseInfo),
		SourceStrategy: s.Name(),
		FetchedAt:      time.Now(),
		Tags:           s.buildItemTags(item, baseInfo),
	}

	if !opts.DryRun {
		if err := s.deps.WriteDocument(ctx, doc); err != nil {
			s.logger.Warn().Err(err).Str("url", itemURL).Msg("Failed to write document")
			return nil
		}
	}

	return nil
}

func (s *DocsRSStrategy) buildItemURL(item *RustdocItem, baseInfo *DocsRSURL) string {
	name := ""
	if item.Name != nil {
		name = *item.Name
	}

	itemType := ""
	if mod := item.GetModule(); mod != nil {
		if mod.IsCrate {
			return fmt.Sprintf("https://docs.rs/%s/%s/%s/",
				baseInfo.CrateName, baseInfo.Version, baseInfo.CrateName)
		}
		itemType = "mod"
	} else if item.GetStruct() != nil {
		itemType = "struct"
	} else if item.GetEnum() != nil {
		itemType = "enum"
	} else if item.GetTrait() != nil {
		itemType = "trait"
	} else if item.GetFunction() != nil {
		itemType = "fn"
	} else if item.GetTypeAlias() != nil {
		itemType = "type"
	} else if item.GetConstant() != nil {
		itemType = "constant"
	} else if item.GetMacro() != nil {
		itemType = "macro"
	} else {
		itemType = "item"
	}

	path := baseInfo.CrateName
	if item.Span != nil && item.Span.Filename != "" {
		spanPath := strings.TrimPrefix(item.Span.Filename, "src/")
		spanPath = strings.TrimSuffix(spanPath, ".rs")
		spanPath = strings.TrimSuffix(spanPath, "/mod")
		if spanPath != "lib" && spanPath != "" {
			path = baseInfo.CrateName + "/" + strings.ReplaceAll(spanPath, "/", "::")
		}
	}

	return fmt.Sprintf("https://docs.rs/%s/%s/%s/%s.%s.html",
		baseInfo.CrateName, baseInfo.Version, path, itemType, name)
}

func (s *DocsRSStrategy) buildItemTitle(item *RustdocItem) string {
	name := ""
	if item.Name != nil {
		name = *item.Name
	}

	if mod := item.GetModule(); mod != nil {
		if mod.IsCrate {
			return fmt.Sprintf("Crate %s", name)
		}
		return fmt.Sprintf("Module %s", name)
	}
	if item.GetStruct() != nil {
		return fmt.Sprintf("Struct %s", name)
	}
	if item.GetEnum() != nil {
		return fmt.Sprintf("Enum %s", name)
	}
	if item.GetTrait() != nil {
		return fmt.Sprintf("Trait %s", name)
	}
	if item.GetFunction() != nil {
		return fmt.Sprintf("Function %s", name)
	}
	if item.GetTypeAlias() != nil {
		return fmt.Sprintf("Type %s", name)
	}
	if item.GetMacro() != nil {
		return fmt.Sprintf("Macro %s", name)
	}
	return name
}

func (s *DocsRSStrategy) buildItemDescription(item *RustdocItem, baseInfo *DocsRSURL) string {
	itemType := s.getItemTypeName(item)
	stability := "stable"
	if item.Deprecation != nil {
		stability = "deprecated"
	}

	return fmt.Sprintf("crate:%s version:%s type:%s stability:%s",
		baseInfo.CrateName, baseInfo.Version, itemType, stability)
}

func (s *DocsRSStrategy) buildItemTags(item *RustdocItem, baseInfo *DocsRSURL) []string {
	itemType := s.getItemTypeName(item)

	tags := []string{
		"docs.rs",
		"rust",
		baseInfo.CrateName,
		itemType,
	}

	if item.Deprecation != nil {
		tags = append(tags, "deprecated")
	}

	return tags
}

func (s *DocsRSStrategy) getItemTypeName(item *RustdocItem) string {
	if item.GetModule() != nil {
		return "module"
	}
	if item.GetStruct() != nil {
		return "struct"
	}
	if item.GetEnum() != nil {
		return "enum"
	}
	if item.GetTrait() != nil {
		return "trait"
	}
	if item.GetFunction() != nil {
		return "function"
	}
	if item.GetTypeAlias() != nil {
		return "type"
	}
	if item.GetMacro() != nil {
		return "macro"
	}
	return "item"
}

func ParseDocsRSPathForTest(rawURL string) (*DocsRSURL, error) {
	return parseDocsRSPath(rawURL)
}

// BuildItemURLForTest exposes buildItemURL for testing
func (s *DocsRSStrategy) BuildItemURLForTest(item *RustdocItem, baseInfo *DocsRSURL) string {
	return s.buildItemURL(item, baseInfo)
}

// BuildItemTitleForTest exposes buildItemTitle for testing
func (s *DocsRSStrategy) BuildItemTitleForTest(item *RustdocItem) string {
	return s.buildItemTitle(item)
}

// GetItemTypeNameForTest exposes getItemTypeName for testing
func (s *DocsRSStrategy) GetItemTypeNameForTest(item *RustdocItem) string {
	return s.getItemTypeName(item)
}

// BuildItemDescriptionForTest exposes buildItemDescription for testing
func (s *DocsRSStrategy) BuildItemDescriptionForTest(item *RustdocItem, baseInfo *DocsRSURL) string {
	return s.buildItemDescription(item, baseInfo)
}

// BuildItemTagsForTest exposes buildItemTags for testing
func (s *DocsRSStrategy) BuildItemTagsForTest(item *RustdocItem, baseInfo *DocsRSURL) []string {
	return s.buildItemTags(item, baseInfo)
}
