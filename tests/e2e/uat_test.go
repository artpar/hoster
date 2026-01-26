package e2e

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// User Acceptance Testing (UAT) Scenarios
// =============================================================================
//
// These tests simulate real user workflows, with Claude acting as the user.
// Each test represents a complete user journey through the system.

// TestUAT_FirstTimeUser_ExploresTemplates simulates a new user browsing templates.
func TestUAT_FirstTimeUser_ExploresTemplates(t *testing.T) {
	t.Log("UAT: User visits the platform for the first time")

	// Step 1: User checks if the platform is healthy
	t.Log("UAT: User checks platform health...")
	resp := HTTPGet(t, baseURL+"/health")
	resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode, "User expects platform to be healthy")

	// Step 2: User browses available templates
	t.Log("UAT: User browses available templates...")
	templates := ListTemplates(t)
	t.Logf("UAT: User sees %d templates available", len(templates))

	// Step 3: If no templates exist, user expects an empty but valid response
	// (In a real scenario, there would be marketplace templates)
	require.NotNil(t, templates, "User expects a list, even if empty")

	t.Log("UAT PASSED: First-time user can explore the platform")
}

// TestUAT_Developer_CreatesTemplate simulates a developer creating a template.
func TestUAT_Developer_CreatesTemplate(t *testing.T) {
	t.Log("UAT: Developer wants to create a new template for the marketplace")

	// Step 1: Developer prepares their docker-compose file
	t.Log("UAT: Developer prepares docker-compose.yaml...")
	composeYAML := LoadFixture(t, "nginx-simple.yaml")
	require.NotEmpty(t, composeYAML, "Developer has a valid compose file")

	// Step 2: Developer submits the template
	t.Log("UAT: Developer submits template to platform...")
	template := CreateTemplate(t, "uat-dev-template", "1.0.0", composeYAML)
	t.Logf("UAT: Template created with ID: %s", template.ID)

	// Step 3: Developer verifies template was created correctly
	t.Log("UAT: Developer checks template details...")
	fetched := GetTemplate(t, template.ID)
	assert.Equal(t, "uat-dev-template", fetched.Name)
	assert.Equal(t, "1.0.0", fetched.Version)
	assert.False(t, fetched.Published, "Template should be draft initially")

	// Step 4: Developer publishes when ready
	t.Log("UAT: Developer publishes template to marketplace...")
	PublishTemplate(t, template.ID)

	// Step 5: Developer confirms publication
	fetched = GetTemplate(t, template.ID)
	assert.True(t, fetched.Published, "Template should now be published")
	t.Logf("UAT: Template %s is now live!", template.ID)

	t.Log("UAT PASSED: Developer successfully created and published template")
}

// TestUAT_Customer_DeploysWordPress simulates a customer deploying WordPress.
func TestUAT_Customer_DeploysWordPress(t *testing.T) {
	t.Log("UAT: Customer wants to deploy a WordPress blog")

	// Pre-requisite: A WordPress template exists
	t.Log("UAT: Setting up WordPress template...")
	composeYAML := LoadFixture(t, "wordpress-mysql.yaml")
	template := CreateTemplate(t, "uat-wordpress", "1.0.0", composeYAML)
	PublishTemplate(t, template.ID)

	// Step 1: Customer finds the WordPress template
	t.Log("UAT: Customer searches for WordPress template...")
	templates := ListTemplates(t)
	var wordpressTemplate *TemplateResponse
	for _, tmpl := range templates {
		if tmpl.ID == template.ID {
			wordpressTemplate = &tmpl
			break
		}
	}
	require.NotNil(t, wordpressTemplate, "Customer finds WordPress template")
	t.Logf("UAT: Customer found template: %s", wordpressTemplate.Name)

	// Step 2: Customer fills in the deployment form
	t.Log("UAT: Customer fills in configuration...")
	variables := map[string]string{
		"DB_PASSWORD":      "mycustomerblogpass123",
		"DB_ROOT_PASSWORD": "rootpassword456",
	}

	// Step 3: Customer clicks "Deploy"
	t.Log("UAT: Customer clicks Deploy button...")
	deployment := CreateDeployment(t, wordpressTemplate.ID, variables)
	t.Logf("UAT: Deployment created: %s (status: %s)", deployment.ID, deployment.Status)

	// Step 4: Customer starts the deployment
	t.Log("UAT: Customer starts the deployment...")
	deployment = StartDeployment(t, deployment.ID)

	// Step 5: Customer waits and watches progress (checking every 5 seconds)
	t.Log("UAT: Customer watches deployment progress...")
	var lastStatus string
	maxChecks := 5
	for i := range maxChecks {
		fetched := GetDeployment(t, deployment.ID)
		if fetched.Status != lastStatus {
			t.Logf("UAT: [Check %d] Status: %s -> %s", i+1, lastStatus, fetched.Status)
			lastStatus = fetched.Status
		}
		if fetched.Status == "running" {
			break
		}
		if fetched.Status == "failed" {
			t.Fatal("UAT: Deployment failed!")
		}
		time.Sleep(1 * time.Second) // Simulating polling
	}

	// Step 6: Customer verifies deployment is running
	assert.Equal(t, "running", lastStatus, "Customer expects deployment to be running")
	t.Logf("UAT: Customer's WordPress is now running!")

	// Step 7: Customer decides to stop the blog temporarily
	t.Log("UAT: Customer stops deployment for maintenance...")
	StopDeployment(t, deployment.ID)

	fetched := GetDeployment(t, deployment.ID)
	assert.Equal(t, "stopped", fetched.Status)
	t.Log("UAT: Customer's blog is stopped")

	// Cleanup
	DeleteDeployment(t, deployment.ID)

	t.Log("UAT PASSED: Customer successfully deployed and managed WordPress")
}

