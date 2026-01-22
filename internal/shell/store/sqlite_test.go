package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/artpar/hoster/internal/core/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Test Helpers
// =============================================================================

func setupTestStore(t *testing.T) Store {
	t.Helper()
	store, err := NewSQLiteStore(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() {
		store.Close()
	})
	return store
}

// setupEmptyTestStore creates a test store and clears all default templates
// Use this for tests that expect an empty database
func setupEmptyTestStore(t *testing.T) *SQLiteStore {
	t.Helper()
	store, err := NewSQLiteStore(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() {
		store.Close()
	})
	// Clear default templates created by migrations
	_, err = store.db.Exec("DELETE FROM templates")
	require.NoError(t, err)
	return store
}

func createTestTemplate(t *testing.T, store Store) *domain.Template {
	t.Helper()
	template, err := domain.NewTemplate(
		"Test Template",
		"1.0.0",
		"services:\n  web:\n    image: nginx",
	)
	require.NoError(t, err)
	template.CreatorID = "creator-123"

	err = store.CreateTemplate(context.Background(), template)
	require.NoError(t, err)
	return template
}

func createPublishedTemplate(t *testing.T, store Store) *domain.Template {
	t.Helper()
	template := createTestTemplate(t, store)
	err := template.Publish()
	require.NoError(t, err)
	err = store.UpdateTemplate(context.Background(), template)
	require.NoError(t, err)
	return template
}

func createTestDeployment(t *testing.T, store Store, template *domain.Template) *domain.Deployment {
	t.Helper()
	deployment, err := domain.NewDeployment(*template, "customer-123", nil)
	require.NoError(t, err)
	err = store.CreateDeployment(context.Background(), deployment)
	require.NoError(t, err)
	return deployment
}

// =============================================================================
// Template CRUD Tests
// =============================================================================

func TestCreateTemplate_Success(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	template, err := domain.NewTemplate(
		"Test Template",
		"1.0.0",
		"services:\n  web:\n    image: nginx",
	)
	require.NoError(t, err)
	template.CreatorID = "creator-123"

	err = store.CreateTemplate(ctx, template)
	require.NoError(t, err)

	// Verify template was created
	retrieved, err := store.GetTemplate(ctx, template.ID)
	require.NoError(t, err)
	assert.Equal(t, template.ID, retrieved.ID)
	assert.Equal(t, template.Name, retrieved.Name)
	assert.Equal(t, template.Slug, retrieved.Slug)
	assert.Equal(t, template.Version, retrieved.Version)
	assert.Equal(t, template.ComposeSpec, retrieved.ComposeSpec)
	assert.Equal(t, template.CreatorID, retrieved.CreatorID)
}

func TestCreateTemplate_DuplicateID(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	template := createTestTemplate(t, store)

	// Try to create another template with same ID
	duplicate := *template
	duplicate.Slug = "different-slug"

	err := store.CreateTemplate(ctx, &duplicate)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrDuplicateID)
}

func TestCreateTemplate_DuplicateSlug(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	template := createTestTemplate(t, store)

	// Create another template with same slug but different ID
	duplicate, err := domain.NewTemplate(
		template.Name, // Same name = same slug
		"2.0.0",
		"services:\n  web:\n    image: nginx",
	)
	require.NoError(t, err)
	duplicate.CreatorID = "creator-456"

	err = store.CreateTemplate(ctx, duplicate)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrDuplicateSlug)
}

func TestGetTemplate_Success(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	template := createTestTemplate(t, store)

	retrieved, err := store.GetTemplate(ctx, template.ID)
	require.NoError(t, err)
	assert.Equal(t, template.ID, retrieved.ID)
	assert.Equal(t, template.Name, retrieved.Name)
}

