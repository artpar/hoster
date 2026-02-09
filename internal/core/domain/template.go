// Package domain contains the core domain types and validation logic.
// This is part of the Functional Core - all functions are pure with no I/O.
package domain

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// Errors
// =============================================================================

var (
	// Name validation errors
	ErrNameRequired     = errors.New("name is required")
	ErrNameTooShort     = errors.New("name must be at least 3 characters")
	ErrNameTooLong      = errors.New("name must be at most 100 characters")
	ErrNameInvalidChars = errors.New("name can only contain alphanumeric characters, spaces, and hyphens")

	// Version validation errors
	ErrVersionRequired      = errors.New("version is required")
	ErrVersionInvalidFormat = errors.New("version must be in semver format (X.Y.Z)")

	// Price validation errors
	ErrPriceNegative = errors.New("price cannot be negative")

	// Variable validation errors
	ErrVariableDuplicate       = errors.New("duplicate variable name")
	ErrVariableInvalidType     = errors.New("invalid variable type")
	ErrVariableOptionsRequired = errors.New("options required for select type")

	// Compose validation errors
	ErrComposeRequired    = errors.New("compose spec is required")
	ErrComposeInvalidYAML = errors.New("compose spec is not valid YAML")
	ErrComposeNoServices  = errors.New("compose spec must have at least one service")

	// State transition errors
	ErrPublishRequiresVersion = errors.New("cannot publish template without a version")
)

// =============================================================================
// Variable Types
// =============================================================================

type VariableType string

const (
	VarTypeString   VariableType = "string"
	VarTypeNumber   VariableType = "number"
	VarTypeBoolean  VariableType = "boolean"
	VarTypePassword VariableType = "password"
	VarTypeSelect   VariableType = "select"
)

// IsValid checks if the variable type is valid.
func (vt VariableType) IsValid() bool {
	switch vt {
	case VarTypeString, VarTypeNumber, VarTypeBoolean, VarTypePassword, VarTypeSelect:
		return true
	default:
		return false
	}
}

// =============================================================================
// Variable
// =============================================================================

// Variable represents a configurable variable in a template.
type Variable struct {
	Name        string       `json:"name"`
	Label       string       `json:"label"`
	Description string       `json:"description,omitempty"`
	Type        VariableType `json:"type"`
	Default     string       `json:"default,omitempty"`
	Required    bool         `json:"required"`
	Options     []string     `json:"options,omitempty"`
	Validation  string       `json:"validation,omitempty"`
}

// =============================================================================
// ConfigFile
// =============================================================================

// ConfigFile represents a configuration file to be mounted in containers.
// These are stored with the template and written to disk at deployment time.
type ConfigFile struct {
	// Name is a human-readable identifier (e.g., "nginx.conf")
	Name string `json:"name"`

	// Path is the absolute path where the file will be mounted in the container
	// (e.g., "/etc/nginx/nginx.conf")
	Path string `json:"path"`

	// Content is the actual file content
	Content string `json:"content"`

	// Mode is the file permission mode (e.g., "0644"). Defaults to "0644" if empty.
	Mode string `json:"mode,omitempty"`
}

// =============================================================================
// Resources
// =============================================================================

// Resources represents resource requirements.
type Resources struct {
	CPUCores float64 `json:"cpu_cores"`
	MemoryMB int64   `json:"memory_mb"`
	DiskMB   int64   `json:"disk_mb"`
}

// =============================================================================
// Template
// =============================================================================

// Template represents a deployable package definition.
type Template struct {
	ID                   int          `json:"-"`
	ReferenceID          string       `json:"id"`
	Name                 string       `json:"name"`
	Slug                 string       `json:"slug"`
	Description          string       `json:"description,omitempty"`
	Version              string       `json:"version"`
	ComposeSpec          string       `json:"compose_spec"`
	Variables            []Variable   `json:"variables,omitempty"`
	ConfigFiles          []ConfigFile `json:"config_files,omitempty"`
	ResourceRequirements Resources    `json:"resource_requirements"`
	RequiredCapabilities []string     `json:"required_capabilities,omitempty"` // Node capabilities required (e.g., ["gpu"])
	PriceMonthly         int64        `json:"price_monthly_cents"`
	Category             string       `json:"category,omitempty"`
	Tags                 []string     `json:"tags,omitempty"`
	Published            bool         `json:"published"`
	CreatorID            int          `json:"-"`
	CreatedAt            time.Time    `json:"created_at"`
	UpdatedAt            time.Time    `json:"updated_at"`
}

