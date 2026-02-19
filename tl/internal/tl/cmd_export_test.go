// ABOUTME: Tests for `tl export --to <path>` command verifying JSONL output, metadata promotion, and atomic write.
// ABOUTME: Uses initDir + appendEventsToFile to set up test data, then validates exported file contents.

package tl

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupExportGraph creates a .tl dir with test issues and returns the .tl path.
func setupExportGraph(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	require.NoError(t, initDir(root))
	dir := filepath.Join(root, tlDirName)
	eventsPath := filepath.Join(dir, eventsFileName)

	ts := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)

	// Issue 1: plain open task
	create1, err := json.Marshal(CreateEventData{
		Title:     "Open task",
		Status:    string(StatusOpen),
		Priority:  1,
		IssueType: string(TypeTask),
	})
	require.NoError(t, err)

	// Issue 2: task with metadata (beads-specific fields)
	create2, err := json.Marshal(CreateEventData{
		Title:     "Hooked task",
		Status:    string(StatusOpen),
		Priority:  2,
		IssueType: string(TypeFeature),
		Metadata: map[string]json.RawMessage{
			"hook_bead":   json.RawMessage(`"bd-xxx"`),
			"mol_type":    json.RawMessage(`"swarm"`),
			"agent_state": json.RawMessage(`"idle"`),
		},
	})
	require.NoError(t, err)

	// Issue 3: task that depends on issue 1
	create3, err := json.Marshal(CreateEventData{
		Title:     "Blocked task",
		Status:    string(StatusOpen),
		Priority:  1,
		IssueType: string(TypeTask),
	})
	require.NoError(t, err)

	depAdd, err := json.Marshal(DepAddEventData{
		DependsOnID: "tl-0001",
		DepType:     string(DepBlocks),
	})
	require.NoError(t, err)

	events := []Event{
		{Type: EventCreate, ID: "tl-0001", Timestamp: ts, Actor: "test", Data: create1},
		{Type: EventCreate, ID: "tl-0002", Timestamp: ts.Add(time.Minute), Actor: "test", Data: create2},
		{Type: EventCreate, ID: "tl-0003", Timestamp: ts.Add(2 * time.Minute), Actor: "test", Data: create3},
		{Type: EventDepAdd, ID: "tl-0003", Timestamp: ts.Add(3 * time.Minute), Actor: "test", Data: depAdd},
	}
	require.NoError(t, appendEventsToFile(eventsPath, events))

	return dir
}

func TestExportBasic(t *testing.T) {
	dir := setupExportGraph(t)
	dest := filepath.Join(t.TempDir(), "out", "issues.jsonl")

	graph, err := loadGraph(dir)
	require.NoError(t, err)

	// Temporarily set globals for the command
	oldDir := tlDirFlag
	tlDirFlag = dir
	defer func() { tlDirFlag = oldDir }()

	oldExportTo := exportTo
	exportTo = dest
	defer func() { exportTo = oldExportTo }()

	err = runExport(exportCmd, nil)
	require.NoError(t, err)

	// Verify output file exists
	_, err = os.Stat(dest)
	require.NoError(t, err)

	// Read and parse each line
	f, err := os.Open(dest)
	require.NoError(t, err)
	defer f.Close()

	var parsed []map[string]interface{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var m map[string]interface{}
		require.NoError(t, json.Unmarshal(scanner.Bytes(), &m))
		parsed = append(parsed, m)
	}
	require.NoError(t, scanner.Err())

	// Should have 3 issues
	assert.Len(t, parsed, len(graph.Tasks))
	assert.Len(t, parsed, 3)

	// Sorted by ID: tl-0001, tl-0002, tl-0003
	assert.Equal(t, "tl-0001", parsed[0]["id"])
	assert.Equal(t, "tl-0002", parsed[1]["id"])
	assert.Equal(t, "tl-0003", parsed[2]["id"])
}

func TestExportMetadataPromotion(t *testing.T) {
	dir := setupExportGraph(t)
	dest := filepath.Join(t.TempDir(), "export.jsonl")

	tlDirFlag = dir
	defer func() { tlDirFlag = "" }()
	exportTo = dest
	defer func() { exportTo = ".beads/issues.jsonl" }()

	require.NoError(t, runExport(exportCmd, nil))

	f, err := os.Open(dest)
	require.NoError(t, err)
	defer f.Close()

	var lines []map[string]interface{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var m map[string]interface{}
		require.NoError(t, json.Unmarshal(scanner.Bytes(), &m))
		lines = append(lines, m)
	}
	require.NoError(t, scanner.Err())
	require.Len(t, lines, 3)

	// tl-0002 has metadata fields that should be promoted
	hooked := lines[1]
	assert.Equal(t, "tl-0002", hooked["id"])
	assert.Equal(t, "bd-xxx", hooked["hook_bead"], "hook_bead should be promoted to top level")
	assert.Equal(t, "swarm", hooked["mol_type"], "mol_type should be promoted to top level")
	assert.Equal(t, "idle", hooked["agent_state"], "agent_state should be promoted to top level")

	// "metadata" key should NOT exist in output
	_, hasMetadata := hooked["metadata"]
	assert.False(t, hasMetadata, "metadata key should be removed from exported JSON")

	// tl-0001 has no metadata — should still export cleanly without metadata key
	plain := lines[0]
	_, hasMetadata = plain["metadata"]
	assert.False(t, hasMetadata, "issues without metadata should not have metadata key")
}

