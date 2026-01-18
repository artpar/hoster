package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Template Creation Tests
// =============================================================================

func TestNewTemplate_ValidInput(t *testing.T) {
	template, err := NewTemplate("WordPress Blog", "1.0.0", validComposeSpec)
	require.NoError(t, err)

	assert.NotEmpty(t, template.ID)
	assert.Equal(t, "WordPress Blog", template.Name)
	assert.Equal(t, "wordpress-blog", template.Slug)
	assert.Equal(t, "1.0.0", template.Version)
	assert.False(t, template.Published)
	assert.NotZero(t, template.CreatedAt)
	assert.NotZero(t, template.UpdatedAt)
}

// =============================================================================
// Name Validation Tests
// =============================================================================

func TestValidateName_Empty(t *testing.T) {
	err := ValidateName("")
	assert.ErrorIs(t, err, ErrNameRequired)
}

func TestValidateName_TooShort(t *testing.T) {
	err := ValidateName("WP")
	assert.ErrorIs(t, err, ErrNameTooShort)
}

func TestValidateName_TooLong(t *testing.T) {
	longName := make([]byte, 101)
	for i := range longName {
		longName[i] = 'a'
	}
	err := ValidateName(string(longName))
	assert.ErrorIs(t, err, ErrNameTooLong)
}

func TestValidateName_InvalidChars(t *testing.T) {
	err := ValidateName("WordPress@Blog!")
	assert.ErrorIs(t, err, ErrNameInvalidChars)
}

func TestValidateName_Valid(t *testing.T) {
	testCases := []string{
		"WordPress Blog",
		"my-app-123",
		"Simple App",
		"App",
	}
	for _, name := range testCases {
		t.Run(name, func(t *testing.T) {
			err := ValidateName(name)
			assert.NoError(t, err)
		})
	}
}

// =============================================================================
// Slug Generation Tests
// =============================================================================

func TestGenerateSlug(t *testing.T) {
	testCases := []struct {
		name     string
		expected string
	}{
		{"WordPress Blog", "wordpress-blog"},
		{"My App 123", "my-app-123"},
		{"UPPERCASE", "uppercase"},
		{"multiple   spaces", "multiple-spaces"},
		{"trailing-dash-", "trailing-dash"},
		{"-leading-dash", "leading-dash"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			slug := GenerateSlug(tc.name)
			assert.Equal(t, tc.expected, slug)
		})
	}
}

// =============================================================================
// Version Validation Tests
// =============================================================================

func TestValidateVersion_Empty(t *testing.T) {
	err := ValidateVersion("")
	assert.ErrorIs(t, err, ErrVersionRequired)
}

func TestValidateVersion_InvalidFormat(t *testing.T) {
	invalidVersions := []string{
		"1.0",
		"1",
		"v1.0.0",
		"1.0.0.0",
		"1.0.0-beta",
		"abc",
	}
	for _, version := range invalidVersions {
		t.Run(version, func(t *testing.T) {
			err := ValidateVersion(version)
			assert.ErrorIs(t, err, ErrVersionInvalidFormat)
		})
	}
}

func TestValidateVersion_Valid(t *testing.T) {
	validVersions := []string{
		"0.0.1",
		"1.0.0",
		"1.2.3",
		"10.20.30",
	}
	for _, version := range validVersions {
		t.Run(version, func(t *testing.T) {
			err := ValidateVersion(version)
			assert.NoError(t, err)
		})
	}
}

