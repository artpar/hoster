package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Keycloak + PostgreSQL Tests
// =============================================================================

// TestE2E_Keycloak_TemplateCreation tests creating a Keycloak template.
func TestE2E_Keycloak_TemplateCreation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Keycloak tests in short mode")
	}

	composeYAML := LoadFixture(t, "keycloak-postgres.yaml")
	require.NotEmpty(t, composeYAML)

	template := CreateTemplate(t, "keycloak-auth", "1.0.0", composeYAML)
	require.NotEmpty(t, template.ID)
	assert.Equal(t, "keycloak-auth", template.Name)
	assert.Contains(t, template.ComposeSpec, "keycloak")
	assert.Contains(t, template.ComposeSpec, "postgres")

	t.Log("PASS: Keycloak template created successfully")
}

// TestE2E_Keycloak_DeploymentWithSecrets tests deployment with admin password.
func TestE2E_Keycloak_DeploymentWithSecrets(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Keycloak tests in short mode")
	}

	// Create and publish template
	composeYAML := LoadFixture(t, "keycloak-postgres.yaml")
	template := CreateTemplate(t, "keycloak-secrets", "1.0.0", composeYAML)
	PublishTemplate(t, template.ID)

	// Deploy with secrets
	variables := map[string]string{
		"DB_USER":        "keycloakuser",
		"DB_PASSWORD":    "keycloakdbpass",
		"ADMIN_PASSWORD": "superadminpass123",
	}
	deployment := CreateDeployment(t, template.ID, "customer-kc-1", variables)
	require.NotEmpty(t, deployment.ID)

	// Verify secrets stored
	assert.Equal(t, "keycloakuser", deployment.Variables["DB_USER"])
	assert.Equal(t, "keycloakdbpass", deployment.Variables["DB_PASSWORD"])
	assert.Equal(t, "superadminpass123", deployment.Variables["ADMIN_PASSWORD"])

	// Start
	deployment = StartDeployment(t, deployment.ID)
	assert.Equal(t, "running", deployment.Status)

	// Stop and cleanup
	StopDeployment(t, deployment.ID)
	DeleteDeployment(t, deployment.ID)

	t.Log("PASS: Keycloak deployment with secrets completed")
}

// TestE2E_Keycloak_VersionedTemplates tests creating multiple versions of a template.
func TestE2E_Keycloak_VersionedTemplates(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Keycloak tests in short mode")
	}

	composeYAML := LoadFixture(t, "keycloak-postgres.yaml")

	// Create multiple versions
	v1 := CreateTemplate(t, "keycloak-versioned-v1", "1.0.0", composeYAML)
	v2 := CreateTemplate(t, "keycloak-versioned-v2", "2.0.0", composeYAML)
	v3 := CreateTemplate(t, "keycloak-versioned-v3", "3.0.0", composeYAML)

	// Verify all have unique IDs and correct versions
	assert.NotEqual(t, v1.ID, v2.ID)
	assert.NotEqual(t, v2.ID, v3.ID)
	assert.Equal(t, "1.0.0", v1.Version)
	assert.Equal(t, "2.0.0", v2.Version)
	assert.Equal(t, "3.0.0", v3.Version)

	// Publish v2 only
	PublishTemplate(t, v2.ID)

	// Verify only v2 is published
	assert.False(t, GetTemplate(t, v1.ID).Published)
	assert.True(t, GetTemplate(t, v2.ID).Published)
	assert.False(t, GetTemplate(t, v3.ID).Published)

	t.Log("PASS: Keycloak versioned templates completed")
}

// TestE2E_Keycloak_LongRunningDeployment simulates a long-running auth service.
func TestE2E_Keycloak_LongRunningDeployment(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Keycloak tests in short mode")
	}

	// Setup
	composeYAML := LoadFixture(t, "keycloak-postgres.yaml")
	template := CreateTemplate(t, "keycloak-longrun", "1.0.0", composeYAML)
	PublishTemplate(t, template.ID)

	variables := map[string]string{
		"DB_PASSWORD":    "longrundbpass",
		"ADMIN_PASSWORD": "longrunAdmin",
	}
	deployment := CreateDeployment(t, template.ID, "customer-longrun-1", variables)

	// Start and verify running
	deployment = StartDeployment(t, deployment.ID)
	assert.Equal(t, "running", deployment.Status)

	// Fetch multiple times to simulate checking status
	for i := 0; i < 3; i++ {
		fetched := GetDeployment(t, deployment.ID)
		assert.Equal(t, "running", fetched.Status)
	}

	// Stop and cleanup
	StopDeployment(t, deployment.ID)
	DeleteDeployment(t, deployment.ID)

	t.Log("PASS: Keycloak long-running deployment completed")
}
