package strategies_test

import (
	"testing"

	"github.com/quantmind-br/repodocs-go/internal/output"
	"github.com/quantmind-br/repodocs-go/internal/strategies"
	"github.com/quantmind-br/repodocs-go/internal/utils"
	"github.com/stretchr/testify/assert"
)

func TestNewDocsRSStrategy(t *testing.T) {
	t.Run("with nil deps", func(t *testing.T) {
		strategy := strategies.NewDocsRSStrategy(nil)
		assert.NotNil(t, strategy)
		assert.Equal(t, "docsrs", strategy.Name())
	})

	t.Run("with valid deps", func(t *testing.T) {
		logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})
		tmpDir := t.TempDir()
		writer := output.NewWriter(output.WriterOptions{
			BaseDir: tmpDir,
			Force:   true,
		})

		deps := &strategies.Dependencies{
			Logger: logger,
			Writer: writer,
		}

		strategy := strategies.NewDocsRSStrategy(deps)
		assert.NotNil(t, strategy)
		assert.Equal(t, "docsrs", strategy.Name())
	})
}

func TestDocsRSStrategy_Name(t *testing.T) {
	strategy := strategies.NewDocsRSStrategy(nil)
	assert.Equal(t, "docsrs", strategy.Name())
}

func TestDocsRSStrategy_CanHandle(t *testing.T) {
	strategy := strategies.NewDocsRSStrategy(nil)

	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{"crate root", "https://docs.rs/serde", true},
		{"crate with version", "https://docs.rs/serde/1.0.0", true},
		{"crate latest", "https://docs.rs/serde/latest/serde/", true},
		{"module path", "https://docs.rs/serde/1.0.0/serde/de", true},
		{"struct page", "https://docs.rs/tokio/1.0.0/tokio/net/struct.TcpStream.html", true},
		{"trait page", "https://docs.rs/serde/1.0.0/serde/trait.Serialize.html", true},
		{"crate info page", "https://docs.rs/crate/serde/1.0.0", true},
		{"with trailing slash", "https://docs.rs/tokio/latest/tokio/", true},
		{"nested module", "https://docs.rs/serde/1.0.0/serde/de/value/index.html", true},

		{"source view", "https://docs.rs/serde/1.0.0/src/serde/lib.rs.html", false},
		{"source view alt", "https://docs.rs/crate/serde/1.0.0/source/", false},
		{"github", "https://github.com/serde-rs/serde", false},
		{"crates.io", "https://crates.io/crates/serde", false},
		{"pkg.go.dev", "https://pkg.go.dev/encoding/json", false},
		{"empty", "", false},
		{"malformed", "not-a-url", false},
		{"other domain", "https://example.com/docs.rs/serde", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := strategy.CanHandle(tc.url)
			assert.Equal(t, tc.expected, result, "URL: %s", tc.url)
		})
	}
}