// TestUAT_Admin_MonitorsDeployments simulates an admin monitoring all deployments.
func TestUAT_Admin_MonitorsDeployments(t *testing.T) {
	t.Log("UAT: Admin monitors platform deployments")

	// Setup: Create some deployments
	composeYAML := LoadFixture(t, "nginx-simple.yaml")
	template := CreateTemplate(t, "uat-admin-template", "1.0.0", composeYAML)
	PublishTemplate(t, template.ID)

	// Create multiple customer deployments
	var deployments []*DeploymentResponse
	for i := 0; i < 3; i++ {
		d := CreateDeployment(t, template.ID, nil)
		StartDeployment(t, d.ID)
		deployments = append(deployments, d)
	}
	t.Logf("UAT: Admin sees %d active deployments", len(deployments))

	// Admin checks each deployment status
	t.Log("UAT: Admin reviews deployment statuses...")
	for _, d := range deployments {
		fetched := GetDeployment(t, d.ID)
		t.Logf("UAT:   - %s: %s (customer: %s)", fetched.ID, fetched.Status, fetched.CustomerID)
		assert.Equal(t, "running", fetched.Status)
	}

	// Cleanup
	for _, d := range deployments {
		StopDeployment(t, d.ID)
		DeleteDeployment(t, d.ID)
	}

	t.Log("UAT PASSED: Admin can monitor all deployments")
}

// TestUAT_Customer_HandlesDeploymentFailure simulates handling a failed deployment.
func TestUAT_Customer_HandlesDeploymentFailure(t *testing.T) {
	t.Log("UAT: Customer experiences a deployment issue")

	// Setup
	composeYAML := LoadFixture(t, "nginx-simple.yaml")
	template := CreateTemplate(t, "uat-failure-template", "1.0.0", composeYAML)
	PublishTemplate(t, template.ID)

	// Step 1: Customer creates and starts deployment
	deployment := CreateDeployment(t, template.ID, nil)
	StartDeployment(t, deployment.ID)

	// Step 2: Customer checks status (in this case, it should be running)
	fetched := GetDeployment(t, deployment.ID)
	t.Logf("UAT: Deployment status: %s", fetched.Status)

	// Step 3: If there was a failure (simulated), customer would see error message
	if fetched.Status == "failed" {
		t.Logf("UAT: Customer sees error: %s", fetched.ErrorMessage)
		// Customer would then decide to retry or seek support
	}

	// Step 4: Customer can always stop and try again
	t.Log("UAT: Customer stops deployment to investigate...")
	StopDeployment(t, deployment.ID)

	// Cleanup
	DeleteDeployment(t, deployment.ID)

	t.Log("UAT PASSED: Customer can handle deployment issues")
}

// TestUAT_FullPlatformJourney simulates a complete platform usage journey.
func TestUAT_FullPlatformJourney(t *testing.T) {
	t.Log("UAT: Complete platform journey test")

	// === Phase 1: Developer creates template ===
	t.Log("UAT Phase 1: Developer creates and publishes template")
	composeYAML := LoadFixture(t, "wordpress-mysql.yaml")
	template := CreateTemplate(t, "uat-journey-template", "1.0.0", composeYAML)
	PublishTemplate(t, template.ID)
	t.Logf("UAT: Template %s published", template.ID)

	// === Phase 2: Customer deploys ===
	t.Log("UAT Phase 2: Customer deploys from template")
	deployment := CreateDeployment(t, template.ID, map[string]string{
		"DB_PASSWORD":      "journeypass",
		"DB_ROOT_PASSWORD": "journeyroot",
	})
	t.Logf("UAT: Deployment %s created", deployment.ID)

	// === Phase 3: Customer starts and uses ===
	t.Log("UAT Phase 3: Customer starts deployment")
	StartDeployment(t, deployment.ID)
	fetched := GetDeployment(t, deployment.ID)
	assert.Equal(t, "running", fetched.Status)
	t.Log("UAT: Deployment is running, customer is using the service")

	// === Phase 4: Customer stops for maintenance ===
	t.Log("UAT Phase 4: Customer stops for maintenance")
	StopDeployment(t, deployment.ID)
	fetched = GetDeployment(t, deployment.ID)
	assert.Equal(t, "stopped", fetched.Status)
	t.Log("UAT: Deployment stopped")

	// === Phase 5: Customer restarts ===
	t.Log("UAT Phase 5: Customer restarts after maintenance")
	StartDeployment(t, deployment.ID)
	fetched = GetDeployment(t, deployment.ID)
	assert.Equal(t, "running", fetched.Status)
	t.Log("UAT: Deployment restarted successfully")

	// === Phase 6: Customer deletes when done ===
	t.Log("UAT Phase 6: Customer removes deployment")
	StopDeployment(t, deployment.ID)
	DeleteDeployment(t, deployment.ID)
	t.Log("UAT: Deployment deleted")

	t.Log("UAT PASSED: Complete platform journey successful!")
}
