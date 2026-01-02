package renderer_test

import (
	"testing"

	"github.com/quantmind-br/repodocs-go/internal/renderer"
	"github.com/stretchr/testify/assert"
)

func TestNeedsJSRendering(t *testing.T) {
	tests := []struct {
		name string
		html string
		want bool
	}{
		{
			name: "static HTML page",
			html: `<!DOCTYPE html>
<html><head><title>Static Page</title></head>
<body><h1>Hello World</h1><p>This is a static page with no JavaScript.</p>
</body></html>`,
			want: false,
		},
		{
			name: "React root div pattern",
			html: `<!DOCTYPE html>
<html><head><title>React App</title></head>
<body><div id="root"></div>
<script src="app.js"></script></body></html>`,
			want: true,
		},
		{
			name: "React self-closing root div",
			html: `<!DOCTYPE html>
<html><head><title>React App</title></head>
<body><div id="root"/></body></html>`,
			want: true,
		},
		{
			name: "React app div pattern",
			html: `<!DOCTYPE html>
<html><head><title>React App</title></head>
<body><div id="app"></div></body></html>`,
			want: true,
		},
		{
			name: "React data-reactroot attribute",
			html: `<!DOCTYPE html>
<html><body><div data-reactroot=""></div></body></html>`,
			want: true,
		},
		{
			name: "React devtools hook",
			html: `<!DOCTYPE html>
<html><body><script>__REACT_DEVTOOLS_GLOBAL_HOOK__ = {};</script></body></html>`,
			want: true,
		},
		{
			name: "Vue app div pattern",
			html: `<!DOCTYPE html>
<html><body><div id="app"></div>
<script src="vue.js"></script></body></html>`,
			want: true,
		},
		{
			name: "Vue v-cloak directive",
			html: `<!DOCTYPE html>
<html><body><div v-cloak>{{ message }}</div></body></html>`,
			want: true,
		},
		{
			name: "Vue global object",
			html: `<!DOCTYPE html>
<html><body><script>window.__VUE__ = {};</script></body></html>`,
			want: true,
		},
		{
			name: "Next.js div pattern",
			html: `<!DOCTYPE html>
<html><body><div id="__next"></div></body></html>`,
			want: true,
		},
		{
			name: "Next.js data script",
			html: `<!DOCTYPE html>
<html><body><script id="__NEXT_DATA__" type="application/json">{}",
			want: true,
		},
		{
			name: "Next.js static files",
			html: `<!DOCTYPE html>
<html><body><script src="/_next/static/chunks/main.js"></script></body></html>`,
			want: true,
		},
		{
			name: "Nuxt global object",
			html: `<!DOCTYPE html>
<html><body><script>window.__NUXT__ = {};</script></body></html>`,
			want: true,
		},
		{
			name: "Nuxt div pattern",
			html: `<!DOCTYPE html>
<html><body><div id="__nuxt"></div></body></html>`,
			want: true,
		},
		{
			name: "Angular ng-version attribute",
			html: `<!DOCTYPE html>
<html><body><app-root ng-version="15.0.0"></app-root></body></html>`,
			want: true,
		},
		{
			name: "Angular ng-app directive",
			html: `<!DOCTYPE html>
<html><body><div ng-app="myApp"></div></body></html>`,
			want: true,
		},
		{
			name: "Angular ng-controller directive",
			html: `<!DOCTYPE html>
<html><body><div ng-controller="MyController"></div></body></html>`,
			want: true,
		},
		{
			name: "Angular app-root element",
			html: `<!DOCTYPE html>
<html><body><app-root></app-root></body></html>`,
			want: true,
		},
		{
			name: "Svelte global object",
			html: `<!DOCTYPE html>
<html><body><script>window.__svelte = {};</script></body></html>`,
			want: true,
		},
		{
			name: "Svelte class prefix",
			html: `<!DOCTYPE html>
<html><body><div class="svelte-1a2b3c"></div></body></html>`,
			want: true,
		},
		{
			name: "Generic SPA initial state",
			html: `<!DOCTYPE html>
<html><body><script>window.__INITIAL_STATE__ = {};</script></body></html>`,
			want: true,
		},
		{
			name: "Generic SPA state object",
			html: `<!DOCTYPE html>
<html><body><script>window.__STATE__ = {};</script></body></html>`,
			want: true,
		},
		{
			name: "Generic SPA preloaded state",
			html: `<!DOCTYPE html>
<html><body><script>window.__PRELOADED_STATE__ = {};</script></body></html>`,
			want: true,
		},
		{
			name: "minimal content with many scripts",
			html: `<!DOCTYPE html>
<html><head>
<script src="bundle1.js"></script>
<script src="bundle2.js"></script>
<script src="bundle3.js"></script>
<script src="bundle4.js"></script>
</head>
<body><div>Hi</div></body></html>`,
			want: true,
		},
		{
			name: "short content below threshold with 3 scripts",
			html: `<!DOCTYPE html>
<html><head>
<script src="script1.js"></script>
<script src="script2.js"></script>
<script src="script3.js"></script>
</head>
<body><p>Short content</p></body></html>`,
			want: true,
		},
		{
			name: "short content with 2 scripts - not SPA",
			html: `<!DOCTYPE html>
<html><head>
<script src="script1.js"></script>
<script src="script2.js"></script>
</head>
<body><p>Short content</p></body></html>`,
			want: false,
		},
		{
			name: "long content without SPA patterns",
			html: `<!DOCTYPE html>
<html><body>
<h1>Very Long Article Title That Goes On and On</h1>
<p>Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.
Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.
Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur.
Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.</p>
<p>Sed ut perspiciatis unde omnis iste natus error sit voluptatem accusantium doloremque laudantium, totam rem aperiam.
Eaque ipsa quae ab illo inventore veritatis et quasi architecto beatae vitae dicta sunt explicabo.</p>
<p>Nemo enim ipsam voluptatem quia voluptas sit aspernatur aut odit aut fugit, sed quia consequuntur magni dolores eos qui ratione voluptatem sequi nesciunt.</p>
</body></html>`,
			want: false,
		},
		{
			name: "empty HTML",
			html: "",
			want: false,
		},
		{
			name: "HTML with only whitespace",
			html: "   \n\t  \n  ",
			want: false,
		},
		{
			name: "HTML with only script tags and no content",
			html: `<!DOCTYPE html>
<html><head><script src="app.js"></script></head><body></body></html>`,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderer.NeedsJSRendering(tt.html)
			assert.Equal(t, tt.want, got, "NeedsJSRendering() = %v, want %v", got, tt.want)
		})
	}
}

func TestDetectFramework(t *testing.T) {
	tests := []struct {
		name string
		html string
		want string
	}{
		{
			name: "Next.js detection",
			html: `<!DOCTYPE html>
<html><body><div id="__next"></div></body></html>`,
			want: "Next.js",
		},
		{
			name: "Next.js with data script",
			html: `<!DOCTYPE html>
<html><body><script id="__NEXT_DATA__">{}",
			want: "Next.js",
		},
		{
			name: "Next.js with static files",
			html: `<!DOCTYPE html>
<html><body><script src="/_next/static/chunks/main.js"></script></body></html>`,
			want: "Next.js",
		},
		{
			name: "Nuxt detection",
			html: `<!DOCTYPE html>
<html><body><script>window.__NUXT__ = {};</script></body></html>`,
			want: "Nuxt",
		},
		{
			name: "Nuxt with div",
			html: `<!DOCTYPE html>
<html><body><div id="__nuxt"></div></body></html>`,
			want: "Nuxt",
		},
		{
			name: "React detection",
			html: `<!DOCTYPE html>
<html><body><div id="root"></div></body></html>`,
			want: "React",
		},
		{
			name: "React with self-closing tag",
			html: `<!DOCTYPE html>
<html><body><div id="root"/></body></html>`,
			want: "React",
		},
		{
			name: "React with data-reactroot",
			html: `<!DOCTYPE html>
<html><body><div data-reactroot=""></div></body></html>`,
			want: "React",
		},
		{
			name: "React with devtools hook",
			html: `<!DOCTYPE html>
<html><body><script>__REACT_DEVTOOLS_GLOBAL_HOOK__ = {};</script></body></html>`,
			want: "React",
		},
		{
			name: "Vue detection",
			html: `<!DOCTYPE html>
<html><body><div id="app"></div></body></html>`,
			want: "Vue",
		},
		{
			name: "Vue with v-cloak",
			html: `<!DOCTYPE html>
<html><body><div v-cloak>{{ message }}</div></body></html>`,
			want: "Vue",
		},
		{
			name: "Vue with global object",
			html: `<!DOCTYPE html>
<html><body><script>window.__VUE__ = {};</script></body></html>`,
			want: "Vue",
		},
		{
			name: "Angular detection",
			html: `<!DOCTYPE html>
<html><body><app-root ng-version="15.0.0"></app-root></body></html>`,
			want: "Angular",
		},
		{
			name: "Angular with ng-app",
			html: `<!DOCTYPE html>
<html><body><div ng-app="myApp"></div></body></html>`,
			want: "Angular",
		},
		{
			name: "Angular with ng-controller",
			html: `<!DOCTYPE html>
<html><body><div ng-controller="MyController"></div></body></html>`,
			want: "Angular",
		},
		{
			name: "Angular with app-root element",
			html: `<!DOCTYPE html>
<html><body><app-root></app-root></body></html>`,
			want: "Angular",
		},
		{
			name: "Svelte detection",
			html: `<!DOCTYPE html>
<html><body><script>window.__svelte = {};</script></body></html>`,
			want: "Svelte",
		},
		{
			name: "Svelte with class prefix",
			html: `<!DOCTYPE html>
<html><body><div class="svelte-1a2b3c"></div></body></html>`,
			want: "Svelte",
		},
		{
			name: "unknown framework - static HTML",
			html: `<!DOCTYPE html>
<html><body><h1>Static Page</h1><p>No framework detected.</p></body></html>`,
			want: "Unknown",
		},
		{
			name: "unknown framework - generic SPA",
			html: `<!DOCTYPE html>
<html><body><script>window.__INITIAL_STATE__ = {};</script></body></html>`,
			want: "Unknown",
		},
		{
			name: "unknown framework - empty HTML",
			html: "",
			want: "Unknown",
		},
		{
			name: "Next.js takes precedence over React",
			html: `<!DOCTYPE html>
<html><body><div id="__next"></div><div id="root"></div></body></html>`,
			want: "Next.js",
		},
		{
			name: "Nuxt takes precedence over Vue",
			html: `<!DOCTYPE html>
<html><body><div id="__nuxt"></div><div id="app"></div></body></html>`,
			want: "Nuxt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderer.DetectFramework(tt.html)
			assert.Equal(t, tt.want, got, "DetectFramework() = %v, want %v", got, tt.want)
		})
	}
}