func TestCompareVersions(t *testing.T) {
	testCases := []struct {
		v1       string
		v2       string
		expected int // -1: v1 < v2, 0: v1 == v2, 1: v1 > v2
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.0.0", "1.0.1", -1},
		{"1.0.1", "1.0.0", 1},
		{"1.0.0", "2.0.0", -1},
		{"1.1.0", "1.0.0", 1},
	}

	for _, tc := range testCases {
		t.Run(tc.v1+" vs "+tc.v2, func(t *testing.T) {
			result := CompareVersions(tc.v1, tc.v2)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// =============================================================================
// Price Validation Tests
// =============================================================================

func TestValidatePrice_Negative(t *testing.T) {
	err := ValidatePrice(-100)
	assert.ErrorIs(t, err, ErrPriceNegative)
}

func TestValidatePrice_Valid(t *testing.T) {
	testCases := []int64{0, 100, 999, 10000}
	for _, price := range testCases {
		err := ValidatePrice(price)
		assert.NoError(t, err)
	}
}

// =============================================================================
// Variable Validation Tests
// =============================================================================

func TestValidateVariables_DuplicateNames(t *testing.T) {
	vars := []Variable{
		{Name: "DB_PASSWORD", Label: "Password", Type: VarTypePassword, Required: true},
		{Name: "DB_PASSWORD", Label: "Password Again", Type: VarTypePassword, Required: true},
	}
	errs := ValidateVariables(vars)
	assert.Len(t, errs, 1)
	assert.ErrorIs(t, errs[0], ErrVariableDuplicate)
}

func TestValidateVariables_InvalidType(t *testing.T) {
	vars := []Variable{
		{Name: "VAR", Label: "Variable", Type: "invalid", Required: true},
	}
	errs := ValidateVariables(vars)
	assert.Len(t, errs, 1)
	assert.ErrorIs(t, errs[0], ErrVariableInvalidType)
}

func TestValidateVariables_SelectWithoutOptions(t *testing.T) {
	vars := []Variable{
		{Name: "CHOICE", Label: "Choice", Type: VarTypeSelect, Required: true, Options: nil},
	}
	errs := ValidateVariables(vars)
	assert.Len(t, errs, 1)
	assert.ErrorIs(t, errs[0], ErrVariableOptionsRequired)
}

func TestValidateVariables_Valid(t *testing.T) {
	vars := []Variable{
		{Name: "DB_PASSWORD", Label: "Database Password", Type: VarTypePassword, Required: true},
		{Name: "SITE_NAME", Label: "Site Name", Type: VarTypeString, Required: false, Default: "My Site"},
		{Name: "PORT", Label: "Port", Type: VarTypeNumber, Required: true},
		{Name: "DEBUG", Label: "Debug Mode", Type: VarTypeBoolean, Required: false, Default: "false"},
		{Name: "ENV", Label: "Environment", Type: VarTypeSelect, Required: true, Options: []string{"dev", "prod"}},
	}
	errs := ValidateVariables(vars)
	assert.Empty(t, errs)
}

// =============================================================================
// Template Validation Tests (Full)
// =============================================================================

func TestValidateTemplate_MultipleErrors(t *testing.T) {
	template := Template{
		Name:             "WP", // Too short
		Version:          "1.0", // Invalid format
		PriceMonthly:     -100, // Negative
		ComposeSpec:      "", // Empty
	}

	errs := ValidateTemplate(template)
	assert.GreaterOrEqual(t, len(errs), 3)
}

func TestValidateTemplate_Valid(t *testing.T) {
	template := Template{
		Name:         "WordPress Blog",
		Version:      "1.0.0",
		ComposeSpec:  validComposeSpec,
		PriceMonthly: 999,
		Variables: []Variable{
			{Name: "DB_PASSWORD", Label: "Database Password", Type: VarTypePassword, Required: true},
		},
	}

	errs := ValidateTemplate(template)
	assert.Empty(t, errs)
}

// =============================================================================
// State Transition Tests
// =============================================================================

func TestTemplate_Publish_FromDraft(t *testing.T) {
	template := Template{Published: false, Version: "1.0.0"}
	err := template.Publish()
	assert.NoError(t, err)
	assert.True(t, template.Published)
}

func TestTemplate_Publish_WithoutVersion(t *testing.T) {
	template := Template{Published: false, Version: ""}
	err := template.Publish()
	assert.ErrorIs(t, err, ErrPublishRequiresVersion)
	assert.False(t, template.Published)
}

func TestTemplate_Unpublish(t *testing.T) {
	template := Template{Published: true}
	template.Unpublish()
	assert.False(t, template.Published)
}

// =============================================================================
// Test Fixtures
// =============================================================================

const validComposeSpec = `
services:
  wordpress:
    image: wordpress:latest
    ports:
      - "80:80"
    environment:
      WORDPRESS_DB_HOST: db
      WORDPRESS_DB_PASSWORD: ${DB_PASSWORD}
  db:
    image: mysql:8
    environment:
      MYSQL_ROOT_PASSWORD: ${DB_PASSWORD}
`
