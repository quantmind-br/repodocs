package strategies

import (
	"context"
	"testing"

	"github.com/quantmind-br/repodocs-go/internal/output"
	"github.com/quantmind-br/repodocs-go/internal/utils"
	"github.com/stretchr/testify/assert"
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
		name         string
		url          string
		expectedHost string
		wantErr      bool
	}{
		{
			name:         "matches custom host",
			url:          "https://custom.docs.rs/serde",
			expectedHost: "custom.docs.rs",
			wantErr:      false,
		},
		{
			name:         "doesn't match custom host",
			url:          "https://other.docs.rs/serde",
			expectedHost: "custom.docs.rs",
			wantErr:      true,
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

// TestDocsRSStrategy_checkFormatVersion tests format version validation
func TestDocsRSStrategy_checkFormatVersion(t *testing.T) {
	tests := []struct {
		name    string
		version int
		wantErr bool
	}{
		{
			name:    "valid version - middle range",
			version: 40,
			wantErr: false,
		},
		{
			name:    "valid version - min boundary",
			version: 30,
			wantErr: false,
		},
		{
			name:    "valid version - max boundary",
			version: 60,
			wantErr: false,
		},
		{
			name:    "too old version",
			version: 20,
			wantErr: true,
		},
		{
			name:    "future version - warning but ok",
			version: 70,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewDocsRSStrategy(&Dependencies{
				Logger: utils.NewDefaultLogger(),
			})
			err := s.checkFormatVersion(tt.version)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestDocsRSStrategy_GetItemByID tests item retrieval from index
func TestDocsRSStrategy_GetItemByID(t *testing.T) {
	index := &RustdocIndex{
		Index: map[string]*RustdocItem{
			"123": {Name: strPtr("item1")},
			"456": {Name: strPtr("item2")},
		},
	}

	s := NewDocsRSStrategy(nil)

	t.Run("get by string", func(t *testing.T) {
		result := s.getItemByID(index, "123")
		assert.NotNil(t, result)
		assert.Equal(t, "item1", *result.Name)
	})

	t.Run("get by float64", func(t *testing.T) {
		result := s.getItemByID(index, 456.0)
		assert.NotNil(t, result)
		assert.Equal(t, "item2", *result.Name)
	})

	t.Run("get by int", func(t *testing.T) {
		result := s.getItemByID(index, 123)
		assert.NotNil(t, result)
		assert.Equal(t, "item1", *result.Name)
	})

	t.Run("not found", func(t *testing.T) {
		result := s.getItemByID(index, "999")
		assert.Nil(t, result)
	})
}

// TestRustdocItem_IsPublic tests public visibility detection
func TestRustdocItem_IsPublic(t *testing.T) {
	tests := []struct {
		name       string
		visibility interface{}
		want       bool
	}{
		{
			name:       "public string",
			visibility: "public",
			want:       true,
		},
		{
			name:       "private string",
			visibility: "private",
			want:       false,
		},
		{
			name:       "restricted visibility (pub(crate))",
			visibility: map[string]interface{}{"restricted": map[string]string{"parent": "crate"}},
			want:       false,
		},
		{
			name:       "nil visibility",
			visibility: nil,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := &RustdocItem{Visibility: tt.visibility}
			got := item.IsPublic()
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestRustdocItem_GetItemType tests item type detection
func TestRustdocItem_GetItemType(t *testing.T) {
	tests := []struct {
		name     string
		inner    map[string]interface{}
		expected string
	}{
		{
			name:     "module type",
			inner:    map[string]interface{}{"module": map[string]interface{}{}},
			expected: "module",
		},
		{
			name:     "struct type",
			inner:    map[string]interface{}{"struct": map[string]interface{}{}},
			expected: "struct",
		},
		{
			name:     "function type",
			inner:    map[string]interface{}{"function": map[string]interface{}{}},
			expected: "function",
		},
		{
			name:     "unknown type",
			inner:    nil,
			expected: "unknown",
		},
		{
			name:     "empty inner",
			inner:    map[string]interface{}{},
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := &RustdocItem{Inner: tt.inner}
			got := item.GetItemType()
			assert.Equal(t, tt.expected, got)
		})
	}
}

// TestRustdocItem_Getters tests all getter methods
func TestRustdocItem_Getters(t *testing.T) {
	t.Run("GetModule", func(t *testing.T) {
		item := &RustdocItem{
			Inner: map[string]interface{}{
				"module": map[string]interface{}{
					"is_crate": true,
					"items":    []interface{}{"1", "2"},
				},
			},
		}
		mod := item.GetModule()
		assert.NotNil(t, mod)
		assert.True(t, mod.IsCrate)
		assert.Len(t, mod.Items, 2)
	})

	t.Run("GetFunction", func(t *testing.T) {
		item := &RustdocItem{
			Inner: map[string]interface{}{
				"function": map[string]interface{}{
					"sig": map[string]interface{}{
						"inputs": []interface{}{},
						"output": nil,
					},
				},
			},
		}
		fn := item.GetFunction()
		assert.NotNil(t, fn)
		assert.NotNil(t, fn.Sig)
	})

	t.Run("GetStruct", func(t *testing.T) {
		item := &RustdocItem{
			Inner: map[string]interface{}{
				"struct": map[string]interface{}{
					"kind": "plain",
				},
			},
		}
		st := item.GetStruct()
		assert.NotNil(t, st)
	})

	t.Run("GetEnum", func(t *testing.T) {
		item := &RustdocItem{
			Inner: map[string]interface{}{
				"enum": map[string]interface{}{
					"variants": []interface{}{},
				},
			},
		}
		en := item.GetEnum()
		assert.NotNil(t, en)
	})

	t.Run("GetTrait", func(t *testing.T) {
		item := &RustdocItem{
			Inner: map[string]interface{}{
				"trait": map[string]interface{}{
					"is_auto": false,
					"items":   []interface{}{},
				},
			},
		}
		tr := item.GetTrait()
		assert.NotNil(t, tr)
	})

	t.Run("GetUse", func(t *testing.T) {
		item := &RustdocItem{
			Inner: map[string]interface{}{
				"use": map[string]interface{}{
					"source": "std::collections",
					"name":   "HashMap",
				},
			},
		}
		use := item.GetUse()
		assert.NotNil(t, use)
		assert.Equal(t, "std::collections", use.Source)
	})

	t.Run("nil inner returns nil", func(t *testing.T) {
		item := &RustdocItem{Inner: nil}
		assert.Nil(t, item.GetModule())
		assert.Nil(t, item.GetFunction())
		assert.Nil(t, item.GetStruct())
		assert.Nil(t, item.GetEnum())
		assert.Nil(t, item.GetTrait())
		assert.Nil(t, item.GetUse())
	})
}

// TestDocsRSStrategy_HasDocumentableChildren tests documentable children detection
func TestDocsRSStrategy_HasDocumentableChildren(t *testing.T) {
	s := NewDocsRSStrategy(nil)

	t.Run("module with items", func(t *testing.T) {
		item := &RustdocItem{
			Inner: map[string]interface{}{
				"module": map[string]interface{}{
					"items": []interface{}{"1", "2", "3"},
				},
			},
		}
		assert.True(t, s.hasDocumentableChildren(item))
	})

	t.Run("module without items", func(t *testing.T) {
		item := &RustdocItem{
			Inner: map[string]interface{}{
				"module": map[string]interface{}{
					"items": []interface{}{},
				},
			},
		}
		assert.False(t, s.hasDocumentableChildren(item))
	})

	t.Run("trait with items", func(t *testing.T) {
		item := &RustdocItem{
			Inner: map[string]interface{}{
				"trait": map[string]interface{}{
					"items": []interface{}{"1"},
				},
			},
		}
		assert.True(t, s.hasDocumentableChildren(item))
	})

	t.Run("struct with impls", func(t *testing.T) {
		item := &RustdocItem{
			Inner: map[string]interface{}{
				"struct": map[string]interface{}{
					"impls": []interface{}{"impl1", "impl2"},
				},
			},
		}
		assert.True(t, s.hasDocumentableChildren(item))
	})

	t.Run("enum with variants", func(t *testing.T) {
		item := &RustdocItem{
			Inner: map[string]interface{}{
				"enum": map[string]interface{}{
					"variants": []interface{}{"v1", "v2"},
				},
			},
		}
		assert.True(t, s.hasDocumentableChildren(item))
	})

	t.Run("function has no children", func(t *testing.T) {
		item := &RustdocItem{
			Inner: map[string]interface{}{
				"function": map[string]interface{}{},
			},
		}
		assert.False(t, s.hasDocumentableChildren(item))
	})
}

// TestParseRustdocJSON tests JSON parsing
func TestParseRustdocJSON(t *testing.T) {
	t.Run("valid JSON", func(t *testing.T) {
		jsonData := []byte(`{
			"root": "0",
			"crate_version": "1.0.0",
			"format_version": 30,
			"index": {}
		}`)

		index, err := ParseRustdocJSON(jsonData)
		assert.NoError(t, err)
		assert.NotNil(t, index)
		assert.Equal(t, "1.0.0", index.CrateVersion)
		assert.Equal(t, 30, index.FormatVersion)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		jsonData := []byte(`{invalid json}`)
		_, err := ParseRustdocJSON(jsonData)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse")
	})
}

// TestDocsRSStrategy_Execute_Errors tests error cases in Execute
func TestDocsRSStrategy_Execute_Errors(t *testing.T) {
	t.Run("nil fetcher", func(t *testing.T) {
		deps := &Dependencies{
			Logger: utils.NewDefaultLogger(),
			Writer: output.NewWriter(output.WriterOptions{}),
		}
		s := NewDocsRSStrategy(deps)

		ctx := context.Background()
		err := s.Execute(ctx, "https://docs.rs/serde/", Options{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "fetcher is nil")
	})

	t.Run("nil writer", func(t *testing.T) {
		mockFetcher := &mockFetcher{}
		deps := &Dependencies{
			Fetcher: mockFetcher,
			Logger:  utils.NewDefaultLogger(),
		}
		s := NewDocsRSStrategy(deps)

		ctx := context.Background()
		err := s.Execute(ctx, "https://docs.rs/serde/", Options{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "writer is nil")
	})

	t.Run("invalid URL", func(t *testing.T) {
		mockFetcher := &mockFetcher{}
		deps := &Dependencies{
			Fetcher: mockFetcher,
			Writer:  output.NewWriter(output.WriterOptions{}),
			Logger:  utils.NewDefaultLogger(),
		}
		s := NewDocsRSStrategy(deps)

		ctx := context.Background()
		err := s.Execute(ctx, "://invalid", Options{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid docs.rs URL")
	})
}

// TestDocsRSStrategy_GetItemTypeName tests item type name extraction
func TestDocsRSStrategy_GetItemTypeName(t *testing.T) {
	s := NewDocsRSStrategy(nil)

	tests := []struct {
		name     string
		inner    map[string]interface{}
		expected string
	}{
		{
			name:     "module",
			inner:    map[string]interface{}{"module": map[string]interface{}{}},
			expected: "module",
		},
		{
			name:     "struct",
			inner:    map[string]interface{}{"struct": map[string]interface{}{}},
			expected: "struct",
		},
		{
			name:     "enum",
			inner:    map[string]interface{}{"enum": map[string]interface{}{}},
			expected: "enum",
		},
		{
			name:     "trait",
			inner:    map[string]interface{}{"trait": map[string]interface{}{}},
			expected: "trait",
		},
		{
			name:     "function",
			inner:    map[string]interface{}{"function": map[string]interface{}{}},
			expected: "function",
		},
		{
			name:     "type alias",
			inner:    map[string]interface{}{"type_alias": map[string]interface{}{}},
			expected: "type",
		},
		{
			name:     "macro",
			inner:    map[string]interface{}{"macro": map[string]interface{}{}},
			expected: "macro",
		},
		{
			name:     "unknown",
			inner:    map[string]interface{}{},
			expected: "item",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := &RustdocItem{Inner: tt.inner}
			result := s.GetItemTypeNameForTest(item)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestDocsRSStrategy_BuildItemTitle tests title building
func TestDocsRSStrategy_BuildItemTitle(t *testing.T) {
	s := NewDocsRSStrategy(nil)

	tests := []struct {
		name     string
		item     *RustdocItem
		expected string
	}{
		{
			name: "crate module",
			item: &RustdocItem{
				Name: strPtr("my_crate"),
				Inner: map[string]interface{}{
					"module": map[string]interface{}{"is_crate": true},
				},
			},
			expected: "Crate my_crate",
		},
		{
			name: "regular module",
			item: &RustdocItem{
				Name: strPtr("my_module"),
				Inner: map[string]interface{}{
					"module": map[string]interface{}{"is_crate": false},
				},
			},
			expected: "Module my_module",
		},
		{
			name: "struct",
			item: &RustdocItem{
				Name: strPtr("MyStruct"),
				Inner: map[string]interface{}{
					"struct": map[string]interface{}{},
				},
			},
			expected: "Struct MyStruct",
		},
		{
			name: "enum",
			item: &RustdocItem{
				Name: strPtr("MyEnum"),
				Inner: map[string]interface{}{
					"enum": map[string]interface{}{},
				},
			},
			expected: "Enum MyEnum",
		},
		{
			name: "trait",
			item: &RustdocItem{
				Name: strPtr("MyTrait"),
				Inner: map[string]interface{}{
					"trait": map[string]interface{}{},
				},
			},
			expected: "Trait MyTrait",
		},
		{
			name: "function",
			item: &RustdocItem{
				Name: strPtr("my_function"),
				Inner: map[string]interface{}{
					"function": map[string]interface{}{},
				},
			},
			expected: "Function my_function",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.BuildItemTitleForTest(tt.item)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestDocsRSStrategy_BuildItemTags tests tag building
func TestDocsRSStrategy_BuildItemTags(t *testing.T) {
	s := NewDocsRSStrategy(nil)
	baseInfo := &DocsRSURL{
		CrateName: "serde",
		Version:   "1.0.0",
	}

	t.Run("struct tags", func(t *testing.T) {
		item := &RustdocItem{
			Name: strPtr("Serialize"),
			Inner: map[string]interface{}{
				"struct": map[string]interface{}{},
			},
		}
		tags := s.BuildItemTagsForTest(item, baseInfo)
		assert.Contains(t, tags, "docs.rs")
		assert.Contains(t, tags, "rust")
		assert.Contains(t, tags, "serde")
		assert.Contains(t, tags, "struct")
	})

	t.Run("deprecated item", func(t *testing.T) {
		item := &RustdocItem{
			Name: strPtr("OldStruct"),
			Inner: map[string]interface{}{
				"struct": map[string]interface{}{},
			},
			Deprecation: &RustdocDeprecation{
				Since: "1.0.0",
				Note:  "Use NewStruct instead",
			},
		}
		tags := s.BuildItemTagsForTest(item, baseInfo)
		assert.Contains(t, tags, "deprecated")
	})
}

// TestDocsRSStrategy_BuildItemDescription tests description building
func TestDocsRSStrategy_BuildItemDescription(t *testing.T) {
	s := NewDocsRSStrategy(nil)
	baseInfo := &DocsRSURL{
		CrateName: "serde",
		Version:   "1.0.0",
	}

	t.Run("stable item", func(t *testing.T) {
		item := &RustdocItem{
			Name: strPtr("Serialize"),
			Inner: map[string]interface{}{
				"struct": map[string]interface{}{},
			},
		}
		desc := s.BuildItemDescriptionForTest(item, baseInfo)
		assert.Contains(t, desc, "crate:serde")
		assert.Contains(t, desc, "version:1.0.0")
		assert.Contains(t, desc, "type:struct")
		assert.Contains(t, desc, "stability:stable")
	})

	t.Run("deprecated item", func(t *testing.T) {
		item := &RustdocItem{
			Name: strPtr("OldStruct"),
			Inner: map[string]interface{}{
				"struct": map[string]interface{}{},
			},
			Deprecation: &RustdocDeprecation{
				Since: "1.0.0",
				Note:  "Use NewStruct instead",
			},
		}
		desc := s.BuildItemDescriptionForTest(item, baseInfo)
		assert.Contains(t, desc, "stability:deprecated")
	})
}

// Helper function
func strPtr(s string) *string {
	return &s
}
