// ABOUTME: Blocked command implementation for listing currently unworkable tasks.
// ABOUTME: Shows each blocked issue and direct unresolved dependency IDs that keep it blocked.

package tl

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

type blockedIssue struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Status   Status   `json:"status"`
	Blockers []string `json:"blockers"`
}

func init() {
	blockedCmd.RunE = runBlocked
}

func runBlocked(cmd *cobra.Command, args []string) error {
	dir, err := tlDir(GlobalOptions{JSON: jsonOutput, Dir: tlDirFlag})
	if err != nil {
		return err
	}

	graph, err := loadGraph(dir)
	if err != nil {
		return err
	}

	blockedSet := computeBlockedSet(graph)
	rows := collectBlockedIssues(graph, blockedSet)

	if jsonOutput {
		data, err := json.Marshal(rows)
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(data))
		return nil
	}

	for _, row := range rows {
		fmt.Fprintf(
			cmd.OutOrStdout(),
			"%s [%s] %s (blocked by: %s)\n",
			row.ID,
			string(row.Status),
			strings.TrimSpace(row.Title),
			strings.Join(row.Blockers, " "),
		)
	}

	return nil
}

func collectBlockedIssues(graph *Graph, blockedSet map[string]bool) []blockedIssue {
	rows := make([]blockedIssue, 0)
	if graph == nil {
		return rows
	}

	issues := make([]*Issue, 0)
	for _, issue := range graph.Tasks {
		if issue.Status == StatusClosed {
			continue
		}
		if !blockedSet[issue.ID] {
			continue
		}
		issues = append(issues, issue)
	}

	sort.Slice(issues, func(i, j int) bool {
		if issues[i].Priority != issues[j].Priority {
			return issues[i].Priority < issues[j].Priority
		}
		return issues[i].CreatedAt.Before(issues[j].CreatedAt)
	})

	for _, issue := range issues {
		rows = append(rows, blockedIssue{
			ID:       issue.ID,
			Title:    issue.Title,
			Status:   issue.Status,
			Blockers: blockerIDsForIssue(issue, graph, blockedSet),
		})
	}

	return rows
}

func blockerIDsForIssue(issue *Issue, graph *Graph, blockedSet map[string]bool) []string {
	ids := make(map[string]struct{})

	for _, dep := range issue.Dependencies {
		if dep == nil || !dep.Type.AffectsReadyWork() {
			continue
		}
		depIssue, ok := graph.Tasks[dep.DependsOnID]
		if !ok {
			continue
		}

		if issueStatusBlocksReady(depIssue.Status) {
			ids[dep.DependsOnID] = struct{}{}
			continue
		}

		if dep.Type == DepParentChild && blockedSet[dep.DependsOnID] {
			ids[dep.DependsOnID] = struct{}{}
		}
	}

	blockers := make([]string, 0, len(ids))
	for id := range ids {
		blockers = append(blockers, id)
	}
	sort.Strings(blockers)
	return blockers
}
