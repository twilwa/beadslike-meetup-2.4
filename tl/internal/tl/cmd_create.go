// ABOUTME: Create command — creates a new task in the tl task graph.
// ABOUTME: Implements `tl create` with title, type, priority, and description flags.

package tl

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var (
	createTitle       string
	createType        string
	createPriority    int
	createDescription string
)

func init() {
	createCmd.Flags().StringVar(&createTitle, "title", "", "Task title")
	createCmd.Flags().StringVar(&createType, "type", "task", "Issue type (task, bug, feature, chore, epic, decision)")
	createCmd.Flags().IntVar(&createPriority, "priority", 2, "Priority (0=critical, 1=high, 2=medium, 3=low)")
	createCmd.Flags().StringVar(&createDescription, "description", "", "Task description")

	createCmd.RunE = runCreate
}

func runCreate(cmd *cobra.Command, args []string) error {
	// Title from flag or first positional arg
	title := createTitle
	if title == "" && len(args) > 0 {
		title = args[0]
	}
	if title == "" {
		return fmt.Errorf("title is required (use --title or pass as first argument)")
	}

	opts := GlobalOptions{JSON: jsonOutput, Dir: tlDirFlag}
	dir, err := tlDir(opts)
	if err != nil {
		return err
	}

	var created *Issue
	err = mutate(dir, func(_ *Graph) ([]Event, error) {
		id := generateID()
		data := CreateEventData{
			Title:       title,
			Description: createDescription,
			Status:      string(StatusOpen),
			Priority:    createPriority,
			IssueType:   createType,
		}
		evt, err := newEvent(EventCreate, id, data)
		if err != nil {
			return nil, err
		}
		created = &Issue{
			ID:          id,
			Title:       data.Title,
			Description: data.Description,
			Status:      StatusOpen,
			Priority:    data.Priority,
			IssueType:   IssueType(data.IssueType),
			CreatedAt:   evt.Timestamp,
			UpdatedAt:   evt.Timestamp,
		}
		return []Event{evt}, nil
	})
	if err != nil {
		return err
	}

	if opts.JSON {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(created)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Created %s: %s\n", created.ID, created.Title)
	return nil
}
