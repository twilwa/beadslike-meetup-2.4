// ABOUTME: Tests for import command behavior when ingesting beads JSONL fixtures.
// ABOUTME: Verifies dedup counts, ID preservation, and unknown metadata retention.

package tl

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImportFromFixture(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, initDir(root))

	dir := filepath.Join(root, tlDirName)
	fixture := filepath.Join("testdata", "beads_sample.jsonl")

	originalDir := tlDirFlag
	originalJSON := jsonOutput
	originalFrom := importFromPath
	t.Cleanup(func() {
		tlDirFlag = originalDir
		jsonOutput = originalJSON
		importFromPath = originalFrom
	})

	tlDirFlag = root
	jsonOutput = false
	importFromPath = fixture

	var out strings.Builder
	cmd := &cobra.Command{}
	cmd.SetOut(&out)

	err := runImport(cmd, nil)
	require.NoError(t, err)
	assert.Equal(t, "Imported 5, Updated 0, Skipped 0 from testdata/beads_sample.jsonl\n", out.String())

	graph, err := loadGraph(dir)
	require.NoError(t, err)
	require.Len(t, graph.Tasks, 5)

	for id := range graph.Tasks {
		assert.True(t, strings.HasPrefix(id, "bd-"), "expected beads ID prefix: %s", id)
	}

	issue := graph.Tasks["bd-aaa5"]
	require.NotNil(t, issue)
	require.NotNil(t, issue.Metadata)
	assert.Contains(t, issue.Metadata, "hook_bead")
	assert.Contains(t, issue.Metadata, "mol_type")
	assert.Contains(t, issue.Metadata, "agent_state")

	var hookBead string
	require.NoError(t, json.Unmarshal(issue.Metadata["hook_bead"], &hookBead))
	assert.Equal(t, "bd-xxx", hookBead)
}

func TestImportDeduplicatesByContentHash(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, initDir(root))

	fixture := filepath.Join("testdata", "beads_sample.jsonl")

	originalDir := tlDirFlag
	originalJSON := jsonOutput
	originalFrom := importFromPath
	t.Cleanup(func() {
		tlDirFlag = originalDir
		jsonOutput = originalJSON
		importFromPath = originalFrom
	})

	tlDirFlag = root
	jsonOutput = false
	importFromPath = fixture

	cmdFirst := &cobra.Command{}
	cmdFirst.SetOut(&strings.Builder{})
	require.NoError(t, runImport(cmdFirst, nil))

	var out strings.Builder
	cmdSecond := &cobra.Command{}
	cmdSecond.SetOut(&out)
	require.NoError(t, runImport(cmdSecond, nil))
	assert.Equal(t, "Imported 0, Updated 0, Skipped 5 from testdata/beads_sample.jsonl\n", out.String())
}