func TestGetTemplate_NotFound(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	_, err := store.GetTemplate(ctx, "nonexistent-id")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestGetTemplateBySlug_Success(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	template := createTestTemplate(t, store)

	retrieved, err := store.GetTemplateBySlug(ctx, template.Slug)
	require.NoError(t, err)
	assert.Equal(t, template.ID, retrieved.ID)
	assert.Equal(t, template.Slug, retrieved.Slug)
}

func TestUpdateTemplate_Success(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	template := createTestTemplate(t, store)

	// Update the template
	template.Description = "Updated description"
	template.UpdatedAt = time.Now()

	err := store.UpdateTemplate(ctx, template)
	require.NoError(t, err)

	// Verify update
	retrieved, err := store.GetTemplate(ctx, template.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated description", retrieved.Description)
}

func TestDeleteTemplate_Success(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	template := createTestTemplate(t, store)

	err := store.DeleteTemplate(ctx, template.ID)
	require.NoError(t, err)

	// Verify deletion
	_, err = store.GetTemplate(ctx, template.ID)
	assert.ErrorIs(t, err, ErrNotFound)
}

// =============================================================================
// Deployment CRUD Tests
// =============================================================================

func TestCreateDeployment_Success(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	template := createPublishedTemplate(t, store)

	deployment, err := domain.NewDeployment(*template, "customer-123", nil)
	require.NoError(t, err)

	err = store.CreateDeployment(ctx, deployment)
	require.NoError(t, err)

	// Verify deployment was created
	retrieved, err := store.GetDeployment(ctx, deployment.ID)
	require.NoError(t, err)
	assert.Equal(t, deployment.ID, retrieved.ID)
	assert.Equal(t, deployment.Name, retrieved.Name)
	assert.Equal(t, deployment.TemplateID, retrieved.TemplateID)
	assert.Equal(t, deployment.CustomerID, retrieved.CustomerID)
}

func TestCreateDeployment_ForeignKeyError(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	// Create a deployment struct manually with fake template ID
	deployment := &domain.Deployment{
		ID:         "test-deployment-id",
		Name:       "Test Deployment",
		TemplateID: "nonexistent-template-id",
		CustomerID: "customer-123",
		Status:     domain.StatusPending,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	err := store.CreateDeployment(ctx, deployment)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrForeignKey)
}

func TestGetDeployment_Success(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	template := createPublishedTemplate(t, store)
	deployment := createTestDeployment(t, store, template)

	retrieved, err := store.GetDeployment(ctx, deployment.ID)
	require.NoError(t, err)
	assert.Equal(t, deployment.ID, retrieved.ID)
}

func TestGetDeployment_NotFound(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	_, err := store.GetDeployment(ctx, "nonexistent-id")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestUpdateDeployment_Success(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	template := createPublishedTemplate(t, store)
	deployment := createTestDeployment(t, store, template)

	// Transition status - need to assign node first, then transition
	deployment.NodeID = "node-123"
	err := deployment.Transition(domain.StatusScheduled)
	require.NoError(t, err)
	deployment.UpdatedAt = time.Now()

	err = store.UpdateDeployment(ctx, deployment)
	require.NoError(t, err)

	// Verify update
	retrieved, err := store.GetDeployment(ctx, deployment.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.StatusScheduled, retrieved.Status)
}

func TestDeleteDeployment_Success(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	template := createPublishedTemplate(t, store)
	deployment := createTestDeployment(t, store, template)

	err := store.DeleteDeployment(ctx, deployment.ID)
	require.NoError(t, err)

	// Verify deletion
	_, err = store.GetDeployment(ctx, deployment.ID)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestListDeploymentsByTemplate_Success(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	template := createPublishedTemplate(t, store)
	deployment1 := createTestDeployment(t, store, template)
	deployment2 := createTestDeployment(t, store, template)

	deployments, err := store.ListDeploymentsByTemplate(ctx, template.ID, DefaultListOptions())
	require.NoError(t, err)
	assert.Len(t, deployments, 2)

	// Verify both deployments are in the list
	ids := make(map[string]bool)
	for _, d := range deployments {
		ids[d.ID] = true
	}
	assert.True(t, ids[deployment1.ID])
	assert.True(t, ids[deployment2.ID])
}

func TestListDeploymentsByCustomer_Success(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	template := createPublishedTemplate(t, store)
	deployment := createTestDeployment(t, store, template)

	deployments, err := store.ListDeploymentsByCustomer(ctx, deployment.CustomerID, DefaultListOptions())
	require.NoError(t, err)
	assert.Len(t, deployments, 1)
	assert.Equal(t, deployment.ID, deployments[0].ID)
}

// =============================================================================
// JSON Serialization Tests
// =============================================================================

func TestTemplate_VariablesSerialization(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	template, err := domain.NewTemplate(
		"Test Template",
		"1.0.0",
		"services:\n  web:\n    image: nginx\n    environment:\n      DB_PASSWORD: ${DB_PASSWORD}",
	)
	require.NoError(t, err)
	template.CreatorID = "creator-123"
	template.Variables = []domain.Variable{
		{Name: "DB_PASSWORD", Label: "Database Password", Type: domain.VarTypePassword, Required: true},
		{Name: "API_KEY", Label: "API Key", Type: domain.VarTypeString},
	}

	err = store.CreateTemplate(ctx, template)
	require.NoError(t, err)

	retrieved, err := store.GetTemplate(ctx, template.ID)
	require.NoError(t, err)
	require.Len(t, retrieved.Variables, 2)
	assert.Equal(t, "DB_PASSWORD", retrieved.Variables[0].Name)
	assert.Equal(t, domain.VarTypePassword, retrieved.Variables[0].Type)
}

func TestDeployment_VariablesSerialization(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	template := createPublishedTemplate(t, store)

	deployment, err := domain.NewDeployment(*template, "customer-123", map[string]string{
		"DB_PASSWORD": "secret123",
		"API_KEY":     "abc123",
	})
	require.NoError(t, err)

	err = store.CreateDeployment(ctx, deployment)
	require.NoError(t, err)

	retrieved, err := store.GetDeployment(ctx, deployment.ID)
	require.NoError(t, err)
	assert.Equal(t, "secret123", retrieved.Variables["DB_PASSWORD"])
	assert.Equal(t, "abc123", retrieved.Variables["API_KEY"])
}

func TestDeployment_ContainersSerialization(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	template := createPublishedTemplate(t, store)
	deployment := createTestDeployment(t, store, template)

	// Add containers
	deployment.Containers = []domain.ContainerInfo{
		{ID: "abc123", ServiceName: "web", Status: "running"},
		{ID: "def456", ServiceName: "db", Status: "running"},
	}
	deployment.UpdatedAt = time.Now()

	err := store.UpdateDeployment(ctx, deployment)
	require.NoError(t, err)

	retrieved, err := store.GetDeployment(ctx, deployment.ID)
	require.NoError(t, err)
	require.Len(t, retrieved.Containers, 2)
	assert.Equal(t, "abc123", retrieved.Containers[0].ID)
	assert.Equal(t, "web", retrieved.Containers[0].ServiceName)
}

func TestTemplate_EmptyVariables(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	template := createTestTemplate(t, store)

	retrieved, err := store.GetTemplate(ctx, template.ID)
	require.NoError(t, err)
	// Empty slice or nil - both are acceptable
	assert.Empty(t, retrieved.Variables)
}

// =============================================================================
// List Operations Tests
// =============================================================================

func TestListTemplates_WithPagination(t *testing.T) {
	store := setupEmptyTestStore(t)
	ctx := context.Background()

	// Create 5 templates
	for i := 0; i < 5; i++ {
		template, err := domain.NewTemplate(
			"Template "+string(rune('A'+i)),
			"1.0.0",
			"services:\n  web:\n    image: nginx",
		)
		require.NoError(t, err)
		template.CreatorID = "creator-123"
		err = store.CreateTemplate(ctx, template)
		require.NoError(t, err)
	}

	// Get first page
	templates, err := store.ListTemplates(ctx, ListOptions{Limit: 2, Offset: 0})
	require.NoError(t, err)
	assert.Len(t, templates, 2)

	// Get second page
	templates, err = store.ListTemplates(ctx, ListOptions{Limit: 2, Offset: 2})
	require.NoError(t, err)
	assert.Len(t, templates, 2)

	// Get last page
	templates, err = store.ListTemplates(ctx, ListOptions{Limit: 2, Offset: 4})
	require.NoError(t, err)
	assert.Len(t, templates, 1)
}

func TestListDeployments_WithPagination(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	template := createPublishedTemplate(t, store)

	// Create 3 deployments
	for i := 0; i < 3; i++ {
		deployment, err := domain.NewDeployment(*template, "customer-123", nil)
		require.NoError(t, err)
		err = store.CreateDeployment(ctx, deployment)
		require.NoError(t, err)
	}

	// Get with limit
	deployments, err := store.ListDeployments(ctx, ListOptions{Limit: 2, Offset: 0})
	require.NoError(t, err)
	assert.Len(t, deployments, 2)
}

func TestListTemplates_EmptyResult(t *testing.T) {
	store := setupEmptyTestStore(t)
	ctx := context.Background()

	templates, err := store.ListTemplates(ctx, DefaultListOptions())
	require.NoError(t, err)
	assert.Empty(t, templates)
}

func TestListTemplates_OffsetBeyondData(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	createTestTemplate(t, store)

	templates, err := store.ListTemplates(ctx, ListOptions{Limit: 10, Offset: 100})
	require.NoError(t, err)
	assert.Empty(t, templates)
}

func TestListOptions_Normalize(t *testing.T) {
	// Test default limit
	opts := ListOptions{Limit: 0, Offset: 0}
	normalized := opts.Normalize()
	assert.Equal(t, 100, normalized.Limit)

	// Test max limit
	opts = ListOptions{Limit: 5000, Offset: 0}
	normalized = opts.Normalize()
	assert.Equal(t, 1000, normalized.Limit)

	// Test negative offset
	opts = ListOptions{Limit: 10, Offset: -5}
	normalized = opts.Normalize()
	assert.Equal(t, 0, normalized.Offset)
}

// =============================================================================
// Transaction Tests
// =============================================================================

func TestWithTx_CommitSuccess(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	var createdID string

	err := store.WithTx(ctx, func(txStore Store) error {
		template, err := domain.NewTemplate(
			"Transaction Test",
			"1.0.0",
			"services:\n  web:\n    image: nginx",
		)
		if err != nil {
			return err
		}
		template.CreatorID = "creator-123"
		createdID = template.ID
		return txStore.CreateTemplate(ctx, template)
	})
	require.NoError(t, err)

	// Verify template was persisted
	retrieved, err := store.GetTemplate(ctx, createdID)
	require.NoError(t, err)
	assert.Equal(t, "Transaction Test", retrieved.Name)
}

func TestWithTx_RollbackOnError(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	var createdID string

	err := store.WithTx(ctx, func(txStore Store) error {
		template, err := domain.NewTemplate(
			"Rollback Test",
			"1.0.0",
			"services:\n  web:\n    image: nginx",
		)
		if err != nil {
			return err
		}
		template.CreatorID = "creator-123"
		createdID = template.ID

		if err := txStore.CreateTemplate(ctx, template); err != nil {
			return err
		}

		// Return error to trigger rollback
		return assert.AnError
	})
	require.Error(t, err)

	// Verify template was NOT persisted
	_, err = store.GetTemplate(ctx, createdID)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestWithTx_ContextCancellation(t *testing.T) {
	store := setupTestStore(t)
	ctx, cancel := context.WithCancel(context.Background())

	err := store.WithTx(ctx, func(txStore Store) error {
		// Cancel context during transaction
		cancel()

		template, err := domain.NewTemplate(
			"Cancelled Test",
			"1.0.0",
			"services:\n  web:\n    image: nginx",
		)
		if err != nil {
			return err
		}
		template.CreatorID = "creator-123"
		return txStore.CreateTemplate(ctx, template)
	})
	// Should get context cancelled error
	require.Error(t, err)
}

// TestWithTx_AllTemplateOperations exercises all template operations within a transaction.
func TestWithTx_AllTemplateOperations(t *testing.T) {
	store := setupEmptyTestStore(t)
	ctx := context.Background()

	var templateID, templateSlug string

	err := store.WithTx(ctx, func(txStore Store) error {
		// Create template
		template, err := domain.NewTemplate(
			"Tx Template Operations",
			"1.0.0",
			"services:\n  web:\n    image: nginx",
		)
		if err != nil {
			return err
		}
		template.CreatorID = "creator-123"
		templateID = template.ID
		templateSlug = template.Slug

		if err := txStore.CreateTemplate(ctx, template); err != nil {
			return err
		}

		// Get template
		retrieved, err := txStore.GetTemplate(ctx, templateID)
		if err != nil {
			return err
		}
		if retrieved.Name != "Tx Template Operations" {
			return assert.AnError
		}

		// Get template by slug
		retrievedBySlug, err := txStore.GetTemplateBySlug(ctx, templateSlug)
		if err != nil {
			return err
		}
		if retrievedBySlug.ID != templateID {
			return assert.AnError
		}

		// Update template
		retrieved.Description = "Updated in transaction"
		retrieved.UpdatedAt = time.Now()
		if err := txStore.UpdateTemplate(ctx, retrieved); err != nil {
			return err
		}

		// List templates
		templates, err := txStore.ListTemplates(ctx, DefaultListOptions())
		if err != nil {
			return err
		}
		if len(templates) != 1 {
			return assert.AnError
		}

		// Delete template
		if err := txStore.DeleteTemplate(ctx, templateID); err != nil {
			return err
		}

		// Verify deleted
		_, err = txStore.GetTemplate(ctx, templateID)
		if !errors.Is(err, ErrNotFound) {
			return assert.AnError
		}

		return nil
	})
	require.NoError(t, err)
}

// TestWithTx_AllDeploymentOperations exercises all deployment operations within a transaction.
func TestWithTx_AllDeploymentOperations(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	// First create a template outside the transaction
	template := createPublishedTemplate(t, store)

	var deploymentID string

	err := store.WithTx(ctx, func(txStore Store) error {
		// Create deployment
		deployment, err := domain.NewDeployment(*template, "customer-tx", nil)
		if err != nil {
			return err
		}
		deploymentID = deployment.ID

		if err := txStore.CreateDeployment(ctx, deployment); err != nil {
			return err
		}

		// Get deployment
		retrieved, err := txStore.GetDeployment(ctx, deploymentID)
		if err != nil {
			return err
		}
		if retrieved.CustomerID != "customer-tx" {
			return assert.AnError
		}

		// Update deployment
		retrieved.NodeID = "node-123"
		retrieved.UpdatedAt = time.Now()
		if err := txStore.UpdateDeployment(ctx, retrieved); err != nil {
			return err
		}

		// List deployments
		deployments, err := txStore.ListDeployments(ctx, DefaultListOptions())
		if err != nil {
			return err
		}
		if len(deployments) != 1 {
			return assert.AnError
		}

		// List by template
		deploymentsByTemplate, err := txStore.ListDeploymentsByTemplate(ctx, template.ID, DefaultListOptions())
		if err != nil {
			return err
		}
		if len(deploymentsByTemplate) != 1 {
			return assert.AnError
		}

		// List by customer
		deploymentsByCustomer, err := txStore.ListDeploymentsByCustomer(ctx, "customer-tx", DefaultListOptions())
		if err != nil {
			return err
		}
		if len(deploymentsByCustomer) != 1 {
			return assert.AnError
		}

		// Delete deployment
		if err := txStore.DeleteDeployment(ctx, deploymentID); err != nil {
			return err
		}

		// Verify deleted
		_, err = txStore.GetDeployment(ctx, deploymentID)
		if !errors.Is(err, ErrNotFound) {
			return assert.AnError
		}

		return nil
	})
	require.NoError(t, err)
}

// TestWithTx_NestedTx tests nested transaction handling.
func TestWithTx_NestedTx(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	var templateID string

	err := store.WithTx(ctx, func(txStore Store) error {
		// Create template
		template, err := domain.NewTemplate(
			"Nested Tx Test",
			"1.0.0",
			"services:\n  web:\n    image: nginx",
		)
		if err != nil {
			return err
		}
		template.CreatorID = "creator-123"
		templateID = template.ID

		if err := txStore.CreateTemplate(ctx, template); err != nil {
			return err
		}

		// Nested transaction (should just run the function)
		return txStore.WithTx(ctx, func(nestedTxStore Store) error {
			// Should be able to access the template created above
			_, err := nestedTxStore.GetTemplate(ctx, templateID)
			return err
		})
	})
	require.NoError(t, err)

	// Verify template was persisted
	retrieved, err := store.GetTemplate(ctx, templateID)
	require.NoError(t, err)
	assert.Equal(t, "Nested Tx Test", retrieved.Name)
}

// TestTxStore_Close tests the Close method of txSQLiteStore.
func TestWithTx_TxStoreClose(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	err := store.WithTx(ctx, func(txStore Store) error {
		// Close should be a no-op for tx store
		return txStore.Close()
	})
	require.NoError(t, err)
}

// TestWithTx_GetTemplateBySlugNotFound tests GetTemplateBySlug error path in transaction.
func TestWithTx_GetTemplateBySlugNotFound(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	err := store.WithTx(ctx, func(txStore Store) error {
		_, err := txStore.GetTemplateBySlug(ctx, "nonexistent-slug")
		if !errors.Is(err, ErrNotFound) {
			return assert.AnError
		}
		return nil
	})
	require.NoError(t, err)
}

// TestWithTx_UpdateTemplateNotFound tests UpdateTemplate error path in transaction.
func TestWithTx_UpdateTemplateNotFound(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	err := store.WithTx(ctx, func(txStore Store) error {
		template, err := domain.NewTemplate(
			"Nonexistent Template",
			"1.0.0",
			"services:\n  web:\n    image: nginx",
		)
		if err != nil {
			return err
		}
		template.CreatorID = "creator-123"

		// Try to update without creating
		err = txStore.UpdateTemplate(ctx, template)
		if !errors.Is(err, ErrNotFound) {
			return assert.AnError
		}
		return nil
	})
	require.NoError(t, err)
}

// TestWithTx_DeleteTemplateNotFound tests DeleteTemplate error path in transaction.
func TestWithTx_DeleteTemplateNotFound(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	err := store.WithTx(ctx, func(txStore Store) error {
		err := txStore.DeleteTemplate(ctx, "nonexistent-id")
		if !errors.Is(err, ErrNotFound) {
			return assert.AnError
		}
		return nil
	})
	require.NoError(t, err)
}

// =============================================================================
// Error Handling Tests
// =============================================================================

func TestStoreError_Error(t *testing.T) {
	// With all fields
	err := NewStoreError("CreateTemplate", "template", "abc-123", "failed to insert", ErrDuplicateID)
	assert.Equal(t, "CreateTemplate template abc-123: failed to insert", err.Error())

	// Without ID
	err = NewStoreError("ListTemplates", "template", "", "database error", ErrConnectionFailed)
	assert.Equal(t, "ListTemplates template: database error", err.Error())

	// Without entity
	err = NewStoreError("Close", "", "", "connection closed", nil)
	assert.Equal(t, "Close: connection closed", err.Error())
}

func TestStoreError_Unwrap(t *testing.T) {
	err := NewStoreError("CreateTemplate", "template", "abc-123", "failed", ErrDuplicateID)
	assert.ErrorIs(t, err, ErrDuplicateID)
}

// =============================================================================
// Edge Cases
// =============================================================================

func TestTemplate_UnicodeFields(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	template, err := domain.NewTemplate(
		"Test Template Unicode",
		"1.0.0",
		"services:\n  web:\n    image: nginx",
	)
	require.NoError(t, err)
	template.CreatorID = "creator-123"
	template.Description = "テスト Template 中文 émoji 🚀"

	err = store.CreateTemplate(ctx, template)
	require.NoError(t, err)

	retrieved, err := store.GetTemplate(ctx, template.ID)
	require.NoError(t, err)
	assert.Equal(t, "テスト Template 中文 émoji 🚀", retrieved.Description)
}

func TestTemplate_VeryLongComposeSpec(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	// Create a very long compose spec
	longSpec := "services:\n  web:\n    image: nginx\n    environment:\n"
	for i := 0; i < 1000; i++ {
		longSpec += "      VAR_" + string(rune('A'+i%26)) + "_" + string(rune('0'+i%10)) + ": value\n"
	}

	template, err := domain.NewTemplate(
		"Long Spec Template",
		"1.0.0",
		longSpec,
	)
	require.NoError(t, err)
	template.CreatorID = "creator-123"

	err = store.CreateTemplate(ctx, template)
	require.NoError(t, err)

	retrieved, err := store.GetTemplate(ctx, template.ID)
	require.NoError(t, err)
	assert.Equal(t, longSpec, retrieved.ComposeSpec)
}

// =============================================================================
// Deployment StartedAt/StoppedAt Coverage Tests
// =============================================================================

func TestDeployment_WithStartedAtAndStoppedAt(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	// Create published template and deployment
	template := createPublishedTemplate(t, store)
	deployment := createTestDeployment(t, store, template)

	// Transition to running state with StartedAt
	deployment.NodeID = "node-123"
	err := deployment.Transition(domain.StatusScheduled)
	require.NoError(t, err)
	err = deployment.Transition(domain.StatusStarting)
	require.NoError(t, err)
	err = deployment.Transition(domain.StatusRunning)
	require.NoError(t, err)

	now := time.Now().UTC().Truncate(time.Second)
	deployment.StartedAt = &now
	deployment.UpdatedAt = now

	err = store.UpdateDeployment(ctx, deployment)
	require.NoError(t, err)

	// Retrieve and verify StartedAt
	retrieved, err := store.GetDeployment(ctx, deployment.ID)
	require.NoError(t, err)
	require.NotNil(t, retrieved.StartedAt)
	assert.Equal(t, now.Format(time.RFC3339), retrieved.StartedAt.Format(time.RFC3339))
}

func TestDeployment_WithStoppedAt(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	// Create published template and deployment
	template := createPublishedTemplate(t, store)
	deployment := createTestDeployment(t, store, template)

	// Transition through states to stopped
	deployment.NodeID = "node-123"
	err := deployment.Transition(domain.StatusScheduled)
	require.NoError(t, err)
	err = deployment.Transition(domain.StatusStarting)
	require.NoError(t, err)
	err = deployment.Transition(domain.StatusRunning)
	require.NoError(t, err)
	err = deployment.Transition(domain.StatusStopping)
	require.NoError(t, err)
	err = deployment.Transition(domain.StatusStopped)
	require.NoError(t, err)

	now := time.Now().UTC().Truncate(time.Second)
	startedAt := now.Add(-1 * time.Hour)
	deployment.StartedAt = &startedAt
	deployment.StoppedAt = &now
	deployment.UpdatedAt = now

	err = store.UpdateDeployment(ctx, deployment)
	require.NoError(t, err)

	// Retrieve and verify both timestamps
	retrieved, err := store.GetDeployment(ctx, deployment.ID)
	require.NoError(t, err)
	require.NotNil(t, retrieved.StartedAt)
	require.NotNil(t, retrieved.StoppedAt)
	assert.Equal(t, startedAt.Format(time.RFC3339), retrieved.StartedAt.Format(time.RFC3339))
	assert.Equal(t, now.Format(time.RFC3339), retrieved.StoppedAt.Format(time.RFC3339))
}

func TestDeployment_CreateWithStartedAt(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	// Create published template first
	template := createPublishedTemplate(t, store)

	// Create deployment with StartedAt already set (edge case)
	deployment, err := domain.NewDeployment(*template, "customer-123", nil)
	require.NoError(t, err)
	now := time.Now().UTC().Truncate(time.Second)
	deployment.StartedAt = &now

	err = store.CreateDeployment(ctx, deployment)
	require.NoError(t, err)

	retrieved, err := store.GetDeployment(ctx, deployment.ID)
	require.NoError(t, err)
	require.NotNil(t, retrieved.StartedAt)
	assert.Equal(t, now.Format(time.RFC3339), retrieved.StartedAt.Format(time.RFC3339))
}

// =============================================================================
// Corrupted JSON Data Tests (for unmarshal error paths)
// =============================================================================

func TestTemplate_CorruptedVariablesJSON(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	// Create a valid template first
	template := createTestTemplate(t, store)

	// Directly corrupt the variables JSON in the database
	_, err := store.(*SQLiteStore).db.ExecContext(ctx,
		`UPDATE templates SET variables = ? WHERE id = ?`,
		`{"invalid json`, template.ID)
	require.NoError(t, err)

	// Try to retrieve - should fail with parse error
	_, err = store.GetTemplate(ctx, template.ID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidData))
}

func TestTemplate_CorruptedTagsJSON(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	// Create a valid template first
	template := createTestTemplate(t, store)

	// Directly corrupt the tags JSON in the database
	_, err := store.(*SQLiteStore).db.ExecContext(ctx,
		`UPDATE templates SET tags = ? WHERE id = ?`,
		`[invalid json`, template.ID)
	require.NoError(t, err)

	// Try to retrieve - should fail with parse error
	_, err = store.GetTemplate(ctx, template.ID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidData))
}

func TestDeployment_CorruptedVariablesJSON(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	// Create published template and deployment
	template := createPublishedTemplate(t, store)
	deployment := createTestDeployment(t, store, template)

	// Corrupt the variables JSON
	_, err := store.(*SQLiteStore).db.ExecContext(ctx,
		`UPDATE deployments SET variables = ? WHERE id = ?`,
		`{invalid`, deployment.ID)
	require.NoError(t, err)

	// Try to retrieve - should fail
	_, err = store.GetDeployment(ctx, deployment.ID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidData))
}

func TestDeployment_CorruptedDomainsJSON(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	// Create published template and deployment
	template := createPublishedTemplate(t, store)
	deployment := createTestDeployment(t, store, template)

	// Corrupt the domains JSON
	_, err := store.(*SQLiteStore).db.ExecContext(ctx,
		`UPDATE deployments SET domains = ? WHERE id = ?`,
		`[{broken`, deployment.ID)
	require.NoError(t, err)

	// Try to retrieve - should fail
	_, err = store.GetDeployment(ctx, deployment.ID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidData))
}

func TestDeployment_CorruptedContainersJSON(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	// Create published template and deployment
	template := createPublishedTemplate(t, store)
	deployment := createTestDeployment(t, store, template)

	// Corrupt the containers JSON
	_, err := store.(*SQLiteStore).db.ExecContext(ctx,
		`UPDATE deployments SET containers = ? WHERE id = ?`,
		`[{invalid`, deployment.ID)
	require.NoError(t, err)

	// Try to retrieve - should fail
	_, err = store.GetDeployment(ctx, deployment.ID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidData))
}

