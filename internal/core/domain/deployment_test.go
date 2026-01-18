package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Deployment Creation Tests
// =============================================================================

func TestNewDeployment_ValidInput(t *testing.T) {
	template := createValidTemplate()
	variables := map[string]string{"DB_PASSWORD": "secret123"}

	deployment, err := NewDeployment(template, "customer-123", variables)
	require.NoError(t, err)

	assert.NotEmpty(t, deployment.ID)
	assert.Contains(t, deployment.Name, template.Slug)
	assert.Equal(t, template.ID, deployment.TemplateID)
	assert.Equal(t, template.Version, deployment.TemplateVersion)
	assert.Equal(t, "customer-123", deployment.CustomerID)
	assert.Equal(t, StatusPending, deployment.Status)
	assert.Equal(t, "secret123", deployment.Variables["DB_PASSWORD"])
	assert.NotZero(t, deployment.CreatedAt)
}

func TestNewDeployment_MissingRequiredVariable(t *testing.T) {
	template := createValidTemplate()
	variables := map[string]string{} // Missing DB_PASSWORD

	_, err := NewDeployment(template, "customer-123", variables)
	assert.ErrorIs(t, err, ErrMissingVariable)
}

func TestNewDeployment_UnpublishedTemplate(t *testing.T) {
	template := createValidTemplate()
	template.Published = false
	variables := map[string]string{"DB_PASSWORD": "secret123"}

	_, err := NewDeployment(template, "customer-123", variables)
	assert.ErrorIs(t, err, ErrTemplateNotPublished)
}

// =============================================================================
// Name Generation Tests
// =============================================================================

func TestGenerateDeploymentName(t *testing.T) {
	name := GenerateDeploymentName("wordpress-blog")

	assert.Contains(t, name, "wordpress-blog-")
	assert.Len(t, name, len("wordpress-blog-")+6) // 6 char suffix
}

func TestGenerateDeploymentName_UniqueSuffix(t *testing.T) {
	name1 := GenerateDeploymentName("test")
	name2 := GenerateDeploymentName("test")

	assert.NotEqual(t, name1, name2)
}

// =============================================================================
// Status Transition Tests
// =============================================================================

func TestDeployment_Transition_PendingToScheduled(t *testing.T) {
	deployment := createPendingDeployment()

	err := deployment.Transition(StatusScheduled)
	assert.NoError(t, err)
	assert.Equal(t, StatusScheduled, deployment.Status)
}

func TestDeployment_Transition_ScheduledToStarting(t *testing.T) {
	deployment := createPendingDeployment()
	deployment.Status = StatusScheduled
	deployment.NodeID = "node-123"

	err := deployment.Transition(StatusStarting)
	assert.NoError(t, err)
	assert.Equal(t, StatusStarting, deployment.Status)
}

func TestDeployment_Transition_StartingToRunning(t *testing.T) {
	deployment := createPendingDeployment()
	deployment.Status = StatusStarting

	err := deployment.Transition(StatusRunning)
	assert.NoError(t, err)
	assert.Equal(t, StatusRunning, deployment.Status)
	assert.NotZero(t, deployment.StartedAt)
}

func TestDeployment_Transition_RunningToStopping(t *testing.T) {
	deployment := createPendingDeployment()
	deployment.Status = StatusRunning

	err := deployment.Transition(StatusStopping)
	assert.NoError(t, err)
	assert.Equal(t, StatusStopping, deployment.Status)
}

func TestDeployment_Transition_StoppingToStopped(t *testing.T) {
	deployment := createPendingDeployment()
	deployment.Status = StatusStopping

	err := deployment.Transition(StatusStopped)
	assert.NoError(t, err)
	assert.Equal(t, StatusStopped, deployment.Status)
	assert.NotZero(t, deployment.StoppedAt)
}

func TestDeployment_Transition_StoppedToStarting(t *testing.T) {
	deployment := createPendingDeployment()
	deployment.Status = StatusStopped
	deployment.NodeID = "node-123"

	err := deployment.Transition(StatusStarting)
	assert.NoError(t, err)
	assert.Equal(t, StatusStarting, deployment.Status)
}

func TestDeployment_Transition_ToFailed(t *testing.T) {
	statuses := []DeploymentStatus{StatusStarting, StatusRunning, StatusStopping}
	for _, status := range statuses {
		t.Run(string(status), func(t *testing.T) {
			deployment := createPendingDeployment()
			deployment.Status = status

			err := deployment.TransitionToFailed("something went wrong")
			assert.NoError(t, err)
			assert.Equal(t, StatusFailed, deployment.Status)
			assert.Equal(t, "something went wrong", deployment.ErrorMessage)
		})
	}
}

func TestDeployment_Transition_FailedToStarting(t *testing.T) {
	deployment := createPendingDeployment()
	deployment.Status = StatusFailed
	deployment.NodeID = "node-123"
	deployment.ErrorMessage = "previous error"

	err := deployment.Transition(StatusStarting)
	assert.NoError(t, err)
	assert.Equal(t, StatusStarting, deployment.Status)
	assert.Empty(t, deployment.ErrorMessage) // Error cleared on retry
}

// =============================================================================
// Invalid Transition Tests
// =============================================================================

func TestDeployment_Transition_PendingToRunning_Invalid(t *testing.T) {
	deployment := createPendingDeployment()

	err := deployment.Transition(StatusRunning)
	assert.ErrorIs(t, err, ErrInvalidTransition)
	assert.Equal(t, StatusPending, deployment.Status) // Unchanged
}

