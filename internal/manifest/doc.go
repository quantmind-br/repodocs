// Package manifest provides types and utilities for loading and validating
// RepoDocs manifest files. A manifest defines multiple documentation sources
// with per-source configurations, enabling batch processing.
//
// # Manifest Format
//
// Manifests can be written in YAML or JSON format:
//
//	sources:
//	  - url: https://docs.example.com
//	    strategy: crawler
//	    content_selector: "article.main"
//	  - url: https://github.com/org/repo
//	    strategy: git
//	    include: ["docs/**/*.md"]
//	options:
//	  continue_on_error: true
//	  output: ./knowledge-base
//
// # Usage
//
// Load a manifest file:
//
//	loader := manifest.NewLoader()
//	cfg, err := loader.Load("sources.yaml")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	for _, source := range cfg.Sources {
//	    // Process each source
//	}
//
// # Error Handling
//
// The package defines sentinel errors for common failure cases:
//   - ErrNoSources: manifest has no sources defined
//   - ErrEmptyURL: source is missing required URL field
//   - ErrInvalidFormat: file is not valid YAML/JSON
//   - ErrFileNotFound: manifest file does not exist
//   - ErrUnsupportedExt: unsupported file extension
package manifest