func TestDocsRSStrategy_ParseURL(t *testing.T) {
	tests := []struct {
		name         string
		url          string
		wantCrate    string
		wantVersion  string
		wantIsCrate  bool
		wantIsSource bool
		wantErr      bool
	}{
		{
			name:        "crate root",
			url:         "https://docs.rs/serde",
			wantCrate:   "serde",
			wantVersion: "latest",
		},
		{
			name:        "crate with version",
			url:         "https://docs.rs/serde/1.0.0",
			wantCrate:   "serde",
			wantVersion: "1.0.0",
		},
		{
			name:        "crate info page",
			url:         "https://docs.rs/crate/serde/1.0.0",
			wantCrate:   "serde",
			wantVersion: "1.0.0",
			wantIsCrate: true,
		},
		{
			name:        "crate info page latest",
			url:         "https://docs.rs/crate/tokio/latest",
			wantCrate:   "tokio",
			wantVersion: "latest",
			wantIsCrate: true,
		},
		{
			name:         "source view",
			url:          "https://docs.rs/serde/1.0.0/src/serde/lib.rs.html",
			wantCrate:    "serde",
			wantVersion:  "1.0.0",
			wantIsSource: true,
		},
		{
			name:         "crate source view",
			url:          "https://docs.rs/crate/serde/1.0.0/source/",
			wantCrate:    "serde",
			wantVersion:  "1.0.0",
			wantIsCrate:  true,
			wantIsSource: true,
		},
		{
			name:    "not docs.rs",
			url:     "https://github.com/serde-rs/serde",
			wantErr: true,
		},
		{
			name:    "empty path",
			url:     "https://docs.rs/",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := strategies.ParseDocsRSPathForTest(tc.url)

			if tc.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tc.wantCrate, result.CrateName, "CrateName mismatch")
			assert.Equal(t, tc.wantVersion, result.Version, "Version mismatch")
			assert.Equal(t, tc.wantIsCrate, result.IsCratePage, "IsCratePage mismatch")
			assert.Equal(t, tc.wantIsSource, result.IsSourceView, "IsSourceView mismatch")
		})
	}
}

func TestDocsRSJSONEndpoint(t *testing.T) {
	tests := []struct {
		crate   string
		version string
		want    string
	}{
		{"serde", "1.0.0", "https://docs.rs/crate/serde/1.0.0/json"},
		{"tokio", "latest", "https://docs.rs/crate/tokio/latest/json"},
		{"ratatui", "0.30.0", "https://docs.rs/crate/ratatui/0.30.0/json"},
	}

	for _, tc := range tests {
		t.Run(tc.crate+"_"+tc.version, func(t *testing.T) {
			result := strategies.DocsRSJSONEndpoint(tc.crate, tc.version)
			assert.Equal(t, tc.want, result)
		})
	}
}

func TestDocsRSStrategy_BuildItemURL(t *testing.T) {
	strategy := strategies.NewDocsRSStrategy(nil)
	baseInfo := &strategies.DocsRSURL{
		CrateName: "serde",
		Version:   "1.0.0",
	}

	tests := []struct {
		name     string
		item     *strategies.RustdocItem
		wantURL  string
	}{
		{
			name: "struct item",
			item: &strategies.RustdocItem{
				Name:  ptrString("Deserialize"),
				Inner: map[string]interface{}{"struct": map[string]interface{}{}},
			},
			wantURL: "https://docs.rs/serde/1.0.0/serde/struct.Deserialize.html",
		},
		{
			name: "enum item",
			item: &strategies.RustdocItem{
				Name:  ptrString("Value"),
				Inner: map[string]interface{}{"enum": map[string]interface{}{}},
			},
			wantURL: "https://docs.rs/serde/1.0.0/serde/enum.Value.html",
		},
		{
			name: "trait item",
			item: &strategies.RustdocItem{
				Name:  ptrString("Serialize"),
				Inner: map[string]interface{}{"trait": map[string]interface{}{}},
			},
			wantURL: "https://docs.rs/serde/1.0.0/serde/trait.Serialize.html",
		},
		{
			name: "function item",
			item: &strategies.RustdocItem{
				Name:  ptrString("to_string"),
				Inner: map[string]interface{}{"function": map[string]interface{}{}},
			},
			wantURL: "https://docs.rs/serde/1.0.0/serde/fn.to_string.html",
		},
		{
			name: "module crate root",
			item: &strategies.RustdocItem{
				Name:  ptrString("serde"),
				Inner: map[string]interface{}{"module": map[string]interface{}{"is_crate": true}},
			},
			wantURL: "https://docs.rs/serde/1.0.0/serde/",
		},
		{
			name: "module non-crate",
			item: &strategies.RustdocItem{
				Name:  ptrString("de"),
				Inner: map[string]interface{}{"module": map[string]interface{}{"is_crate": false}},
			},
			wantURL: "https://docs.rs/serde/1.0.0/serde/mod.de.html",
		},
		{
			name: "type alias",
			item: &strategies.RustdocItem{
				Name:  ptrString("Result"),
				Inner: map[string]interface{}{"type_alias": map[string]interface{}{}},
			},
			wantURL: "https://docs.rs/serde/1.0.0/serde/type.Result.html",
		},
		{
			name: "constant",
			item: &strategies.RustdocItem{
				Name:  ptrString("VERSION"),
				Inner: map[string]interface{}{"constant": map[string]interface{}{}},
			},
			wantURL: "https://docs.rs/serde/1.0.0/serde/constant.VERSION.html",
		},
		{
			name: "macro item",
			item: &strategies.RustdocItem{
				Name:  ptrString("derive_serialize"),
				Inner: map[string]interface{}{"macro": map[string]interface{}{}},
			},
			wantURL: "https://docs.rs/serde/1.0.0/serde/macro.derive_serialize.html",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			url := strategy.BuildItemURLForTest(tc.item, baseInfo)
			assert.Equal(t, tc.wantURL, url)
		})
	}
}

