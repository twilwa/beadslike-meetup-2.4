// ABOUTME: Dependency management commands for adding and removing task dependencies.
// ABOUTME: Implements `tl dep add` and `tl dep remove` with cycle detection.

package tl

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

var depType string

func init() {
	depAddCmd.Args = cobra.ExactArgs(2)
	depRemoveCmd.Args = cobra.ExactArgs(2)
	depAddCmd.Flags().StringVar(&depType, "type", string(DepBlocks), "Dependency type")

	depAddCmd.RunE = runDepAdd
	depRemoveCmd.RunE = runDepRemove
}

func runDepAdd(cmd *cobra.Command, args []string) error {
	issueID := args[0]
	dependsOnID := args[1]

	opts := GlobalOptions{JSON: jsonOutput, Dir: tlDirFlag}
	dir, err := tlDir(opts)
	if err != nil {
		return err
	}

	err = mutate(dir, func(graph *Graph) ([]Event, error) {
		if issueID == dependsOnID {
			return nil, errors.New("cannot depend on self")
		}
		if _, ok := graph.Tasks[issueID]; !ok {
			return nil, fmt.Errorf("issue %q: %w", issueID, ErrNotFound)
		}
		if hasCycle(graph, issueID, dependsOnID) {
			return nil, ErrCycle
		}

		evt, err := newEvent(EventDepAdd, issueID, DepAddEventData{
			DependsOnID: dependsOnID,
			DepType:     depType,
		})
		if err != nil {
			return nil, err
		}
		return []Event{evt}, nil
	})
	if err != nil {
		return err
	}

	graph, err := loadGraph(dir)
	if err != nil {
		return err
	}
	issue, ok := graph.Tasks[issueID]
	if !ok {
		return fmt.Errorf("issue %q: %w", issueID, ErrNotFound)
	}

	if opts.JSON {
		return printIssueJSON(cmd, issue)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Added dependency: %s depends on %s\n", issueID, dependsOnID)
	return nil
}

func runDepRemove(cmd *cobra.Command, args []string) error {
	issueID := args[0]
	dependsOnID := args[1]

	opts := GlobalOptions{JSON: jsonOutput, Dir: tlDirFlag}
	dir, err := tlDir(opts)
	if err != nil {
		return err
	}

	err = mutate(dir, func(graph *Graph) ([]Event, error) {
		if _, ok := graph.Tasks[issueID]; !ok {
			return nil, fmt.Errorf("issue %q: %w", issueID, ErrNotFound)
		}

		found := false
		for _, depID := range graph.Deps[issueID] {
			if depID == dependsOnID {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("dependency %q -> %q: %w", issueID, dependsOnID, ErrNotFound)
		}

		evt, err := newEvent(EventDepRemove, issueID, DepRemoveEventData{DependsOnID: dependsOnID})
		if err != nil {
			return nil, err
		}
		return []Event{evt}, nil
	})
	if err != nil {
		return err
	}

	graph, err := loadGraph(dir)
	if err != nil {
		return err
	}
	issue, ok := graph.Tasks[issueID]
	if !ok {
		return fmt.Errorf("issue %q: %w", issueID, ErrNotFound)
	}

	if opts.JSON {
		return printIssueJSON(cmd, issue)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Removed dependency: %s no longer depends on %s\n", issueID, dependsOnID)
	return nil
}

func printIssueJSON(cmd *cobra.Command, issue *Issue) error {
	data, err := json.Marshal(issue)
	if err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), string(data))
	return nil
}
