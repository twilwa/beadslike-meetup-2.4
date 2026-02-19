// ABOUTME: Tests cycle detection logic for dependency edges in the in-memory graph.
// ABOUTME: Verifies reachability checks used before writing dep_add events.

package tl

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHasCycle(t *testing.T) {
	graph := &Graph{
		Deps: map[string][]string{
			"tl-a": {"tl-b"},
			"tl-b": {"tl-a"},
		},
	}

	assert.True(t, hasCycle(graph, "tl-a", "tl-b"))
}

func TestHasCycleNoCycle(t *testing.T) {
	graph := &Graph{
		Deps: map[string][]string{
			"tl-a": {"tl-b"},
			"tl-b": {"tl-c"},
		},
	}

	assert.False(t, hasCycle(graph, "tl-a", "tl-c"))
}
