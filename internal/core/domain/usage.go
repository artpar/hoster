// Package domain defines core domain types for Hoster.
package domain

import "time"

// =============================================================================
// Usage Event Types - Following F009: Billing Integration
// =============================================================================

// EventType represents the type of usage event.
type EventType string

const (
	// EventDeploymentCreated is recorded when a deployment is created.
	// Uses dot notation to match APIGate's JSON:API format.
	EventDeploymentCreated EventType = "deployment.created"

	// EventDeploymentStarted is recorded when a deployment starts running.
	// Uses dot notation to match APIGate's JSON:API format.
	EventDeploymentStarted EventType = "deployment.started"

	// EventDeploymentStopped is recorded when a deployment stops.
	// Uses dot notation to match APIGate's JSON:API format.
	EventDeploymentStopped EventType = "deployment.stopped"

	// EventDeploymentDeleted is recorded when a deployment is deleted.
	// Uses dot notation to match APIGate's JSON:API format.
	EventDeploymentDeleted EventType = "deployment.deleted"
)

// MeterEvent represents a usage event to be reported to APIGate for billing.
// Events are stored locally and batch-reported to APIGate.
type MeterEvent struct {
	// ID is the unique identifier for this event.
	ID string `json:"id"`

	// UserID is the user who triggered the event.
	UserID string `json:"user_id"`

	// EventType is the type of usage event.
	EventType EventType `json:"event_type"`

	// ResourceID is the ID of the resource (e.g., deployment ID).
	ResourceID string `json:"resource_id"`

	// ResourceType is the type of resource (e.g., "deployment").
	ResourceType string `json:"resource_type"`

	// Quantity is the amount for metered usage (default 1).
	Quantity int64 `json:"quantity"`

	// Metadata contains additional event data.
	Metadata map[string]string `json:"metadata,omitempty"`

	// Timestamp is when the event occurred.
	Timestamp time.Time `json:"timestamp"`

	// ReportedAt is when the event was reported to APIGate (nil if unreported).
	ReportedAt *time.Time `json:"reported_at,omitempty"`

	// CreatedAt is when the event record was created.
	CreatedAt time.Time `json:"created_at"`
}

// NewMeterEvent creates a new meter event with sensible defaults.
func NewMeterEvent(id, userID string, eventType EventType, resourceID, resourceType string) MeterEvent {
	now := time.Now()
	return MeterEvent{
		ID:           id,
		UserID:       userID,
		EventType:    eventType,
		ResourceID:   resourceID,
		ResourceType: resourceType,
		Quantity:     1,
		Metadata:     make(map[string]string),
		Timestamp:    now,
		CreatedAt:    now,
	}
}

// WithMetadata adds metadata to the event and returns it for chaining.
func (e MeterEvent) WithMetadata(key, value string) MeterEvent {
	if e.Metadata == nil {
		e.Metadata = make(map[string]string)
	}
	e.Metadata[key] = value
	return e
}

// WithQuantity sets the quantity and returns the event for chaining.
func (e MeterEvent) WithQuantity(qty int64) MeterEvent {
	e.Quantity = qty
	return e
}

// IsReported returns true if the event has been reported to APIGate.
func (e MeterEvent) IsReported() bool {
	return e.ReportedAt != nil
}
