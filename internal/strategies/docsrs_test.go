package strategies

import (
	"testing"

	"github.com/quantmind-br/repodocs-go/internal/utils"
)

// TestNewDocsRSStrategy tests strategy creation
func TestNewDocsRSStrategy(t *testing.T) {
	t.Run("nil dependencies", func(t *testing.T) {
		s := NewDocsRSStrategy(nil)
		if s == nil {
			t.Fatal("Expected non-nil strategy")
		}
		if s.Name() != "docsrs" {
			t.Errorf("Expected name 'docsrs', got '%s'", s.Name())
		}
		if s.baseHost != "docs.rs" {
			t.Errorf("Expected baseHost 'docs.rs', got '%s'", s.baseHost)
		}
	})

	t.Run("with dependencies", func(t *testing.T) {
		deps := &Dependencies{
			Logger: utils.NewDefaultLogger(),
		}
		s := NewDocsRSStrategy(deps)
		if s == nil {
			t.Fatal("Expected non-nil strategy")
		}
		if s.deps != deps {
			t.Error("Expected deps to be set")
		}
		if s.baseHost != "docs.rs" {
			t.Errorf("Expected baseHost 'docs.rs', got '%s'", s.baseHost)
		}
	})
}

// TestDocsRSStrategyName tests strategy name
func TestDocsRSStrategyName(t *testing.T) {
	s := NewDocsRSStrategy(nil)
	if s.Name() != "docsrs" {
		t.Errorf("Expected name 'docsrs', got '%s'", s.Name())
	}
}

// TestDocsRSStrategySetFetcher tests fetcher setter
func TestDocsRSStrategySetFetcher(t *testing.T) {
	s := NewDocsRSStrategy(nil)
	mockFetcher := &mockFetcher{}
	s.SetFetcher(mockFetcher)
	if s.fetcher != mockFetcher {
		t.Error("Expected fetcher to be set")
	}
}

// TestDocsRSStrategySetBaseHost tests base host setter
func TestDocsRSStrategySetBaseHost(t *testing.T) {
	s := NewDocsRSStrategy(nil)
	s.SetBaseHost("custom.docs.rs")
	if s.baseHost != "custom.docs.rs" {
		t.Errorf("Expected baseHost 'custom.docs.rs', got '%s'", s.baseHost)
	}
}

// TestDocsRSStrategyCanHandle tests URL handling detection
func TestDocsRSStrategyCanHandle(t *testing.T) {
	s := NewDocsRSStrategy(nil)

	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{"valid crate URL", "https://docs.rs/serde", true},
		{"valid crate with version", "https://docs.rs/serde/1.0", true},
		{"valid crate with module", "https://docs.rs/serde/1.0/serde", true},
		{"invalid domain", "https://example.com/crate", false},
		{"source view", "https://docs.rs/serde/1.0/src/serde/lib.rs", false},
		{"empty string", "", false},
		{"invalid URL", "not-a-url", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.CanHandle(tt.url)
			if result != tt.expected {
				t.Errorf("CanHandle(%q) = %v, want %v", tt.url, result, tt.expected)
			}
		})
	}
}

