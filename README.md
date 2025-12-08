# RepoDocs-Go

A CLI tool for extracting documentation from websites, Git repositories, sitemaps, pkg.go.dev, and llms.txt files. Built with Go, featuring stealth mode for avoiding bot detection and JavaScript rendering for single-page applications.

## Features

- **Multiple Extraction Strategies**: Automatically detects and uses the appropriate strategy based on URL
  - **Crawler**: Web crawling with Colly, includes stealth transport
  - **Sitemap**: Parse XML sitemaps (including sitemap index and gzipped)
  - **Git**: Clone and extract documentation from Git repositories
  - **pkg.go.dev**: Extract Go package documentation
  - **llms.txt**: Parse and download referenced documentation

- **Stealth Mode**: TLS fingerprinting with Chrome profile, User-Agent rotation, and header randomization
- **JavaScript Rendering**: Headless browser with Rod for SPAs (React, Vue, Next.js, Nuxt)
- **Caching**: BadgerDB-based caching with TTL support
- **Output Formats**: Markdown with YAML frontmatter, optional JSON metadata
- **Progress Tracking**: Real-time progress bars and logging

## Installation

### From Source

```bash
go install github.com/quantmind-br/repodocs-go/cmd/repodocs@latest
```

### Build Locally

```bash
git clone https://github.com/quantmind-br/repodocs-go.git
cd repodocs-go
make build
./build/repodocs --help
```

## Usage

### Basic Usage

```bash
# Crawl a website
repodocs https://docs.example.com

# Parse a sitemap
repodocs https://example.com/sitemap.xml

# Clone and extract from a Git repository
repodocs https://github.com/user/repo

# Extract Go package documentation
repodocs https://pkg.go.dev/github.com/user/package

# Parse llms.txt
repodocs https://example.com/llms.txt
```

### Common Options

```bash
# Specify output directory
repodocs https://example.com -o ./output

# Limit number of pages
repodocs https://example.com -l 100

# Set crawl depth
repodocs https://example.com -d 2

# Force JS rendering
repodocs https://example.com --render-js

# Dry run (simulate without writing)
repodocs https://example.com --dry-run

# Verbose output
repodocs https://example.com -v
```

### All Options

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--output` | `-o` | `./docs` | Output directory |
| `--concurrency` | `-j` | `5` | Number of concurrent workers |
| `--limit` | `-l` | `0` | Max pages to process (0=unlimited) |
| `--max-depth` | `-d` | `3` | Max crawl depth |
| `--exclude` | | `[]` | Regex patterns to exclude |
| `--nofolders` | | `false` | Flat output structure |
| `--force` | | `false` | Overwrite existing files |
| `--verbose` | `-v` | `false` | Verbose output |
| `--no-cache` | | `false` | Disable caching |
| `--cache-ttl` | | `24h` | Cache TTL |
| `--refresh-cache` | | `false` | Force cache refresh |
| `--render-js` | | `false` | Force JS rendering |
| `--timeout` | | `30s` | Request timeout |
| `--json-meta` | | `false` | Generate JSON metadata files |
| `--dry-run` | | `false` | Simulate without writing files |
| `--split` | | `false` | Split output by sections (pkg.go.dev) |
| `--include-assets` | | `false` | Include referenced images (git) |
| `--user-agent` | | `""` | Custom User-Agent |
| `--content-selector` | | `""` | CSS selector for main content |

## Configuration

Create a configuration file at `~/.repodocs/config.yaml`:

```yaml
output:
  directory: "./docs"
  flat: false
  json_metadata: true
  overwrite: false

concurrency:
  workers: 5
  timeout: 30s
  max_depth: 3

cache:
  enabled: true
  ttl: 24h

rendering:
  force_js: false
  js_timeout: 60s
  scroll_to_end: true

stealth:
  user_agent: ""
  random_delay_min: 1s
  random_delay_max: 3s

exclude:
  - ".*\\.pdf$"
  - ".*/login.*"
  - ".*/admin.*"

logging:
  level: "info"
  format: "pretty"
```

## Commands

### Doctor

Check system dependencies:

```bash
repodocs doctor
```

This will verify:
- Internet connectivity
- Chrome/Chromium availability (for JS rendering)
- Write permissions
- Configuration file validity
- Cache directory

### Version

```bash
repodocs version
```

## Output Format

### Markdown with Frontmatter

Each extracted page is saved as a Markdown file with YAML frontmatter:

```markdown
---
title: "Page Title"
url: "https://example.com/docs/page"
source: "crawler"
fetched_at: "2024-01-15T10:30:00Z"
rendered_js: false
word_count: 1523
---

# Page Title

Content here...
```

### JSON Metadata

With `--json-meta`, a companion JSON file is created:

```json
{
  "url": "https://example.com/docs/page",
  "title": "Page Title",
  "description": "Page description",
  "fetched_at": "2024-01-15T10:30:00Z",
  "word_count": 1523,
  "char_count": 8432,
  "links": ["https://example.com/other"],
  "headers": {
    "h1": ["Page Title"],
    "h2": ["Section 1", "Section 2"]
  },
  "rendered_with_js": false,
  "source_strategy": "crawler"
}
```

## Development

### Prerequisites

- Go 1.21+
- Chrome/Chromium (for JS rendering)

### Building

```bash
make build
```

### Testing

```bash
# Run all tests
make test

# Run with coverage
make coverage

# Run linter
make lint
```

### Project Structure

```
cmd/repodocs/         # CLI entrypoint
internal/
  app/                # Application orchestrator
  cache/              # BadgerDB cache implementation
  config/             # Configuration management
  converter/          # HTML to Markdown pipeline
  domain/             # Interfaces and models
  fetcher/            # HTTP client with stealth features
  output/             # Output writer
  renderer/           # Headless browser (Rod)
  strategies/         # Extraction strategies
  utils/              # Utilities
pkg/version/          # Version information
tests/                # Test suites
```

## Requirements

- **Chrome/Chromium**: Required for JavaScript rendering
- **Internet**: Required for web extraction

## License

MIT License
