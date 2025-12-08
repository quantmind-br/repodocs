package app_test

import (
	"testing"

	"github.com/quantmind-br/repodocs-go/internal/converter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsUTF8(t *testing.T) {
	// Note: IsUTF8 relies on DetectEncoding which uses charset.DetermineEncoding
	// from golang.org/x/net/html/charset. The detection may return different
	// encodings based on content analysis, not just meta tags.
	tests := []struct {
		name  string
		input []byte
		want  bool
	}{
		{
			name:  "valid utf8 with unicode",
			input: []byte("Hello, 世界"),
			want:  true,
		},
		{
			name:  "utf8 with charset meta tag",
			input: []byte(`<html><head><meta charset="utf-8"></head><body>Test</body></html>`),
			want:  true,
		},
		{
			name:  "utf8 with UTF-8 charset meta tag",
			input: []byte(`<html><head><meta charset="UTF-8"></head><body>Test</body></html>`),
			want:  true,
		},
		{
			name:  "utf8 with cyrillic",
			input: []byte("Привет мир"),
			want:  true,
		},
		{
			name:  "utf8 with japanese",
			input: []byte("こんにちは"),
			want:  true,
		},
		{
			name:  "iso-8859-1 declared",
			input: []byte(`<html><head><meta charset="iso-8859-1"></head><body>Test</body></html>`),
			want:  false,
		},
		{
			name:  "windows-1252 declared",
			input: []byte(`<html><head><meta charset="windows-1252"></head><body>Test</body></html>`),
			want:  false,
		},
		// Note: ASCII-only and empty content may be detected as windows-1252
		// by charset.DetermineEncoding, which is expected behavior
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := converter.IsUTF8(tc.input)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestGetEncoder(t *testing.T) {
	tests := []struct {
		name       string
		charset    string
		wantErr    bool
		wantNotNil bool
	}{
		{
			name:       "utf-8 lowercase",
			charset:    "utf-8",
			wantErr:    false,
			wantNotNil: true,
		},
		{
			name:       "UTF-8 uppercase",
			charset:    "UTF-8",
			wantErr:    false,
			wantNotNil: true,
		},
		{
			name:       "iso-8859-1",
			charset:    "iso-8859-1",
			wantErr:    false,
			wantNotNil: true,
		},
		{
			name:       "windows-1252",
			charset:    "windows-1252",
			wantErr:    false,
			wantNotNil: true,
		},
		{
			name:       "latin1",
			charset:    "latin1",
			wantErr:    false,
			wantNotNil: true,
		},
		{
			name:       "shift_jis",
			charset:    "shift_jis",
			wantErr:    false,
			wantNotNil: true,
		},
		{
			name:       "euc-jp",
			charset:    "euc-jp",
			wantErr:    false,
			wantNotNil: true,
		},
		{
			name:       "gbk",
			charset:    "gbk",
			wantErr:    false,
			wantNotNil: true,
		},
		{
			name:       "unknown-charset",
			charset:    "unknown-charset",
			wantErr:    true,
			wantNotNil: false,
		},
		{
			name:       "completely-made-up",
			charset:    "completely-made-up",
			wantErr:    true,
			wantNotNil: false,
		},
		{
			name:       "empty string",
			charset:    "",
			wantErr:    true,
			wantNotNil: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			enc, err := converter.GetEncoder(tc.charset)

			if tc.wantErr {
				assert.Error(t, err, "Expected error for charset: %s", tc.charset)
				assert.Nil(t, enc)
			} else {
				assert.NoError(t, err, "Expected no error for charset: %s", tc.charset)
				assert.NotNil(t, enc, "Expected encoder for charset: %s", tc.charset)
			}
		})
	}
}

func TestDetectEncoding(t *testing.T) {
	// Note: charset.DetermineEncoding may return different default encodings
	// when no explicit charset is declared. The function extracts charset from
	// meta tags first, then falls back to charset detection.
	tests := []struct {
		name    string
		content []byte
		want    string
	}{
		{
			name:    "explicit utf-8 meta charset",
			content: []byte(`<html><head><meta charset="utf-8"></head><body>Test</body></html>`),
			want:    "utf-8",
		},
		{
			name:    "explicit iso-8859-1 meta charset",
			content: []byte(`<html><head><meta charset="iso-8859-1"></head><body>Test</body></html>`),
			want:    "iso-8859-1",
		},
		{
			name:    "content-type meta tag",
			content: []byte(`<html><head><meta http-equiv="Content-Type" content="text/html; charset=utf-8"></head></html>`),
			want:    "utf-8",
		},
		// Note: Plain text without meta charset may be detected as windows-1252
		// by charset.DetermineEncoding, which is acceptable behavior
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := converter.DetectEncoding(tc.content)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestConvertToUTF8(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		wantErr bool
	}{
		{
			name:    "utf-8 content passes through",
			input:   []byte(`<html><head><meta charset="utf-8"></head><body>Hello, 世界</body></html>`),
			wantErr: false,
		},
		{
			name:    "ascii content passes through",
			input:   []byte("Plain ASCII text"),
			wantErr: false,
		},
		{
			name:    "empty content",
			input:   []byte{},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := converter.ConvertToUTF8(tc.input)

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

func TestExtractCharsetFromMetaTag(t *testing.T) {
	// These test cases exercise the extractCharsetFromMeta function indirectly
	// through DetectEncoding since extractCharsetFromMeta is not exported

	tests := []struct {
		name    string
		html    []byte
		wantEnc string
	}{
		{
			name:    "charset with double quotes",
			html:    []byte(`<html><head><meta charset="iso-8859-1"></head></html>`),
			wantEnc: "iso-8859-1",
		},
		{
			name:    "charset with single quotes",
			html:    []byte(`<html><head><meta charset='windows-1252'></head></html>`),
			wantEnc: "windows-1252",
		},
		{
			name:    "charset without quotes",
			html:    []byte(`<html><head><meta charset=utf-8></head></html>`),
			wantEnc: "utf-8",
		},
		{
			name:    "http-equiv content-type",
			html:    []byte(`<html><head><meta http-equiv="Content-Type" content="text/html; charset=iso-8859-1"></head></html>`),
			wantEnc: "iso-8859-1",
		},
		{
			name:    "mixed case charset",
			html:    []byte(`<html><head><meta charset="UTF-8"></head></html>`),
			wantEnc: "utf-8",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			enc := converter.DetectEncoding(tc.html)
			assert.Equal(t, tc.wantEnc, enc)
		})
	}
}
