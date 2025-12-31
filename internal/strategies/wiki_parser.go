package strategies

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type WikiPage struct {
	Filename  string
	Title     string
	Content   string
	Section   string
	Order     int
	Links     []string
	IsHome    bool
	IsSpecial bool
}

type WikiStructure struct {
	Sections   []WikiSection
	Pages      map[string]*WikiPage
	HasSidebar bool
}

type WikiSection struct {
	Name  string
	Order int
	Pages []string
}

type WikiInfo struct {
	Owner      string
	Repo       string
	CloneURL   string
	Platform   string
	TargetPage string
}

func ParseWikiURL(rawURL string) (*WikiInfo, error) {
	url := strings.TrimSuffix(rawURL, "/")

	// github.com/{owner}/{repo}/wiki[/{page}] or {repo}.wiki.git
	wikiPattern := regexp.MustCompile(
		`github\.com[:/]([^/]+)/([^/]+?)(?:\.wiki)?(?:/wiki)?(?:/([^/]+))?(?:\.git)?$`,
	)

	matches := wikiPattern.FindStringSubmatch(url)
	if len(matches) < 3 {
		return nil, fmt.Errorf("invalid wiki URL format: %s", rawURL)
	}

	owner := matches[1]
	repo := strings.TrimSuffix(matches[2], ".wiki")

	var targetPage string
	if len(matches) > 3 && matches[3] != "" {
		targetPage = matches[3]
	}

	cloneURL := fmt.Sprintf("https://github.com/%s/%s.wiki.git", owner, repo)

	return &WikiInfo{
		Owner:      owner,
		Repo:       repo,
		CloneURL:   cloneURL,
		Platform:   "github",
		TargetPage: targetPage,
	}, nil
}

func FilenameToTitle(filename string) string {
	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	name = strings.ReplaceAll(name, "-", " ")
	name = strings.ReplaceAll(name, "_", " ")

	words := strings.Fields(name)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}

	return strings.Join(words, " ")
}

func TitleToFilename(title string) string {
	return strings.ReplaceAll(title, " ", "-")
}

func ParseSidebarContent(content string, pages map[string]*WikiPage) []WikiSection {
	var sections []WikiSection
	var currentSection *WikiSection

	lines := strings.Split(content, "\n")
	sectionOrder := 0
	pageOrder := 0

	headerPattern := regexp.MustCompile(`^#+\s*(.+)$`)
	wikiLinkPattern := regexp.MustCompile(`\[\[([^\]|]+)(?:\|[^\]]+)?\]\]`)
	mdLinkPattern := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if matches := headerPattern.FindStringSubmatch(trimmed); len(matches) > 1 {
			if currentSection != nil && len(currentSection.Pages) > 0 {
				sections = append(sections, *currentSection)
			}

			sectionOrder++
			pageOrder = 0
			currentSection = &WikiSection{
				Name:  strings.TrimSpace(matches[1]),
				Order: sectionOrder,
				Pages: []string{},
			}
			continue
		}

		if wikiMatches := wikiLinkPattern.FindAllStringSubmatch(trimmed, -1); len(wikiMatches) > 0 {
			for _, match := range wikiMatches {
				pageName := match[1]
				filename := findPageFilename(pageName, pages)
				if filename != "" {
					pageOrder++
					if page, exists := pages[filename]; exists {
						page.Section = currentSection.Name
						page.Order = pageOrder
					}
					if currentSection != nil {
						currentSection.Pages = append(currentSection.Pages, filename)
					}
				}
			}
			continue
		}

		if mdMatches := mdLinkPattern.FindAllStringSubmatch(trimmed, -1); len(mdMatches) > 0 {
			for _, match := range mdMatches {
				pageName := match[2]
				pageName = strings.TrimSuffix(pageName, ".md")
				filename := findPageFilename(pageName, pages)
				if filename != "" {
					pageOrder++
					if page, exists := pages[filename]; exists {
						page.Section = currentSection.Name
						page.Order = pageOrder
					}
					if currentSection != nil {
						currentSection.Pages = append(currentSection.Pages, filename)
					}
				}
			}
		}
	}

	if currentSection != nil && len(currentSection.Pages) > 0 {
		sections = append(sections, *currentSection)
	}

	return sections
}

