# RepoDocs

A professional CLI tool for extracting documentation from GitHub repositories with security-first design and comprehensive progress tracking.

## Features

- 🔒 **Security-first**: URL validation, certificate checks, path sanitization
- 📊 **Professional UI**: Colored output, progress bars, structured JSON reporting
- ⚡ **Performance optimized**: Efficient file scanning, parallel operations
- 🛡️ **Safe operations**: Resource limits, graceful shutdown, temporary file cleanup
- 🔧 **Configurable**: TOML configuration files, CLI overrides, custom filters
- 📈 **Comprehensive**: Support for 14+ file types including extensionless documentation

## Installation

### From Source

```bash
git clone https://github.com/quantmind-br/repodocs.git
cd repodocs
cargo build --release
cp target/release/repodocs ~/.local/bin/
```

### From Cargo (when published)

```bash
cargo install repodocs
```

## Quick Start

```bash
# Extract documentation from a repository
repodocs https://github.com/rust-lang/book

# Custom output directory
repodocs --output my-docs https://github.com/microsoft/vscode

# JSON output for automation
repodocs --output-format json --output report.json https://github.com/octocat/Hello-World

# Dry run to preview what would be extracted
repodocs --dry-run https://github.com/rust-lang/book
```

## Configuration

Create a `repodocs.toml` configuration file:

```toml
[filters]
extensions = ["md", "markdown", "rst", "txt"]
max_file_size = 10485760  # 10MB
exclude_dirs = [".git", "node_modules", "target"]
max_depth = 10

[output]
preserve_structure = true
create_index = true
generate_report = true
base_directory = "/path/to/output"

[git]
timeout = 300
```

## Supported File Types

- Markdown: `.md`, `.markdown`, `.mdown`
- reStructuredText: `.rst`, `.rest`
- AsciiDoc: `.adoc`, `.asciidoc`, `.asc`
- Plain text: `.txt`, `.text`
- Org mode: `.org`
- Wiki: `.wiki`
- LaTeX: `.tex`, `.latex`
- Extensionless: `README`, `LICENSE`, `CHANGELOG`, etc.

## Security Features

- **URL Validation**: Only accepts GitHub URLs with proper format
- **Certificate Verification**: Validates SSL certificates for Git operations
- **Path Sanitization**: Prevents directory traversal attacks
- **Resource Limits**: Configurable file size and depth limits
- **Temporary Files**: Secure temporary directory handling with automatic cleanup
- **No Credentials**: Never exposes tokens or credentials in logs

## CLI Options

```
Usage: repodocs [OPTIONS] <REPOSITORY_URL>

Arguments:
  <REPOSITORY_URL>  GitHub repository URL (e.g., https://github.com/owner/repo)

Options:
  -o, --output <OUTPUT_DIR>         Output directory [default: ./docs]
  -c, --config <CONFIG_FILE>        Configuration file (TOML format)
      --output-format <FORMAT>      Output format [default: human] [possible values: human, json, plain]
      --dry-run                     Preview extraction without downloading files
      --formats <EXTENSIONS>         Comma-separated list of file extensions
      --max-size <SIZE>             Maximum file size (e.g., 10MB, 1GB)
      --max-depth <DEPTH>           Maximum directory depth to scan
      --exclude <DIRS>              Comma-separated directories to exclude
      --branch <BRANCH>             Specific branch to clone
      --quiet                       Suppress all output except errors
      --verbose                     Enable verbose output
      --generate-config             Generate sample configuration file
  -v, --version                    Print version information
  -h, --help                       Print help
```

## Examples

### Basic Usage

```bash
# Extract with default settings
repodocs https://github.com/rust-lang/book

# Verbose output with custom directory
repodocs --verbose --output rust-book https://github.com/rust-lang/book
```

### Advanced Usage

```bash
# Custom file types and size limit
repodocs --formats md,rst,txt --max-size 5MB https://github.com/microsoft/vscode

# Exclude specific directories
repodocs --exclude "node_modules,target,dist" https://github.com/facebook/react

# Specific branch
repodocs --branch stable https://github.com/torvalds/linux
```

### JSON Output for Automation

```bash
# Generate JSON report for CI/CD
repodocs --output-format json --output build/docs-report.json https://github.com/vuejs/core

# Pipe to other tools
repodocs --output-format json https://github.com/tailwindlabs/tailwindcss | jq '.extraction_summary.total_files_processed'
```

## Output Structure

```
output_directory/
├── docs_repository_name/
│   ├── README.md
│   ├── docs/
│   │   ├── guide.md
│   │   └── api.md
│   └── src/
│       └── lib.rs
└── .repodocs/
    ├── extraction_report.json
    └── extraction_report.txt
```

## JSON Report Format

```json
{
  "repository_info": {
    "name": "repository-name",
    "owner": "owner",
    "default_branch": "main",
    "url": "https://github.com/owner/repo"
  },
  "extraction_summary": {
    "total_files_processed": 42,
    "total_bytes_processed": 1048576,
    "files_by_extension": {
      "md": 30,
      "txt": 12
    }
  },
  "files": [
    {
      "filename": "README.md",
      "relative_path": "README.md",
      "extension": "md",
      "size": 2048
    }
  ]
}
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

# Run clippy
cargo clippy
```

### Testing

```bash
# Unit tests
cargo test

# Integration tests
cargo test --test integration

# Run with coverage
cargo tarpaulin
```

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Changelog

### v1.0.0
- Initial release
- GitHub repository extraction
- Multiple output formats
- Configuration file support
- Comprehensive security features
- Progress tracking and reporting

## Support

- 📚 [Documentation](https://github.com/quantmind-br/repodocs/wiki)
- 🐛 [Report Issues](https://github.com/quantmind-br/repodocs/issues)
- 💬 [Discussions](https://github.com/quantmind-br/repodocs/discussions)

## Acknowledgments

Built with ❤️ using Rust and the amazing crates in the Rust ecosystem.