func TestDocsRSStrategy_BuildItemTitle(t *testing.T) {
	strategy := strategies.NewDocsRSStrategy(nil)

	tests := []struct {
		name      string
		item      *strategies.RustdocItem
		wantTitle string
	}{
		{
			name: "struct",
			item: &strategies.RustdocItem{
				Name:  ptrString("MyStruct"),
				Inner: map[string]interface{}{"struct": map[string]interface{}{}},
			},
			wantTitle: "Struct MyStruct",
		},
		{
			name: "enum",
			item: &strategies.RustdocItem{
				Name:  ptrString("MyEnum"),
				Inner: map[string]interface{}{"enum": map[string]interface{}{}},
			},
			wantTitle: "Enum MyEnum",
		},
		{
			name: "trait",
			item: &strategies.RustdocItem{
				Name:  ptrString("MyTrait"),
				Inner: map[string]interface{}{"trait": map[string]interface{}{}},
			},
			wantTitle: "Trait MyTrait",
		},
		{
			name: "function",
			item: &strategies.RustdocItem{
				Name:  ptrString("my_func"),
				Inner: map[string]interface{}{"function": map[string]interface{}{}},
			},
			wantTitle: "Function my_func",
		},
		{
			name: "module crate",
			item: &strategies.RustdocItem{
				Name:  ptrString("serde"),
				Inner: map[string]interface{}{"module": map[string]interface{}{"is_crate": true}},
			},
			wantTitle: "Crate serde",
		},
		{
			name: "module non-crate",
			item: &strategies.RustdocItem{
				Name:  ptrString("de"),
				Inner: map[string]interface{}{"module": map[string]interface{}{"is_crate": false}},
			},
			wantTitle: "Module de",
		},
		{
			name: "type alias",
			item: &strategies.RustdocItem{
				Name:  ptrString("Result"),
				Inner: map[string]interface{}{"type_alias": map[string]interface{}{}},
			},
			wantTitle: "Type Result",
		},
		{
			name: "macro",
			item: &strategies.RustdocItem{
				Name:  ptrString("my_macro"),
				Inner: map[string]interface{}{"macro": map[string]interface{}{}},
			},
			wantTitle: "Macro my_macro",
		},
		{
			name: "unknown type",
			item: &strategies.RustdocItem{
				Name:  ptrString("something"),
				Inner: map[string]interface{}{"unknown": map[string]interface{}{}},
			},
			wantTitle: "something",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			title := strategy.BuildItemTitleForTest(tc.item)
			assert.Equal(t, tc.wantTitle, title)
		})
	}
}

