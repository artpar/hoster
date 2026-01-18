package deployment

import (
	"testing"

	"github.com/artpar/hoster/internal/core/compose"
	"github.com/stretchr/testify/assert"
)

// =============================================================================
// TopologicalSort Tests
// =============================================================================

func TestTopologicalSort_Empty(t *testing.T) {
	services := []compose.Service{}
	result := TopologicalSort(services)
	assert.Empty(t, result)
}

func TestTopologicalSort_SingleService(t *testing.T) {
	services := []compose.Service{
		{Name: "web"},
	}
	result := TopologicalSort(services)
	assert.Len(t, result, 1)
	assert.Equal(t, "web", result[0].Name)
}

func TestTopologicalSort_NoDependencies(t *testing.T) {
	services := []compose.Service{
		{Name: "web"},
		{Name: "api"},
		{Name: "db"},
	}
	result := TopologicalSort(services)
	assert.Len(t, result, 3)
	// All should be present (order may vary since no dependencies)
	names := make(map[string]bool)
	for _, s := range result {
		names[s.Name] = true
	}
	assert.True(t, names["web"])
	assert.True(t, names["api"])
	assert.True(t, names["db"])
}

func TestTopologicalSort_LinearDependencies(t *testing.T) {
	// web depends on api, api depends on db
	services := []compose.Service{
		{Name: "web", DependsOn: []string{"api"}},
		{Name: "api", DependsOn: []string{"db"}},
		{Name: "db"},
	}
	result := TopologicalSort(services)

	// db must come before api, api before web
	dbIdx, apiIdx, webIdx := -1, -1, -1
	for i, s := range result {
		switch s.Name {
		case "db":
			dbIdx = i
		case "api":
			apiIdx = i
		case "web":
			webIdx = i
		}
	}
	assert.Less(t, dbIdx, apiIdx, "db should come before api")
	assert.Less(t, apiIdx, webIdx, "api should come before web")
}

func TestTopologicalSort_DiamondDependencies(t *testing.T) {
	// web depends on api and cache, both depend on db
	//       web
	//      /   \
	//    api   cache
	//      \   /
	//       db
	services := []compose.Service{
		{Name: "web", DependsOn: []string{"api", "cache"}},
		{Name: "api", DependsOn: []string{"db"}},
		{Name: "cache", DependsOn: []string{"db"}},
		{Name: "db"},
	}
	result := TopologicalSort(services)

	// Find indices
	indices := make(map[string]int)
	for i, s := range result {
		indices[s.Name] = i
	}

	// db must come first
	assert.Equal(t, 0, indices["db"], "db should be first")
	// web must come last
	assert.Equal(t, 3, indices["web"], "web should be last")
	// api and cache must come after db, before web
	assert.Less(t, indices["db"], indices["api"])
	assert.Less(t, indices["db"], indices["cache"])
	assert.Less(t, indices["api"], indices["web"])
	assert.Less(t, indices["cache"], indices["web"])
}

func TestTopologicalSort_MultipleRoots(t *testing.T) {
	// Two independent chains: web→api and worker→db
	services := []compose.Service{
		{Name: "web", DependsOn: []string{"api"}},
		{Name: "api"},
		{Name: "worker", DependsOn: []string{"db"}},
		{Name: "db"},
	}
	result := TopologicalSort(services)

	indices := make(map[string]int)
	for i, s := range result {
		indices[s.Name] = i
	}

	// api before web, db before worker
	assert.Less(t, indices["api"], indices["web"])
	assert.Less(t, indices["db"], indices["worker"])
}

func TestTopologicalSort_CycleFallback(t *testing.T) {
	// Note: cycles should be caught by compose parser
	// This tests the fallback behavior
	services := []compose.Service{
		{Name: "a", DependsOn: []string{"b"}},
		{Name: "b", DependsOn: []string{"a"}},
	}
	result := TopologicalSort(services)
	// Should return all services even with cycle
	assert.Len(t, result, 2)

	names := make(map[string]bool)
	for _, s := range result {
		names[s.Name] = true
	}
	assert.True(t, names["a"])
	assert.True(t, names["b"])
}

func TestTopologicalSort_PartialCycle(t *testing.T) {
	// c has no dependencies, a and b form a cycle
	services := []compose.Service{
		{Name: "a", DependsOn: []string{"b"}},
		{Name: "b", DependsOn: []string{"a"}},
		{Name: "c"},
	}
	result := TopologicalSort(services)
	// Should return all services
	assert.Len(t, result, 3)

	// c should be first (no dependencies)
	assert.Equal(t, "c", result[0].Name)
}

func TestTopologicalSort_DeepChain(t *testing.T) {
	// a → b → c → d → e
	services := []compose.Service{
		{Name: "a", DependsOn: []string{"b"}},
		{Name: "b", DependsOn: []string{"c"}},
		{Name: "c", DependsOn: []string{"d"}},
		{Name: "d", DependsOn: []string{"e"}},
		{Name: "e"},
	}
	result := TopologicalSort(services)

	// Should be in reverse order: e, d, c, b, a
	expected := []string{"e", "d", "c", "b", "a"}
	for i, name := range expected {
		assert.Equal(t, name, result[i].Name)
	}
}

func TestTopologicalSort_PreservesServiceData(t *testing.T) {
	// Ensure all service fields are preserved
	services := []compose.Service{
		{
			Name:        "web",
			Image:       "nginx:latest",
			DependsOn:   []string{"api"},
			Environment: map[string]string{"PORT": "80"},
		},
		{
			Name:  "api",
			Image: "myapp:1.0",
		},
	}
	result := TopologicalSort(services)

	// Find web service in result
	var webService compose.Service
	for _, s := range result {
		if s.Name == "web" {
			webService = s
			break
		}
	}

	assert.Equal(t, "nginx:latest", webService.Image)
	assert.Equal(t, []string{"api"}, webService.DependsOn)
	assert.Equal(t, "80", webService.Environment["PORT"])
}

func TestTopologicalSort_MissingDependency(t *testing.T) {
	// web depends on "api" but api is not in the list
	// This shouldn't happen after parsing, but test graceful handling
	services := []compose.Service{
		{Name: "web", DependsOn: []string{"api"}},
	}
	result := TopologicalSort(services)

	// web should still be returned (it has in-degree 1, never becomes 0)
	// Our fallback appends remaining services
	assert.Len(t, result, 1)
	assert.Equal(t, "web", result[0].Name)
}