// =============================================================================
// ListTemplates with corrupted data
// =============================================================================

func TestListTemplates_CorruptedVariablesJSON(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	// Create a valid template first
	template := createTestTemplate(t, store)

	// Directly corrupt the variables JSON
	_, err := store.(*SQLiteStore).db.ExecContext(ctx,
		`UPDATE templates SET variables = ? WHERE id = ?`,
		`{"broken`, template.ID)
	require.NoError(t, err)

	// ListTemplates should fail when converting corrupted row
	_, err = store.ListTemplates(ctx, ListOptions{})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidData))
}

func TestListDeployments_CorruptedVariablesJSON(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	// Create published template and deployment
	template := createPublishedTemplate(t, store)
	deployment := createTestDeployment(t, store, template)

	// Corrupt the variables JSON
	_, err := store.(*SQLiteStore).db.ExecContext(ctx,
		`UPDATE deployments SET variables = ? WHERE id = ?`,
		`{corrupt`, deployment.ID)
	require.NoError(t, err)

	// ListDeployments should fail
	_, err = store.ListDeployments(ctx, ListOptions{})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidData))
}

func TestListDeploymentsByTemplate_CorruptedData(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	// Create published template and deployment
	template := createPublishedTemplate(t, store)
	deployment := createTestDeployment(t, store, template)

	// Corrupt the domains JSON
	_, err := store.(*SQLiteStore).db.ExecContext(ctx,
		`UPDATE deployments SET domains = ? WHERE id = ?`,
		`{broken`, deployment.ID)
	require.NoError(t, err)

	// ListDeploymentsByTemplate should fail
	_, err = store.ListDeploymentsByTemplate(ctx, template.ID, ListOptions{})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidData))
}

