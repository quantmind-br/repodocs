package renderer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseProxyURL(t *testing.T) {
	tests := []struct {
		name         string
		raw          string
		wantEnabled  bool
		wantServer   string
		wantScheme   string
		wantUsername string
		wantPassword string
		wantErr      bool
	}{
		{
			name:        "empty disables proxy",
			raw:         "",
			wantEnabled: false,
		},
		{
			name:         "socks5 with credentials",
			raw:          "socks5://user:pass@p.webshare.io:80",
			wantEnabled:  true,
			wantServer:   "socks5://p.webshare.io:80",
			wantScheme:   "socks5",
			wantUsername: "user",
			wantPassword: "pass",
		},
		{
			name:        "http without credentials",
			raw:         "http://proxy.local:3128",
			wantEnabled: true,
			wantServer:  "http://proxy.local:3128",
			wantScheme:  "http",
		},
		{
			// Chrome's --proxy-server rejects "socks5h"; the server string must
			// be mapped to "socks5" while the original scheme is preserved.
			name:         "socks5h maps to socks5 for chrome but keeps scheme",
			raw:          "socks5h://user:pass@host:1080",
			wantEnabled:  true,
			wantServer:   "socks5://host:1080",
			wantScheme:   "socks5h",
			wantUsername: "user",
			wantPassword: "pass",
		},
		{
			name:    "missing host errors",
			raw:     "socks5://",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := parseProxyURL(tt.raw)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantEnabled, info.enabled)
			assert.Equal(t, tt.wantServer, info.server)
			assert.Equal(t, tt.wantScheme, info.scheme)
			assert.Equal(t, tt.wantUsername, info.username)
			assert.Equal(t, tt.wantPassword, info.password)
		})
	}
}
