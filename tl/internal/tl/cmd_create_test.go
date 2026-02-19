// ABOUTME: Tests for the create command — title flag, positional arg, JSON output, error handling.
// ABOUTME: Verifies task creation and persistence via event store replay.
package tl

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// resetCreateGlobals sets create-related globals to defaults and schedules cleanup.
func resetCreateGlobals(t *testing.T, tlDir string) {
	t.Helper()
	jsonOutput = false
	tlDirFlag = tlDir
	createTitle = ""
	createType = "task"
	createPriority = 2
	createDescription = ""
	t.Cleanup(func() {
		jsonOutput = false
		tlDirFlag = ""
		createTitle = ""
		createType = "task"
		createPriority = 2
		createDescription = ""
	})
}

func TestCreateWithTitleFlag(t *testing.T) {
	t.Setenv("TL_ACTOR", "test")
	tmp := t.TempDir()
	require.NoError(t, initDir(tmp))
	resetCreateGlobals(t, tmp)

	createTitle = "My Task"

	cmd := &cobra.Command{}
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	err := runCreate(cmd, nil)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Created tl-")
	assert.Contains(t, output, "My Task")
}

func TestCreateWithPositionalArg(t *testing.T) {
	t.Setenv("TL_ACTOR", "test")
	tmp := t.TempDir()
	require.NoError(t, initDir(tmp))
	resetCreateGlobals(t, tmp)

	cmd := &cobra.Command{}
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	err := runCreate(cmd, []string{"Positional Title"})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Created tl-")
	assert.Contains(t, output, "Positional Title")
}

func TestCreateTitleFlagOverridesPositional(t *testing.T) {
	t.Setenv("TL_ACTOR", "test")
	tmp := t.TempDir()
	require.NoError(t, initDir(tmp))
	resetCreateGlobals(t, tmp)

	createTitle = "Flag Title"

	cmd := &cobra.Command{}
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	err := runCreate(cmd, []string{"Positional Title"})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Flag Title")
	assert.NotContains(t, output, "Positional Title")
}

func TestCreateJSON(t *testing.T) {
	t.Setenv("TL_ACTOR", "test")
	tmp := t.TempDir()
	require.NoError(t, initDir(tmp))
	resetCreateGlobals(t, tmp)

	jsonOutput = true
	createTitle = "JSON Task"
	createDescription = "A test task"
	createType = "feature"
	createPriority = 1

	cmd := &cobra.Command{}
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	err := runCreate(cmd, nil)
	require.NoError(t, err)

	var issue Issue
	err = json.Unmarshal(buf.Bytes(), &issue)
	require.NoError(t, err)

	assert.Contains(t, issue.ID, "tl-")
	assert.Equal(t, "JSON Task", issue.Title)
	assert.Equal(t, "A test task", issue.Description)
	assert.Equal(t, StatusOpen, issue.Status)
	assert.Equal(t, 1, issue.Priority)
	assert.Equal(t, IssueType("feature"), issue.IssueType)
	assert.False(t, issue.CreatedAt.IsZero())
	assert.False(t, issue.UpdatedAt.IsZero())
}

func TestCreateNoTitle(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, initDir(tmp))
	resetCreateGlobals(t, tmp)

	cmd := &cobra.Command{}
	cmd.SetOut(new(bytes.Buffer))

	err := runCreate(cmd, nil)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "title is required")
}

func TestCreatePersists(t *testing.T) {
	t.Setenv("TL_ACTOR", "test")
	tmp := t.TempDir()
	require.NoError(t, initDir(tmp))
	resetCreateGlobals(t, tmp)

	createTitle = "Persistent Task"
	createType = "bug"
	createPriority = 1

	cmd := &cobra.Command{}
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	err := runCreate(cmd, nil)
	require.NoError(t, err)

	// Verify the task was persisted by replaying the event store
	graph, err := loadGraph(filepath.Join(tmp, ".tl"))
	require.NoError(t, err)
	assert.Len(t, graph.Tasks, 1)

	for _, issue := range graph.Tasks {
		assert.Equal(t, "Persistent Task", issue.Title)
		assert.Equal(t, StatusOpen, issue.Status)
		assert.Equal(t, 1, issue.Priority)
		assert.Equal(t, IssueType("bug"), issue.IssueType)
	}
}

func TestCreateDefaultPriority(t *testing.T) {
	t.Setenv("TL_ACTOR", "test")
	tmp := t.TempDir()
	require.NoError(t, initDir(tmp))
	resetCreateGlobals(t, tmp)

	jsonOutput = true
	createTitle = "Default Priority"

	cmd := &cobra.Command{}
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	err := runCreate(cmd, nil)
	require.NoError(t, err)

	var issue Issue
	require.NoError(t, json.Unmarshal(buf.Bytes(), &issue))
	assert.Equal(t, 2, issue.Priority)
	assert.Equal(t, IssueType("task"), issue.IssueType)
}