func TestListDeploymentsByCustomer_CorruptedData(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	// Create published template and deployment
	template := createPublishedTemplate(t, store)
	deployment := createTestDeployment(t, store, template)

	// Corrupt the containers JSON
	_, err := store.(*SQLiteStore).db.ExecContext(ctx,
		`UPDATE deployments SET containers = ? WHERE id = ?`,
		`{broken`, deployment.ID)
	require.NoError(t, err)

	// ListDeploymentsByCustomer should fail
	_, err = store.ListDeploymentsByCustomer(ctx, deployment.CustomerID, ListOptions{})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidData))
}

// =============================================================================
// GetTemplateBySlug with corrupted data
// =============================================================================

func TestGetTemplateBySlug_CorruptedTags(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	// Create a valid template first
	template := createTestTemplate(t, store)

	// Corrupt the tags JSON
	_, err := store.(*SQLiteStore).db.ExecContext(ctx,
		`UPDATE templates SET tags = ? WHERE id = ?`,
		`{notarray`, template.ID)
	require.NoError(t, err)

	// GetTemplateBySlug should fail
	_, err = store.GetTemplateBySlug(ctx, template.Slug)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidData))
}

// =============================================================================
// CreateDeployment with StoppedAt Coverage
// =============================================================================

func TestDeployment_CreateWithStoppedAt(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	// Create published template
	template := createPublishedTemplate(t, store)

	// Create deployment with both StartedAt and StoppedAt set
	deployment, err := domain.NewDeployment(*template, "customer-123", nil)
	require.NoError(t, err)
	now := time.Now().UTC().Truncate(time.Second)
	startedAt := now.Add(-1 * time.Hour)
	deployment.StartedAt = &startedAt
	deployment.StoppedAt = &now

	err = store.CreateDeployment(ctx, deployment)
	require.NoError(t, err)

	retrieved, err := store.GetDeployment(ctx, deployment.ID)
	require.NoError(t, err)
	require.NotNil(t, retrieved.StartedAt)
	require.NotNil(t, retrieved.StoppedAt)
	assert.Equal(t, startedAt.Format(time.RFC3339), retrieved.StartedAt.Format(time.RFC3339))
	assert.Equal(t, now.Format(time.RFC3339), retrieved.StoppedAt.Format(time.RFC3339))
}

