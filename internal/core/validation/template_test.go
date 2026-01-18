package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// =============================================================================
// ValidateCreateTemplateFields Tests
// =============================================================================

func TestValidateCreateTemplateFields_AllValid(t *testing.T) {
	field, msg := ValidateCreateTemplateFields("My App", "1.0.0", "services:", "user-123")
	assert.Empty(t, field)
	assert.Empty(t, msg)
}

func TestValidateCreateTemplateFields_MissingName(t *testing.T) {
	field, msg := ValidateCreateTemplateFields("", "1.0.0", "services:", "user-123")
	assert.Equal(t, "name", field)
	assert.Equal(t, "name is required", msg)
}

func TestValidateCreateTemplateFields_MissingVersion(t *testing.T) {
	field, msg := ValidateCreateTemplateFields("My App", "", "services:", "user-123")
	assert.Equal(t, "version", field)
	assert.Equal(t, "version is required", msg)
}

func TestValidateCreateTemplateFields_MissingComposeSpec(t *testing.T) {
	field, msg := ValidateCreateTemplateFields("My App", "1.0.0", "", "user-123")
	assert.Equal(t, "compose_spec", field)
	assert.Equal(t, "compose_spec is required", msg)
}

func TestValidateCreateTemplateFields_MissingCreatorID(t *testing.T) {
	field, msg := ValidateCreateTemplateFields("My App", "1.0.0", "services:", "")
	assert.Equal(t, "creator_id", field)
	assert.Equal(t, "creator_id is required", msg)
}

func TestValidateCreateTemplateFields_ChecksInOrder(t *testing.T) {
	// When multiple fields are missing, first one is reported
	field, _ := ValidateCreateTemplateFields("", "", "", "")
	assert.Equal(t, "name", field, "should check name first")
}

// =============================================================================
// CanUpdateTemplate Tests
// =============================================================================

func TestCanUpdateTemplate_Unpublished(t *testing.T) {
	allowed, reason := CanUpdateTemplate(false)
	assert.True(t, allowed)
	assert.Empty(t, reason)
}

func TestCanUpdateTemplate_Published(t *testing.T) {
	allowed, reason := CanUpdateTemplate(true)
	assert.False(t, allowed)
	assert.Equal(t, "published templates cannot be modified", reason)
}

// =============================================================================
// CanCreateDeployment Tests
// =============================================================================

func TestCanCreateDeployment_Published(t *testing.T) {
	allowed, reason := CanCreateDeployment(true)
	assert.True(t, allowed)
	assert.Empty(t, reason)
}

func TestCanCreateDeployment_Unpublished(t *testing.T) {
	allowed, reason := CanCreateDeployment(false)
	assert.False(t, allowed)
	assert.Equal(t, "template is not published", reason)
}

// =============================================================================
// Table-Driven Tests
// =============================================================================

func TestValidateCreateTemplateFields_TableDriven(t *testing.T) {
	tests := []struct {
		name        string
		inputName   string
		version     string
		composeSpec string
		creatorID   string
		wantField   string
		wantMsg     string
	}{
		{
			name:        "all valid",
			inputName:   "Test App",
			version:     "2.0.0",
			composeSpec: "services:\n  web:",
			creatorID:   "creator-1",
			wantField:   "",
			wantMsg:     "",
		},
		{
			name:        "whitespace name is valid",
			inputName:   "  ",
			version:     "1.0.0",
			composeSpec: "services:",
			creatorID:   "user-1",
			wantField:   "",
			wantMsg:     "",
		},
		{
			name:        "empty name fails",
			inputName:   "",
			version:     "1.0.0",
			composeSpec: "services:",
			creatorID:   "user-1",
			wantField:   "name",
			wantMsg:     "name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field, msg := ValidateCreateTemplateFields(tt.inputName, tt.version, tt.composeSpec, tt.creatorID)
			assert.Equal(t, tt.wantField, field)
			assert.Equal(t, tt.wantMsg, msg)
		})
	}
}
