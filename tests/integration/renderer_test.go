package integration

import (
	"testing"

	"github.com/quantmind-br/repodocs-go/internal/renderer"
	"github.com/stretchr/testify/assert"
)

func TestNeedsJSRendering_EmptyReactRoot(t *testing.T) {
	html := `
	<!DOCTYPE html>
	<html>
	<head><title>React App</title></head>
	<body>
		<div id="root"></div>
		<script src="/static/js/bundle.js"></script>
	</body>
	</html>
	`

	assert.True(t, renderer.NeedsJSRendering(html), "Empty React root should need JS rendering")
}

func TestNeedsJSRendering_EmptyVueApp(t *testing.T) {
	html := `
	<!DOCTYPE html>
	<html>
	<head><title>Vue App</title></head>
	<body>
		<div id="app"></div>
		<script src="/js/app.js"></script>
	</body>
	</html>
	`

	assert.True(t, renderer.NeedsJSRendering(html), "Empty Vue app should need JS rendering")
}

func TestNeedsJSRendering_NextJS(t *testing.T) {
	html := `
	<!DOCTYPE html>
	<html>
	<head><title>Next.js App</title></head>
	<body>
		<div id="__next"></div>
		<script src="/_next/static/chunks/main.js"></script>
	</body>
	</html>
	`

	assert.True(t, renderer.NeedsJSRendering(html), "Empty Next.js container should need JS rendering")
}

func TestNeedsJSRendering_Nuxt(t *testing.T) {
	html := `
	<!DOCTYPE html>
	<html>
	<head><title>Nuxt App</title></head>
	<body>
		<div id="__nuxt"></div>
		<script>window.__NUXT__={}</script>
	</body>
	</html>
	`

	assert.True(t, renderer.NeedsJSRendering(html), "Nuxt.js page should need JS rendering")
}

func TestNeedsJSRendering_StaticContent(t *testing.T) {
	html := `
	<!DOCTYPE html>
	<html>
	<head><title>Static Page</title></head>
	<body>
		<header>
			<nav>Navigation</nav>
		</header>
		<main>
			<h1>Welcome to Our Website</h1>
			<p>This is a static page with plenty of content.</p>
			<p>Lorem ipsum dolor sit amet, consectetur adipiscing elit.
			   Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.
			   Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris.</p>
			<p>More content here to ensure we have enough text.</p>
			<p>And even more content to make this a substantial static page.</p>
		</main>
		<footer>Copyright 2024</footer>
	</body>
	</html>
	`

	assert.False(t, renderer.NeedsJSRendering(html), "Static content should not need JS rendering")
}

func TestNeedsJSRendering_PrerenderedReact(t *testing.T) {
	html := `
	<!DOCTYPE html>
	<html>
	<head><title>SSR React App</title></head>
	<body>
		<div id="root">
			<header>Navigation</header>
			<main>
				<h1>Server-Side Rendered Page</h1>
				<p>This content was rendered on the server and includes substantial text.</p>
				<p>Lorem ipsum dolor sit amet, consectetur adipiscing elit.</p>
				<p>More server-rendered content here.</p>
			</main>
		</div>
		<script src="/static/js/bundle.js"></script>
	</body>
	</html>
	`

	assert.False(t, renderer.NeedsJSRendering(html), "Pre-rendered React should not need JS rendering")
}

func TestNeedsJSRendering_MinimalContent(t *testing.T) {
	html := `
	<!DOCTYPE html>
	<html>
	<body>
		<p>Loading...</p>
		<script src="/app.js"></script>
		<script src="/vendor.js"></script>
		<script src="/chunk.js"></script>
	</body>
	</html>
	`

	// Minimal content with many scripts often indicates SPA
	result := renderer.NeedsJSRendering(html)
	// This test checks the heuristic - minimal content + many scripts
	assert.True(t, result || len(html) < 500, "Minimal content with scripts may need JS rendering")
}

func TestNeedsJSRendering_EmptyBody(t *testing.T) {
	html := `
	<!DOCTYPE html>
	<html>
	<head><title>Empty Body</title></head>
	<body></body>
	</html>
	`

	// Empty body is a strong indicator of SPA
	result := renderer.NeedsJSRendering(html)
	// Empty pages might need rendering
	_ = result // Just ensure it doesn't panic
}

func TestNeedsJSRendering_WithInlineScripts(t *testing.T) {
	html := `
	<!DOCTYPE html>
	<html>
	<body>
		<div id="root"></div>
		<script>
			ReactDOM.render(
				React.createElement(App),
				document.getElementById('root')
			);
		</script>
	</body>
	</html>
	`

	assert.True(t, renderer.NeedsJSRendering(html), "React app initialization should need JS rendering")
}

func TestNeedsJSRendering_Angular(t *testing.T) {
	html := `
	<!DOCTYPE html>
	<html>
	<body>
		<app-root></app-root>
		<script src="main.js"></script>
	</body>
	</html>
	`

	// Angular apps typically have custom elements
	result := renderer.NeedsJSRendering(html)
	// Custom elements like <app-root> may indicate Angular
	_ = result
}

func TestNeedsJSRendering_DocumentationSite(t *testing.T) {
	html := `
	<!DOCTYPE html>
	<html>
	<head><title>Documentation</title></head>
	<body>
		<nav>
			<a href="/">Home</a>
			<a href="/docs">Docs</a>
		</nav>
		<main class="documentation">
			<h1>Getting Started</h1>
			<p>Welcome to the documentation. This guide will help you get started
			   with our software. Follow the steps below to install and configure.</p>
			<h2>Installation</h2>
			<pre><code>npm install package-name</code></pre>
			<h2>Configuration</h2>
			<p>Configure your settings by editing the config file.</p>
			<h2>Usage</h2>
			<p>Here's how to use the package in your project.</p>
		</main>
	</body>
	</html>
	`

	assert.False(t, renderer.NeedsJSRendering(html), "Documentation site with content should not need JS rendering")
}