// =============================================================================
// Context Cancellation Tests (for DB error paths)
// =============================================================================

func TestGetTemplate_ContextCancelled(t *testing.T) {
	store := setupTestStore(t)

	// Create a template first
	template := createTestTemplate(t, store)

	// Cancel context immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Should fail with context error
	_, err := store.GetTemplate(ctx, template.ID)
	require.Error(t, err)
}

func TestGetDeployment_ContextCancelled(t *testing.T) {
	store := setupTestStore(t)

	template := createPublishedTemplate(t, store)
	deployment := createTestDeployment(t, store, template)

	// Cancel context immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Should fail with context error
	_, err := store.GetDeployment(ctx, deployment.ID)
	require.Error(t, err)
}

func TestGetTemplateBySlug_ContextCancelled(t *testing.T) {
	store := setupTestStore(t)

	// Create a template first
	template := createTestTemplate(t, store)

	// Cancel context immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Should fail with context error
	_, err := store.GetTemplateBySlug(ctx, template.Slug)
	require.Error(t, err)
}

func TestDeleteTemplate_ContextCancelled(t *testing.T) {
	store := setupTestStore(t)

	// Create a template first
	template := createTestTemplate(t, store)

	// Cancel context immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Should fail with context error
	err := store.DeleteTemplate(ctx, template.ID)
	require.Error(t, err)
}

