// ABOUTME: Init command — creates .tl/ directory structure for tl task management.
// ABOUTME: Implements `tl init` to initialize a new task repository in the current directory.

package tl

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func init() {
	initCmd.RunE = runInit
}

func runInit(cmd *cobra.Command, args []string) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	if err := initDir(wd); err != nil {
		return err
	}
	if jsonOutput {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(struct {
			Initialized bool   `json:"initialized"`
			Path        string `json:"path"`
		}{true, ".tl/"})
	}
	fmt.Fprintln(cmd.OutOrStdout(), "Initialized .tl/")
	return nil
}
