// ABOUTME: End-to-end smoke tests for the tl CLI covering the full agent workflow.
// ABOUTME: Builds the real tl binary and exercises all daily-use commands with real data.

package tl

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type e2eRunner struct {
	t       *testing.T
	bin     string
	tlDir   string
	rootDir string
}

func newE2ERunner(t *testing.T, bin string) *e2eRunner {
	t.Helper()
	root := t.TempDir()
	tlD := filepath.Join(root, tlDirName)
	return &e2eRunner{t: t, bin: bin, tlDir: tlD, rootDir: root}
}

func (r *e2eRunner) run(args ...string) (string, error) {
	r.t.Helper()
	allArgs := append([]string{"--dir", r.tlDir}, args...)
	cmd := exec.Command(r.bin, allArgs...)
	cmd.Dir = r.rootDir
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func (r *e2eRunner) mustRun(args ...string) string {
	r.t.Helper()
	out, err := r.run(args...)
	require.NoError(r.t, err, "command failed: tl %s\noutput: %s", strings.Join(args, " "), out)
	return out
}

func buildTestBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "tl")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	cmd := exec.Command("go", "build", "-o", bin, "github.com/twilwa/tl/cmd/tl")
	cmd.Dir = filepath.Join(findModuleRoot(t), "tl")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "failed to build tl binary: %s", out)
	return bin
}

func findModuleRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	dir := filepath.Dir(file)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return filepath.Dir(dir)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find module root")
		}
		dir = parent
	}
}

func TestE2EFullAgentWorkflow(t *testing.T) {
	bin := buildTestBinary(t)
	r := newE2ERunner(t, bin)

	// Init
	out := r.mustRun("init")
	assert.Contains(t, out, "Initialized")

	// Create tasks
	out = r.mustRun("create", "--title", "Task A", "--priority", "1", "--json")
	var taskA Issue
	require.NoError(t, json.Unmarshal([]byte(out), &taskA))
	assert.True(t, strings.HasPrefix(taskA.ID, "tl-"), "ID should have tl- prefix")
	assert.Equal(t, "Task A", taskA.Title)
	assert.Equal(t, StatusOpen, taskA.Status)

	out = r.mustRun("create", "--title", "Task B", "--priority", "2", "--json")
	var taskB Issue
	require.NoError(t, json.Unmarshal([]byte(out), &taskB))

	out = r.mustRun("create", "--title", "Blocker", "--priority", "0", "--json")
	var blocker Issue
	require.NoError(t, json.Unmarshal([]byte(out), &blocker))

	// List — should have 3 tasks
	out = r.mustRun("list", "--json")
	var tasks []Issue
	require.NoError(t, json.Unmarshal([]byte(out), &tasks))
	assert.Len(t, tasks, 3)

	// List with status filter
	out = r.mustRun("list", "--status", "open", "--json")
	require.NoError(t, json.Unmarshal([]byte(out), &tasks))
	assert.Len(t, tasks, 3)

	// Show
	out = r.mustRun("show", taskA.ID, "--json")
	var shown Issue
	require.NoError(t, json.Unmarshal([]byte(out), &shown))
	assert.Equal(t, taskA.ID, shown.ID)

	// Update status
	out = r.mustRun("update", taskA.ID, "--status", "in_progress", "--json")
	var updated Issue
	require.NoError(t, json.Unmarshal([]byte(out), &updated))
	assert.Equal(t, StatusInProgress, updated.Status)

	// Stats
	out = r.mustRun("stats", "--json")
	var stats map[string]int
	require.NoError(t, json.Unmarshal([]byte(out), &stats))
	assert.Equal(t, 3, stats["total"])
	assert.Equal(t, 1, stats["in_progress"])

	// Dep add: taskB depends on blocker
	out = r.mustRun("dep", "add", taskB.ID, blocker.ID, "--type", "blocks", "--json")
	var withDep Issue
	require.NoError(t, json.Unmarshal([]byte(out), &withDep))
	assert.Len(t, withDep.Dependencies, 1)

	// Cycle detection: adding reverse dep should fail
	_, err := r.run("dep", "add", blocker.ID, taskB.ID, "--type", "blocks")
	assert.Error(t, err, "cycle should be rejected")

	// Ready queue: taskB is blocked by blocker, taskA is in_progress (not ready)
	out = r.mustRun("ready", "--json")
	var readyTasks []Issue
	require.NoError(t, json.Unmarshal([]byte(out), &readyTasks))
	readyIDs := make(map[string]bool)
	for _, task := range readyTasks {
		readyIDs[task.ID] = true
	}
	assert.True(t, readyIDs[blocker.ID], "blocker should be ready")
	assert.False(t, readyIDs[taskB.ID], "taskB should be blocked")
	assert.False(t, readyIDs[taskA.ID], "taskA is in_progress, not in ready queue")

	// Blocked command
	out = r.mustRun("blocked", "--json")
	var blockedOutput []map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &blockedOutput))
	assert.Len(t, blockedOutput, 1)

	// Close blocker — taskB should become ready
	r.mustRun("close", blocker.ID, "--reason", "done")
	out = r.mustRun("ready", "--json")
	require.NoError(t, json.Unmarshal([]byte(out), &readyTasks))
	readyIDs = make(map[string]bool)
	for _, task := range readyTasks {
		readyIDs[task.ID] = true
	}
	assert.True(t, readyIDs[taskB.ID], "taskB should be ready after blocker closed")

	// Claim taskB
	out = r.mustRun("claim", taskB.ID, "--json")
	var claimed Issue
	require.NoError(t, json.Unmarshal([]byte(out), &claimed))
	assert.Equal(t, StatusInProgress, claimed.Status)

	// Close taskA
	r.mustRun("close", taskA.ID, "--reason", "finished")
	out = r.mustRun("show", taskA.ID, "--json")
	var closedA Issue
	require.NoError(t, json.Unmarshal([]byte(out), &closedA))
	assert.Equal(t, StatusClosed, closedA.Status)
	assert.Equal(t, "finished", closedA.CloseReason)

	// Reopen taskA
	r.mustRun("reopen", taskA.ID)
	out = r.mustRun("show", taskA.ID, "--json")
	var reopened Issue
	require.NoError(t, json.Unmarshal([]byte(out), &reopened))
	assert.Equal(t, StatusOpen, reopened.Status)
}

