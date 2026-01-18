// Package compose contains pure functions for parsing Docker Compose specifications.
// This is part of the Functional Core - all functions are pure with no I/O.
package compose

import (
	"errors"
	"fmt"
)

// =============================================================================
// Error Types
// =============================================================================

var (
	// Input validation errors
	ErrEmptyInput = errors.New("compose spec is empty")

	// YAML parsing errors
	ErrInvalidYAML = errors.New("invalid YAML syntax")

	// Compose structure errors
	ErrNoServices = errors.New("compose spec must define at least one service")

	// Service validation errors
	ErrServiceNoImage       = errors.New("service must have image or build")
	ErrServiceInvalidPort   = errors.New("invalid port configuration")
	ErrServiceInvalidVolume = errors.New("invalid volume configuration")
	ErrCircularDependency   = errors.New("circular dependency detected")

	// Resource validation errors
	ErrInvalidCPU    = errors.New("invalid CPU value")
	ErrInvalidMemory = errors.New("invalid memory value")

	// Unsupported feature errors
	ErrUnsupportedFeature = errors.New("unsupported compose feature")
)

// ParseError wraps errors with context about where parsing failed.
type ParseError struct {
	Field   string // e.g., "services.web.ports[0]"
	Message string
	Err     error
}

func (e *ParseError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%s: %s", e.Field, e.Message)
	}
	return e.Message
}

func (e *ParseError) Unwrap() error {
	return e.Err
}

// NewParseError creates a new ParseError.
func NewParseError(field, message string, err error) *ParseError {
	return &ParseError{
		Field:   field,
		Message: message,
		Err:     err,
	}
}
