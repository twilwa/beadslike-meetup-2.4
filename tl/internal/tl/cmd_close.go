// ABOUTME: Close and reopen commands — transition tasks to closed/open status.
// ABOUTME: Implements `tl close <id>` and `tl reopen <id>` with transition validation.

package tl

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	closeCmd.Flags().String("reason", "", "Reason for closing")

	closeCmd.RunE = runClose
	reopenCmd.RunE = runReopen
}

func runClose(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: tl close <id>")
	}
	id := args[0]

	dir, err := tlDir(GlobalOptions{Dir: tlDirFlag})
	if err != nil {
		return err
	}

	reason, _ := cmd.Flags().GetString("reason")

	var updatedIssue Issue
	err = mutate(dir, func(g *Graph) ([]Event, error) {
		issue, ok := g.Tasks[id]
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrNotFound, id)
		}

		if err := validateTransition(issue.Status, StatusClosed); err != nil {
			return nil, err
		}

		evt, err := newEvent(EventClose, id, CloseEventData{Reason: reason})
		if err != nil {
			return nil, err
		}

		// Apply close state to in-memory issue for post-mutate capture
		issue.Status = StatusClosed
		issue.CloseReason = reason
		closedAt := evt.Timestamp
		issue.ClosedAt = &closedAt
		issue.UpdatedAt = evt.Timestamp
		updatedIssue = *issue

		return []Event{evt}, nil
	})
	if err != nil {
		return err
	}

	if jsonOutput {
		out, err := json.MarshalIndent(updatedIssue, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(out))
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "Closed %s\n", id)
	}

	return nil
}

func runReopen(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: tl reopen <id>")
	}
	id := args[0]

	dir, err := tlDir(GlobalOptions{Dir: tlDirFlag})
	if err != nil {
		return err
	}

	var updatedIssue Issue
	err = mutate(dir, func(g *Graph) ([]Event, error) {
		issue, ok := g.Tasks[id]
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrNotFound, id)
		}

		if err := validateTransition(issue.Status, StatusOpen); err != nil {
			return nil, err
		}

		evt, err := newEvent(EventReopen, id, ReopenEventData{})
		if err != nil {
			return nil, err
		}

		// Apply reopen state to in-memory issue for post-mutate capture
		issue.Status = StatusOpen
		issue.ClosedAt = nil
		issue.CloseReason = ""
		issue.Assignee = ""
		issue.UpdatedAt = evt.Timestamp
		updatedIssue = *issue

		return []Event{evt}, nil
	})
	if err != nil {
		return err
	}

	if jsonOutput {
		out, err := json.MarshalIndent(updatedIssue, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(out))
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "Reopened %s\n", id)
	}

	return nil
}