// NewTemplate creates a new template with the given name, version, and compose spec.
// Returns an error if validation fails.
func NewTemplate(name, version, composeSpec string) (*Template, error) {
	if err := ValidateName(name); err != nil {
		return nil, err
	}
	if err := ValidateVersion(version); err != nil {
		return nil, err
	}
	if err := ValidateComposeSpec(composeSpec); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	return &Template{
		ReferenceID: "tmpl_" + uuid.New().String()[:8],
		Name:        name,
		Slug:        GenerateSlug(name),
		Version:     version,
		ComposeSpec: composeSpec,
		Published:   false,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// Publish marks the template as published.
func (t *Template) Publish() error {
	if t.Version == "" {
		return ErrPublishRequiresVersion
	}
	t.Published = true
	t.UpdatedAt = time.Now().UTC()
	return nil
}

// Unpublish marks the template as unpublished.
func (t *Template) Unpublish() {
	t.Published = false
	t.UpdatedAt = time.Now().UTC()
}

// =============================================================================
// Validation Functions (Pure)
// =============================================================================

var (
	nameRegex    = regexp.MustCompile(`^[a-zA-Z0-9\s\-]+$`)
	versionRegex = regexp.MustCompile(`^\d+\.\d+\.\d+$`)
)

// ValidateName validates a template name.
func ValidateName(name string) error {
	if name == "" {
		return ErrNameRequired
	}
	if len(name) < 3 {
		return ErrNameTooShort
	}
	if len(name) > 100 {
		return ErrNameTooLong
	}
	if !nameRegex.MatchString(name) {
		return ErrNameInvalidChars
	}
	return nil
}

// GenerateSlug generates a URL-safe slug from a name.
func GenerateSlug(name string) string {
	// Lowercase
	slug := strings.ToLower(name)
	// Replace spaces with hyphens
	slug = strings.ReplaceAll(slug, " ", "-")
	// Remove consecutive hyphens
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	// Trim leading/trailing hyphens
	slug = strings.Trim(slug, "-")
	return slug
}

// ValidateVersion validates a version string (must be semver X.Y.Z).
func ValidateVersion(version string) error {
	if version == "" {
		return ErrVersionRequired
	}
	if !versionRegex.MatchString(version) {
		return ErrVersionInvalidFormat
	}
	return nil
}

// CompareVersions compares two version strings.
// Returns -1 if v1 < v2, 0 if v1 == v2, 1 if v1 > v2.
func CompareVersions(v1, v2 string) int {
	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	for i := 0; i < 3; i++ {
		n1, _ := strconv.Atoi(parts1[i])
		n2, _ := strconv.Atoi(parts2[i])
		if n1 < n2 {
			return -1
		}
		if n1 > n2 {
			return 1
		}
	}
	return 0
}

// ValidatePrice validates a price (must be non-negative).
func ValidatePrice(price int64) error {
	if price < 0 {
		return ErrPriceNegative
	}
	return nil
}

// ValidateComposeSpec validates a compose spec string.
// For now, just checks it's not empty. Full validation will use compose-go.
func ValidateComposeSpec(spec string) error {
	if strings.TrimSpace(spec) == "" {
		return ErrComposeRequired
	}
	// TODO: Use compose-go for full validation
	return nil
}

// ValidateVariables validates a slice of variables.
// Returns all validation errors found.
func ValidateVariables(vars []Variable) []error {
	var errs []error
	seen := make(map[string]bool)

	for _, v := range vars {
		// Check for duplicates
		if seen[v.Name] {
			errs = append(errs, ErrVariableDuplicate)
			continue
		}
		seen[v.Name] = true

		// Check type
		if !v.Type.IsValid() {
			errs = append(errs, ErrVariableInvalidType)
			continue
		}

		// Check options for select type
		if v.Type == VarTypeSelect && len(v.Options) == 0 {
			errs = append(errs, ErrVariableOptionsRequired)
		}
	}

	return errs
}

// ValidateTemplate validates a template and returns all validation errors.
func ValidateTemplate(t Template) []error {
	var errs []error

	if err := ValidateName(t.Name); err != nil {
		errs = append(errs, err)
	}
	if err := ValidateVersion(t.Version); err != nil {
		errs = append(errs, err)
	}
	if err := ValidatePrice(t.PriceMonthly); err != nil {
		errs = append(errs, err)
	}
	if err := ValidateComposeSpec(t.ComposeSpec); err != nil {
		errs = append(errs, err)
	}

	varErrs := ValidateVariables(t.Variables)
	errs = append(errs, varErrs...)

	return errs
}
