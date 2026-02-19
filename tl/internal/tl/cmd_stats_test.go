// ABOUTME: Tests `tl stats` command for status totals and computed blocked-set counts.
// ABOUTME: Ensures blocked metrics derive from dependency state, not only status labels.

package tl

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatsCountsBlockedSetAndStatuses(t *testing.T) {
	ts := time.Date(2026, 1, 5, 9, 0, 0, 0, time.UTC)
	dir := seedCommandRepoWithEvents(
		t,
		createIssueEvent(t, "tl-open", "Open", StatusOpen, 0, ts),
		createIssueEvent(t, "tl-inprog", "In Progress", StatusOpen, 0, ts.Add(time.Minute)),
		statusUpdateEvent(t, "tl-inprog", StatusInProgress, ts.Add(2*time.Minute)),
		createIssueEvent(t, "tl-closed", "Closed", StatusOpen, 0, ts.Add(3*time.Minute)),
		closeIssueEvent(t, "tl-closed", "done", ts.Add(4*time.Minute)),
		createIssueEvent(t, "tl-deferred", "Deferred", StatusDeferred, 0, ts.Add(5*time.Minute)),
		createIssueEvent(t, "tl-blocked", "Blocked by dependency", StatusOpen, 1, ts.Add(6*time.Minute)),
		depAddEvent(t, "tl-blocked", "tl-open", DepBlocks, ts.Add(7*time.Minute)),
	)

	setCommandGlobals(t, dir, false)
	cmd := newTestCommand()
	require.NoError(t, runStats(cmd, nil))

	text := cmd.OutOrStdout().(*bytes.Buffer).String()
	assert.Equal(t, "Open: 2 | In Progress: 1 | Blocked: 1 | Closed: 1 | Total: 5\n", text)

	setCommandGlobals(t, dir, true)
	cmd = newTestCommand()
	require.NoError(t, runStats(cmd, nil))

	var out statsOutput
	require.NoError(t, json.Unmarshal(bytes.TrimSpace(cmd.OutOrStdout().(*bytes.Buffer).Bytes()), &out))
	assert.Equal(t, 2, out.Open)
	assert.Equal(t, 1, out.InProgress)
	assert.Equal(t, 1, out.Blocked)
	assert.Equal(t, 1, out.Closed)
	assert.Equal(t, 1, out.Deferred)
	assert.Equal(t, 5, out.Total)
}
