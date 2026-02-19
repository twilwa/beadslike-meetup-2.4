// ABOUTME: Tests `tl blocked` command output for blocked issue discovery and blockers listing.
// ABOUTME: Covers both text and JSON output forms with real dependency events.

package tl

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBlockedShowsBlockingIssueIDsTextAndJSON(t *testing.T) {
	ts := time.Date(2026, 1, 4, 9, 0, 0, 0, time.UTC)
	dir := seedCommandRepoWithEvents(
		t,
		createIssueEvent(t, "tl-blocker", "Blocker", StatusOpen, 0, ts),
		createIssueEvent(t, "tl-target", "Target", StatusOpen, 1, ts.Add(time.Minute)),
		depAddEvent(t, "tl-target", "tl-blocker", DepBlocks, ts.Add(2*time.Minute)),
	)

	setCommandGlobals(t, dir, false)
	cmd := newTestCommand()
	require.NoError(t, runBlocked(cmd, nil))

	text := cmd.OutOrStdout().(*bytes.Buffer).String()
	assert.Contains(t, text, "tl-target [open] Target (blocked by: tl-blocker)")
	assert.NotContains(t, text, "tl-blocker [open]")

	setCommandGlobals(t, dir, true)
	cmd = newTestCommand()
	require.NoError(t, runBlocked(cmd, nil))

	var rows []blockedIssue
	require.NoError(t, json.Unmarshal(bytes.TrimSpace(cmd.OutOrStdout().(*bytes.Buffer).Bytes()), &rows))
	require.Len(t, rows, 1)
	assert.Equal(t, "tl-target", rows[0].ID)
	assert.Equal(t, []string{"tl-blocker"}, rows[0].Blockers)
}