func TestE2EImportExportRoundTrip(t *testing.T) {
	bin := buildTestBinary(t)
	r := newE2ERunner(t, bin)

	r.mustRun("init")

	// Import from beads sample fixture
	samplePath := filepath.Join(findModuleRoot(t), "tl", "internal", "tl", "testdata", "beads_sample.jsonl")
	if _, err := os.Stat(samplePath); os.IsNotExist(err) {
		t.Skip("beads_sample.jsonl not found, skipping import test")
	}

	out := r.mustRun("import", "--from", samplePath, "--json")
	var counts struct {
		Imported int `json:"imported"`
		Updated  int `json:"updated"`
		Skipped  int `json:"skipped"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &counts))
	assert.Greater(t, counts.Imported, 0, "should have imported issues")

	// List to verify imported tasks
	out = r.mustRun("list", "--json")
	var tasks []Issue
	require.NoError(t, json.Unmarshal([]byte(out), &tasks))
	assert.Greater(t, len(tasks), 0)

	// Verify beads IDs preserved (bd- prefix)
	for _, task := range tasks {
		assert.True(t, strings.HasPrefix(task.ID, "bd-"), "imported IDs should preserve bd- prefix: %s", task.ID)
	}

	// Second import should skip all (dedup)
	out = r.mustRun("import", "--from", samplePath, "--json")
	require.NoError(t, json.Unmarshal([]byte(out), &counts))
	assert.Equal(t, 0, counts.Imported, "second import should import nothing")
	assert.Greater(t, counts.Skipped, 0, "second import should skip all")

	// Export
	exportPath := filepath.Join(t.TempDir(), "exported.jsonl")
	out = r.mustRun("export", "--to", exportPath, "--json")
	var exportResult struct {
		Exported int    `json:"exported"`
		Path     string `json:"path"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &exportResult))
	assert.Greater(t, exportResult.Exported, 0)

	// Verify exported file is valid JSONL
	data, err := os.ReadFile(exportPath)
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	for i, line := range lines {
		var obj map[string]interface{}
		assert.NoError(t, json.Unmarshal([]byte(line), &obj), "line %d should be valid JSON", i+1)
	}
}

func TestE2EDepRemove(t *testing.T) {
	bin := buildTestBinary(t)
	r := newE2ERunner(t, bin)
	r.mustRun("init")

	out := r.mustRun("create", "--title", "A", "--json")
	var a Issue
	require.NoError(t, json.Unmarshal([]byte(out), &a))

	out = r.mustRun("create", "--title", "B", "--json")
	var b Issue
	require.NoError(t, json.Unmarshal([]byte(out), &b))

	// Add dep
	r.mustRun("dep", "add", b.ID, a.ID, "--type", "blocks")

	// Verify B is blocked
	out = r.mustRun("ready", "--json")
	var readyTasks []Issue
	require.NoError(t, json.Unmarshal([]byte(out), &readyTasks))
	readyIDs := map[string]bool{}
	for _, task := range readyTasks {
		readyIDs[task.ID] = true
	}
	assert.False(t, readyIDs[b.ID], "B should be blocked")

	// Remove dep
	r.mustRun("dep", "remove", b.ID, a.ID)

	// B should be ready now
	out = r.mustRun("ready", "--json")
	require.NoError(t, json.Unmarshal([]byte(out), &readyTasks))
	readyIDs = map[string]bool{}
	for _, task := range readyTasks {
		readyIDs[task.ID] = true
	}
	assert.True(t, readyIDs[b.ID], "B should be ready after dep removed")
}
