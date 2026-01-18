package validation

// =============================================================================
// Template Validation Functions
// =============================================================================

// ValidateCreateTemplateFields validates required fields for template creation.
// Returns the field name and error message if validation fails.
// Returns empty strings if all fields are valid.
//
// Example:
//
//	field, msg := ValidateCreateTemplateFields("My App", "1.0.0", "services:", "user-123")
//	if field != "" {
//	    // Handle validation error
//	}
func ValidateCreateTemplateFields(name, version, composeSpec, creatorID string) (field, message string) {
	if name == "" {
		return "name", "name is required"
	}
	if version == "" {
		return "version", "version is required"
	}
	if composeSpec == "" {
		return "compose_spec", "compose_spec is required"
	}
	if creatorID == "" {
		return "creator_id", "creator_id is required"
	}
	return "", ""
}

// CanUpdateTemplate checks if a template can be updated based on its published status.
// Published templates cannot be modified.
// Returns whether the update is allowed and an optional reason if not.
//
// Example:
//
//	allowed, reason := CanUpdateTemplate(template.Published)
//	if !allowed {
//	    // Return 409 Conflict with reason
//	}
func CanUpdateTemplate(published bool) (allowed bool, reason string) {
	if published {
		return false, "published templates cannot be modified"
	}
	return true, ""
}

// CanCreateDeployment checks if a deployment can be created from a template.
// Only published templates can be used for deployments.
// Returns whether deployment creation is allowed and an optional reason if not.
//
// Example:
//
//	allowed, reason := CanCreateDeployment(template.Published)
//	if !allowed {
//	    // Return 409 Conflict with reason
//	}
func CanCreateDeployment(templatePublished bool) (allowed bool, reason string) {
	if !templatePublished {
		return false, "template is not published"
	}
	return true, ""
}
