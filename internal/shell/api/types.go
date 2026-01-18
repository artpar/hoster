package api

import "time"

// =============================================================================
// Request Types
// =============================================================================

// CreateTemplateRequest is the request body for creating a template.
type CreateTemplateRequest struct {
	Name             string            `json:"name"`
	Version          string            `json:"version"`
	ComposeSpec      string            `json:"compose_spec"`
	CreatorID        string            `json:"creator_id"`
	Description      string            `json:"description,omitempty"`
	Category         string            `json:"category,omitempty"`
	Tags             []string          `json:"tags,omitempty"`
	PriceMonthly     int               `json:"price_monthly_cents,omitempty"`
	Variables        []VariableRequest `json:"variables,omitempty"`
}

// VariableRequest represents a variable in a template creation request.
type VariableRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Type        string `json:"type,omitempty"`
	Default     string `json:"default,omitempty"`
	Required    bool   `json:"required,omitempty"`
	Secret      bool   `json:"secret,omitempty"`
}

// UpdateTemplateRequest is the request body for updating a template.
type UpdateTemplateRequest struct {
	Name         string   `json:"name,omitempty"`
	Description  string   `json:"description,omitempty"`
	Category     string   `json:"category,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	PriceMonthly int      `json:"price_monthly_cents,omitempty"`
}

// CreateDeploymentRequest is the request body for creating a deployment.
type CreateDeploymentRequest struct {
	TemplateID string            `json:"template_id"`
	CustomerID string            `json:"customer_id"`
	Name       string            `json:"name,omitempty"`
	Variables  map[string]string `json:"variables,omitempty"`
}

// =============================================================================
// Response Types
// =============================================================================

// TemplateResponse is the response for template operations.
type TemplateResponse struct {
	ID                   string             `json:"id"`
	Name                 string             `json:"name"`
	Slug                 string             `json:"slug"`
	Description          string             `json:"description"`
	Version              string             `json:"version"`
	ComposeSpec          string             `json:"compose_spec"`
	Variables            []VariableResponse `json:"variables"`
	ResourceRequirements ResourcesResponse  `json:"resource_requirements"`
	PriceMonthly         int                `json:"price_monthly_cents"`
	Category             string             `json:"category"`
	Tags                 []string           `json:"tags"`
	Published            bool               `json:"published"`
	CreatorID            string             `json:"creator_id"`
	CreatedAt            time.Time          `json:"created_at"`
	UpdatedAt            time.Time          `json:"updated_at"`
}

// VariableResponse represents a variable in a template response.
type VariableResponse struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Type        string `json:"type,omitempty"`
	Default     string `json:"default,omitempty"`
	Required    bool   `json:"required,omitempty"`
	Secret      bool   `json:"secret,omitempty"`
}

// ResourcesResponse represents resource requirements.
type ResourcesResponse struct {
	CPUCores int `json:"cpu_cores"`
	MemoryMB int `json:"memory_mb"`
	DiskMB   int `json:"disk_mb"`
}

// DeploymentResponse is the response for deployment operations.
type DeploymentResponse struct {
	ID              string              `json:"id"`
	Name            string              `json:"name"`
	TemplateID      string              `json:"template_id"`
	TemplateVersion string              `json:"template_version"`
	CustomerID      string              `json:"customer_id"`
	Status          string              `json:"status"`
	Variables       map[string]string   `json:"variables"`
	Domains         []DomainResponse    `json:"domains"`
	Containers      []ContainerResponse `json:"containers"`
	Resources       ResourcesResponse   `json:"resources"`
	ErrorMessage    string              `json:"error_message,omitempty"`
	CreatedAt       time.Time           `json:"created_at"`
	UpdatedAt       time.Time           `json:"updated_at"`
	StartedAt       *time.Time          `json:"started_at,omitempty"`
	StoppedAt       *time.Time          `json:"stopped_at,omitempty"`
}

// DomainResponse represents a domain in a deployment response.
type DomainResponse struct {
	Domain   string `json:"domain"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

// ContainerResponse represents a container in a deployment response.
type ContainerResponse struct {
	ServiceName string `json:"service_name"`
	ContainerID string `json:"container_id"`
	Status      string `json:"status"`
}

// ListTemplatesResponse is the response for listing templates.
type ListTemplatesResponse struct {
	Templates []TemplateResponse `json:"templates"`
	Total     int                `json:"total"`
	Limit     int                `json:"limit"`
	Offset    int                `json:"offset"`
}

// ListDeploymentsResponse is the response for listing deployments.
type ListDeploymentsResponse struct {
	Deployments []DeploymentResponse `json:"deployments"`
	Total       int                  `json:"total"`
	Limit       int                  `json:"limit"`
	Offset      int                  `json:"offset"`
}

// ErrorResponse is the error response format.
type ErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

// HealthResponse is the health check response.
type HealthResponse struct {
	Status string `json:"status"`
}

// ReadyResponse is the readiness check response.
type ReadyResponse struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks"`
}
