# RepoDocs

A professional CLI tool for extracting documentation from GitHub repositories, designed with a security-first mindset and a focus on providing a comprehensive and user-friendly experience.

[![Latest Version](https://img.shields.io/crates/v/repodocs.svg)](https://crates.io/crates/repodocs)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

RepoDocs clones a GitHub repository and extracts all documentation files into a local directory, making it easy to browse, search, and analyze project documentation offline.

## Key Features

- 🔒 **Security-First Design**:
  - **URL Validation**: Strictly enforces `https://`, `ssh://`, or `git://` protocols and only allows GitHub URLs.
  - **Path Sanitization**: Prevents directory traversal and other filesystem-based attacks.
  - **Resource Limits**: Configurable limits for file size and scan depth to prevent abuse.
  - **Safe Operations**: Uses secure temporary directories with automatic cleanup.

- 📊 **Professional UI & Reporting**:
  - **Rich Terminal UI**: Provides colored output, progress bars for downloads and file operations, and structured logging.
  - **Multiple Output Formats**: Choose between human-readable, JSON, or plain text output.
  - **Detailed JSON Reports**: Generates a comprehensive `extraction_report.json` with repository info, extraction stats, and file details for CI/CD integration.

- ⚡ **Performance & Efficiency**:
  - **Optimized Scanning**: Efficiently scans repository files based on your criteria.
  - **Graceful Shutdown**: Responds to `Ctrl+C` (SIGINT) to terminate operations cleanly.

- 🔧 **Highly Configurable**:
  - **TOML Configuration**: Use a `repodocs.toml` file for project-specific settings.
  - **CLI Overrides**: All configuration file settings can be overridden via command-line flags.
  - **Custom Filters**: Define which file extensions to include, directories and patterns to exclude, and set size/depth limits.

- 📈 **Comprehensive Extraction**:
  - **Wide File Support**: Supports over 14 file types by default, including Markdown, reStructuredText, AsciiDoc, and more.
  - **Preserves Structure**: Maintains the original directory structure of the repository.
  - **Index Generation**: Automatically creates an index file listing all extracted documents.

## Installation

### From Source

```bash
git clone https://github.com/quantmind-br/repodocs.git
cd repodocs
cargo build --release
# Copy the binary to a directory in your PATH
cp target/release/repodocs ~/.local/bin/
```

### From Crates.io (Once Published)

```bash
cargo install repodocs
```

## Quick Start

1.  **Extract documentation from a repository:**
    ```bash
    # Uses default settings, saves to ./docs_book/
    repodocs https://github.com/rust-lang/book
    ```

2.  **Specify a custom output directory:**
    ```bash
    repodocs --output my-vscode-docs https://github.com/microsoft/vscode
    ```

3.  **Generate a JSON report for automation:**
    ```bash
    repodocs --output-format json --output report.json https://github.com/octocat/Hello-World
    ```

4.  **Perform a dry run to see what would be extracted:**
    ```bash
    repodocs --dry-run https://github.com/rust-lang/book
    ```

5.  **Generate a sample configuration file:**
    ```bash
    repodocs --generate-config
    # This creates a `repodocs.toml` file in the current directory
    ```

## CLI Options

The command-line interface is designed to be intuitive and powerful, allowing you to control all aspects of the extraction process.

```
Usage: repodocs [OPTIONS] <REPOSITORY_URL>

Arguments:
  <REPOSITORY_URL>  GitHub repository URL (e.g., https://github.com/owner/repo)

Options:
  -o, --output <OUTPUT_DIR>
          Output directory name (defaults to docs_{repo_name})

  -c, --config <CONFIG_FILE>
          Path to a `repodocs.toml` configuration file.

  -f, --formats <EXTENSIONS>
          Comma-separated file extensions to extract (e.g., "md,rst,txt"). Overrides config file.

  -e, --exclude <DIRS>
          Comma-separated list of directories to exclude. Appends to the default exclude list.

      --max-size <SIZE>
          Maximum file size to process in megabytes (e.g., 10 for 10MB).

      --branch <BRANCH>
          Specific git branch to clone (defaults to the repository's default branch).

      --output-format <FORMAT>
          Output format for results.
          [default: human] [possible values: human, json, plain]

      --timeout <SECONDS>
          Timeout for the git clone operation in seconds.

      --force
          Force overwrite of an existing output directory.

      --dry-run
          Show what would be extracted without cloning the repository or writing any files.

      --preserve-structure <true|false>
          Preserve the original directory structure in the output.

  -v, --verbose
          Enable verbose output. Use -vv or -vvv for more detail.

  -q, --quiet
          Suppress all output except for errors.

      --generate-config
          Generate a sample `repodocs.toml` file with default settings.

  -h, --help
          Print help information.

      --version
          Print version information.
```

## Configuration File

For advanced and project-specific settings, create a `repodocs.toml` file. `repodocs` automatically detects it in the current directory if named `repodocs.toml`, `.repodocs.toml`, or `repodocs.config.toml`.

You can generate a default configuration file using `repodocs --generate-config`.

### Example `repodocs.toml`

```toml
# repodocs.toml

[filters]
# A list of file extensions to extract.
extensions = [
    "md", "markdown", "mdown", "rst", "rest", "adoc", "asciidoc", "asc",
    "txt", "text", "org", "wiki", "tex", "latex"
]

# Maximum file size in bytes (e.g., 10 * 1024 * 1024 for 10MB).
max_file_size = 10485760

# A list of directory names to exclude from the scan.
exclude_dirs = [
    ".git", "node_modules", "target", "build", "dist", "vendor",
    ".vscode", ".idea"
]

# A list of regex patterns to exclude files.
exclude_patterns = [".*\.min\..*", ".*\.lock"]

# Maximum directory depth to scan.
max_depth = 10

[output]
# If true, mirrors the repository's directory structure.
preserve_structure = true

# If true, creates an `_index.md` file with a list of all extracted files.
create_index = true

# If true, generates a `extraction_report.json` file.
generate_report = true

# The base directory where the output folder will be created.
# Defaults to the current working directory.
base_directory = "."

[git]
# Specifies the depth of the git clone. `None` for a full clone.
clone_depth = 1

# Timeout for the git clone operation in seconds.
timeout = 300

# The specific branch to clone. `None` for the repository's default branch.
branch = "main"
```

## Examples

### Basic Usage

```bash
# Extract docs from a repository with default settings
repodocs https://github.com/rust-lang/book

# Use verbose output and specify a custom output directory
repodocs --verbose --output rust-book-docs https://github.com/rust-lang/book
```

### Advanced Usage

```bash
# Only extract Markdown and reStructuredText files, with a 5MB size limit
repodocs --formats md,rst --max-size 5 https://github.com/microsoft/vscode

# Exclude additional directories from the scan
repodocs --exclude "vendor,third_party" https://github.com/facebook/react

# Clone a specific branch of a repository
repodocs --branch stable https://github.com/torvalds/linux
```

### Automation and CI/CD

```bash
# Generate a JSON report for use in a CI/CD pipeline
repodocs --output-format json --output build/docs-report.json https://github.com/vuejs/core

# Pipe the JSON output to `jq` to extract specific information
repodocs --output-format json https://github.com/tailwindlabs/tailwindcss | jq '.extraction_summary.total_files_processed'
```

## Output Structure

When you run `repodocs`, it creates a structured output directory.

```
./
├── my-docs/  (Your specified output, e.g., --output my-docs)
│   ├── docs_repository_name/
│   │   ├── _index.md
│   │   ├── README.md
│   │   ├── docs/
│   │   │   ├── guide.md
│   │   │   └── api.md
│   │   └── src/
│   │       └── lib.rs
│   └── .repodocs/
│       ├── extraction_report.json
│       └── extraction_report.txt
```

## Development

### Prerequisites

- Rust 1.70+
- Git

### Building

```bash
# Development build
cargo build

# Release build
cargo build --release

# Run tests
cargo test

# Run clippy for linting
cargo clippy
```

## Contributing

Contributions are welcome! Please follow these steps:

1.  Fork the repository.
2.  Create a new feature branch (`git checkout -b feature/my-new-feature`).
3.  Commit your changes (`git commit -am 'Add some feature'`).
4.  Push to the branch (`git push origin feature/my-new-feature`).
5.  Open a new Pull Request.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
