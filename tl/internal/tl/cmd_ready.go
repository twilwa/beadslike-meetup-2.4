// ABOUTME: Ready command implementation for selecting actionable open tasks.
// ABOUTME: Computes blocked/deferred/pinned filters and prints ready queue in text or JSON.

package tl

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

type readyIssue struct {
	ID       string `json:"id"`
	Priority int    `json:"priority"`
	Title    string `json:"title"`
}

func init() {
	readyCmd.RunE = runReady
}

func runReady(cmd *cobra.Command, args []string) error {
	dir, err := tlDir(GlobalOptions{JSON: jsonOutput, Dir: tlDirFlag})
	if err != nil {
		return err
	}

	graph, err := loadGraph(dir)
	if err != nil {
		return err
	}

	blockedSet := computeBlockedSet(graph)
	ready := collectReadyIssues(graph, blockedSet, time.Now())

	if jsonOutput {
		rows := make([]readyIssue, 0, len(ready))
		for _, issue := range ready {
			rows = append(rows, readyIssue{ID: issue.ID, Priority: issue.Priority, Title: issue.Title})
		}
		data, err := json.Marshal(rows)
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(data))
		return nil
	}

	for _, issue := range ready {
		fmt.Fprintf(cmd.OutOrStdout(), "%s P%d %s\n", issue.ID, issue.Priority, strings.TrimSpace(issue.Title))
	}

	return nil
}
