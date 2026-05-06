// Package strategies provides documentation extraction strategies for different
// source types.
//
// Strategy implementations share common dependencies and cover crawler, git,
// sitemap, docs.rs, pkg.go.dev, GitHub Pages, wiki, and llms.txt sources. The
// app detector chooses among them in the configured detection order.
package strategies
