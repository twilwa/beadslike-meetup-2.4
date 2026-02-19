// ABOUTME: List command — displays tasks with optional status/type/assignee/priority filters.
// ABOUTME: Implements `tl list` to query the task graph and output matching issues.

package tl

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

var (
	listStatus   string
	listType     string
	listAssignee string
	listPriority int
	listLimit    int
)

func init() {
	listCmd.Flags().StringVar(&listStatus, "status", "", "Filter by status")
	listCmd.Flags().StringVar(&listType, "type", "", "Filter by issue type")
	listCmd.Flags().StringVar(&listAssignee, "assignee", "", "Filter by assignee")
	listCmd.Flags().IntVar(&listPriority, "priority", -1, "Filter by priority (-1 = no filter)")
	listCmd.Flags().IntVar(&listLimit, "limit", 0, "Maximum number of results (0 = all)")

	listCmd.RunE = runList
}

func runList(cmd *cobra.Command, args []string) error {
	opts := GlobalOptions{JSON: jsonOutput, Dir: tlDirFlag}
	dir, err := tlDir(opts)
	if err != nil {
		return err
	}

	graph, err := loadGraph(dir)
	if err != nil {
		return err
	}

	issues := filterIssues(graph)
	sortIssues(issues)

	if listLimit > 0 && len(issues) > listLimit {
		issues = issues[:listLimit]
	}

	if opts.JSON {
		return printListJSON(cmd, issues)
	}
	return printListText(cmd, issues)
}

func filterIssues(graph *Graph) []*Issue {
	var result []*Issue
	for _, issue := range graph.Tasks {
		if listStatus != "" && string(issue.Status) != listStatus {
			continue
		}
		if listType != "" && string(issue.IssueType) != listType {
			continue
		}
		if listAssignee != "" && issue.Assignee != listAssignee {
			continue
		}
		if listPriority >= 0 && issue.Priority != listPriority {
			continue
		}
		result = append(result, issue)
	}
	return result
}

func sortIssues(issues []*Issue) {
	sort.Slice(issues, func(i, j int) bool {
		if issues[i].Priority != issues[j].Priority {
			return issues[i].Priority < issues[j].Priority
		}
		return issues[i].CreatedAt.Before(issues[j].CreatedAt)
	})
}

func printListJSON(cmd *cobra.Command, issues []*Issue) error {
	if issues == nil {
		issues = []*Issue{}
	}
	data, err := json.Marshal(issues)
	if err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), string(data))
	return nil
}

func printListText(cmd *cobra.Command, issues []*Issue) error {
	w := cmd.OutOrStdout()
	for _, issue := range issues {
		fmt.Fprintf(w, "%s [%s] P%d %s\n",
			issue.ID,
			string(issue.Status),
			issue.Priority,
			strings.TrimSpace(issue.Title))
	}
	return nil
}
