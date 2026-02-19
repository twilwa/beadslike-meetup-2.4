// ABOUTME: Claim command — atomically transitions an open task to in_progress under flock.
// ABOUTME: Implements `tl claim <id>` ensuring only one agent can claim a task concurrently.

package tl

import (
	"fmt"

	"github.com/spf13/cobra"
)

var claimAgent string

func init() {
	claimCmd.Args = cobra.ExactArgs(1)
	claimCmd.Flags().StringVar(&claimAgent, "agent", resolveActor(), "Agent claiming the task")
	claimCmd.RunE = runClaim
}

func runClaim(cmd *cobra.Command, args []string) error {
	id := args[0]
	agent := claimAgent
	if agent == "" {
		agent = resolveActor()
	}

	opts := GlobalOptions{JSON: jsonOutput, Dir: tlDirFlag}
	dir, err := tlDir(opts)
	if err != nil {
		return err
	}

	var claimed Issue
	err = mutate(dir, func(g *Graph) ([]Event, error) {
		issue, ok := g.Tasks[id]
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrNotFound, id)
		}
		if issue.Status != StatusOpen {
			return nil, fmt.Errorf("task %s is not open (status: %s)", id, issue.Status)
		}

		evt, err := newEvent(EventClaim, id, ClaimEventData{Agent: agent})
		if err != nil {
			return nil, err
		}

		issue.Status = StatusInProgress
		issue.Assignee = agent
		issue.UpdatedAt = evt.Timestamp
		claimed = *issue

		return []Event{evt}, nil
	})
	if err != nil {
		return err
	}

	if opts.JSON {
		return printIssueJSON(cmd, &claimed)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Claimed %s by %s\n", claimed.ID, claimed.Assignee)
	return nil
}
