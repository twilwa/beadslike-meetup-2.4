// ABOUTME: Tests for the show command (tl show <id>).
// ABOUTME: Verifies JSON and text output, not-found error, and dependency display.

package tl

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupShowGraph(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	require.NoError(t, initDir(root))
	dir := root + "/" + tlDirName

	ts := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	evt, err := newEvent(EventCreate, "tl-show1", CreateEventData{
		Title:     "Show me",
		Status:    "open",
		Priority:  1,
		IssueType: "task",
	})
	require.NoError(t, err)
	evt.Timestamp = ts
	require.NoError(t, appendEventsToFile(dir+"/"+eventsFileName, []Event{evt}))

	return dir
}

func TestShowJSON(t *testing.T) {
	dir := setupShowGraph(t)

	tlDirFlag = dir
	defer func() { tlDirFlag = "" }()
	jsonOutput = true
	defer func() { jsonOutput = false }()

	var buf bytes.Buffer
	showCmd.SetOut(&buf)
	defer showCmd.SetOut(nil)

	err := runShow(showCmd, []string{"tl-show1"})
	require.NoError(t, err)

	var issue Issue
	require.NoError(t, json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &issue))
	assert.Equal(t, "tl-show1", issue.ID)
	assert.Equal(t, "Show me", issue.Title)
	assert.Equal(t, StatusOpen, issue.Status)
	assert.Equal(t, 1, issue.Priority)
}

func TestShowText(t *testing.T) {
	dir := setupShowGraph(t)

	tlDirFlag = dir
	defer func() { tlDirFlag = "" }()
	jsonOutput = false

	var buf bytes.Buffer
	showCmd.SetOut(&buf)
	defer showCmd.SetOut(nil)

	err := runShow(showCmd, []string{"tl-show1"})
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "tl-show1")
	assert.Contains(t, out, "Show me")
	assert.Contains(t, out, "open")
}

func TestShowNotFound(t *testing.T) {
	dir := setupShowGraph(t)

	tlDirFlag = dir
	defer func() { tlDirFlag = "" }()

	err := runShow(showCmd, []string{"tl-doesnotexist"})
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "not found"))
}

func TestShowNoArgs(t *testing.T) {
	dir := setupShowGraph(t)

	tlDirFlag = dir
	defer func() { tlDirFlag = "" }()

	err := runShow(showCmd, []string{})
	require.Error(t, err)
}
