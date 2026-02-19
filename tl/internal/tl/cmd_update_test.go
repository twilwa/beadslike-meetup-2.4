// ABOUTME: Tests for tl update command — verifies selective field updates, status transitions, and error cases.
// ABOUTME: Uses t.TempDir() with real event store for end-to-end mutation verification.

package tl

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// seedIssue creates a test issue in the store and returns the .tl dir path
func seedIssue(t *testing.T, id, title string, status Status) string {
	t.Helper()
	root := t.TempDir()
	err := initDir(root)
	require.NoError(t, err)
	dir := filepath.Join(root, tlDirName)

	err = mutate(dir, func(_ *Graph) ([]Event, error) {
		data, err := json.Marshal(CreateEventData{
			Title:     title,
			Status:    string(status),
			Priority:  1,
			IssueType: string(TypeTask),
		})
		if err != nil {
			return nil, err
		}
		ts := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		return []Event{{Type: EventCreate, ID: id, Timestamp: ts, Actor: "test", Data: data}}, nil
	})
	require.NoError(t, err)
	return dir
}

func TestUpdateTitle(t *testing.T) {
	dir := seedIssue(t, "tl-u001", "Original Title", StatusOpen)

	var updated Issue
	err := mutate(dir, func(g *Graph) ([]Event, error) {
		issue, ok := g.Tasks["tl-u001"]
		if !ok {
			return nil, ErrNotFound
		}

		fields := map[string]json.RawMessage{
			"title": json.RawMessage(`"New Title"`),
		}

		evt, err := newEvent(EventUpdate, "tl-u001", UpdateEventData{Fields: fields})
		if err != nil {
			return nil, err
		}

		issue.Title = "New Title"
		issue.UpdatedAt = evt.Timestamp
		updated = *issue
		return []Event{evt}, nil
	})
	require.NoError(t, err)
	assert.Equal(t, "New Title", updated.Title)

	// Verify persisted state via reload
	g, err := loadGraph(dir)
	require.NoError(t, err)
	assert.Equal(t, "New Title", g.Tasks["tl-u001"].Title)
}

func TestUpdateStatus(t *testing.T) {
	dir := seedIssue(t, "tl-u002", "Status Test", StatusOpen)

	err := mutate(dir, func(g *Graph) ([]Event, error) {
		issue := g.Tasks["tl-u002"]
		if err := validateTransition(issue.Status, StatusInProgress); err != nil {
			return nil, err
		}
		fields := map[string]json.RawMessage{
			"status": json.RawMessage(`"in_progress"`),
		}
		evt, err := newEvent(EventUpdate, "tl-u002", UpdateEventData{Fields: fields})
		if err != nil {
			return nil, err
		}
		issue.Status = StatusInProgress
		return []Event{evt}, nil
	})
	require.NoError(t, err)

	g, err := loadGraph(dir)
	require.NoError(t, err)
	assert.Equal(t, StatusInProgress, g.Tasks["tl-u002"].Status)
}

func TestUpdateInvalidTransition(t *testing.T) {
	dir := seedIssue(t, "tl-u003", "Bad Transition", StatusOpen)

	err := mutate(dir, func(g *Graph) ([]Event, error) {
		issue := g.Tasks["tl-u003"]
		// open → pinned is not a valid transition
		if err := validateTransition(issue.Status, StatusPinned); err != nil {
			return nil, err
		}
		return nil, nil
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid transition")
}

func TestUpdateMultipleFields(t *testing.T) {
	dir := seedIssue(t, "tl-u004", "Multi Update", StatusOpen)

	err := mutate(dir, func(g *Graph) ([]Event, error) {
		issue := g.Tasks["tl-u004"]
		fields := map[string]json.RawMessage{
			"title":    json.RawMessage(`"Updated Title"`),
			"priority": json.RawMessage(`3`),
			"assignee": json.RawMessage(`"alice"`),
		}
		evt, err := newEvent(EventUpdate, "tl-u004", UpdateEventData{Fields: fields})
		if err != nil {
			return nil, err
		}
		issue.Title = "Updated Title"
		issue.Priority = 3
		issue.Assignee = "alice"
		issue.UpdatedAt = evt.Timestamp
		return []Event{evt}, nil
	})
	require.NoError(t, err)

	g, err := loadGraph(dir)
	require.NoError(t, err)
	issue := g.Tasks["tl-u004"]
	assert.Equal(t, "Updated Title", issue.Title)
	assert.Equal(t, 3, issue.Priority)
	assert.Equal(t, "alice", issue.Assignee)
}

func TestUpdateNotFound(t *testing.T) {
	dir := seedIssue(t, "tl-u005", "Exists", StatusOpen)

	err := mutate(dir, func(g *Graph) ([]Event, error) {
		_, ok := g.Tasks["tl-nope"]
		if !ok {
			return nil, ErrNotFound
		}
		return nil, nil
	})
	assert.ErrorIs(t, err, ErrNotFound)
}
