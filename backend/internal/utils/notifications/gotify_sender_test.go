package notifications

import (
	"testing"

	"github.com/getarcaneapp/arcane/types/notification"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildGotifyURL(t *testing.T) {
	tests := []struct {
		name     string
		config   notification.GotifyConfig
		wantErr  bool
		expected string
	}{
		{
			name: "basic config (host + token)",
			config: notification.GotifyConfig{
				Host:  "gotify.example.com",
				Token: "A12345678901234",
			},
			wantErr:  false,
			expected: "gotify://gotify.example.com/A12345678901234?priority=0",
		},
		{
			name: "config with port",
			config: notification.GotifyConfig{
				Host:  "gotify.example.com",
				Port:  8443,
				Token: "A12345678901234",
			},
			wantErr:  false,
			expected: "gotify://gotify.example.com:8443/A12345678901234?priority=0",
		},
		{
			name: "config with path",
			config: notification.GotifyConfig{
				Host:  "gotify.example.com",
				Path:  "/gotify",
				Token: "A12345678901234",
			},
			wantErr:  false,
			expected: "gotify://gotify.example.com/gotify/A12345678901234?priority=0",
		},
		{
			name: "config with all options",
			config: notification.GotifyConfig{
				Host:       "gotify.example.com",
				Port:       80,
				Path:       "mysubpath",
				Token:      "A12345678901234",
				Priority:   5,
				Title:      "My Title",
				DisableTLS: true,
			},
			wantErr:  false,
			expected: "gotify://gotify.example.com:80/mysubpath/A12345678901234?disabletls=yes&priority=5&title=My+Title",
		},
		{
			name: "missing host",
			config: notification.GotifyConfig{
				Token: "A12345678901234",
			},
			wantErr: true,
		},
		{
			name: "missing token",
			config: notification.GotifyConfig{
				Host: "gotify.example.com",
			},
			wantErr: true,
		},
		{
			name: "custom token format",
			config: notification.GotifyConfig{
				Host:  "gotify.example.com",
				Token: "custom_token_123",
			},
			wantErr:  false,
			expected: "gotify://gotify.example.com/custom_token_123?priority=0",
		},
		{
			name: "negative priority valid",
			config: notification.GotifyConfig{
				Host:     "gotify.example.com",
				Token:    "A12345678901234",
				Priority: -1,
			},
			wantErr:  false,
			expected: "gotify://gotify.example.com/A12345678901234?priority=-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotURL, err := BuildGotifyURL(tt.config)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, gotURL)
			}
		})
	}
}