func TestDeployment_Transition_RunningToStarting_Invalid(t *testing.T) {
	deployment := createPendingDeployment()
	deployment.Status = StatusRunning

	err := deployment.Transition(StatusStarting)
	assert.ErrorIs(t, err, ErrInvalidTransition)
}

func TestDeployment_Transition_DeletedToAnything_Invalid(t *testing.T) {
	deployment := createPendingDeployment()
	deployment.Status = StatusDeleted

	err := deployment.Transition(StatusStarting)
	assert.ErrorIs(t, err, ErrInvalidTransition)
}

func TestDeployment_Transition_StartingWithoutNode_Invalid(t *testing.T) {
	deployment := createPendingDeployment()
	deployment.Status = StatusScheduled
	deployment.NodeID = "" // No node assigned

	err := deployment.Transition(StatusStarting)
	assert.ErrorIs(t, err, ErrNodeRequired)
}

// =============================================================================
// ValidateTransition Tests
// =============================================================================

func TestValidateTransition_AllValid(t *testing.T) {
	validTransitions := []struct {
		from DeploymentStatus
		to   DeploymentStatus
	}{
		{StatusPending, StatusScheduled},
		{StatusScheduled, StatusStarting},
		{StatusStarting, StatusRunning},
		{StatusStarting, StatusFailed},
		{StatusRunning, StatusStopping},
		{StatusRunning, StatusFailed},
		{StatusStopping, StatusStopped},
		{StatusStopped, StatusStarting},
		{StatusStopped, StatusDeleting},
		{StatusDeleting, StatusDeleted},
		{StatusFailed, StatusStarting},
		{StatusFailed, StatusDeleting},
	}

	for _, tc := range validTransitions {
		t.Run(string(tc.from)+"->"+string(tc.to), func(t *testing.T) {
			err := ValidateTransition(tc.from, tc.to)
			assert.NoError(t, err)
		})
	}
}

func TestValidateTransition_AllInvalid(t *testing.T) {
	invalidTransitions := []struct {
		from DeploymentStatus
		to   DeploymentStatus
	}{
		{StatusPending, StatusRunning},
		{StatusPending, StatusStopped},
		{StatusRunning, StatusPending},
		{StatusRunning, StatusStarting},
		{StatusStopped, StatusRunning},
		{StatusDeleted, StatusRunning},
		{StatusDeleted, StatusStarting},
	}

	for _, tc := range invalidTransitions {
		t.Run(string(tc.from)+"->"+string(tc.to), func(t *testing.T) {
			err := ValidateTransition(tc.from, tc.to)
			assert.ErrorIs(t, err, ErrInvalidTransition)
		})
	}
}

// =============================================================================
// Domain Tests
// =============================================================================

func TestGenerateDomain(t *testing.T) {
	domain := GenerateDomain("wordpress-blog-a1b2c3", "apps.hoster.io")

	assert.Equal(t, "wordpress-blog-a1b2c3.apps.hoster.io", domain.Hostname)
	assert.Equal(t, DomainTypeAuto, domain.Type)
	assert.False(t, domain.SSLEnabled) // SSL enabled after provisioning
}

func TestGenerateDomain_SlugifiesName(t *testing.T) {
	// Names with spaces and mixed case should be slugified
	domain := GenerateDomain("My WordPress Blog", "apps.localhost")

	assert.Equal(t, "my-wordpress-blog.apps.localhost", domain.Hostname)
	assert.Equal(t, DomainTypeAuto, domain.Type)
}

// =============================================================================
// Variable Validation Tests
// =============================================================================

func TestValidateDeploymentVariables_AllRequired(t *testing.T) {
	template := createValidTemplate()
	variables := map[string]string{
		"DB_PASSWORD": "secret123",
	}

	errs := ValidateDeploymentVariables(template.Variables, variables)
	assert.Empty(t, errs)
}

func TestValidateDeploymentVariables_MissingRequired(t *testing.T) {
	template := createValidTemplate()
	variables := map[string]string{} // Missing DB_PASSWORD

	errs := ValidateDeploymentVariables(template.Variables, variables)
	assert.Len(t, errs, 1)
	assert.ErrorIs(t, errs[0], ErrMissingVariable)
}

func TestValidateDeploymentVariables_OptionalMissing(t *testing.T) {
	template := createValidTemplate()
	template.Variables = append(template.Variables, Variable{
		Name:     "OPTIONAL_VAR",
		Label:    "Optional",
		Type:     VarTypeString,
		Required: false,
	})
	variables := map[string]string{
		"DB_PASSWORD": "secret123",
		// OPTIONAL_VAR not provided - should be fine
	}

	errs := ValidateDeploymentVariables(template.Variables, variables)
	assert.Empty(t, errs)
}

// =============================================================================
// Test Helpers
// =============================================================================

func createValidTemplate() Template {
	return Template{
		ID:        "template-123",
		Name:      "WordPress Blog",
		Slug:      "wordpress-blog",
		Version:   "1.0.0",
		Published: true,
		Variables: []Variable{
			{Name: "DB_PASSWORD", Label: "Database Password", Type: VarTypePassword, Required: true},
		},
	}
}

func createPendingDeployment() *Deployment {
	return &Deployment{
		ID:              "deployment-123",
		Name:            "wordpress-blog-abc123",
		TemplateID:      "template-123",
		TemplateVersion: "1.0.0",
		CustomerID:      "customer-123",
		Status:          StatusPending,
		Variables:       map[string]string{"DB_PASSWORD": "secret123"},
	}
}
