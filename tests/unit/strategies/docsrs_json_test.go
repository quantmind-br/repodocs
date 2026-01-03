package strategies_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/quantmind-br/repodocs-go/internal/strategies"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRustdocJSON(t *testing.T) {
	t.Run("parse minimal crate fixture", func(t *testing.T) {
		fixturePath := filepath.Join("..", "..", "testdata", "docsrs", "minimal_crate.json")
		data, err := os.ReadFile(fixturePath)
		require.NoError(t, err)

		index, err := strategies.ParseRustdocJSON(data)
		require.NoError(t, err)

		assert.Equal(t, "1.0.0", index.CrateVersion)
		assert.Equal(t, 57, index.FormatVersion)
		assert.False(t, index.IncludesPrivate)
		assert.NotEmpty(t, index.Index)
	})

	t.Run("parse crate root item", func(t *testing.T) {
		fixturePath := filepath.Join("..", "..", "testdata", "docsrs", "minimal_crate.json")
		data, err := os.ReadFile(fixturePath)
		require.NoError(t, err)

		index, err := strategies.ParseRustdocJSON(data)
		require.NoError(t, err)

		rootItem := index.Index["0"]
		require.NotNil(t, rootItem)
		require.NotNil(t, rootItem.Name)
		assert.Equal(t, "example", *rootItem.Name)

		mod := rootItem.GetModule()
		require.NotNil(t, mod)
		assert.True(t, mod.IsCrate)
		assert.Len(t, mod.Items, 4)
	})

	t.Run("parse function item", func(t *testing.T) {
		fixturePath := filepath.Join("..", "..", "testdata", "docsrs", "minimal_crate.json")
		data, err := os.ReadFile(fixturePath)
		require.NoError(t, err)

		index, err := strategies.ParseRustdocJSON(data)
		require.NoError(t, err)

		funcItem := index.Index["1"]
		require.NotNil(t, funcItem)
		require.NotNil(t, funcItem.Name)
		assert.Equal(t, "hello", *funcItem.Name)
		assert.True(t, funcItem.IsPublic())

		fn := funcItem.GetFunction()
		require.NotNil(t, fn)
		require.NotNil(t, fn.Header)
		assert.False(t, fn.Header.IsAsync)
		assert.False(t, fn.Header.IsUnsafe)
		assert.False(t, fn.Header.IsConst)
	})

	t.Run("parse struct item", func(t *testing.T) {
		fixturePath := filepath.Join("..", "..", "testdata", "docsrs", "minimal_crate.json")
		data, err := os.ReadFile(fixturePath)
		require.NoError(t, err)

		index, err := strategies.ParseRustdocJSON(data)
		require.NoError(t, err)

		structItem := index.Index["3"]
		require.NotNil(t, structItem)
		require.NotNil(t, structItem.Name)
		assert.Equal(t, "Config", *structItem.Name)

		st := structItem.GetStruct()
		require.NotNil(t, st)
		assert.NotEmpty(t, st.Impls)
	})

	t.Run("parse enum item", func(t *testing.T) {
		fixturePath := filepath.Join("..", "..", "testdata", "docsrs", "minimal_crate.json")
		data, err := os.ReadFile(fixturePath)
		require.NoError(t, err)

		index, err := strategies.ParseRustdocJSON(data)
		require.NoError(t, err)

		enumItem := index.Index["4"]
		require.NotNil(t, enumItem)
		require.NotNil(t, enumItem.Name)
		assert.Equal(t, "Status", *enumItem.Name)

		en := enumItem.GetEnum()
		require.NotNil(t, en)
		assert.Len(t, en.Variants, 3)
	})

	t.Run("parse invalid JSON", func(t *testing.T) {
		_, err := strategies.ParseRustdocJSON([]byte("not valid json"))
		assert.Error(t, err)
	})

	t.Run("item type detection", func(t *testing.T) {
		fixturePath := filepath.Join("..", "..", "testdata", "docsrs", "minimal_crate.json")
		data, err := os.ReadFile(fixturePath)
		require.NoError(t, err)

		index, err := strategies.ParseRustdocJSON(data)
		require.NoError(t, err)

		assert.Equal(t, "module", index.Index["0"].GetItemType())
		assert.Equal(t, "function", index.Index["1"].GetItemType())
		assert.Equal(t, "struct", index.Index["3"].GetItemType())
		assert.Equal(t, "enum", index.Index["4"].GetItemType())
	})
}