// TestParseDocsRSPath tests docs.rs URL parsing
func TestParseDocsRSPath(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		wantErr   bool
		checkFunc func(*testing.T, *DocsRSURL)
	}{
		{
			name:    "simple crate",
			url:     "https://docs.rs/serde",
			wantErr: false,
			checkFunc: func(t *testing.T, p *DocsRSURL) {
				if p.CrateName != "serde" {
					t.Errorf("Expected CrateName 'serde', got '%s'", p.CrateName)
				}
			},
		},
		{
			name:    "crate with version",
			url:     "https://docs.rs/serde/1.0.0",
			wantErr: false,
			checkFunc: func(t *testing.T, p *DocsRSURL) {
				if p.CrateName != "serde" {
					t.Errorf("Expected CrateName 'serde', got '%s'", p.CrateName)
				}
				if p.Version != "1.0.0" {
					t.Errorf("Expected Version '1.0.0', got '%s'", p.Version)
				}
			},
		},
		{
			name:    "crate with version",
			url:     "https://docs.rs/serde/1.0.0",
			wantErr: false,
			checkFunc: func(t *testing.T, p *DocsRSURL) {
				if p.CrateName != "serde" {
					t.Errorf("Expected CrateName 'serde', got '%s'", p.CrateName)
				}
				if p.Version != "1.0.0" {
					t.Errorf("Expected Version '1.0.0', got '%s'", p.Version)
				}
			},
		},
		{
			name:    "crate with module path",
			url:     "https://docs.rs/serde/1.0.0/serde/Serialize",
			wantErr: false,
			checkFunc: func(t *testing.T, p *DocsRSURL) {
				if p.CrateName != "serde" {
					t.Errorf("Expected CrateName 'serde', got '%s'", p.CrateName)
				}
				if p.ModulePath != "Serialize" {
					t.Errorf("Expected ModulePath 'Serialize', got '%s'", p.ModulePath)
				}
			},
		},
		{
			name:    "crate page",
			url:     "https://docs.rs/crate/serde",
			wantErr: false,
			checkFunc: func(t *testing.T, p *DocsRSURL) {
				if !p.IsCratePage {
					t.Error("Expected IsCratePage to be true")
				}
				if p.CrateName != "serde" {
					t.Errorf("Expected CrateName 'serde', got '%s'", p.CrateName)
				}
			},
		},
		{
			name:    "source view",
			url:     "https://docs.rs/serde/1.0.0/src/serde/lib.rs",
			wantErr: false,
			checkFunc: func(t *testing.T, p *DocsRSURL) {
				if !p.IsSourceView {
					t.Error("Expected IsSourceView to be true")
				}
			},
		},
		{
			name:    "invalid domain",
			url:     "https://example.com/serde",
			wantErr: true,
		},
		{
			name:    "empty path",
			url:     "https://docs.rs",
			wantErr: true,
		},
		{
			name:    "invalid URL",
			url:     "://invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseDocsRSPath(tt.url)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, result)
			}
		})
	}
}

// TestParseDocsRSPathWithHost tests docs.rs URL parsing with custom host
func TestParseDocsRSPathWithHost(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expectedHost string
		wantErr     bool
	}{
		{
			name:        "matches custom host",
			url:         "https://custom.docs.rs/serde",
			expectedHost: "custom.docs.rs",
			wantErr:     false,
		},
		{
			name:        "doesn't match custom host",
			url:         "https://other.docs.rs/serde",
			expectedHost: "custom.docs.rs",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseDocsRSPathWithHost(tt.url, tt.expectedHost)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// TestDocsRSStrategy_ParseURL tests the parseURL method
func TestDocsRSStrategy_ParseURL(t *testing.T) {
	s := NewDocsRSStrategy(nil)

	t.Run("parses valid URL", func(t *testing.T) {
		parsed, err := s.parseURL("https://docs.rs/serde/1.0")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if parsed.CrateName != "serde" {
			t.Errorf("Expected CrateName 'serde', got '%s'", parsed.CrateName)
		}
		if parsed.Version != "1.0" {
			t.Errorf("Expected Version '1.0', got '%s'", parsed.Version)
		}
	})

	t.Run("returns error for invalid URL", func(t *testing.T) {
		_, err := s.parseURL("://invalid")
		if err == nil {
			t.Error("Expected error but got none")
		}
	})
}

// TestDocsRSJSONEndpoint tests JSON endpoint construction
func TestDocsRSJSONEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		crate    string
		version  string
		expected string
	}{
		{
			name:     "simple crate",
			crate:    "serde",
			version:  "1.0.0",
			expected: "https://docs.rs/crate/serde/1.0.0/json",
		},
		{
			name:     "crate with hyphen",
			crate:    "tokio-util",
			version:  "0.7.0",
			expected: "https://docs.rs/crate/tokio-util/0.7.0/json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DocsRSJSONEndpoint(tt.crate, tt.version)
			if result != tt.expected {
				t.Errorf("DocsRSJSONEndpoint(%q, %q) = %q, want %q",
					tt.crate, tt.version, result, tt.expected)
			}
		})
	}
}
