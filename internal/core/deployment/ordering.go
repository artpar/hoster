package deployment

import (
	"github.com/artpar/hoster/internal/core/compose"
)

// =============================================================================
// Service Ordering Functions
// =============================================================================

// TopologicalSort sorts services by their dependencies using Kahn's algorithm.
// Services with no dependencies come first.
//
// The function implements a BFS-based topological sort:
//  1. Build a map of service dependencies (in-degree)
//  2. Start with services that have no dependencies (in-degree = 0)
//  3. Process each service, reducing the in-degree of its dependents
//  4. When a dependent's in-degree reaches 0, add it to the queue
//
// If a cycle exists (which should be caught at parse time), remaining
// services are appended to the result as a fallback.
//
// Example:
//
//	// Services: web → api → db
//	services := []compose.Service{
//	    {Name: "web", DependsOn: []string{"api"}},
//	    {Name: "api", DependsOn: []string{"db"}},
//	    {Name: "db"},
//	}
//	sorted := TopologicalSort(services)
//	// Result: [db, api, web]
func TopologicalSort(services []compose.Service) []compose.Service {
	if len(services) == 0 {
		return services
	}

	// Build dependency graph
	serviceMap := make(map[string]compose.Service)
	inDegree := make(map[string]int)
	dependents := make(map[string][]string)

	for _, svc := range services {
		serviceMap[svc.Name] = svc
		inDegree[svc.Name] = len(svc.DependsOn)
		for _, dep := range svc.DependsOn {
			dependents[dep] = append(dependents[dep], svc.Name)
		}
	}

	// Start with services that have no dependencies
	var queue []string
	for name, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, name)
		}
	}

	// Process queue (BFS)
	var result []compose.Service
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]

		if svc, ok := serviceMap[name]; ok {
			result = append(result, svc)
		}

		// Reduce in-degree for dependents
		for _, dep := range dependents[name] {
			inDegree[dep]--
			if inDegree[dep] == 0 {
				queue = append(queue, dep)
			}
		}
	}

	// If we didn't get all services, there's a cycle (shouldn't happen after parsing)
	// Just append remaining services as fallback
	if len(result) < len(services) {
		for _, svc := range services {
			found := false
			for _, r := range result {
				if r.Name == svc.Name {
					found = true
					break
				}
			}
			if !found {
				result = append(result, svc)
			}
		}
	}

	return result
}
