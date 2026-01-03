# RepoDocs

RepoDocs is a powerful Go-based CLI tool and library designed to extract documentation from diverse sources—including websites, Git repositories, Sitemaps, and Wikis—and convert them into clean, structured Markdown. It is built to facilitate the creation of high-quality datasets for LLM training, RAG (Retrieval-Augmented Generation) pipelines, or local documentation mirrors.

## Features

-   **Multi-Source Extraction**: Automatically detects and handles various source types:
    -   **Web Crawler**: Recursive crawling of documentation sites.
    -   **Git/GitHub**: Cloning repositories or fetching specific paths.
    -   **Sitemaps**: Systematic discovery via `sitemap.xml`.
    -   **llms.txt**: Support for the emerging `llms.txt` standard for LLM-friendly discovery.
    -   **Package Docs**: Specialized handling for `pkg.go.dev`.
-   **Advanced Processing**:
    -   **HTML to Markdown**: Converts complex HTML into clean Markdown using a multi-stage pipeline.
    -   **Content Extraction**: Uses "readability" logic and CSS selectors to isolate main content and remove noise (navbars, footers, scripts).
    -   **JS Rendering**: Headless browser support (via `go-rod`) for Single Page Applications (SPAs) and JavaScript-heavy sites.
-   **Stealth & Robustness**:
    -   **Bot Avoidance**: User-Agent rotation and TLS fingerprinting to bypass basic bot detection.
    -   **Caching**: Persistent caching using BadgerDB to minimize network load and respect rate limits.
    -   **Retries**: Exponential backoff for transient network errors.
-   **AI Integration**: Optional metadata enrichment using LLMs (OpenAI, Anthropic, Google) to generate summaries, tags, and categories.
-   **Structured Output**: Generates Markdown files with YAML frontmatter and a consolidated `repodocs.json` index.

## Installation

### Prerequisites

-   **Go**: 1.21 or later.
-   **Chrome/Chromium**: Required if using the `--render-js` feature for JavaScript rendering.

### From Source

```bash
git clone https://github.com/yourusername/repodocs.git
cd repodocs
go build -o repodocs ./cmd/repodocs
```

### Dependency Check
Use the built-in "doctor" command to verify your environment:
```bash
./repodocs doctor
```

## Quick Start

Extract documentation from a URL to the default `./docs` directory:
```bash
repodocs https://docs.example.com
```

Extract a specific GitHub repository with a maximum depth of 2:
```bash
repodocs https://github.com/user/repo --max-depth 2
```

Force JavaScript rendering for a React-based documentation site:
```bash
repodocs https://spa-docs.com --render-js
```

Generate JSON metadata and limit to 10 pages:
```bash
repodocs https://example.com --json-meta --limit 10
```

## Architecture

RepoDocs follows a decoupled, interface-driven architecture structured as a processing pipeline:

1.  **Detection**: The `Orchestrator` uses a `Strategy Factory` to identify the correct approach (Git, Crawler, Sitemap, etc.) based on the input URL.
2.  **Execution**: The selected `Strategy` orchestrates fetching or cloning content.
3.  **Processing**: The `Converter Pipeline` transforms raw content:
    -   **Encoding**: Normalizes text to UTF-8.
    -   **Sanitization**: Removes unwanted HTML tags and noise.
    -   **Conversion**: Transforms cleaned HTML into Markdown.
4.  **Enhancement**: The `MetadataEnhancer` (optional) uses LLMs to enrich the document with summaries and tags.
5.  **Output**: The `Writer` persists the final Markdown and metadata to the local filesystem.

### Core Components

-   **Internal Domain**: Defines core models (`Document`, `Page`) and interfaces (`Fetcher`, `Renderer`, `Cache`).
-   **Fetcher**: High-level HTTP client with stealth capabilities and caching.
-   **Renderer**: Manages a pool of headless browser tabs for dynamic content.
-   **Strategies**: Specialized logic for different documentation sources.

## Configuration

RepoDocs can be configured via CLI flags or a configuration file (default: `~/.repodocs/config.yaml`).

### Common Flags

| Flag | Short | Description | Default |
| :--- | :--- | :--- | :--- |
| `--output` | `-o` | Output directory | `./docs` |
| `--concurrency` | `-j` | Number of concurrent workers | `5` |
| `--max-depth` | `-d` | Maximum crawl depth | `4` |
| `--limit` | `-l` | Maximum number of pages to process | `0` (unlimited) |
| `--render-js` | | Force JavaScript rendering | `false` |
| `--no-cache` | | Disable the BadgerDB caching layer | `false` |
| `--exclude` | | Regex patterns to exclude specific paths | |
| `--json-meta` | | Generate individual `.json` metadata files | `false` |

## Development

### Running Tests
Execute the test suite:
```bash
go test ./...
```

### Linting
The project follows standard Go formatting. Run the linter:
```bash
go vet ./...
```

### Building
Build the binary for your local architecture:
```bash
go build -o repodocs ./cmd/repodocs
```

## Contributing

1.  Ensure all core services are defined via interfaces in the `internal/domain` package.
2.  When adding new extraction logic, implement the `Strategy` interface in `internal/strategies`.
3.  Add unit tests for new components using `go.uber.org/mock` for dependency mocking.
4.  Ensure any changes to the HTML-to-Markdown pipeline are reflected in the `internal/converter` package.

## License

Refer to the `LICENSE` file for details.