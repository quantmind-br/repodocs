package strategies

import (
	"bytes"
	"context"
	"net/url"
	"strings"
	"sync"

	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/utils"
)

type SitemapDiscoveryResult struct {
	SitemapURL string
	Method     string
}

type SitemapProbe struct {
	Path string
	Name string
}

func ParseRobotsTxt(content []byte, baseURL string) []string {
	urls := make([]string, 0)
	if len(content) == 0 {
		return urls
	}

	baseParsed, _ := url.Parse(baseURL)

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSuffix(line, "\r")

		if idx := strings.Index(line, "#"); idx >= 0 {
			line = line[:idx]
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if len(line) < len("sitemap:") || !strings.EqualFold(line[:len("sitemap:")], "sitemap:") {
			continue
		}

		raw := strings.TrimSpace(line[len("sitemap:"):])
		if raw == "" {
			continue
		}

		parsedURL, err := url.Parse(raw)
		if err != nil {
			continue
		}

		if baseParsed != nil {
			urls = append(urls, baseParsed.ResolveReference(parsedURL).String())
			continue
		}

		if parsedURL.IsAbs() {
			urls = append(urls, parsedURL.String())
		}
	}

	return urls
}

func IsSitemapContent(body []byte) bool {
	if len(body) == 0 {
		return false
	}

	segment := body
	if len(segment) > 1024 {
		segment = segment[:1024]
	}

	bom := []byte{0xEF, 0xBB, 0xBF}
	segment = bytes.TrimPrefix(segment, bom)
	segment = bytes.TrimLeft(segment, " \t\r\n")
	if len(segment) == 0 {
		return false
	}

	lower := bytes.ToLower(segment)
	return bytes.Contains(lower, []byte("<urlset")) || bytes.Contains(lower, []byte("<sitemapindex"))
}

func GetSitemapProbes() []SitemapProbe {
	return []SitemapProbe{
		{"/robots.txt", "robots.txt"},
		{"/sitemap.xml", "sitemap.xml"},
		{"/sitemap-0.xml", "sitemap-0.xml"},
		{"/sitemap_index.xml", "sitemap_index.xml"},
		{"/sitemap/sitemap-index.xml", "sitemap-index.xml"},
		{"/server-sitemap.xml", "server-sitemap.xml"},
	}
}

func DiscoverSitemap(ctx context.Context, fetcher domain.Fetcher, baseURL string, logger *utils.Logger) (*SitemapDiscoveryResult, error) {
	probes := GetSitemapProbes()

	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	origin := parsed.Scheme + "://" + parsed.Host

	type probeResult struct {
		priority   int
		sitemapURL string
		method     string
	}

	results := make(chan probeResult, len(probes))
	var wg sync.WaitGroup

	for i, probe := range probes {
		wg.Add(1)
		go func(priority int, p SitemapProbe) {
			defer wg.Done()

			select {
			case <-ctx.Done():
				return
			default:
			}

			probeURL := origin + p.Path
			resp, err := fetcher.Get(ctx, probeURL)
			if err != nil {
				if logger != nil {
					logger.Debug().Str("probe", p.Name).Str("url", probeURL).Err(err).Msg("Sitemap probe failed")
				}
				return
			}

			if resp.StatusCode != 200 {
				if logger != nil {
					logger.Debug().Str("probe", p.Name).Int("status", resp.StatusCode).Msg("Sitemap probe returned non-200")
				}
				return
			}

			if p.Path == "/robots.txt" {
				sitemapURLs := ParseRobotsTxt(resp.Body, origin)
				if len(sitemapURLs) == 0 {
					if logger != nil {
						logger.Debug().Str("probe", p.Name).Msg("No sitemap directive found in robots.txt")
					}
					return
				}

				if logger != nil {
					logger.Debug().Str("probe", p.Name).Str("sitemap_url", sitemapURLs[0]).Msg("Sitemap discovered via robots.txt")
				}

				select {
				case <-ctx.Done():
					return
				case results <- probeResult{priority: priority, sitemapURL: sitemapURLs[0], method: "robots.txt"}:
				}
				return
			}

			if !IsSitemapContent(resp.Body) {
				if logger != nil {
					logger.Debug().Str("probe", p.Name).Msg("Probe content does not look like sitemap XML")
				}
				return
			}

			if logger != nil {
				logger.Debug().Str("probe", p.Name).Str("sitemap_url", probeURL).Msg("Sitemap discovered via direct probe")
			}

			select {
			case <-ctx.Done():
				return
			case results <- probeResult{priority: priority, sitemapURL: probeURL, method: "probe:" + p.Path}:
			}
		}(i, probe)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var best *probeResult
	for r := range results {
		if best == nil || r.priority < best.priority {
			best = &r
		}
	}

	if best == nil {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return nil, nil
	}

	return &SitemapDiscoveryResult{
		SitemapURL: best.sitemapURL,
		Method:     best.method,
	}, nil
}
