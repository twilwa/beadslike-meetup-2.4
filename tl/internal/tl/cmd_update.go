// ABOUTME: Update command — modifies task fields including status, title, description, priority, and assignee.
// ABOUTME: Implements `tl update <id>` with selective field updates under write lock.

package tl

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	updateCmd.Flags().String("status", "", "New status")
	updateCmd.Flags().String("title", "", "New title")
	updateCmd.Flags().String("description", "", "New description")
	updateCmd.Flags().Int("priority", -1, "New priority (0-5)")
	updateCmd.Flags().String("assignee", "", "New assignee")
	updateCmd.Flags().String("type", "", "New issue type")

	updateCmd.RunE = runUpdate
}

func runUpdate(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: tl update <id>")
	}
	id := args[0]

	dir, err := tlDir(GlobalOptions{Dir: tlDirFlag})
	if err != nil {
		return err
	}

	// Build the fields map from explicitly-set flags only
	fields := make(map[string]json.RawMessage)

	if cmd.Flags().Changed("status") {
		val, _ := cmd.Flags().GetString("status")
		raw, _ := json.Marshal(val)
		fields["status"] = raw
	}
	if cmd.Flags().Changed("title") {
		val, _ := cmd.Flags().GetString("title")
		raw, _ := json.Marshal(val)
		fields["title"] = raw
	}
	if cmd.Flags().Changed("description") {
		val, _ := cmd.Flags().GetString("description")
		raw, _ := json.Marshal(val)
		fields["description"] = raw
	}
	if cmd.Flags().Changed("priority") {
		val, _ := cmd.Flags().GetInt("priority")
		raw, _ := json.Marshal(val)
		fields["priority"] = raw
	}
	if cmd.Flags().Changed("assignee") {
		val, _ := cmd.Flags().GetString("assignee")
		raw, _ := json.Marshal(val)
		fields["assignee"] = raw
	}
	if cmd.Flags().Changed("type") {
		val, _ := cmd.Flags().GetString("type")
		raw, _ := json.Marshal(val)
		fields["issue_type"] = raw
	}

	if len(fields) == 0 {
		return fmt.Errorf("no fields to update")
	}

	var updatedIssue Issue
	err = mutate(dir, func(g *Graph) ([]Event, error) {
		issue, ok := g.Tasks[id]
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrNotFound, id)
		}

		// Validate status transition if status is being changed
		if raw, ok := fields["status"]; ok {
			var newStatus Status
			if err := json.Unmarshal(raw, &newStatus); err != nil {
				return nil, err
			}
			if err := validateTransition(issue.Status, newStatus); err != nil {
				return nil, err
			}
		}

		evt, err := newEvent(EventUpdate, id, UpdateEventData{Fields: fields})
		if err != nil {
			return nil, err
		}

		// Apply field changes to the in-memory issue for post-mutate capture
		for field, value := range fields {
			switch field {
			case "status":
				var v Status
				_ = json.Unmarshal(value, &v)
				issue.Status = v
			case "title":
				var v string
				_ = json.Unmarshal(value, &v)
				issue.Title = v
			case "description":
				var v string
				_ = json.Unmarshal(value, &v)
				issue.Description = v
			case "priority":
				var v int
				_ = json.Unmarshal(value, &v)
				issue.Priority = v
			case "assignee":
				var v string
				_ = json.Unmarshal(value, &v)
				issue.Assignee = v
			case "issue_type":
				var v IssueType
				_ = json.Unmarshal(value, &v)
				issue.IssueType = v
			}
		}
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
		fmt.Fprintf(cmd.OutOrStdout(), "Updated %s\n", id)
	}

	return nil
}