func TestRustdocRenderer_RenderType(t *testing.T) {
	renderer := strategies.NewRustdocRenderer(nil, "test", "1.0.0")

	tests := []struct {
		name     string
		typeData map[string]interface{}
		expected string
	}{
		{
			name:     "primitive i32",
			typeData: map[string]interface{}{"primitive": "i32"},
			expected: "i32",
		},
		{
			name:     "primitive str",
			typeData: map[string]interface{}{"primitive": "str"},
			expected: "str",
		},
		{
			name:     "generic T",
			typeData: map[string]interface{}{"generic": "T"},
			expected: "T",
		},
		{
			name: "borrowed ref immutable",
			typeData: map[string]interface{}{
				"borrowed_ref": map[string]interface{}{
					"is_mutable": false,
					"type":       map[string]interface{}{"primitive": "str"},
				},
			},
			expected: "&str",
		},
		{
			name: "borrowed ref mutable",
			typeData: map[string]interface{}{
				"borrowed_ref": map[string]interface{}{
					"is_mutable": true,
					"type":       map[string]interface{}{"generic": "T"},
				},
			},
			expected: "&mut T",
		},
		{
			name: "resolved path simple",
			typeData: map[string]interface{}{
				"resolved_path": map[string]interface{}{
					"path": "Vec",
					"id":   10,
				},
			},
			expected: "Vec",
		},
		{
			name: "slice",
			typeData: map[string]interface{}{
				"slice": map[string]interface{}{"primitive": "u8"},
			},
			expected: "[u8]",
		},
		{
			name:     "empty tuple",
			typeData: map[string]interface{}{"tuple": []interface{}{}},
			expected: "()",
		},
		{
			name: "tuple with elements",
			typeData: map[string]interface{}{
				"tuple": []interface{}{
					map[string]interface{}{"primitive": "i32"},
					map[string]interface{}{"primitive": "bool"},
				},
			},
			expected: "(i32, bool)",
		},
		{
			name: "array",
			typeData: map[string]interface{}{
				"array": map[string]interface{}{
					"type": map[string]interface{}{"primitive": "u8"},
					"len":  "32",
				},
			},
			expected: "[u8; 32]",
		},
		{
			name: "raw pointer const",
			typeData: map[string]interface{}{
				"raw_pointer": map[string]interface{}{
					"is_mutable": false,
					"type":       map[string]interface{}{"primitive": "u8"},
				},
			},
			expected: "*const u8",
		},
		{
			name: "raw pointer mut",
			typeData: map[string]interface{}{
				"raw_pointer": map[string]interface{}{
					"is_mutable": true,
					"type":       map[string]interface{}{"primitive": "u8"},
				},
			},
			expected: "*mut u8",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := renderer.RenderTypeMap(tc.typeData)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestRustdocRenderer_RenderItem(t *testing.T) {
	fixturePath := filepath.Join("..", "..", "testdata", "docsrs", "minimal_crate.json")
	data, err := os.ReadFile(fixturePath)
	require.NoError(t, err)

	index, err := strategies.ParseRustdocJSON(data)
	require.NoError(t, err)

	renderer := strategies.NewRustdocRenderer(index, "example", "1.0.0")

	t.Run("render crate root", func(t *testing.T) {
		item := index.Index["0"]
		result := renderer.RenderItem(item)

		assert.Contains(t, result, "# Crate `example`")
		assert.Contains(t, result, "Example Crate")
		assert.Contains(t, result, "## Contents")
	})

	t.Run("render function", func(t *testing.T) {
		item := index.Index["1"]
		result := renderer.RenderItem(item)

		assert.Contains(t, result, "# Function `hello`")
		assert.Contains(t, result, "```rust")
		assert.Contains(t, result, "pub fn hello()")
		assert.Contains(t, result, "Says hello")
	})

	t.Run("render function with args", func(t *testing.T) {
		item := index.Index["2"]
		result := renderer.RenderItem(item)

		assert.Contains(t, result, "# Function `greet`")
		assert.Contains(t, result, "name: &str")
		assert.Contains(t, result, "-> String")
	})

	t.Run("render struct", func(t *testing.T) {
		item := index.Index["3"]
		result := renderer.RenderItem(item)

		assert.Contains(t, result, "# Struct `Config`")
		assert.Contains(t, result, "pub struct Config")
		assert.Contains(t, result, "Configuration struct")
	})

	t.Run("render enum", func(t *testing.T) {
		item := index.Index["4"]
		result := renderer.RenderItem(item)

		assert.Contains(t, result, "# Enum `Status`")
		assert.Contains(t, result, "pub enum Status")
		assert.Contains(t, result, "Status enum")
	})
}
