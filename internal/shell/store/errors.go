// Package store provides persistence for Hoster entities.
package store

import (
	"errors"
	"fmt"
)

// =============================================================================
// Error Types
// =============================================================================

var (
	// ErrNotFound is returned when an entity is not found.
	ErrNotFound = errors.New("entity not found")

	// ErrDuplicateID is returned when creating an entity with an existing ID.
	ErrDuplicateID = errors.New("entity with this ID already exists")

	// ErrDuplicateSlug is returned when creating a template with an existing slug.
	ErrDuplicateSlug = errors.New("template with this slug already exists")

	// ErrForeignKey is returned when a foreign key constraint is violated.
	ErrForeignKey = errors.New("foreign key constraint violated")

	// ErrConnectionFailed is returned when database connection fails.
	ErrConnectionFailed = errors.New("database connection failed")

	// ErrMigrationFailed is returned when database migration fails.
	ErrMigrationFailed = errors.New("database migration failed")

	// ErrInvalidData is returned when JSON serialization/deserialization fails.
	ErrInvalidData = errors.New("invalid data format")

	// ErrTxFailed is returned when a transaction operation fails.
	ErrTxFailed = errors.New("transaction failed")
)

// StoreError wraps errors with additional context.
type StoreError struct {
	Op      string // Operation that failed (e.g., "CreateTemplate")
	Entity  string // Entity type (e.g., "template", "deployment")
	ID      string // Entity ID if applicable
	Message string
	Err     error
}

func (e *StoreError) Error() string {
	if e.ID != "" {
		return fmt.Sprintf("%s %s %s: %s", e.Op, e.Entity, e.ID, e.Message)
	}
	if e.Entity != "" {
		return fmt.Sprintf("%s %s: %s", e.Op, e.Entity, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.Op, e.Message)
}

func (e *StoreError) Unwrap() error {
	return e.Err
}

// NewStoreError creates a new StoreError.
func NewStoreError(op, entity, id, message string, err error) *StoreError {
	return &StoreError{
		Op:      op,
		Entity:  entity,
		ID:      id,
		Message: message,
		Err:     err,
	}
}
