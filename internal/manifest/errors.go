package manifest

import "errors"

// Sentinel errors for the manifest package
var (
	// ErrNoSources indicates the manifest has no sources defined
	ErrNoSources = errors.New("manifest must contain at least one source")

	// ErrEmptyURL indicates a source is missing the required URL field
	ErrEmptyURL = errors.New("source URL cannot be empty")

	// ErrInvalidFormat indicates the manifest file is not valid YAML or JSON
	ErrInvalidFormat = errors.New("manifest must be valid YAML or JSON")

	// ErrFileNotFound indicates the manifest file does not exist
	ErrFileNotFound = errors.New("manifest file not found")

	// ErrUnsupportedExt indicates an unsupported file extension
	ErrUnsupportedExt = errors.New("unsupported file extension (use .yaml, .yml, or .json)")
)
