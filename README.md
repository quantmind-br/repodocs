# repodocs-go: Documentation Extraction CLI

## Project Overview

`repodocs-go` is a powerful Command Line Interface (CLI) tool designed to automate the extraction, cleaning, and conversion of documentation from diverse sources into standardized Markdown files. It acts as a robust client capable of handling complex web environments, including JavaScript-heavy Single Page Applications (SPAs) and version control systems.

**Purpose and Main Functionality**
The primary goal of `repodocs-go` is to provide a reliable, configurable pipeline for transforming unstructured or semi-structured content (HTML, Git files) into clean, readable documentation, suitable for integration into knowledge bases or static site generators.

**Key Features and Capabilities**

*   **Multi-Strategy Extraction:** Supports various input sources, including standard web pages (Crawler), sitemaps, Git repositories, and GitHub wikis.
*   **SPA Rendering:** Utilizes a headless browser (Chromium via `rod`) to render JavaScript-heavy content before extraction.
*   **Content Cleaning:** Employs a content pipeline to sanitize HTML, extract the main article body (readability), and convert it to Markdown.
*   **Resilient Fetching:** Includes a custom HTTP client with retry logic (exponential backoff) for transient network errors and specific status codes (429, 5xx).
*   **Persistent Caching:** Uses BadgerDB for local, persistent caching of fetched resources, reducing external load and improving performance.
*   **Stealth Capabilities:** Integrates specialized HTTP clients and browser automation techniques to handle anti-bot measures.

**Likely Intended Use Cases**

*   Automated documentation generation for microservices by reading files directly from Git repositories.
*   Archiving or mirroring external documentation websites.
*   Creating local, searchable documentation from complex, modern web applications (SPAs).

## Table of Contents

