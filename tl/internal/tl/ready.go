// ABOUTME: Blocked set computation and ready queue logic for tl task graph.
// ABOUTME: computeBlockedSet traverses dependency edges to determine which tasks are unworkable.

package tl

import (
	"sort"
	"time"
)

func computeBlockedSet(graph *Graph) map[string]bool {
	blocked := make(map[string]bool)
	if graph == nil {
		return blocked
	}

	changed := true
	for changed {
		changed = false
		for id, issue := range graph.Tasks {
			if blocked[id] {
				continue
			}

			for _, dep := range issue.Dependencies {
				if dep == nil || !dep.Type.AffectsReadyWork() {
					continue
				}

				depIssue, ok := graph.Tasks[dep.DependsOnID]
				if !ok {
					continue
				}

				if issueStatusBlocksReady(depIssue.Status) {
					blocked[id] = true
					changed = true
					break
				}

				if dep.Type == DepParentChild && blocked[dep.DependsOnID] {
					blocked[id] = true
					changed = true
					break
				}
			}
		}
	}

	return blocked
}

func collectReadyIssues(graph *Graph, blockedSet map[string]bool, now time.Time) []*Issue {
	if graph == nil {
		return nil
	}

	ready := make([]*Issue, 0)
	for _, issue := range graph.Tasks {
		if issue.Status != StatusOpen {
			continue
		}
		if blockedSet[issue.ID] {
			continue
		}
		if issue.Pinned {
			continue
		}
		if issue.DeferUntil != nil && !issue.DeferUntil.Before(now) {
			continue
		}
		ready = append(ready, issue)
	}

	sort.Slice(ready, func(i, j int) bool {
		if ready[i].Priority != ready[j].Priority {
			return ready[i].Priority < ready[j].Priority
		}
		return ready[i].CreatedAt.Before(ready[j].CreatedAt)
	})

	return ready
}

func issueStatusBlocksReady(status Status) bool {
	return status == StatusOpen || status == StatusInProgress || status == StatusBlocked
}
