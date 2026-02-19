// ABOUTME: Tests blocked-set and ready-queue selection logic for dependency and defer rules.
// ABOUTME: Verifies transitive parent-child blocking and DeferUntil gating behavior.

package tl

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeBlockedSetMarksBlockingDependency(t *testing.T) {
	now := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	graph := &Graph{
		Tasks: map[string]*Issue{
			"tl-a": {
				ID:        "tl-a",
				Title:     "Blocked task",
				Status:    StatusOpen,
				CreatedAt: now,
				Dependencies: []*Dependency{
					{IssueID: "tl-a", DependsOnID: "tl-b", Type: DepBlocks},
				},
			},
			"tl-b": {
				ID:        "tl-b",
				Title:     "Blocking task",
				Status:    StatusOpen,
				CreatedAt: now,
			},
		},
	}

	blockedSet := computeBlockedSet(graph)
	require.True(t, blockedSet["tl-a"])
	assert.False(t, blockedSet["tl-b"])
}

func TestComputeBlockedSetParentChildTransitive(t *testing.T) {
	now := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	graph := &Graph{
		Tasks: map[string]*Issue{
			"tl-parent": {
				ID:        "tl-parent",
				Title:     "Parent",
				Status:    StatusDeferred,
				CreatedAt: now,
				Dependencies: []*Dependency{
					{IssueID: "tl-parent", DependsOnID: "tl-root", Type: DepParentChild},
				},
			},
			"tl-child": {
				ID:        "tl-child",
				Title:     "Child",
				Status:    StatusOpen,
				CreatedAt: now,
				Dependencies: []*Dependency{
					{IssueID: "tl-child", DependsOnID: "tl-parent", Type: DepParentChild},
				},
			},
			"tl-root": {
				ID:        "tl-root",
				Title:     "Root",
				Status:    StatusOpen,
				CreatedAt: now,
			},
		},
	}

	blockedSet := computeBlockedSet(graph)
	require.True(t, blockedSet["tl-parent"])
	assert.True(t, blockedSet["tl-child"])
}

func TestReadyExcludesFutureDeferredIssue(t *testing.T) {
	now := time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC)
	future := now.Add(2 * time.Hour)
	graph := &Graph{
		Tasks: map[string]*Issue{
			"tl-ready": {
				ID:        "tl-ready",
				Title:     "Ready task",
				Status:    StatusOpen,
				Priority:  1,
				CreatedAt: now,
			},
			"tl-deferred": {
				ID:         "tl-deferred",
				Title:      "Deferred task",
				Status:     StatusOpen,
				Priority:   0,
				CreatedAt:  now,
				DeferUntil: &future,
			},
		},
	}

	ready := collectReadyIssues(graph, map[string]bool{}, now)
	require.Len(t, ready, 1)
	assert.Equal(t, "tl-ready", ready[0].ID)
}