1.  [Project Overview](#project-overview)
2.  [Installation and Usage](#installation-and-usage)
3.  [Architecture](#architecture)
4.  [C4 Model Architecture](#c4-model-architecture)
5.  [Repository Structure](#repository-structure)
6.  [Dependencies and Integration](#dependencies-and-integration)
7.  [API Documentation](#api-documentation)
8.  [Development Notes](#development-notes)
9.  [Known Issues and Limitations](#known-issues-and-limitations)
10. [Additional Documentation](#additional-documentation)

## Architecture

### High-Level Architecture Overview

The `repodocs-go` architecture follows a layered, modular design, closely resembling a **Hexagonal Architecture** (Ports and Adapters). The core logic is decoupled from infrastructure concerns through interfaces defined in the `internal/domain` package.

The system is centered around the **Orchestrator**, which acts as the application coordinator. It loads configuration, detects the input source type, selects the appropriate **Strategy** (e.g., `CrawlerStrategy`, `GitStrategy`), and executes the content extraction and conversion **Pipeline**.

### Technology Stack and Frameworks

| Category | Technology/Framework | Purpose |
| :--- | :--- | :--- |
| **Language** | Go | Primary development language. |
| **CLI/Config** | `cobra`, `viper` | Command-line interface and configuration management. |
| **Caching** | BadgerDB | Embedded, persistent key-value store for caching fetched content. |
| **Web Scraping** | `gocolly/colly`, `fhttp`, `tls-client` | Web crawling and specialized HTTP requests (stealth). |
| **Rendering** | `go-rod`, Chromium | Headless browser automation for JavaScript rendering. |
| **Content Processing** | `go-readability`, `html-to-markdown` | HTML cleaning, core content extraction, and Markdown conversion. |

### Key Design Patterns

| Pattern | Implementation | Description |
| :--- | :--- | :--- |
| **Strategy Pattern** | `internal/strategies` | Allows the `Orchestrator` to dynamically select the extraction logic (`Git`, `Wiki`, `Sitemap`, `Crawler`) based on the input URL. |
| **Pipeline Pattern** | `internal/converter/pipeline.go` | Defines a fixed, sequential chain of responsibility for content transformation (Sanitize -> Readability -> Markdown). |
| **Dependency Injection** | `internal/strategies/Dependencies` | A central Composition Root that instantiates and aggregates all infrastructure services, injecting them into the specific `Strategy` implementations. |

### Component Relationships

The following diagram illustrates the flow of control and data through the main components of the application.

```mermaid
graph TD
    subgraph Application Flow
        A[cmd/main.go] --> B(app.Orchestrator);
        B --> C{Detect Strategy};
        C --> D(Strategy Implementation);
        D --> E(strategy.Execute());
    end

    subgraph Infrastructure Services
        F(fetcher.Client)
        G(renderer.Renderer)
        H(converter.Pipeline)
        I(cache.BadgerCache)
        J(output.Writer)
    end

    E --> |Uses| F;
    E --> |Uses| G;
    E --> |Uses| H;
    E --> |Uses| J;

    F --> |Caches/Retrieves| I;
    F --> |Fetches Raw Content| K[External Web/Git];
    G --> |Renders JS Content| L[Chromium/Headless Browser];
    H --> |Transforms HTML to Markdown| M[Content Processing Libraries];
    J --> |Writes Final Output| N[Local Filesystem];

    style B fill:#f9f,stroke:#333
    style E fill:#ccf,stroke:#333
    style H fill:#ccf,stroke:#333
```

## C4 Model Architecture

### Context Diagram

The Context diagram shows the `repodocs-go` system and its primary interactions with external entities.

<details>
<summary>C4 Context Diagram</summary>

```mermaid
C4Context
    title Context Diagram for repodocs-go
    Person(user, "Developer/User", "Operates the CLI tool to generate documentation.")
    System(repodocs, "repodocs-go CLI Tool", "A command-line application that extracts, cleans, and converts documentation from various sources into Markdown.")

    System_Ext(web_services, "External Web Services", "Public or private websites, APIs, and sitemaps.")
    System_Ext(git_repos, "Git Repositories", "Remote or local Git repositories (e.g., GitHub, GitLab).")
    System_Ext(filesystem, "Local Filesystem", "The local disk where configuration, cache, and final output files are stored.")

    user -- "Runs command with URL" --> repodocs
    repodocs --> "Fetches content (HTTP/HTTPS)" --> web_services
    repodocs --> "Clones/Reads content (Git Protocol/HTTP)" --> git_repos
    repodocs --> "Reads config, cache, Writes final Markdown" --> filesystem

    Rel(repodocs, web_services, "Consumes content from", "HTTP/HTTPS")
    Rel(repodocs, git_repos, "Consumes content from", "Git Protocol")
    Rel(repodocs, filesystem, "Persists data to", "File I/O")
```
</details>

### Container Diagram

The Container diagram breaks down the `repodocs-go` system into its main technical building blocks.

<details>
<summary>C4 Container Diagram</summary>

```mermaid
C4Container
    title Container Diagram for repodocs-go
    System_Boundary(repodocs, "repodocs-go")
        Container(cli, "CLI Application", "Go Executable", "The main application binary containing the Orchestrator, Strategies, and Conversion Pipeline.")
        Container(cache, "BadgerDB Cache", "Embedded Key-Value Store", "Stores fetched raw content (HTML/XML) to minimize repeated external requests.")
        Container(renderer, "Headless Browser Pool", "Chromium/Rod", "Manages instances of a headless browser for rendering JavaScript-heavy pages.")
    System_Boundary(repodocs)

    System_Ext(web_services, "External Web Services", "Public or private websites, APIs, and sitemaps.")
    System_Ext(git_repos, "Git Repositories", "Remote or local Git repositories.")
    System_Ext(filesystem, "Local Filesystem", "The local disk for final output.")

    cli --> cache "Reads and Writes cached content"
    cli --> renderer "Sends URLs for rendering"
    cli --> web_services "Fetches content (HTTP/Stealth Client)"
    cli --> git_repos "Clones/Reads repository content"
    cli --> filesystem "Writes final Markdown documentation"
```
</details>

## Installation and Usage

### Prerequisites

- **Go 1.21+** (for building from source)
- **Chromium/Chrome** (optional, for JavaScript rendering)
- **Git** (for cloning repositories)

### Installation

#### Option 1: User Installation (~/.local/bin)

Install for the current user without sudo:

```bash
git clone https://github.com/quantmind-br/repodocs-go.git
cd repodocs-go/repodocs-go
make install
```

This installs the binary to `~/.local/bin/repodocs`. Make sure this directory is in your PATH:

```bash
export PATH="$HOME/.local/bin:$PATH"
# Add to ~/.bashrc or ~/.zshrc to persist
```

#### Option 2: Global Installation (/usr/local/bin)

Install system-wide (requires sudo):

```bash
git clone https://github.com/quantmind-br/repodocs-go.git
cd repodocs-go/repodocs-go
make install-global
```

#### Option 3: Build Only

Build without installing:

```bash
make build
# Binary will be in ./build/repodocs
./build/repodocs --help
```

### Uninstallation

```bash
# Remove user installation
make uninstall

# Remove global installation (requires sudo)
make uninstall-global

# Check installation status
make check-install
```

### Quick Start

```bash
# Extract documentation from a website
repodocs https://example.com/docs

# Extract from a Git repository
repodocs https://github.com/user/repo

# Extract from a GitHub wiki
repodocs https://github.com/user/repo/wiki

# Extract from a sitemap
repodocs https://example.com/sitemap.xml

# Specify output directory
repodocs https://example.com/docs -o ./my-docs

# Enable JavaScript rendering for SPAs
repodocs https://spa-app.com --render-js

# Limit crawl depth and pages
repodocs https://example.com/docs --max-depth 2 --limit 50

# Check system dependencies
repodocs doctor
```

### GitHub Wiki Extraction

The wiki strategy automatically extracts documentation from GitHub wiki repositories:

```bash
# Extract entire wiki with hierarchical structure
repodocs https://github.com/Alexays/Waybar/wiki

# Extract with custom output directory
repodocs https://github.com/owner/repo/wiki -o ./wiki-docs

# Flat structure (no section folders)
repodocs https://github.com/owner/repo/wiki --nofolders

# Limit number of pages
repodocs https://github.com/owner/repo/wiki --limit 10

# Dry run (preview without writing files)
repodocs https://github.com/owner/repo/wiki --dry-run
```

**Features:**
- Parses `_Sidebar.md` for hierarchical organization
- Converts wiki-style links (`[[Page Name]]`) to standard Markdown
- Transforms `Home.md` to `index.md`
- Supports private wikis via `GITHUB_TOKEN` environment variable

### Configuration

Configuration can be provided via:
1. Command-line flags
2. Configuration file (`~/.repodocs/config.yaml`)
3. Environment variables

Example config file:

```yaml
output:
  directory: "./docs"
  flat: false
  overwrite: false

concurrency:
  workers: 5
  max_depth: 4
  timeout: 30s

cache:
  enabled: true
  ttl: 24h

rendering:
  force_js: false
  wait_stable: 2s
```

For more options, run:

```bash
repodocs --help
```

## Repository Structure

The repository is organized to separate the application entry point, core logic, domain contracts, and infrastructure implementations.

| Directory | Purpose |
| :--- | :--- |
| `cmd/` | Contains the main CLI entry point (`main.go`) and `cobra` command definitions. |
| `internal/app` | Application layer logic, including the `Orchestrator` and `Detector`. |
| `internal/domain` | Core business models, interfaces (contracts), and custom errors. |
| `internal/strategies` | Implementations of the `domain.Strategy` interface (e.g., `crawler`, `git`, `wiki`, `sitemap`). |
| `internal/converter` | The content transformation pipeline (sanitization, readability, markdown conversion). |
| `internal/fetcher` | Network I/O, custom HTTP client, retry logic, and stealth features. |
| `internal/cache` | Concrete implementation of the `domain.Cache` interface using BadgerDB. |
| `internal/renderer` | Headless browser management using the `rod` library. |
| `internal/output` | Logic for writing the final documentation files to the filesystem. |
| `internal/utils` | Shared utilities like logging (`zerolog`) and the worker pool. |

## Dependencies and Integration

The project integrates with several internal and external systems to perform its function.

### Internal Service Dependencies

| Service | Technology | Integration Method | Notes |
| :--- | :--- | :--- | :--- |
| **Local Cache** | BadgerDB | Embedded Key-Value Store | Used by the `fetcher` to store raw HTTP responses, keyed by URL and configuration hash. |
| **Headless Browser** | Chromium/Rod | DevTools Protocol | Managed by the `renderer` package to execute JavaScript and obtain final DOM content for SPAs. |

### External Service Dependencies

| Service | Protocol/Type | Purpose |
| :--- | :--- | :--- |
| **Generic Web Services** | HTTP/HTTPS | Primary source for content fetching (HTML, XML, raw files). |
| **Git Repositories** | Git Protocol/HTTP | Source for file-based documentation extraction. |
| **LLM Services** | TBD (API) | Placeholder strategy (`LLMsStrategy`) for future integration with Large Language Models for processing or generating content. |

### Integration Patterns

*   **Custom HTTP Client:** Uses specialized Go libraries (`fhttp`, `tls-client`) to implement a "stealth" client, allowing the tool to fetch content from sites with anti-bot measures.
*   **Retry Mechanism:** Implements an exponential backoff strategy for retrying network requests on transient errors (e.g., 429, 5xx status codes).
*   **Browser Automation:** The `internal/renderer` package uses `rod` to control a headless browser, which is essential for accurately fetching content from modern, client-side rendered applications.

## API Documentation

As a Command Line Interface (CLI) tool, `repodocs-go` does not expose any external HTTP or network-based APIs. Its primary interface is the command line itself, and its "API" for developers is its configuration and the external services it consumes.

### APIs Served by This Project

N/A (No endpoints served)

### External API Configuration

The tool acts as a client, and its behavior when interacting with external services is highly configurable via CLI flags and configuration files.

| Configuration Parameter | Default Value | Description |
| :--- | :--- | :--- |
| `source-url` | N/A (Required) | The target URL or repository path to extract documentation from. |
| `output` | (URL-based default) | The directory path where the final Markdown files will be written. |
| `concurrency` | TBD | The maximum number of concurrent fetching/rendering tasks (managed by the internal worker pool). |
| `max-retries` | 3 | Maximum number of times to retry a failed network request. |
| `timeout` | 30 seconds | Request timeout for individual network operations. |
| `user-agent` | Custom | The User-Agent string used for all HTTP requests. |
| `cache-ttl` | TBD | Time-to-live for cached items in BadgerDB. |

## Development Notes

### Project-Specific Conventions

*   **Structured Logging:** The project uses `github.com/rs/zerolog` for structured, leveled logging, configured via the `--verbose` flag.
*   **Dependency Management:** Dependencies are managed using a **Composition Root** pattern (`internal/strategies/Dependencies`), ensuring that concrete implementations are instantiated in one central location and injected into consumers via interfaces.
*   **Context Usage:** All major operations and network calls accept a `context.Context` to allow for graceful cancellation (e.g., on `SIGINT`/`SIGTERM`) and timeouts.

### Testing Requirements

*   **Dependency Check:** The `repodocs doctor` command is available to check for required external dependencies, particularly the presence and accessibility of the Chromium binary needed for the `internal/renderer` component.
*   **Integration Testing:** Due to the heavy reliance on external services (HTTP, Git, Chromium), robust integration tests are necessary to ensure the `fetcher`, `renderer`, and various `strategies` function correctly.

### Performance Considerations

*   **Concurrency:** The application utilizes a `WorkerPool` (`internal/utils/workerpool.go`) to manage concurrent fetching and processing tasks, optimizing throughput when dealing with sitemaps or large crawl scopes.
*   **Caching:** The use of BadgerDB provides fast, persistent caching, which is critical for performance, especially when running the tool multiple times against the same source or during development.

## Known Issues and Limitations

*   **LLM Strategy Placeholder:** The `LLMsStrategy` is currently a placeholder and does not contain functional integration logic for Large Language Models.
*   **Git Retry Logic:** The Git strategy relies on the underlying `go-git/v5` library for operations. No specific retry or backoff logic is implemented within the Go code for failed Git operations.
*   **Headless Browser Overhead:** The dependency on a headless browser (Chromium) for rendering significantly increases the application's resource footprint and deployment complexity.
*   **Stealth Client Maintenance:** The reliance on specialized, non-standard HTTP libraries (`fhttp`, `tls-client`) for stealth features may introduce maintenance challenges if these libraries lag behind upstream Go updates or if anti-bot techniques evolve.

## Additional Documentation

*   [Project Plan and Roadmap](./PLAN.md)
*   [Current Development Tasks](./TASKS.md)
*   [Linting and Code Quality Configuration](./.golangci.yml)