// ABOUTME: Stats command implementation for task status and blocked-set counts.
// ABOUTME: Reports workflow counts in either compact text format or machine-readable JSON.

package tl

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

type statsOutput struct {
	Open       int `json:"open"`
	InProgress int `json:"in_progress"`
	Blocked    int `json:"blocked"`
	Closed     int `json:"closed"`
	Deferred   int `json:"deferred"`
	Total      int `json:"total"`
}

func init() {
	statsCmd.RunE = runStats
}

func runStats(cmd *cobra.Command, args []string) error {
	dir, err := tlDir(GlobalOptions{JSON: jsonOutput, Dir: tlDirFlag})
	if err != nil {
		return err
	}

	graph, err := loadGraph(dir)
	if err != nil {
		return err
	}

	stats := computeStats(graph)

	if jsonOutput {
		data, err := json.Marshal(stats)
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(data))
		return nil
	}

	fmt.Fprintf(
		cmd.OutOrStdout(),
		"Open: %d | In Progress: %d | Blocked: %d | Closed: %d | Total: %d\n",
		stats.Open,
		stats.InProgress,
		stats.Blocked,
		stats.Closed,
		stats.Total,
	)
	return nil
}

func computeStats(graph *Graph) statsOutput {
	out := statsOutput{}
	if graph == nil {
		return out
	}

	for _, issue := range graph.Tasks {
		switch issue.Status {
		case StatusOpen:
			out.Open++
		case StatusInProgress:
			out.InProgress++
		case StatusClosed:
			out.Closed++
		case StatusDeferred:
			out.Deferred++
		}
	}

	out.Blocked = len(computeBlockedSet(graph))
	out.Total = len(graph.Tasks)

	return out
}