func TestHasDynamicContent(t *testing.T) {
	tests := []struct {
		name string
		html string
		want bool
	}{
		{
			name: "loading text indicator",
			html: `<!DOCTYPE html>
<html><body><p>Loading...</p></body></html>`,
			want: true,
		},
		{
			name: "loading ellipsis indicator",
			html: `<!DOCTYPE html>
<html><body><p>Loading…</p></body></html>`,
			want: true,
		},
		{
			name: "please wait indicator",
			html: `<!DOCTYPE html>
<html><body><p>Please wait while we load your content.</p></body></html>`,
			want: true,
		},
		{
			name: "spinner class",
			html: `<!DOCTYPE html>
<html><body><div class="spinner"></div></body></html>`,
			want: true,
		},
		{
			name: "skeleton loader class",
			html: `<!DOCTYPE html>
<html><body><div class="skeleton-loader"></div></body></html>`,
			want: true,
		},
		{
			name: "lazy-load attribute",
			html: `<!DOCTYPE html>
<html><body><img src="image.jpg" lazy-load="true" /></body></html>`,
			want: true,
		},
		{
			name: "lazyload class",
			html: `<!DOCTYPE html>
<html><body><img src="image.jpg" class="lazyload" /></body></html>`,
			want: true,
		},
		{
			name: "infinite-scroll class",
			html: `<!DOCTYPE html>
<html><body><div class="infinite-scroll"></div></body></html>`,
			want: true,
		},
		{
			name: "no dynamic content indicators",
			html: `<!DOCTYPE html>
<html><body><h1>Static Content</h1><p>This page is fully rendered.</p></body></html>`,
			want: false,
		},
		{
			name: "empty HTML",
			html: "",
			want: false,
		},
		{
			name: "case insensitive - LOADING",
			html: `<!DOCTYPE html>
<html><body><p>LOADING CONTENT...</p></body></html>`,
			want: true,
		},
		{
			name: "case insensitive - Spinner",
			html: `<!DOCTYPE html>
<html><body><div class="Spinner"></div></body></html>`,
			want: true,
		},
		{
			name: "mixed case indicators",
			html: `<!DOCTYPE html>
<html><body><div class="Skeleton-Loader"></div></body></html>`,
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderer.HasDynamicContent(tt.html)
			assert.Equal(t, tt.want, got, "HasDynamicContent() = %v, want %v", got, tt.want)
		})
	}
}

