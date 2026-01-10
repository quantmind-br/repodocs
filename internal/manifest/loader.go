package manifest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Loader loads and validates manifest files
type Loader struct{}

// NewLoader creates a new manifest loader
func NewLoader() *Loader {
	return &Loader{}
}

// Load reads and parses a manifest file from the given path
func (l *Loader) Load(path string) (*Config, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("%w: %s", ErrFileNotFound, path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest file: %w", err)
	}

	return l.LoadFromBytes(data, filepath.Ext(path))
}

// LoadFromBytes parses manifest configuration from raw bytes
func (l *Loader) LoadFromBytes(data []byte, ext string) (*Config, error) {
	ext = strings.ToLower(ext)

	var cfg Config
	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidFormat, err)
		}
	case ".json":
		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidFormat, err)
		}
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedExt, ext)
	}

	l.applyDefaults(&cfg)

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (l *Loader) applyDefaults(cfg *Config) {
	defaults := DefaultOptions()

	if cfg.Options.Output == "" {
		cfg.Options.Output = defaults.Output
	}
	if cfg.Options.Concurrency == 0 {
		cfg.Options.Concurrency = defaults.Concurrency
	}
	if cfg.Options.CacheTTL == 0 {
		cfg.Options.CacheTTL = defaults.CacheTTL
	}
}
