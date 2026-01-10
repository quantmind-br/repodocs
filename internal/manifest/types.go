package manifest

import (
	"fmt"
	"time"
)

// Config represents the complete manifest configuration
type Config struct {
	Sources []Source `yaml:"sources" json:"sources"`
	Options Options  `yaml:"options" json:"options"`
}

// Source represents an individual documentation source
type Source struct {
	URL             string   `yaml:"url" json:"url"`
	Strategy        string   `yaml:"strategy,omitempty" json:"strategy,omitempty"`
	ContentSelector string   `yaml:"content_selector,omitempty" json:"content_selector,omitempty"`
	ExcludeSelector string   `yaml:"exclude_selector,omitempty" json:"exclude_selector,omitempty"`
	Exclude         []string `yaml:"exclude,omitempty" json:"exclude,omitempty"`
	Include         []string `yaml:"include,omitempty" json:"include,omitempty"`
	MaxDepth        int      `yaml:"max_depth,omitempty" json:"max_depth,omitempty"`
	RenderJS        *bool    `yaml:"render_js,omitempty" json:"render_js,omitempty"`
	Limit           int      `yaml:"limit,omitempty" json:"limit,omitempty"`
}

// Options represents global manifest options
type Options struct {
	ContinueOnError bool          `yaml:"continue_on_error" json:"continue_on_error"`
	Output          string        `yaml:"output,omitempty" json:"output,omitempty"`
	Concurrency     int           `yaml:"concurrency,omitempty" json:"concurrency,omitempty"`
	CacheTTL        time.Duration `yaml:"cache_ttl,omitempty" json:"cache_ttl,omitempty"`
}

// Validate validates the manifest configuration
func (c *Config) Validate() error {
	if len(c.Sources) == 0 {
		return ErrNoSources
	}
	for i, src := range c.Sources {
		if src.URL == "" {
			return fmt.Errorf("source %d: %w", i, ErrEmptyURL)
		}
	}
	return nil
}

// DefaultOptions returns options with sensible defaults
func DefaultOptions() Options {
	return Options{
		ContinueOnError: false,
		Output:          "./docs",
		Concurrency:     5,
		CacheTTL:        24 * time.Hour,
	}
}