func TestDeleteDeployment_ContextCancelled(t *testing.T) {
	store := setupTestStore(t)

	template := createPublishedTemplate(t, store)
	deployment := createTestDeployment(t, store, template)

	// Cancel context immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Should fail with context error
	err := store.DeleteDeployment(ctx, deployment.ID)
	require.Error(t, err)
}

func TestListTemplates_ContextCancelled(t *testing.T) {
	store := setupTestStore(t)

	// Cancel context immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Should fail with context error
	_, err := store.ListTemplates(ctx, ListOptions{})
	require.Error(t, err)
}

func TestListDeployments_ContextCancelled(t *testing.T) {
	store := setupTestStore(t)

	// Cancel context immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Should fail with context error
	_, err := store.ListDeployments(ctx, ListOptions{})
	require.Error(t, err)
}

func TestListDeploymentsByTemplate_ContextCancelled(t *testing.T) {
	store := setupTestStore(t)

	// Cancel context immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Should fail with context error
	_, err := store.ListDeploymentsByTemplate(ctx, "template-123", ListOptions{})
	require.Error(t, err)
}

func TestListDeploymentsByCustomer_ContextCancelled(t *testing.T) {
	store := setupTestStore(t)

	// Cancel context immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Should fail with context error
	_, err := store.ListDeploymentsByCustomer(ctx, "customer-123", ListOptions{})
	require.Error(t, err)
}

func TestCreateTemplate_ContextCancelled(t *testing.T) {
	store := setupTestStore(t)

	template, err := domain.NewTemplate("Test", "1.0.0", "services:\n  web:\n    image: nginx")
	require.NoError(t, err)
	template.CreatorID = "creator-123"

	// Cancel context immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Should fail with context error
	err = store.CreateTemplate(ctx, template)
	require.Error(t, err)
}

func TestUpdateTemplate_ContextCancelled(t *testing.T) {
	store := setupTestStore(t)
	template := createTestTemplate(t, store)

	template.Description = "Updated"

	// Cancel context immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Should fail with context error
	err := store.UpdateTemplate(ctx, template)
	require.Error(t, err)
}

func TestCreateDeployment_ContextCancelled(t *testing.T) {
	store := setupTestStore(t)
	template := createPublishedTemplate(t, store)

	deployment, err := domain.NewDeployment(*template, "customer-123", nil)
	require.NoError(t, err)

	// Cancel context immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Should fail with context error
	err = store.CreateDeployment(ctx, deployment)
	require.Error(t, err)
}

func TestUpdateDeployment_ContextCancelled(t *testing.T) {
	store := setupTestStore(t)
	template := createPublishedTemplate(t, store)
	deployment := createTestDeployment(t, store, template)

	deployment.Name = "Updated"

	// Cancel context immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Should fail with context error
	err := store.UpdateDeployment(ctx, deployment)
	require.Error(t, err)
}