func TestNeedsJSRendering_EdgeCases(t *testing.T) {
	tests := []struct {
		name string
		html string
		want bool
	}{
		{
			name: "mixed case React pattern",
			html: `<!DOCTYPE html>
<html><body><DIV ID="ROOT"></DIV></body></html>`,
			want: true,
		},
		{
			name: "mixed case Next.js pattern",
			html: `<!DOCTYPE html>
<html><body><DIV ID="__NEXT"></DIV></body></html>`,
			want: true,
		},
		{
			name: "script tags with HTML entities in content",
			html: `<!DOCTYPE html>
<html><body><p>&lt;script&gt;alert('test')&lt;/script&gt;</p>
<h1>Actual content here</h1><p>More text that exceeds the minimum length</p>
<p>Even more text to ensure we have enough content</p></body></html>`,
			want: false,
		},
		{
			name: "exactly at content threshold (500 chars)",
			html: `<!DOCTYPE html>
<html><body><p>Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum. Sed ut perspiciatis unde omnis iste natus error sit voluptatem accusantium doloremque laudantium, totam rem aperiam, eaque ipsa quae ab illo inventore veritatis et quasi architecto beatae vitae dicta sunt explicabo. Nemo enim ipsam voluptatem quia voluptas sit aspernatur aut odit aut fugit, sed quia consequuntur magni dolores eos qui ratione voluptatem sequi nesciunt. Neque porro quisquam est, qui dolorem ipsum quia dolor sit amet, consectetur, adipisci velit, sed quia non numquam eius modi tempora incidunt ut labore et dolore magnam aliquam quaerat voluptatem. Ut enim ad minima veniam, quis nostrum exercitationem ullam corporis suscipit laboriosam, nisi ut aliquid ex ea commodi consequatur? Quis autem vel eum iure reprehenderit qui in ea voluptate velit esse quam nihil molestiae consequatur.</p></body></html>`,
			want: false,
		},
		{
			name: "just below content threshold (499 chars)",
			html: `<!DOCTYPE html>
<html><body><p>Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum. Sed ut perspiciatis unde omnis iste natus error sit voluptatem accusantium doloremque laudantium, totam rem aperiam, eaque ipsa quae ab illo inventore veritatis et quasi architecto beatae vitae dicta sunt explicabo. Nemo enim ipsam voluptatem quia voluptas sit aspernatur aut odit aut fugit, sed quia consequuntur magni dolores eos qui ratione voluptatem sequi nesciunt. Neque porro quisquam est, qui dolorem ipsum quia dolor sit amet, consectetur, adipisci velit.</p></body></html>`,
			want: false,
		},
		{
			name: "just below threshold with 4 scripts",
			html: `<!DOCTYPE html>
<html><head><script src="1.js"></script><script src="2.js"></script><script src="3.js"></script><script src="4.js"></script></head>
<body><p>Short content below threshold.</p></body></html>`,
			want: true,
		},
		{
			name: "malformed HTML with SPA pattern",
			html: `<div id="root"><p>Unclosed tags`,
			want: true,
		},
		{
			name: "HTML comments with SPA patterns",
			html: `<!DOCTYPE html>
<html><body><!-- <div id="root"> -->
<h1>Static Page</h1></body></html>`,
			want: false, // Comments should not trigger SPA detection
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderer.NeedsJSRendering(tt.html)
			assert.Equal(t, tt.want, got, "NeedsJSRendering() = %v, want %v", got, tt.want)
		})
	}
}

