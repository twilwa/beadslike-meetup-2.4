// ABOUTME: Tests dependency CLI command handlers for add/remove edge mutations.
// ABOUTME: Covers cycle checks, self-dependency rejection, and permissive dependency types.

package tl

import (
	"bytes"
	"errors"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDepAddRejectsCycle(t *testing.T) {
	dir := setupDepRepo(t, "tl-a", "tl-b")
	setDepCommandGlobals(t, dir)

	require.NoError(t, runDepAdd(newDepCommand(t), []string{"tl-a", "tl-b"}))
	err := runDepAdd(newDepCommand(t), []string{"tl-b", "tl-a"})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrCycle)

	graph, loadErr := loadGraph(dir)
	require.NoError(t, loadErr)
	assert.Equal(t, []string{"tl-b"}, graph.Deps["tl-a"])
	assert.Empty(t, graph.Deps["tl-b"])
}

func TestDepAddAcceptsUnknownType(t *testing.T) {
	dir := setupDepRepo(t, "tl-a", "tl-b")
	setDepCommandGlobals(t, dir)
	depType = "soft-link"

	err := runDepAdd(newDepCommand(t), []string{"tl-a", "tl-b"})
	require.NoError(t, err)

	graph, loadErr := loadGraph(dir)
	require.NoError(t, loadErr)

	require.Contains(t, graph.Tasks, "tl-a")
	require.Len(t, graph.Tasks["tl-a"].Dependencies, 1)
	assert.Equal(t, DependencyType("soft-link"), graph.Tasks["tl-a"].Dependencies[0].Type)
}

func TestDepAddRejectsSelfDependency(t *testing.T) {
	dir := setupDepRepo(t, "tl-a")
	setDepCommandGlobals(t, dir)

	err := runDepAdd(newDepCommand(t), []string{"tl-a", "tl-a"})
	require.Error(t, err)
	assert.EqualError(t, err, "cannot depend on self")
}

func TestDepRemoveWorks(t *testing.T) {
	dir := setupDepRepo(t, "tl-a", "tl-b")
	setDepCommandGlobals(t, dir)

	require.NoError(t, runDepAdd(newDepCommand(t), []string{"tl-a", "tl-b"}))

	cmd := newDepCommand(t)
	err := runDepRemove(cmd, []string{"tl-a", "tl-b"})
	require.NoError(t, err)
	assert.Equal(t, "Removed dependency: tl-a no longer depends on tl-b\n", cmd.OutOrStdout().(*bytes.Buffer).String())

	graph, loadErr := loadGraph(dir)
	require.NoError(t, loadErr)
	assert.Empty(t, graph.Deps["tl-a"])
	assert.Empty(t, graph.RDeps["tl-b"])
	require.Contains(t, graph.Tasks, "tl-a")
	assert.Empty(t, graph.Tasks["tl-a"].Dependencies)
}

func TestDepAddCmdTypeFlagDefault(t *testing.T) {
	flag := depAddCmd.Flags().Lookup("type")
	require.NotNil(t, flag)
	assert.Equal(t, string(DepBlocks), flag.DefValue)
}

func newDepCommand(t *testing.T) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{}
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	return cmd
}

func setDepCommandGlobals(t *testing.T, dir string) {
	t.Helper()
	prevJSON := jsonOutput
	prevDir := tlDirFlag
	prevType := depType
	t.Cleanup(func() {
		jsonOutput = prevJSON
		tlDirFlag = prevDir
		depType = prevType
	})

	jsonOutput = false
	tlDirFlag = dir
	depType = string(DepBlocks)
}

func setupDepRepo(t *testing.T, issueIDs ...string) string {
	t.Helper()

	root := t.TempDir()
	require.NoError(t, initDir(root))
	dir := filepath.Join(root, tlDirName)

	err := mutate(dir, func(_ *Graph) ([]Event, error) {
		events := make([]Event, 0, len(issueIDs))
		for _, issueID := range issueIDs {
			evt, evtErr := newEvent(EventCreate, issueID, CreateEventData{
				Title:  issueID,
				Status: string(StatusOpen),
			})
			if evtErr != nil {
				return nil, evtErr
			}
			events = append(events, evt)
		}
		return events, nil
	})
	require.NoError(t, err)

	return dir
}

func TestDepRemoveMissingDependencyReturnsNotFound(t *testing.T) {
	dir := setupDepRepo(t, "tl-a", "tl-b")
	setDepCommandGlobals(t, dir)

	err := runDepRemove(newDepCommand(t), []string{"tl-a", "tl-b"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrNotFound))
}
