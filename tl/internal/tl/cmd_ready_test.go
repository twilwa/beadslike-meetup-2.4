// ABOUTME: Tests `tl ready` command behavior against real event-log graph replays.
// ABOUTME: Validates blocked filtering and blocker-closure promotion into ready queue.

package tl

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadySkipsBlockedThenIncludesAfterBlockerClosed(t *testing.T) {
	ts := time.Date(2026, 1, 3, 9, 0, 0, 0, time.UTC)
	dir := seedCommandRepoWithEvents(
		t,
		createIssueEvent(t, "tl-blocker", "Blocker", StatusOpen, 0, ts),
		createIssueEvent(t, "tl-child", "Blocked child", StatusOpen, 1, ts.Add(time.Minute)),
		depAddEvent(t, "tl-child", "tl-blocker", DepBlocks, ts.Add(2*time.Minute)),
	)

	setCommandGlobals(t, dir, false)
	cmd := newTestCommand()
	require.NoError(t, runReady(cmd, nil))

	output := cmd.OutOrStdout().(*bytes.Buffer).String()
	assert.Contains(t, output, "tl-blocker P0 Blocker")
	assert.NotContains(t, output, "tl-child")

	require.NoError(t, appendEventsToFile(filepath.Join(dir, eventsFileName), []Event{
		closeIssueEvent(t, "tl-blocker", "done", ts.Add(3*time.Minute)),
	}))

	cmd = newTestCommand()
	require.NoError(t, runReady(cmd, nil))

	output = cmd.OutOrStdout().(*bytes.Buffer).String()
	assert.Contains(t, output, "tl-child P1 Blocked child")
	assert.NotContains(t, output, "tl-blocker")
}

func TestReadyJSONReturnsArray(t *testing.T) {
	ts := time.Date(2026, 1, 3, 10, 0, 0, 0, time.UTC)
	dir := seedCommandRepoWithEvents(
		t,
		createIssueEvent(t, "tl-ready", "Ready now", StatusOpen, 2, ts),
	)

	setCommandGlobals(t, dir, true)
	cmd := newTestCommand()
	require.NoError(t, runReady(cmd, nil))

	var rows []readyIssue
	require.NoError(t, json.Unmarshal(bytes.TrimSpace(cmd.OutOrStdout().(*bytes.Buffer).Bytes()), &rows))
	require.Len(t, rows, 1)
	assert.Equal(t, "tl-ready", rows[0].ID)
}

func newTestCommand() *cobra.Command {
	cmd := &cobra.Command{}
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	return cmd
}

func setCommandGlobals(t *testing.T, dir string, json bool) {
	t.Helper()
	prevJSON := jsonOutput
	prevDir := tlDirFlag
	t.Cleanup(func() {
		jsonOutput = prevJSON
		tlDirFlag = prevDir
	})

	jsonOutput = json
	tlDirFlag = dir
}

func seedCommandRepoWithEvents(t *testing.T, events ...Event) string {
	t.Helper()
	root := t.TempDir()
	require.NoError(t, initDir(root))
	dir := filepath.Join(root, tlDirName)
	require.NoError(t, appendEventsToFile(filepath.Join(dir, eventsFileName), events))
	return dir
}

func createIssueEvent(t *testing.T, id, title string, status Status, priority int, ts time.Time) Event {
	t.Helper()
	data, err := json.Marshal(CreateEventData{Title: title, Status: string(status), Priority: priority})
	require.NoError(t, err)
	return Event{Type: EventCreate, ID: id, Timestamp: ts, Actor: "test", Data: data}
}

func depAddEvent(t *testing.T, issueID, dependsOnID string, depType DependencyType, ts time.Time) Event {
	t.Helper()
	data, err := json.Marshal(DepAddEventData{DependsOnID: dependsOnID, DepType: string(depType)})
	require.NoError(t, err)
	return Event{Type: EventDepAdd, ID: issueID, Timestamp: ts, Actor: "test", Data: data}
}

func closeIssueEvent(t *testing.T, id, reason string, ts time.Time) Event {
	t.Helper()
	data, err := json.Marshal(CloseEventData{Reason: reason})
	require.NoError(t, err)
	return Event{Type: EventClose, ID: id, Timestamp: ts, Actor: "test", Data: data}
}

func statusUpdateEvent(t *testing.T, id string, status Status, ts time.Time) Event {
	t.Helper()
	fields := map[string]json.RawMessage{"status": json.RawMessage(`"` + strings.ReplaceAll(string(status), `"`, ``) + `"`)}
	data, err := json.Marshal(UpdateEventData{Fields: fields})
	require.NoError(t, err)
	return Event{Type: EventUpdate, ID: id, Timestamp: ts, Actor: "test", Data: data}
}
