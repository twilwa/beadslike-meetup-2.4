// ABOUTME: Sync command — exports tl tasks to beads JSONL and stages with git add.
// ABOUTME: Implements `tl sync` to update .beads/issues.jsonl and stage the change.

package tl

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

var syncTo string

func init() {
	syncCmd.Flags().StringVar(&syncTo, "to", ".beads/issues.jsonl", "Destination path for JSONL export")
	syncCmd.RunE = runSync
}

func runSync(cmd *cobra.Command, args []string) error {
	// Step 1: export to the target path (reuse export logic)
	prevExportTo := exportTo
	exportTo = syncTo
	defer func() { exportTo = prevExportTo }()

	if err := runExport(cmd, args); err != nil {
		return fmt.Errorf("export: %w", err)
	}

	// Step 2: git add the exported file
	dest := syncTo
	if !filepath.IsAbs(dest) {
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getting working directory: %w", err)
		}
		dest = filepath.Join(wd, dest)
	}

	gitCmd := exec.Command("git", "add", dest)
	gitCmd.Stdout = cmd.OutOrStdout()
	gitCmd.Stderr = cmd.ErrOrStderr()
	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("git add %s: %w", dest, err)
	}

	if !jsonOutput {
		fmt.Fprintf(cmd.OutOrStdout(), "Synced %s\n", dest)
	}
	return nil
}
