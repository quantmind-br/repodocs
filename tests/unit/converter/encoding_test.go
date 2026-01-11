package converter_test

import (
	"testing"

	"github.com/quantmind-br/repodocs-go/internal/converter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDetectEncoding tests encoding detection
func TestDetectEncoding(t *testing.T) {
	tests := []struct {
		name   string
		html   string
		expect string
	}{
		{"UTF-8_meta", `<meta charset="UTF-8">`, "utf-8"},
		{"utf-8 lowercase", `<meta charset="utf-8">`, "utf-8"},
		{"no_charset", `<!DOCTYPE html><html><body>Content</body></html>`, "utf-8"},
		{"empty", "", "utf-8"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enc := converter.DetectEncoding([]byte(tt.html))
			assert.Equal(t, tt.expect, enc)
		})
	}
}

// TestConvertToUTF8 tests conversion to UTF-8
func TestConvertToUTF8(t *testing.T) {
	tests := []struct {
		name    string
		content []byte
	}{
		{"UTF-8 text", []byte("Hello, world!")},
		{"empty", []byte("")},
		{"UTF-8 special", []byte("Hello, 世界!")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := converter.ConvertToUTF8(tt.content)
			require.NoError(t, err)
			assert.NotNil(t, result)
		})
	}
}

// TestIsUTF8 tests UTF-8 detection
func TestIsUTF8(t *testing.T) {
	tests := []struct {
		name   string
		bytes  []byte
		expect bool
	}{
		{"UTF-8", []byte("Hello"), true},
		{"empty", []byte(""), true},
		{"UTF-8 with meta", []byte(`<meta charset="utf-8">`), true},
		{"UTF-8 content", []byte(`<meta charset="UTF-8">Hello`), true},
		{"ISO-8859-1 detected as non-UTF8", []byte(`<meta charset="iso-8859-1">Hello`), false},
		{"Latin1 content", []byte{0xFF, 0xFE}, false}, // Invalid UTF-8 sequence
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.IsUTF8(tt.bytes)
			assert.Equal(t, tt.expect, result)
		})
	}
}

// TestGetEncoder tests getting encoders for charsets
func TestGetEncoder(t *testing.T) {
	tests := []struct {
		name         string
		charsetName  string
		expectError  bool
	}{
		{"UTF-8", "utf-8", false},
		{"ISO-8859-1", "iso-8859-1", false},
		{"Windows-1252", "windows-1252", false},
		{"Unsupported charset", "unknown-charset", true},
		{"Empty charset", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enc, err := converter.GetEncoder(tt.charsetName)
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, enc)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, enc)
			}
		})
	}
}
