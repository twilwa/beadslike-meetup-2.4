// ABOUTME: Tests for `tl list` command — filtering, sorting, limit, text and JSON output.
// ABOUTME: Verifies list behavior against a temp .tl/ event store with known test data.

package tl

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupListTestTasks creates a temp .tl/ dir with 3 test issues:
//
//	tl-aaaa: P2, open, task, assignee=alice, created t0
//	tl-bbbb: P0, closed, bug, assignee=bob, created t0+1min
//	tl-cccc: P1, open, feature, assignee=alice, created t0+2min
//
// Sorted by priority: tl-bbbb(P0), tl-cccc(P1), tl-aaaa(P2)
func setupListTestTasks(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	require.NoError(t, initDir(root))

	dir := filepath.Join(root, tlDirName)
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	createA, _ := json.Marshal(CreateEventData{Title: "Alpha task", Status: string(StatusOpen), Priority: 2, IssueType: string(TypeTask)})
	createB, _ := json.Marshal(CreateEventData{Title: "Beta bug", Status: string(StatusOpen), Priority: 0, IssueType: string(TypeBug)})
	createC, _ := json.Marshal(CreateEventData{Title: "Charlie feature", Status: string(StatusOpen), Priority: 1, IssueType: string(TypeFeature)})

	assignAlice, _ := json.Marshal(UpdateEventData{Fields: map[string]json.RawMessage{"assignee": json.RawMessage(`"alice"`)}})
	assignBob, _ := json.Marshal(UpdateEventData{Fields: map[string]json.RawMessage{"assignee": json.RawMessage(`"bob"`)}})
	closeData, _ := json.Marshal(CloseEventData{Reason: "fixed"})

	events := []Event{
		{Type: EventCreate, ID: "tl-aaaa", Timestamp: t0, Actor: "tester", Data: createA},
		{Type: EventCreate, ID: "tl-bbbb", Timestamp: t0.Add(time.Minute), Actor: "tester", Data: createB},
		{Type: EventCreate, ID: "tl-cccc", Timestamp: t0.Add(2 * time.Minute), Actor: "tester", Data: createC},
		{Type: EventUpdate, ID: "tl-aaaa", Timestamp: t0.Add(3 * time.Minute), Actor: "tester", Data: assignAlice},
		{Type: EventUpdate, ID: "tl-bbbb", Timestamp: t0.Add(4 * time.Minute), Actor: "tester", Data: assignBob},
		{Type: EventUpdate, ID: "tl-cccc", Timestamp: t0.Add(5 * time.Minute), Actor: "tester", Data: assignAlice},
		{Type: EventClose, ID: "tl-bbbb", Timestamp: t0.Add(6 * time.Minute), Actor: "tester", Data: closeData},
	}

	require.NoError(t, appendEventsToFile(filepath.Join(dir, eventsFileName), events))
	return root
}

func resetListGlobals(dir string) {
	tlDirFlag = dir
	jsonOutput = false
	listStatus = ""
	listType = ""
	listAssignee = ""
	listPriority = -1
	listLimit = 0
}

func runListCapture(t *testing.T) (string, error) {
	t.Helper()
	buf := new(bytes.Buffer)
	listCmd.SetOut(buf)
	err := runList(listCmd, nil)
	return buf.String(), err
}

func TestListEmpty(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, initDir(root))
	resetListGlobals(root)

	out, err := runListCapture(t)
	require.NoError(t, err)
	assert.Equal(t, "", out)
}

func TestListEmptyJSON(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, initDir(root))
	resetListGlobals(root)
	jsonOutput = true

	out, err := runListCapture(t)
	require.NoError(t, err)
	assert.Equal(t, "[]\n", out)
}

func TestListAllSorted(t *testing.T) {
	root := setupListTestTasks(t)
	resetListGlobals(root)

	out, err := runListCapture(t)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(out), "\n")
	require.Len(t, lines, 3)
	assert.Contains(t, lines[0], "tl-bbbb") // P0
	assert.Contains(t, lines[1], "tl-cccc") // P1
	assert.Contains(t, lines[2], "tl-aaaa") // P2
}

func TestListTextFormat(t *testing.T) {
	root := setupListTestTasks(t)
	resetListGlobals(root)

	out, err := runListCapture(t)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(out), "\n")
	require.Len(t, lines, 3)
	assert.Equal(t, "tl-bbbb [closed] P0 Beta bug", lines[0])
	assert.Equal(t, "tl-cccc [open] P1 Charlie feature", lines[1])
	assert.Equal(t, "tl-aaaa [open] P2 Alpha task", lines[2])
}

func TestListFilterStatus(t *testing.T) {
	root := setupListTestTasks(t)
	resetListGlobals(root)
	listStatus = "open"

	out, err := runListCapture(t)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(out), "\n")
	require.Len(t, lines, 2)
	assert.Contains(t, lines[0], "tl-cccc") // P1
	assert.Contains(t, lines[1], "tl-aaaa") // P2
}

func TestListFilterType(t *testing.T) {
	root := setupListTestTasks(t)
	resetListGlobals(root)
	listType = "bug"

	out, err := runListCapture(t)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(out), "\n")
	require.Len(t, lines, 1)
	assert.Contains(t, lines[0], "tl-bbbb")
}

func TestListFilterAssignee(t *testing.T) {
	root := setupListTestTasks(t)
	resetListGlobals(root)
	listAssignee = "alice"

	out, err := runListCapture(t)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(out), "\n")
	require.Len(t, lines, 2)
	assert.Contains(t, lines[0], "tl-cccc") // P1 alice
	assert.Contains(t, lines[1], "tl-aaaa") // P2 alice
}

func TestListFilterPriority(t *testing.T) {
	root := setupListTestTasks(t)
	resetListGlobals(root)
	listPriority = 0

	out, err := runListCapture(t)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(out), "\n")
	require.Len(t, lines, 1)
	assert.Contains(t, lines[0], "tl-bbbb")
}

func TestListFilterPriorityNotSet(t *testing.T) {
	root := setupListTestTasks(t)
	resetListGlobals(root)
	// listPriority = -1 (default, no filter) — all 3 returned

	out, err := runListCapture(t)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(out), "\n")
	assert.Len(t, lines, 3)
}

func TestListLimit(t *testing.T) {
	root := setupListTestTasks(t)
	resetListGlobals(root)
	listLimit = 2

	out, err := runListCapture(t)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(out), "\n")
	require.Len(t, lines, 2)
	assert.Contains(t, lines[0], "tl-bbbb") // P0
	assert.Contains(t, lines[1], "tl-cccc") // P1
}

func TestListJSON(t *testing.T) {
	root := setupListTestTasks(t)
	resetListGlobals(root)
	jsonOutput = true

	out, err := runListCapture(t)
	require.NoError(t, err)

	var issues []Issue
	require.NoError(t, json.Unmarshal([]byte(out), &issues))
	require.Len(t, issues, 3)

	// Verify sort order in JSON output
	assert.Equal(t, "tl-bbbb", issues[0].ID)
	assert.Equal(t, "tl-cccc", issues[1].ID)
	assert.Equal(t, "tl-aaaa", issues[2].ID)
}

func TestListFilterCombined(t *testing.T) {
	root := setupListTestTasks(t)
	resetListGlobals(root)
	listStatus = "open"
	listAssignee = "alice"

	out, err := runListCapture(t)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(out), "\n")
	require.Len(t, lines, 2)
	assert.Contains(t, lines[0], "tl-cccc")
	assert.Contains(t, lines[1], "tl-aaaa")
}
