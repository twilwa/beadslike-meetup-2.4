// ABOUTME: Export command — writes tl tasks to beads JSONL format for interoperability.
// ABOUTME: Implements `tl export --to <path>` with metadata field promotion and atomic write.

package tl

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"
)

var exportTo string

func init() {
	exportCmd.Flags().StringVar(&exportTo, "to", ".beads/issues.jsonl", "Destination path for JSONL export")
	exportCmd.RunE = runExport
}

func runExport(cmd *cobra.Command, args []string) error {
	opts := GlobalOptions{JSON: jsonOutput, Dir: tlDirFlag}
	dir, err := tlDir(opts)
	if err != nil {
		return err
	}

	graph, err := loadGraph(dir)
	if err != nil {
		return err
	}

	// Collect and sort issues by ID for deterministic output
	issues := make([]*Issue, 0, len(graph.Tasks))
	for _, issue := range graph.Tasks {
		issues = append(issues, issue)
	}
	sort.Slice(issues, func(i, j int) bool {
		return issues[i].ID < issues[j].ID
	})

	// Serialize each issue to beads JSONL with metadata promotion
	var lines [][]byte
	for _, issue := range issues {
		line, err := issueToBeadsJSON(issue)
		if err != nil {
			return fmt.Errorf("serializing %s: %w", issue.ID, err)
		}
		lines = append(lines, line)
	}

	dest := exportTo
	if !filepath.IsAbs(dest) {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		dest = filepath.Join(wd, dest)
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	// Atomic write: write to tmp file in same dir, then rename
	tmpPath := dest + ".tmp"
	if err := writeJSONLFile(tmpPath, lines); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, dest); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("atomic rename: %w", err)
	}

	count := len(issues)
	if opts.JSON {
		result := map[string]interface{}{
			"exported": count,
			"path":     dest,
		}
		data, err := json.Marshal(result)
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(data))
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "Exported %d issues to %s\n", count, dest)
	}

	return nil
}

// issueToBeadsJSON serializes an Issue to beads JSONL format with metadata
// fields promoted to the top level. Beads stores fields like hook_bead at the
// top level; tl stores them in Metadata. This function reverses that mapping.
func issueToBeadsJSON(issue *Issue) ([]byte, error) {
	// Marshal issue to intermediate map
	raw, err := json.Marshal(issue)
	if err != nil {
		return nil, err
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}

	// Promote metadata fields to top level
	for k, v := range issue.Metadata {
		m[k] = v
	}

	// Remove the nested metadata key
	delete(m, "metadata")

	return json.Marshal(m)
}

// writeJSONLFile writes lines to a file, one per line, with a trailing newline.
func writeJSONLFile(path string, lines [][]byte) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, line := range lines {
		if _, err := f.Write(line); err != nil {
			return err
		}
		if _, err := f.Write([]byte{'\n'}); err != nil {
			return err
		}
	}
	return nil
}
