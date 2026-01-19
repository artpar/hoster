package proxy

import "fmt"

// ProxyErrorType defines the type of proxy error.
type ProxyErrorType int

const (
	ErrorNotFound ProxyErrorType = iota
	ErrorStopped
	ErrorUnavailable
	ErrorUpstreamTimeout
	ErrorUpstreamError
)

// ProxyError represents an error during proxying.
type ProxyError struct {
	Type       ProxyErrorType
	Hostname   string
	Message    string
	StatusCode int
}

// Error implements the error interface.
func (e ProxyError) Error() string {
	return e.Message
}

// NewNotFoundError creates an error for unknown hostname.
func NewNotFoundError(hostname string) ProxyError {
	return ProxyError{
		Type:       ErrorNotFound,
		Hostname:   hostname,
		Message:    fmt.Sprintf("app not found: %s", hostname),
		StatusCode: 404,
	}
}

// NewStoppedError creates an error for stopped deployment.
func NewStoppedError(hostname string) ProxyError {
	return ProxyError{
		Type:       ErrorStopped,
		Hostname:   hostname,
		Message:    fmt.Sprintf("app is stopped: %s", hostname),
		StatusCode: 503,
	}
}

// NewUnavailableError creates an error for unreachable container.
func NewUnavailableError(hostname string) ProxyError {
	return ProxyError{
		Type:       ErrorUnavailable,
		Hostname:   hostname,
		Message:    fmt.Sprintf("app unavailable: %s", hostname),
		StatusCode: 503,
	}
}
