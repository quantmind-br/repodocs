# RepoDocs

RepoDocs is a modular, high-performance CLI tool and Go library designed to extract documentation from various online sources and convert it into structured, LLM-friendly Markdown. Whether targeting a standard website, a Git repository, a sitemap, or specialized documentation hubs like `pkg.go.dev`, RepoDocs orchestrates the extraction, cleaning, and transformation process automatically.

## Features

-   **Multi-Strategy Extraction**: Automatically detects and handles various source types:
    -   **Web Crawler**: Recursive crawling of standard websites.
    -   **Git Strategy**: Clones or archives Git repositories (GitHub, GitLab, Bitbucket).
    -   **Sitemap Strategy**: Processes URLs defined in `sitemap.xml`.
    -   **Specialized Parsers**: Support for `llms.txt`, `pkg.go.dev`, and `docs.rs` (Rust crate documentation).
-   **Stealth Fetching**: Utilizes `tls-client` and custom User-Agent spoofing to bypass bot detection.
-   **Headless Rendering**: Optional JavaScript execution via Rod/Chrome for dynamic, single-page application (SPA) documentation.
-   **Advanced Conversion Pipeline**: 
    -   Identifies main content using Readability algorithms.
    -   Sanitizes HTML by removing scripts, navigation, and comments.
    -   Converts HTML to GitHub Flavored Markdown (GFM).
-   **AI Metadata Enrichment**: Integrates with OpenAI, Anthropic, and Google Gemini to generate AI-powered summaries, tags, and categories for every document.
-   **Persistent Caching**: Uses BadgerDB to cache raw content locally, respecting TTLs and reducing redundant network requests.
-   **Flexible Output**: Supports hierarchical folder structures or flat file organization with YAML frontmatter and JSON metadata.

## Installation

### Prerequisites
-   **Go**: 1.21 or higher.
-   **Chrome/Chromium**: Required for the `--render-js` feature (JavaScript rendering).

### Build from Source
```bash
git clone https://github.com/youruser/repodocs.git
cd repodocs
go build -o repodocs
```

## Quick Start

### Basic Documentation Extraction
Extract documentation from a website with default settings:
```bash
repodocs https://docs.example.com
```

### Git Repository Extraction
Clone a repository and extract all documentation files:
```bash
repodocs https://github.com/spf13/cobra.git
```

#### Filtering to Specific Directories
Extract documentation from only a specific subdirectory in a Git repository:
```bash
# Using URL path (for repositories with /tree/ in the URL)
repodocs https://github.com/owner/repo/tree/main/docs

# Using --filter flag
repodocs https://github.com/owner/repo --filter docs

# Filter to nested directory
repodocs https://github.com/owner/repo --filter docs/guides
```

**Note:** When both URL path and `--filter` flag are provided, the URL path takes precedence.

### Rust Crate Documentation (docs.rs)
Extract documentation from Rust crates hosted on docs.rs:
```bash
repodocs https://docs.rs/serde

repodocs https://docs.rs/tokio/1.32.0

repodocs https://docs.rs/serde/1.0.0 -o ./serde-docs -j 3 --limit 50
```

### Advanced Usage
Crawl with JavaScript rendering, higher concurrency, and AI metadata:
```bash
repodocs https://react.dev --render-js -j 10 --json-meta
```

### Check System Health
Use the `doctor` command to ensure your environment is set up correctly:
```bash
repodocs doctor
```

## Architecture

RepoDocs follows an interface-driven, modular architecture designed for extensibility:

-   **Orchestrator**: The central engine that detects the input type and manages the lifecycle of the extraction process.
-   **Strategies**: Specialized handlers (Crawler, Git, Sitemap, etc.) that implement the logic for content discovery.
-   **Conversion Pipeline**: A multi-stage processor that handles encoding conversion, content extraction (via CSS selectors or Readability), sanitization, and Markdown generation.
-   **Dependency Container**: A unified structure providing strategies with access to the Fetcher, Renderer (Browser), Cache (BadgerDB), and LLM services.
-   **Metadata Enhancer**: An optional layer that interacts with LLM providers to enrich documents with contextual metadata.

## Configuration

RepoDocs can be configured via command-line flags or a configuration file located at `~/.repodocs/config.yaml`.

### Global Flags
| Flag | Short | Description | Default |
| :--- | :--- | :--- | :--- |
| `--output` | `-o` | Output directory | `./docs` |
| `--concurrency` | `-j` | Number of concurrent workers | `5` |
| `--limit` | `-l` | Max pages to process (0=unlimited) | `0` |
| `--max-depth` | `-d` | Max crawl depth | `4` |
| `--filter` | | Path filter (web: base URL; git: subdirectory) | |
| `--render-js` | | Force JavaScript rendering | `false` |
| `--no-cache` | | Disable the BadgerDB caching layer | `false` |
| `--json-meta` | | Generate additional JSON metadata files | `false` |
| `--exclude` | | Regex patterns to exclude specific paths | |

## Development

### Running Tests
The project uses Go's standard testing tool. Strategies and services are designed with interfaces to facilitate mocking.
```bash
go test ./...
```

### Project Structure
-   `internal/app`: Orchestrator and strategy detection logic.
-   `internal/strategies`: Implementations for various extraction methods.
-   `internal/converter`: The HTML-to-Markdown transformation pipeline.
-   `internal/fetcher`: Stealth HTTP client implementation.
-   `internal/renderer`: Headless browser management.
-   `internal/cache`: BadgerDB persistent storage integration.
-   `internal/domain`: Core interfaces and data models.

## License

This project is licensed under the terms of the LICENSE file included in the repository root.