func TestWithTx_ContextCancelled(t *testing.T) {
	store := setupTestStore(t)

	// Cancel context immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Should fail when starting transaction
	err := store.WithTx(ctx, func(tx Store) error {
		return nil
	})
	require.Error(t, err)
}

// Note: The following paths are not covered because they are impossible to trigger
// with a real SQLite database without mocking:
// - NewSQLiteStore: db.Ping failure (in-memory DB always succeeds)
// - runMigrations: source/migrator creation errors (embedded files are valid)
// - WithTx: tx.Rollback failure after fn error (rollback rarely fails)
// - WithTx: tx.Commit failure (requires unusual DB state)
// - JSON marshal errors for valid Go types (map, slice) never fail
// These are defensive error paths that protect against edge cases.

// =============================================================================
// Node Tests
// =============================================================================

func createTestNode(t *testing.T, store Store, creatorID string) *domain.Node {
	t.Helper()
	// Generate unique name using timestamp
	uniqueName := "test-node-" + time.Now().Format("150405.000000000")
	node, err := domain.NewNode(creatorID, uniqueName, "192.168.1.100", "root", 22, []string{"standard"})
	require.NoError(t, err)
	node.Capabilities = []string{"standard", "gpu"}
	node.Capacity = domain.NodeCapacity{
		CPUCores:     8,
		MemoryMB:     16384,
		DiskMB:       100000,
		CPUUsed:      2,
		MemoryUsedMB: 4096,
		DiskUsedMB:   20000,
	}
	err = store.CreateNode(context.Background(), node)
	require.NoError(t, err)
	return node
}

func TestCreateNode_Success(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	node, err := domain.NewNode("creator-123", "my-node", "10.0.0.1", "ubuntu", 22, []string{"standard"})
	require.NoError(t, err)
	node.Capabilities = []string{"standard", "ssd"}

	err = store.CreateNode(ctx, node)
	require.NoError(t, err)

	// Verify it was stored
	retrieved, err := store.GetNode(ctx, node.ID)
	require.NoError(t, err)
	assert.Equal(t, node.ID, retrieved.ID)
	assert.Equal(t, "my-node", retrieved.Name)
	assert.Equal(t, "creator-123", retrieved.CreatorID)
	assert.Equal(t, "10.0.0.1", retrieved.SSHHost)
	assert.Equal(t, 22, retrieved.SSHPort)
	assert.Equal(t, "ubuntu", retrieved.SSHUser)
	assert.Equal(t, []string{"standard", "ssd"}, retrieved.Capabilities)
	assert.Equal(t, domain.NodeStatusOffline, retrieved.Status)
}

func TestCreateNode_DuplicateID(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	node1, _ := domain.NewNode("creator-123", "node-1", "10.0.0.1", "ubuntu", 22, []string{"standard"})
	err := store.CreateNode(ctx, node1)
	require.NoError(t, err)

	// Try to create with same ID
	node2, _ := domain.NewNode("creator-123", "node-2", "10.0.0.2", "ubuntu", 22, []string{"standard"})
	node2.ID = node1.ID
	err = store.CreateNode(ctx, node2)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrDuplicateID))
}

func TestCreateNode_DuplicateNameSameCreator(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	node1, _ := domain.NewNode("creator-123", "my-node", "10.0.0.1", "ubuntu", 22, []string{"standard"})
	err := store.CreateNode(ctx, node1)
	require.NoError(t, err)

	// Try to create with same name for same creator
	node2, _ := domain.NewNode("creator-123", "my-node", "10.0.0.2", "ubuntu", 22, []string{"standard"})
	err = store.CreateNode(ctx, node2)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrDuplicateKey))
}

func TestCreateNode_SameNameDifferentCreator(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	node1, _ := domain.NewNode("creator-123", "my-node", "10.0.0.1", "ubuntu", 22, []string{"standard"})
	err := store.CreateNode(ctx, node1)
	require.NoError(t, err)

	// Different creator should be able to use same name
	node2, _ := domain.NewNode("creator-456", "my-node", "10.0.0.2", "ubuntu", 22, []string{"standard"})
	err = store.CreateNode(ctx, node2)
	require.NoError(t, err)
}

func TestGetNode_NotFound(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	_, err := store.GetNode(ctx, "nonexistent-id")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrNotFound))
}

func TestUpdateNode_Success(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()
	node := createTestNode(t, store, "creator-123")

	// Update node
	node.Status = domain.NodeStatusOnline
	node.Capabilities = []string{"standard", "gpu", "high-memory"}
	node.Capacity.CPUUsed = 4
	node.Capacity.MemoryUsedMB = 8192

	err := store.UpdateNode(ctx, node)
	require.NoError(t, err)

	// Verify update
	retrieved, err := store.GetNode(ctx, node.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.NodeStatusOnline, retrieved.Status)
	assert.Equal(t, []string{"standard", "gpu", "high-memory"}, retrieved.Capabilities)
	assert.Equal(t, float64(4), retrieved.Capacity.CPUUsed)
	assert.Equal(t, int64(8192), retrieved.Capacity.MemoryUsedMB)
}

func TestUpdateNode_NotFound(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	node, _ := domain.NewNode("creator-123", "ghost-node", "10.0.0.1", "ubuntu", 22, []string{"standard"})
	err := store.UpdateNode(ctx, node)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrNotFound))
}

func TestDeleteNode_Success(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()
	node := createTestNode(t, store, "creator-123")

	err := store.DeleteNode(ctx, node.ID)
	require.NoError(t, err)

	// Verify deleted
	_, err = store.GetNode(ctx, node.ID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrNotFound))
}

func TestDeleteNode_NotFound(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	err := store.DeleteNode(ctx, "nonexistent-id")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrNotFound))
}

func TestListNodesByCreator_Success(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	// Create nodes for different creators
	createTestNode(t, store, "creator-a")
	createTestNode(t, store, "creator-a")
	createTestNode(t, store, "creator-b")

	// List nodes for creator-a
	nodes, err := store.ListNodesByCreator(ctx, "creator-a", DefaultListOptions())
	require.NoError(t, err)
	assert.Len(t, nodes, 2)
	for _, n := range nodes {
		assert.Equal(t, "creator-a", n.CreatorID)
	}

	// List nodes for creator-b
	nodes, err = store.ListNodesByCreator(ctx, "creator-b", DefaultListOptions())
	require.NoError(t, err)
	assert.Len(t, nodes, 1)
}

