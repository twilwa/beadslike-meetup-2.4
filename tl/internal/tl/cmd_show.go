// ABOUTME: Show command — displays full details for a single task by ID.
// ABOUTME: Implements `tl show <id>` to display issue fields and dependencies.

package tl

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	showCmd.RunE = runShow
}

func runShow(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("requires an issue ID argument")
	}
	id := args[0]

	opts := GlobalOptions{JSON: jsonOutput, Dir: tlDirFlag}
	dir, err := tlDir(opts)
	if err != nil {
		return err
	}

	graph, err := loadGraph(dir)
	if err != nil {
		return err
	}

	issue, ok := graph.Tasks[id]
	if !ok {
		return fmt.Errorf("issue %q: %w", id, ErrNotFound)
	}

	if opts.JSON {
		return printShowJSON(cmd, issue)
	}
	return printShowText(cmd, issue)
}

func printShowJSON(cmd *cobra.Command, issue *Issue) error {
	data, err := json.Marshal(issue)
	if err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), string(data))
	return nil
}

func printShowText(cmd *cobra.Command, issue *Issue) error {
	w := cmd.OutOrStdout()
	fmt.Fprintf(w, "id:           %s\n", issue.ID)
	fmt.Fprintf(w, "title:        %s\n", issue.Title)
	fmt.Fprintf(w, "status:       %s\n", string(issue.Status))
	fmt.Fprintf(w, "priority:     %d\n", issue.Priority)
	fmt.Fprintf(w, "type:         %s\n", string(issue.IssueType))
	fmt.Fprintf(w, "assignee:     %s\n", issue.Assignee)
	fmt.Fprintf(w, "description:  %s\n", issue.Description)
	fmt.Fprintf(w, "created_at:   %s\n", issue.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))
	fmt.Fprintf(w, "updated_at:   %s\n", issue.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"))

	var depIDs []string
	for _, dep := range issue.Dependencies {
		depIDs = append(depIDs, dep.DependsOnID)
	}
	fmt.Fprintf(w, "dependencies: %s\n", strings.Join(depIDs, " "))
	return nil
}
