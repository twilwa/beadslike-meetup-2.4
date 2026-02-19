// ABOUTME: Tests for beads JSONL parser covering known/unknown fields, dependencies, and truncation
// ABOUTME: Uses testdata/beads_sample.jsonl fixture with 5 representative beads issues

package tl

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBeadsParserKnownFields(t *testing.T) {
	// Open the fixture file
	fixturePath := filepath.Join("testdata", "beads_sample.jsonl")
	file, err := os.Open(fixturePath)
	require.NoError(t, err, "failed to open fixture file")
	defer file.Close()

	issues, err := ParseBeadsJSONL(file)
	require.NoError(t, err, "ParseBeadsJSONL failed")
	require.Len(t, issues, 5, "expected 5 issues from fixture")

	// Test bd-aaa1: open task with priority 1
	aaa1 := issues[0]
	assert.Equal(t, "bd-aaa1", aaa1.ID)
	assert.Equal(t, "Open task with priority 1", aaa1.Title)
	assert.Equal(t, "A simple open task", aaa1.Description)
	assert.Equal(t, StatusOpen, aaa1.Status)
	assert.Equal(t, 1, aaa1.Priority)
	assert.Equal(t, TypeTask, aaa1.IssueType)
	assert.NotZero(t, aaa1.CreatedAt)
	assert.NotZero(t, aaa1.UpdatedAt)

	// Test bd-aaa2: closed task with reason
	aaa2 := issues[1]
	assert.Equal(t, "bd-aaa2", aaa2.ID)
	assert.Equal(t, "Closed task with reason", aaa2.Title)
	assert.Equal(t, StatusClosed, aaa2.Status)
	assert.Equal(t, 2, aaa2.Priority)
	assert.NotNil(t, aaa2.ClosedAt)
	assert.Equal(t, "completed", aaa2.CloseReason)
}

func TestBeadsParserUnknownFields(t *testing.T) {
	// Open the fixture file
	fixturePath := filepath.Join("testdata", "beads_sample.jsonl")
	file, err := os.Open(fixturePath)
	require.NoError(t, err, "failed to open fixture file")
	defer file.Close()

	issues, err := ParseBeadsJSONL(file)
	require.NoError(t, err, "ParseBeadsJSONL failed")
	require.Len(t, issues, 5, "expected 5 issues from fixture")

	// Test bd-aaa5: task with unknown beads fields
	aaa5 := issues[4]
	assert.Equal(t, "bd-aaa5", aaa5.ID)
	assert.NotNil(t, aaa5.Metadata, "Metadata should not be nil for issue with unknown fields")

	// Check that unknown fields are preserved in Metadata
	assert.Contains(t, aaa5.Metadata, "hook_bead", "hook_bead should be in Metadata")
	assert.Contains(t, aaa5.Metadata, "mol_type", "mol_type should be in Metadata")
	assert.Contains(t, aaa5.Metadata, "agent_state", "agent_state should be in Metadata")

	// Verify the values are correct
	var hookBead string
	err = json.Unmarshal(aaa5.Metadata["hook_bead"], &hookBead)
	require.NoError(t, err)
	assert.Equal(t, "bd-xxx", hookBead)

	var molType string
	err = json.Unmarshal(aaa5.Metadata["mol_type"], &molType)
	require.NoError(t, err)
	assert.Equal(t, "swarm", molType)

	var agentState string
	err = json.Unmarshal(aaa5.Metadata["agent_state"], &agentState)
	require.NoError(t, err)
	assert.Equal(t, "idle", agentState)
}

func TestBeadsParserDependencies(t *testing.T) {
	// Open the fixture file
	fixturePath := filepath.Join("testdata", "beads_sample.jsonl")
	file, err := os.Open(fixturePath)
	require.NoError(t, err, "failed to open fixture file")
	defer file.Close()

	issues, err := ParseBeadsJSONL(file)
	require.NoError(t, err, "ParseBeadsJSONL failed")
	require.Len(t, issues, 5, "expected 5 issues from fixture")

	// Test bd-aaa3: task with dependency
	aaa3 := issues[2]
	assert.Equal(t, "bd-aaa3", aaa3.ID)
	assert.Len(t, aaa3.Dependencies, 1, "aaa3 should have 1 dependency")

	dep := aaa3.Dependencies[0]
	assert.Equal(t, "bd-aaa3", dep.IssueID)
	assert.Equal(t, "bd-aaa1", dep.DependsOnID)
	assert.Equal(t, DepBlocks, dep.Type)
	assert.Equal(t, "test", dep.CreatedBy)
	assert.NotZero(t, dep.CreatedAt)
}

func TestBeadsParserTruncated(t *testing.T) {
	// Create a JSONL with a truncated final line (no closing brace)
	jsonl := `{"id":"bd-test1","title":"First issue","status":"open","priority":1,"created_at":"2025-01-15T10:00:00Z","updated_at":"2025-01-15T10:00:00Z"}
{"id":"bd-test2","title":"Second issue","status":"open","priority":2,"created_at":"2025-01-15T10:00:00Z","updated_at":"2025-01-15T10:00:00Z"}
{"id":"bd-test3","title":"Truncated issue","status":"open","priority":3,"created_at":"2025-01-15T10:00:00Z","updated_at":"2025-01-15T10:00:00Z"`

	// Write to temp file without final newline
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "truncated.jsonl")
	err := os.WriteFile(tmpFile, []byte(jsonl), 0644)
	require.NoError(t, err)

	// Parse the truncated file
	file, err := os.Open(tmpFile)
	require.NoError(t, err)
	defer file.Close()

	issues, err := ParseBeadsJSONL(file)
	// Should succeed and return the first 2 complete issues
	require.NoError(t, err, "ParseBeadsJSONL should tolerate truncated final line")
	assert.Len(t, issues, 2, "should parse 2 complete issues before truncation")
	assert.Equal(t, "bd-test1", issues[0].ID)
	assert.Equal(t, "bd-test2", issues[1].ID)
}