func TestListOnlineNodes_Success(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	// Create nodes with different statuses
	node1 := createTestNode(t, store, "creator-a")
	node1.Status = domain.NodeStatusOnline
	store.UpdateNode(ctx, node1)

	node2 := createTestNode(t, store, "creator-b")
	node2.Status = domain.NodeStatusOnline
	store.UpdateNode(ctx, node2)

	createTestNode(t, store, "creator-c") // offline by default

	// List online nodes
	nodes, err := store.ListOnlineNodes(ctx)
	require.NoError(t, err)
	assert.Len(t, nodes, 2)
	for _, n := range nodes {
		assert.Equal(t, domain.NodeStatusOnline, n.Status)
	}
}

func TestListCheckableNodes_Success(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	// Create nodes with different statuses
	node1 := createTestNode(t, store, "creator-a")
	node1.Status = domain.NodeStatusOnline
	store.UpdateNode(ctx, node1)

	node2 := createTestNode(t, store, "creator-b")
	node2.Status = domain.NodeStatusOffline
	store.UpdateNode(ctx, node2)

	node3 := createTestNode(t, store, "creator-c")
	node3.Status = domain.NodeStatusMaintenance
	store.UpdateNode(ctx, node3)

	// List checkable nodes (should exclude maintenance)
	nodes, err := store.ListCheckableNodes(ctx)
	require.NoError(t, err)
	assert.Len(t, nodes, 2)

	// Verify maintenance node is excluded
	for _, n := range nodes {
		assert.NotEqual(t, domain.NodeStatusMaintenance, n.Status)
	}
}

// =============================================================================
// SSH Key Tests
// =============================================================================

func createTestSSHKey(t *testing.T, store Store, creatorID, name string) *domain.SSHKey {
	t.Helper()
	// Use UUID-like unique ID to avoid collisions
	uniqueID := "key-" + creatorID + "-" + name + "-" + time.Now().Format("150405.000000000")
	key := &domain.SSHKey{
		ID:                  uniqueID,
		CreatorID:           creatorID,
		Name:                name,
		PrivateKeyEncrypted: []byte("encrypted-key-data"),
		Fingerprint:         "SHA256:abc123xyz",
		CreatedAt:           time.Now(),
	}
	err := store.CreateSSHKey(context.Background(), key)
	require.NoError(t, err)
	return key
}

func TestCreateSSHKey_Success(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	key := &domain.SSHKey{
		ID:                  "key-123",
		CreatorID:           "creator-456",
		Name:                "my-ssh-key",
		PrivateKeyEncrypted: []byte("encrypted-private-key"),
		Fingerprint:         "SHA256:fingerprint123",
		CreatedAt:           time.Now(),
	}

	err := store.CreateSSHKey(ctx, key)
	require.NoError(t, err)

	// Verify retrieval
	retrieved, err := store.GetSSHKey(ctx, key.ID)
	require.NoError(t, err)
	assert.Equal(t, key.ID, retrieved.ID)
	assert.Equal(t, "my-ssh-key", retrieved.Name)
	assert.Equal(t, "creator-456", retrieved.CreatorID)
	assert.Equal(t, "SHA256:fingerprint123", retrieved.Fingerprint)
	assert.Equal(t, []byte("encrypted-private-key"), retrieved.PrivateKeyEncrypted)
}

func TestCreateSSHKey_DuplicateNameSameCreator(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	createTestSSHKey(t, store, "creator-123", "my-key")

	// Try to create with same name for same creator
	key2 := &domain.SSHKey{
		ID:                  "key-different",
		CreatorID:           "creator-123",
		Name:                "my-key",
		PrivateKeyEncrypted: []byte("data"),
		Fingerprint:         "SHA256:xyz",
		CreatedAt:           time.Now(),
	}
	err := store.CreateSSHKey(ctx, key2)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrDuplicateKey))
}

func TestGetSSHKey_NotFound(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	_, err := store.GetSSHKey(ctx, "nonexistent-id")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrNotFound))
}

func TestDeleteSSHKey_Success(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()
	key := createTestSSHKey(t, store, "creator-123", "my-key")

	err := store.DeleteSSHKey(ctx, key.ID)
	require.NoError(t, err)

	// Verify deleted
	_, err = store.GetSSHKey(ctx, key.ID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrNotFound))
}

func TestDeleteSSHKey_NotFound(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	err := store.DeleteSSHKey(ctx, "nonexistent-id")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrNotFound))
}

func TestListSSHKeysByCreator_Success(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	// Create keys for different creators
	createTestSSHKey(t, store, "creator-a", "key1")
	createTestSSHKey(t, store, "creator-a", "key2")
	createTestSSHKey(t, store, "creator-b", "key1")

	// List keys for creator-a
	keys, err := store.ListSSHKeysByCreator(ctx, "creator-a", DefaultListOptions())
	require.NoError(t, err)
	assert.Len(t, keys, 2)
	for _, k := range keys {
		assert.Equal(t, "creator-a", k.CreatorID)
	}
}

func TestNode_WithSSHKey(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	// Create SSH key first
	key := createTestSSHKey(t, store, "creator-123", "deploy-key")

	// Create node with SSH key reference
	node, _ := domain.NewNode("creator-123", "my-node", "10.0.0.1", "ubuntu", 22, []string{"standard"})
	node.SSHKeyID = key.ID
	err := store.CreateNode(ctx, node)
	require.NoError(t, err)

	// Retrieve and verify
	retrieved, err := store.GetNode(ctx, node.ID)
	require.NoError(t, err)
	assert.Equal(t, key.ID, retrieved.SSHKeyID)
}

func TestNode_CapabilitiesSerialization(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	node, _ := domain.NewNode("creator-123", "my-node", "10.0.0.1", "ubuntu", 22, []string{"standard"})
	node.Capabilities = []string{"gpu", "high-memory", "ssd", "standard"}

	err := store.CreateNode(ctx, node)
	require.NoError(t, err)

	retrieved, err := store.GetNode(ctx, node.ID)
	require.NoError(t, err)
	assert.Equal(t, []string{"gpu", "high-memory", "ssd", "standard"}, retrieved.Capabilities)
}

func TestNode_LastHealthCheck(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	node := createTestNode(t, store, "creator-123")

	// Set last health check
	now := time.Now().Truncate(time.Second)
	node.LastHealthCheck = &now
	node.Status = domain.NodeStatusOnline

	err := store.UpdateNode(ctx, node)
	require.NoError(t, err)

	retrieved, err := store.GetNode(ctx, node.ID)
	require.NoError(t, err)
	require.NotNil(t, retrieved.LastHealthCheck)
	assert.Equal(t, now.UTC().Truncate(time.Second), retrieved.LastHealthCheck.UTC().Truncate(time.Second))
}
