package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// WordPress + MySQL Tests
// =============================================================================

// TestE2E_WordPress_TemplateCreation tests creating a WordPress template with complex compose.
func TestE2E_WordPress_TemplateCreation(t *testing.T) {
	composeYAML := LoadFixture(t, "wordpress-mysql.yaml")
	require.NotEmpty(t, composeYAML)

	template := CreateTemplate(t, "wordpress-complex", "2.0.0", composeYAML)
	require.NotEmpty(t, template.ID)
	assert.Equal(t, "wordpress-complex", template.Name)
	assert.Equal(t, "2.0.0", template.Version)
	assert.Contains(t, template.ComposeSpec, "wordpress")
	assert.Contains(t, template.ComposeSpec, "mysql")

	t.Log("PASS: WordPress template created successfully")
}

// TestE2E_WordPress_DeploymentWithAllVariables tests deployment with all required variables.
func TestE2E_WordPress_DeploymentWithAllVariables(t *testing.T) {
	// Create and publish template
	composeYAML := LoadFixture(t, "wordpress-mysql.yaml")
	template := CreateTemplate(t, "wordpress-full-vars", "1.0.0", composeYAML)
	PublishTemplate(t, template.ID)

	// Deploy with all variables
	variables := map[string]string{
		"DB_USER":          "wpuser",
		"DB_PASSWORD":      "wppassword123",
		"DB_ROOT_PASSWORD": "rootpass456",
	}
	deployment := CreateDeployment(t, template.ID, variables)
	require.NotEmpty(t, deployment.ID)
	assert.Equal(t, "pending", deployment.Status)

	// Verify all variables stored
	assert.Equal(t, "wpuser", deployment.Variables["DB_USER"])
	assert.Equal(t, "wppassword123", deployment.Variables["DB_PASSWORD"])
	assert.Equal(t, "rootpass456", deployment.Variables["DB_ROOT_PASSWORD"])

	// Start deployment
	deployment = StartDeployment(t, deployment.ID)
	assert.Equal(t, "running", deployment.Status)

	// Fetch and verify state persisted
	fetched := GetDeployment(t, deployment.ID)
	assert.Equal(t, "running", fetched.Status)
	assert.Equal(t, template.ID, fetched.TemplateID)

	// Stop and cleanup
	StopDeployment(t, deployment.ID)
	DeleteDeployment(t, deployment.ID)

	t.Log("PASS: WordPress deployment with variables completed")
}

// TestE2E_WordPress_MultipleDeployments tests creating multiple deployments from same template.
func TestE2E_WordPress_MultipleDeployments(t *testing.T) {
	// Create and publish template
	composeYAML := LoadFixture(t, "wordpress-mysql.yaml")
	template := CreateTemplate(t, "wordpress-multi", "1.0.0", composeYAML)
	PublishTemplate(t, template.ID)

	// Create multiple deployments
	var deployments []*DeploymentResponse
	for i := 0; i < 3; i++ {
		variables := map[string]string{
			"DB_PASSWORD":      "password" + string(rune('A'+i)),
			"DB_ROOT_PASSWORD": "root" + string(rune('A'+i)),
		}
		d := CreateDeployment(t, template.ID, variables)
		deployments = append(deployments, d)
	}

	// Verify all created with unique IDs
	ids := make(map[string]bool)
	for _, d := range deployments {
		assert.False(t, ids[d.ID], "Duplicate deployment ID found")
		ids[d.ID] = true
		assert.Equal(t, "pending", d.Status)
	}

	// Start all deployments
	for _, d := range deployments {
		started := StartDeployment(t, d.ID)
		assert.Equal(t, "running", started.Status)
	}

	// Stop and cleanup all
	for _, d := range deployments {
		StopDeployment(t, d.ID)
		DeleteDeployment(t, d.ID)
	}

	t.Log("PASS: Multiple WordPress deployments completed")
}

// TestE2E_WordPress_StopRestartCycle tests stop and restart cycle.
func TestE2E_WordPress_StopRestartCycle(t *testing.T) {
	// Setup
	composeYAML := LoadFixture(t, "wordpress-mysql.yaml")
	template := CreateTemplate(t, "wordpress-cycle", "1.0.0", composeYAML)
	PublishTemplate(t, template.ID)

	variables := map[string]string{
		"DB_PASSWORD":      "cyclepass",
		"DB_ROOT_PASSWORD": "cycleroot",
	}
	deployment := CreateDeployment(t, template.ID, variables)

	// Start
	deployment = StartDeployment(t, deployment.ID)
	assert.Equal(t, "running", deployment.Status)

	// Stop
	deployment = StopDeployment(t, deployment.ID)
	assert.Equal(t, "stopped", deployment.Status)

	// Restart (start again from stopped state)
	deployment = StartDeployment(t, deployment.ID)
	assert.Equal(t, "running", deployment.Status)

	// Stop again
	deployment = StopDeployment(t, deployment.ID)
	assert.Equal(t, "stopped", deployment.Status)

	// Cleanup
	DeleteDeployment(t, deployment.ID)

	t.Log("PASS: WordPress stop/restart cycle completed")
}
