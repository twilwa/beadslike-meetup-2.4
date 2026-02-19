// ABOUTME: Cycle detection for dependency graph using depth-first search.
// ABOUTME: Used by dep add to reject cycles before appending dep_add events.

package tl

func hasCycle(graph *Graph, issueID, dependsOnID string) bool {
	if graph == nil {
		return false
	}

	visited := map[string]bool{}
	stack := []string{dependsOnID}

	for len(stack) > 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if current == issueID {
			return true
		}
		if visited[current] {
			continue
		}
		visited[current] = true

		for _, next := range graph.Deps[current] {
			if !visited[next] {
				stack = append(stack, next)
			}
		}
	}

	return false
}
