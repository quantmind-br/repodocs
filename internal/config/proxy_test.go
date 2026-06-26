package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProxyConfig_Resolve covers URL resolution from both the explicit URL form
// and the structured Type/Host/Port/Username/Password form.
func TestProxyConfig_Resolve(t *testing.T) {
	tests := []struct {
		name    string
		proxy   ProxyConfig
		want    string
		wantErr bool
	}{
		{
			name:  "disabled returns empty",
			proxy: ProxyConfig{Enabled: false, URL: "socks5://u:p@host:1080"},
			want:  "",
		},
		{
			name:  "explicit socks5 url with credentials",
			proxy: ProxyConfig{Enabled: true, URL: "socks5://user:pass@p.webshare.io:80"},
			want:  "socks5://user:pass@p.webshare.io:80",
		},
		{
			name:  "explicit url with separate credentials",
			proxy: ProxyConfig{Enabled: true, URL: "socks5://p.webshare.io:80", Username: "user", Password: "pass"},
			want:  "socks5://user:pass@p.webshare.io:80",
		},
		{
			name:  "credentials in url take precedence over separate fields",
			proxy: ProxyConfig{Enabled: true, URL: "socks5://urlu:urlp@host:1080", Username: "other", Password: "other"},
			want:  "socks5://urlu:urlp@host:1080",
		},
		{
			name:  "structured form defaults to socks5",
			proxy: ProxyConfig{Enabled: true, Host: "p.webshare.io", Port: 80, Username: "user", Password: "pass"},
			want:  "socks5://user:pass@p.webshare.io:80",
		},
		{
			name:  "structured http with port",
			proxy: ProxyConfig{Enabled: true, Type: "http", Host: "proxy.local", Port: 3128},
			want:  "http://proxy.local:3128",
		},
		{
			name:  "http without port is allowed (tls-client defaults to 80)",
			proxy: ProxyConfig{Enabled: true, Type: "http", Host: "proxy.local"},
			want:  "http://proxy.local",
		},
		{
			name:  "socks alias normalizes to socks5",
			proxy: ProxyConfig{Enabled: true, Type: "SOCKS", Host: "proxy.local", Port: 1080},
			want:  "socks5://proxy.local:1080",
		},
		{
			name:    "socks5 without port is rejected (structured)",
			proxy:   ProxyConfig{Enabled: true, Type: "socks5", Host: "proxy.local"},
			wantErr: true,
		},
		{
			name:    "socks5h without port is rejected (url form)",
			proxy:   ProxyConfig{Enabled: true, URL: "socks5h://proxy.local"},
			wantErr: true,
		},
		{
			name:    "enabled but no url or host",
			proxy:   ProxyConfig{Enabled: true},
			wantErr: true,
		},
		{
			name:    "unsupported scheme in url",
			proxy:   ProxyConfig{Enabled: true, URL: "ftp://host:21"},
			wantErr: true,
		},
		{
			name:    "unsupported structured type",
			proxy:   ProxyConfig{Enabled: true, Type: "ftp", Host: "host", Port: 21},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.proxy.Resolve()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestProxyConfig_Resolve_EncodesSpecialChars ensures credentials with reserved
// characters are percent-encoded so the resulting URL parses correctly.
func TestProxyConfig_Resolve_EncodesSpecialChars(t *testing.T) {
	p := ProxyConfig{
		Enabled:  true,
		Type:     "socks5",
		Host:     "host",
		Port:     1080,
		Username: "user@name",
		Password: "p@ss:word/!",
	}
	got, err := p.Resolve()
	require.NoError(t, err)
	assert.Contains(t, got, "socks5://")
	assert.Contains(t, got, "@host:1080")
	// Reserved characters must be escaped, not present raw in the userinfo.
	assert.Contains(t, got, "user%40name")
}

// TestProxyConfig_Resolve_NoPasswordLeak ensures that error messages produced
// for malformed proxy URLs never echo the password.
func TestProxyConfig_Resolve_NoPasswordLeak(t *testing.T) {
	secret := "sup3rs3cr3t"

	t.Run("missing host", func(t *testing.T) {
		p := ProxyConfig{Enabled: true, URL: "socks5://user:" + secret + "@"}
		_, err := p.Resolve()
		require.Error(t, err)
		assert.NotContains(t, err.Error(), secret)
	})

	t.Run("unparseable url", func(t *testing.T) {
		// A control character makes url.Parse fail.
		p := ProxyConfig{Enabled: true, URL: "socks5://user:" + secret + "@host\x7f:1080"}
		_, err := p.Resolve()
		require.Error(t, err)
		assert.NotContains(t, err.Error(), secret)
	})
}

// TestConfig_Validate_IgnoresProxy documents that proxy resolution is validated
// lazily at point of use, not by Config.Validate (so a broken config proxy never
// blocks unrelated commands or a --proxy override).
func TestConfig_Validate_IgnoresProxy(t *testing.T) {
	cfg := Default()
	cfg.Proxy = ProxyConfig{Enabled: true, Type: "ftp", Host: "host"}
	require.NoError(t, cfg.Validate())
}