func TestDocsRSStrategy_GetItemTypeName(t *testing.T) {
	strategy := strategies.NewDocsRSStrategy(nil)

	tests := []struct {
		name     string
		item     *strategies.RustdocItem
		wantType string
	}{
		{"module", &strategies.RustdocItem{Inner: map[string]interface{}{"module": map[string]interface{}{}}}, "module"},
		{"struct", &strategies.RustdocItem{Inner: map[string]interface{}{"struct": map[string]interface{}{}}}, "struct"},
		{"enum", &strategies.RustdocItem{Inner: map[string]interface{}{"enum": map[string]interface{}{}}}, "enum"},
		{"trait", &strategies.RustdocItem{Inner: map[string]interface{}{"trait": map[string]interface{}{}}}, "trait"},
		{"function", &strategies.RustdocItem{Inner: map[string]interface{}{"function": map[string]interface{}{}}}, "function"},
		{"type_alias", &strategies.RustdocItem{Inner: map[string]interface{}{"type_alias": map[string]interface{}{}}}, "type"},
		{"macro", &strategies.RustdocItem{Inner: map[string]interface{}{"macro": map[string]interface{}{}}}, "macro"},
		{"unknown", &strategies.RustdocItem{Inner: map[string]interface{}{"unknown": map[string]interface{}{}}}, "item"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			typeName := strategy.GetItemTypeNameForTest(tc.item)
			assert.Equal(t, tc.wantType, typeName)
		})
	}
}

func TestDocsRSStrategy_BuildItemDescription(t *testing.T) {
	strategy := strategies.NewDocsRSStrategy(nil)
	baseInfo := &strategies.DocsRSURL{
		CrateName: "tokio",
		Version:   "1.0.0",
	}

	tests := []struct {
		name            string
		item            *strategies.RustdocItem
		wantContains    []string
	}{
		{
			name: "stable struct",
			item: &strategies.RustdocItem{
				Inner: map[string]interface{}{"struct": map[string]interface{}{}},
			},
			wantContains: []string{"crate:tokio", "version:1.0.0", "type:struct", "stability:stable"},
		},
		{
			name: "deprecated function",
			item: &strategies.RustdocItem{
				Inner:       map[string]interface{}{"function": map[string]interface{}{}},
				Deprecation: &strategies.RustdocDeprecation{Since: "1.0.0", Note: "Use new_func instead"},
			},
			wantContains: []string{"crate:tokio", "version:1.0.0", "type:function", "stability:deprecated"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			desc := strategy.BuildItemDescriptionForTest(tc.item, baseInfo)
			for _, want := range tc.wantContains {
				assert.Contains(t, desc, want)
			}
		})
	}
}

func TestDocsRSStrategy_BuildItemTags(t *testing.T) {
	strategy := strategies.NewDocsRSStrategy(nil)
	baseInfo := &strategies.DocsRSURL{
		CrateName: "serde",
		Version:   "1.0.0",
	}

	t.Run("stable item", func(t *testing.T) {
		item := &strategies.RustdocItem{
			Inner: map[string]interface{}{"struct": map[string]interface{}{}},
		}
		tags := strategy.BuildItemTagsForTest(item, baseInfo)

		assert.Contains(t, tags, "docs.rs")
		assert.Contains(t, tags, "rust")
		assert.Contains(t, tags, "serde")
		assert.Contains(t, tags, "struct")
		assert.NotContains(t, tags, "deprecated")
	})

	t.Run("deprecated item", func(t *testing.T) {
		item := &strategies.RustdocItem{
			Inner:       map[string]interface{}{"function": map[string]interface{}{}},
			Deprecation: &strategies.RustdocDeprecation{Since: "1.0.0", Note: "Deprecated"},
		}
		tags := strategy.BuildItemTagsForTest(item, baseInfo)

		assert.Contains(t, tags, "docs.rs")
		assert.Contains(t, tags, "rust")
		assert.Contains(t, tags, "serde")
		assert.Contains(t, tags, "function")
		assert.Contains(t, tags, "deprecated")
	})
}

// Helper function to create string pointer
func ptrString(s string) *string {
	return &s
}