func TestDetectFramework_Priority(t *testing.T) {
	tests := []struct {
		name string
		html string
		want string
	}{
		{
			name: "Next.js detected before React when both present",
			html: `<!DOCTYPE html>
<html><body>
<div id="__next"></div>
<div id="root"></div>
</body></html>`,
			want: "Next.js",
		},
		{
			name: "Nuxt detected before Vue when both present",
			html: `<!DOCTYPE html>
<html><body>
<div id="__nuxt"></div>
<div id="app" v-cloak></div>
</body></html>`,
			want: "Nuxt",
		},
		{
			name: "React detected when only React pattern present",
			html: `<!DOCTYPE html>
<html><body>
<div id="root"></div>
<div id="app"></div>
</body></html>`,
			want: "React",
		},
		{
			name: "Vue detected when only Vue pattern present",
			html: `<!DOCTYPE html>
<html><body>
<div id="app" v-cloak></div>
</body></html>`,
			want: "Vue",
		},
		{
			name: "Angular detected uniquely",
			html: `<!DOCTYPE html>
<html><body>
<app-root ng-version="15.0.0"></app-root>
</body></html>`,
			want: "Angular",
		},
		{
			name: "Svelte detected uniquely",
			html: `<!DOCTYPE html>
<html><body>
<div class="svelte-1a2b3c"></div>
</body></html>`,
			want: "Svelte",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderer.DetectFramework(tt.html)
			assert.Equal(t, tt.want, got, "DetectFramework() = %v, want %v", got, tt.want)
		})
	}
}