func findPageFilename(pageName string, pages map[string]*WikiPage) string {
	if _, exists := pages[pageName+".md"]; exists {
		return pageName + ".md"
	}

	hyphenated := strings.ReplaceAll(pageName, " ", "-") + ".md"
	if _, exists := pages[hyphenated]; exists {
		return hyphenated
	}

	for filename := range pages {
		baseName := strings.TrimSuffix(filename, ".md")
		if strings.EqualFold(baseName, pageName) ||
			strings.EqualFold(baseName, strings.ReplaceAll(pageName, " ", "-")) {
			return filename
		}
	}

	return ""
}

func CreateDefaultStructure(pages map[string]*WikiPage) []WikiSection {
	var pageNames []string
	for filename, page := range pages {
		if !page.IsSpecial {
			pageNames = append(pageNames, filename)
		}
	}

	sort.Strings(pageNames)

	for i, name := range pageNames {
		if strings.EqualFold(name, "Home.md") {
			pageNames = append([]string{name}, append(pageNames[:i], pageNames[i+1:]...)...)
			break
		}
	}

	for i, filename := range pageNames {
		if page, exists := pages[filename]; exists {
			page.Order = i + 1
			page.Section = "Documentation"
		}
	}

	return []WikiSection{
		{
			Name:  "Documentation",
			Order: 1,
			Pages: pageNames,
		},
	}
}

func ConvertWikiLinks(content string, _ map[string]*WikiPage) string {
	// [[Page Name|Custom Text]] -> [Custom Text](./page-name.md)
	pattern1 := regexp.MustCompile(`\[\[([^\]|]+)\|([^\]]+)\]\]`)
	content = pattern1.ReplaceAllStringFunc(content, func(match string) string {
		matches := pattern1.FindStringSubmatch(match)
		if len(matches) == 3 {
			pageName := matches[1]
			linkText := matches[2]
			filename := TitleToFilename(pageName) + ".md"
			return fmt.Sprintf("[%s](./%s)", linkText, strings.ToLower(filename))
		}
		return match
	})

	// [[Page Name#Section]] -> [Page Name](./page-name.md#section)
	pattern2 := regexp.MustCompile(`\[\[([^\]#]+)#([^\]]+)\]\]`)
	content = pattern2.ReplaceAllStringFunc(content, func(match string) string {
		matches := pattern2.FindStringSubmatch(match)
		if len(matches) == 3 {
			pageName := matches[1]
			section := matches[2]
			filename := TitleToFilename(pageName) + ".md"
			anchor := strings.ToLower(strings.ReplaceAll(section, " ", "-"))
			return fmt.Sprintf("[%s](./%s#%s)", pageName, strings.ToLower(filename), anchor)
		}
		return match
	})

	// [[Page Name]] -> [Page Name](./page-name.md)
	pattern3 := regexp.MustCompile(`\[\[([^\]]+)\]\]`)
	content = pattern3.ReplaceAllStringFunc(content, func(match string) string {
		matches := pattern3.FindStringSubmatch(match)
		if len(matches) == 2 {
			pageName := matches[1]
			filename := TitleToFilename(pageName) + ".md"
			return fmt.Sprintf("[%s](./%s)", pageName, strings.ToLower(filename))
		}
		return match
	})

	return content
}

func BuildRelativePath(page *WikiPage, structure *WikiStructure, flat bool) string {
	if page.IsHome {
		return "index.md"
	}

	if flat || len(structure.Sections) == 0 || page.Section == "" {
		return strings.ToLower(page.Filename)
	}

	sectionDir := strings.ToLower(strings.ReplaceAll(page.Section, " ", "-"))
	filename := strings.ToLower(page.Filename)

	return filepath.Join(sectionDir, filename)
}
