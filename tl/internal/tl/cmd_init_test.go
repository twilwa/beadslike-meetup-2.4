// ABOUTME: Tests for the init command — directory creation, JSON and text output, double-init error.
// ABOUTME: Uses temporary directories for test isolation.
package tl

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitText(t *testing.T) {
	tmp := t.TempDir()
	origWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmp))
	defer os.Chdir(origWd)

	jsonOutput = false

	cmd := &cobra.Command{}
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	err = runInit(cmd, nil)
	require.NoError(t, err)
	assert.Equal(t, "Initialized .tl/\n", buf.String())

	// Verify .tl/ directory was created with expected files
	info, err := os.Stat(filepath.Join(tmp, ".tl"))
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	_, err = os.Stat(filepath.Join(tmp, ".tl", "events.jsonl"))
	assert.NoError(t, err)

	_, err = os.Stat(filepath.Join(tmp, ".tl", "lock"))
	assert.NoError(t, err)
}

func TestInitJSON(t *testing.T) {
	tmp := t.TempDir()
	origWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmp))
	defer os.Chdir(origWd)

	jsonOutput = true
	defer func() { jsonOutput = false }()

	cmd := &cobra.Command{}
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	err = runInit(cmd, nil)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, true, result["initialized"])
	assert.Equal(t, ".tl/", result["path"])
}

func TestInitDoubleError(t *testing.T) {
	tmp := t.TempDir()
	origWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmp))
	defer os.Chdir(origWd)

	jsonOutput = false

	cmd := &cobra.Command{}
	cmd.SetOut(new(bytes.Buffer))

	err = runInit(cmd, nil)
	require.NoError(t, err)

	err = runInit(cmd, nil)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "already initialized")
}