func TestBeadsParserEmpty(t *testing.T) {
	// Parse empty reader
	reader := bytes.NewReader([]byte(""))
	issues, err := ParseBeadsJSONL(reader)
	require.NoError(t, err, "ParseBeadsJSONL should handle empty input")
	assert.Empty(t, issues, "should return empty slice for empty input")
}

func TestBeadsParserEmptyLines(t *testing.T) {
	// JSONL with empty lines should skip them
	jsonl := `{"id":"bd-test1","title":"First","status":"open","priority":1,"created_at":"2025-01-15T10:00:00Z","updated_at":"2025-01-15T10:00:00Z"}

{"id":"bd-test2","title":"Second","status":"open","priority":2,"created_at":"2025-01-15T10:00:00Z","updated_at":"2025-01-15T10:00:00Z"}
`

	reader := bytes.NewReader([]byte(jsonl))
	issues, err := ParseBeadsJSONL(reader)
	require.NoError(t, err)
	assert.Len(t, issues, 2, "should skip empty lines")
	assert.Equal(t, "bd-test1", issues[0].ID)
	assert.Equal(t, "bd-test2", issues[1].ID)
}

func TestBeadsParserInvalidJSON(t *testing.T) {
	// JSONL with invalid JSON should error
	jsonl := `{"id":"bd-test1","title":"First","status":"open","priority":1,"created_at":"2025-01-15T10:00:00Z","updated_at":"2025-01-15T10:00:00Z"}
{invalid json`

	reader := bytes.NewReader([]byte(jsonl))
	_, err := ParseBeadsJSONL(reader)
	assert.Error(t, err, "should error on invalid JSON")
}

func TestBeadsParserTimestamps(t *testing.T) {
	// Test that timestamps are parsed correctly
	jsonl := `{"id":"bd-test1","title":"Test","status":"open","priority":1,"created_at":"2025-01-15T10:30:45Z","updated_at":"2025-01-16T14:22:33Z","closed_at":"2025-01-17T09:15:00Z"}`

	reader := bytes.NewReader([]byte(jsonl))
	issues, err := ParseBeadsJSONL(reader)
	require.NoError(t, err)
	require.Len(t, issues, 1)

	issue := issues[0]
	assert.Equal(t, 2025, issue.CreatedAt.Year())
	assert.Equal(t, time.January, issue.CreatedAt.Month())
	assert.Equal(t, 15, issue.CreatedAt.Day())
	assert.Equal(t, 10, issue.CreatedAt.Hour())
	assert.Equal(t, 30, issue.CreatedAt.Minute())
	assert.Equal(t, 45, issue.CreatedAt.Second())

	assert.NotNil(t, issue.ClosedAt)
	assert.Equal(t, 2025, issue.ClosedAt.Year())
	assert.Equal(t, time.January, issue.ClosedAt.Month())
	assert.Equal(t, 17, issue.ClosedAt.Day())
}

func TestBeadsParserLabels(t *testing.T) {
	// Test that labels array is parsed correctly
	jsonl := `{"id":"bd-test1","title":"Test","status":"open","priority":1,"created_at":"2025-01-15T10:00:00Z","updated_at":"2025-01-15T10:00:00Z","labels":["bug","urgent","frontend"]}`

	reader := bytes.NewReader([]byte(jsonl))
	issues, err := ParseBeadsJSONL(reader)
	require.NoError(t, err)
	require.Len(t, issues, 1)

	issue := issues[0]
	assert.Len(t, issue.Labels, 3)
	assert.Equal(t, "bug", issue.Labels[0])
	assert.Equal(t, "urgent", issue.Labels[1])
	assert.Equal(t, "frontend", issue.Labels[2])
}

func TestBeadsParserDeferUntil(t *testing.T) {
	// Test that defer_until is parsed correctly
	jsonl := `{"id":"bd-test1","title":"Test","status":"deferred","priority":1,"created_at":"2025-01-15T10:00:00Z","updated_at":"2025-01-15T10:00:00Z","defer_until":"2025-02-20T15:30:00Z"}`

	reader := bytes.NewReader([]byte(jsonl))
	issues, err := ParseBeadsJSONL(reader)
	require.NoError(t, err)
	require.Len(t, issues, 1)

	issue := issues[0]
	assert.NotNil(t, issue.DeferUntil)
	assert.Equal(t, 2025, issue.DeferUntil.Year())
	assert.Equal(t, time.February, issue.DeferUntil.Month())
	assert.Equal(t, 20, issue.DeferUntil.Day())
}

func TestBeadsParserMultipleUnknownFields(t *testing.T) {
	// Test that multiple unknown fields are all preserved
	jsonl := `{"id":"bd-test1","title":"Test","status":"open","priority":1,"created_at":"2025-01-15T10:00:00Z","updated_at":"2025-01-15T10:00:00Z","custom_field_1":"value1","custom_field_2":42,"custom_field_3":true}`

	reader := bytes.NewReader([]byte(jsonl))
	issues, err := ParseBeadsJSONL(reader)
	require.NoError(t, err)
	require.Len(t, issues, 1)

	issue := issues[0]
	assert.NotNil(t, issue.Metadata)
	assert.Len(t, issue.Metadata, 3)
	assert.Contains(t, issue.Metadata, "custom_field_1")
	assert.Contains(t, issue.Metadata, "custom_field_2")
	assert.Contains(t, issue.Metadata, "custom_field_3")
}
