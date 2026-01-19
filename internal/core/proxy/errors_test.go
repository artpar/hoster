package proxy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProxyError_Error(t *testing.T) {
	tests := []struct {
		name    string
		err     ProxyError
		wantMsg string
	}{
		{
			name:    "not found error",
			err:     NewNotFoundError("my-app.apps.hoster.io"),
			wantMsg: "app not found: my-app.apps.hoster.io",
		},
		{
			name:    "stopped error",
			err:     NewStoppedError("my-app.apps.hoster.io"),
			wantMsg: "app is stopped: my-app.apps.hoster.io",
		},
		{
			name:    "unavailable error",
			err:     NewUnavailableError("my-app.apps.hoster.io"),
			wantMsg: "app unavailable: my-app.apps.hoster.io",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantMsg, tt.err.Error())
		})
	}
}

func TestProxyError_StatusCode(t *testing.T) {
	tests := []struct {
		name       string
		err        ProxyError
		wantCode   int
		wantType   ProxyErrorType
	}{
		{
			name:       "not found returns 404",
			err:        NewNotFoundError("host"),
			wantCode:   404,
			wantType:   ErrorNotFound,
		},
		{
			name:       "stopped returns 503",
			err:        NewStoppedError("host"),
			wantCode:   503,
			wantType:   ErrorStopped,
		},
		{
			name:       "unavailable returns 503",
			err:        NewUnavailableError("host"),
			wantCode:   503,
			wantType:   ErrorUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantCode, tt.err.StatusCode)
			assert.Equal(t, tt.wantType, tt.err.Type)
			assert.NotEmpty(t, tt.err.Hostname)
		})
	}
}

func TestProxyError_Implements_error(t *testing.T) {
	// Verify ProxyError implements the error interface
	var err error = NewNotFoundError("test")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "test")
}