func TestExportDependencies(t *testing.T) {
	dir := setupExportGraph(t)
	dest := filepath.Join(t.TempDir(), "deps.jsonl")

	tlDirFlag = dir
	defer func() { tlDirFlag = "" }()
	exportTo = dest
	defer func() { exportTo = ".beads/issues.jsonl" }()

	require.NoError(t, runExport(exportCmd, nil))

	f, err := os.Open(dest)
	require.NoError(t, err)
	defer f.Close()

	var lines []map[string]json.RawMessage
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var m map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(scanner.Bytes(), &m))
		lines = append(lines, m)
	}
	require.NoError(t, scanner.Err())
	require.Len(t, lines, 3)

	// tl-0003 has dependencies
	blocked := lines[2]
	var deps []map[string]interface{}
	require.NoError(t, json.Unmarshal(blocked["dependencies"], &deps))
	require.Len(t, deps, 1)
	assert.Equal(t, "tl-0003", deps[0]["issue_id"])
	assert.Equal(t, "tl-0001", deps[0]["depends_on_id"])
	assert.Equal(t, "blocks", deps[0]["type"])
}

func TestExportSortedByID(t *testing.T) {
	dir := setupExportGraph(t)
	dest := filepath.Join(t.TempDir(), "sorted.jsonl")

	tlDirFlag = dir
	defer func() { tlDirFlag = "" }()
	exportTo = dest
	defer func() { exportTo = ".beads/issues.jsonl" }()

	require.NoError(t, runExport(exportCmd, nil))

	f, err := os.Open(dest)
	require.NoError(t, err)
	defer f.Close()

	var ids []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var m map[string]interface{}
		require.NoError(t, json.Unmarshal(scanner.Bytes(), &m))
		ids = append(ids, m["id"].(string))
	}
	require.NoError(t, scanner.Err())

	for i := 1; i < len(ids); i++ {
		assert.True(t, ids[i-1] < ids[i], "IDs should be sorted: %s < %s", ids[i-1], ids[i])
	}
}

func TestExportCreatesParentDirs(t *testing.T) {
	dir := setupExportGraph(t)
	dest := filepath.Join(t.TempDir(), "deep", "nested", "dir", "issues.jsonl")

	tlDirFlag = dir
	defer func() { tlDirFlag = "" }()
	exportTo = dest
	defer func() { exportTo = ".beads/issues.jsonl" }()

	require.NoError(t, runExport(exportCmd, nil))

	_, err := os.Stat(dest)
	assert.NoError(t, err, "output file should exist in deeply nested path")
}

func TestExportAtomicWriteNoPartial(t *testing.T) {
	dir := setupExportGraph(t)

	// Point to a path where the parent doesn't exist AND we prevent MkdirAll
	// by using a file as the "parent directory"
	blocker := filepath.Join(t.TempDir(), "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte("I'm a file"), 0644))

	// Destination under a file (not a dir) — MkdirAll will fail
	dest := filepath.Join(blocker, "subdir", "issues.jsonl")

	tlDirFlag = dir
	defer func() { tlDirFlag = "" }()
	exportTo = dest
	defer func() { exportTo = ".beads/issues.jsonl" }()

	err := runExport(exportCmd, nil)
	assert.Error(t, err, "should fail when parent can't be created")

	// No output file should exist (stat will error — either NotExist or NotDir)
	_, statErr := os.Stat(dest)
	assert.Error(t, statErr, "no output file should exist on failure")
}

func TestExportEmptyGraph(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, initDir(root))
	dir := filepath.Join(root, tlDirName)
	dest := filepath.Join(t.TempDir(), "empty.jsonl")

	tlDirFlag = dir
	defer func() { tlDirFlag = "" }()
	exportTo = dest
	defer func() { exportTo = ".beads/issues.jsonl" }()

	require.NoError(t, runExport(exportCmd, nil))

	data, err := os.ReadFile(dest)
	require.NoError(t, err)
	assert.Empty(t, string(data), "empty graph should produce empty file")
}

func TestExportJSONOutput(t *testing.T) {
	dir := setupExportGraph(t)
	dest := filepath.Join(t.TempDir(), "json-out.jsonl")

	tlDirFlag = dir
	defer func() { tlDirFlag = "" }()
	exportTo = dest
	defer func() { exportTo = ".beads/issues.jsonl" }()
	jsonOutput = true
	defer func() { jsonOutput = false }()

	// Capture stdout
	old := exportCmd.OutOrStdout()
	r, w, _ := os.Pipe()
	exportCmd.SetOut(w)
	defer exportCmd.SetOut(old)

	require.NoError(t, runExport(exportCmd, nil))
	w.Close()

	var buf [4096]byte
	n, _ := r.Read(buf[:])
	output := string(buf[:n])

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(output), &result))
	assert.Equal(t, float64(3), result["exported"])
	assert.Equal(t, dest, result["path"])
}

func TestIssueToBeadsJSON(t *testing.T) {
	ts := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	issue := &Issue{
		ID:        "tl-test",
		Title:     "Test issue",
		Status:    StatusOpen,
		Priority:  1,
		IssueType: TypeTask,
		CreatedAt: ts,
		UpdatedAt: ts,
		Metadata: map[string]json.RawMessage{
			"hook_bead": json.RawMessage(`"bd-abc"`),
			"custom":    json.RawMessage(`42`),
		},
	}

	data, err := issueToBeadsJSON(issue)
	require.NoError(t, err)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &m))

	// Standard fields present
	assert.Equal(t, "tl-test", m["id"])
	assert.Equal(t, "Test issue", m["title"])

	// Metadata promoted
	assert.Equal(t, "bd-abc", m["hook_bead"])
	assert.Equal(t, float64(42), m["custom"])

	// No metadata key
	_, hasMetadata := m["metadata"]
	assert.False(t, hasMetadata)
}
