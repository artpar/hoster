// Package validation provides pure validation functions for API handlers.
//
// This package contains the functional core logic for validating API requests
// and checking business rules. All functions are pure (no I/O, no side effects)
// and comply with ADR-002 "Values as Boundaries".
//
// # Functions
//
//   - ValidateCreateTemplateFields: Validate required fields for template creation
//   - CanUpdateTemplate: Check if a template can be updated
//   - CanCreateDeployment: Check if a deployment can be created from a template
//
// # Usage
//
// The API handlers use these functions to validate requests before processing:
//
//	if field, msg := validation.ValidateCreateTemplateFields(name, version, spec, creator); field != "" {
//	    // Return 400 Bad Request with msg
//	}
package validation
