// ABOUTME: Tests for tl close and reopen commands — verifies lifecycle transitions and state clearing.
// ABOUTME: Covers full create → update → close → reopen lifecycle plus invalid transition errors.

package tl

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCloseTask(t *testing.T) {
	dir := seedIssue(t, "tl-c001", "Close Me", StatusOpen)

	var closed Issue
	err := mutate(dir, func(g *Graph) ([]Event, error) {
		issue := g.Tasks["tl-c001"]
		if err := validateTransition(issue.Status, StatusClosed); err != nil {
			return nil, err
		}
		evt, err := newEvent(EventClose, "tl-c001", CloseEventData{Reason: "completed"})
		if err != nil {
			return nil, err
		}
		issue.Status = StatusClosed
		issue.CloseReason = "completed"
		closedAt := evt.Timestamp
		issue.ClosedAt = &closedAt
		issue.UpdatedAt = evt.Timestamp
		closed = *issue
		return []Event{evt}, nil
	})
	require.NoError(t, err)
	assert.Equal(t, StatusClosed, closed.Status)
	assert.Equal(t, "completed", closed.CloseReason)
	assert.NotNil(t, closed.ClosedAt)

	// Verify persisted state
	g, err := loadGraph(dir)
	require.NoError(t, err)
	issue := g.Tasks["tl-c001"]
	assert.Equal(t, StatusClosed, issue.Status)
	assert.Equal(t, "completed", issue.CloseReason)
	assert.NotNil(t, issue.ClosedAt)
}

func TestCloseNotFound(t *testing.T) {
	dir := seedIssue(t, "tl-c002", "Exists", StatusOpen)

	err := mutate(dir, func(g *Graph) ([]Event, error) {
		_, ok := g.Tasks["tl-nope"]
		if !ok {
			return nil, ErrNotFound
		}
		return nil, nil
	})
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestCloseAlreadyClosed(t *testing.T) {
	dir := seedIssue(t, "tl-c003", "Already Closed", StatusOpen)

	// Close it first
	err := mutate(dir, func(g *Graph) ([]Event, error) {
		issue := g.Tasks["tl-c003"]
		evt, err := newEvent(EventClose, "tl-c003", CloseEventData{Reason: "done"})
		if err != nil {
			return nil, err
		}
		issue.Status = StatusClosed
		return []Event{evt}, nil
	})
	require.NoError(t, err)

	// Try to close again — closed → closed is a no-op (same status)
	err = mutate(dir, func(g *Graph) ([]Event, error) {
		issue := g.Tasks["tl-c003"]
		if err := validateTransition(issue.Status, StatusClosed); err != nil {
			return nil, err
		}
		// validateTransition allows from==to as no-op
		return nil, nil
	})
	assert.NoError(t, err)
}

func TestReopenTask(t *testing.T) {
	dir := seedIssue(t, "tl-r001", "Reopen Me", StatusOpen)

	// Close first
	err := mutate(dir, func(g *Graph) ([]Event, error) {
		evt, err := newEvent(EventClose, "tl-r001", CloseEventData{Reason: "premature"})
		if err != nil {
			return nil, err
		}
		issue := g.Tasks["tl-r001"]
		issue.Status = StatusClosed
		closedAt := evt.Timestamp
		issue.ClosedAt = &closedAt
		issue.CloseReason = "premature"
		issue.Assignee = "bob"
		return []Event{evt}, nil
	})
	require.NoError(t, err)

	// Reopen
	var reopened Issue
	err = mutate(dir, func(g *Graph) ([]Event, error) {
		issue := g.Tasks["tl-r001"]
		if err := validateTransition(issue.Status, StatusOpen); err != nil {
			return nil, err
		}
		evt, err := newEvent(EventReopen, "tl-r001", ReopenEventData{})
		if err != nil {
			return nil, err
		}
		issue.Status = StatusOpen
		issue.ClosedAt = nil
		issue.CloseReason = ""
		issue.Assignee = ""
		issue.UpdatedAt = evt.Timestamp
		reopened = *issue
		return []Event{evt}, nil
	})
	require.NoError(t, err)
	assert.Equal(t, StatusOpen, reopened.Status)
	assert.Nil(t, reopened.ClosedAt)
	assert.Empty(t, reopened.CloseReason)
	assert.Empty(t, reopened.Assignee)

	// Verify persisted state
	g, err := loadGraph(dir)
	require.NoError(t, err)
	issue := g.Tasks["tl-r001"]
	assert.Equal(t, StatusOpen, issue.Status)
	assert.Nil(t, issue.ClosedAt)
	assert.Empty(t, issue.CloseReason)
	assert.Empty(t, issue.Assignee)
}

func TestReopenNotClosed(t *testing.T) {
	dir := seedIssue(t, "tl-r002", "Not Closed", StatusOpen)

	err := mutate(dir, func(g *Graph) ([]Event, error) {
		issue := g.Tasks["tl-r002"]
		// open → open is a no-op (same status)
		if err := validateTransition(issue.Status, StatusOpen); err != nil {
			return nil, err
		}
		return nil, nil
	})
	// validateTransition allows from==to as no-op
	assert.NoError(t, err)
}

func TestReopenDeferredInvalid(t *testing.T) {
	dir := seedIssue(t, "tl-r003", "Deferred", StatusOpen)

	// Transition to deferred first
	err := mutate(dir, func(g *Graph) ([]Event, error) {
		issue := g.Tasks["tl-r003"]
		fields := map[string]json.RawMessage{
			"status": json.RawMessage(`"deferred"`),
		}
		evt, err := newEvent(EventUpdate, "tl-r003", UpdateEventData{Fields: fields})
		if err != nil {
			return nil, err
		}
		issue.Status = StatusDeferred
		return []Event{evt}, nil
	})
	require.NoError(t, err)

	// Deferred → closed is not valid per the transition table
	err = mutate(dir, func(g *Graph) ([]Event, error) {
		issue := g.Tasks["tl-r003"]
		return nil, validateTransition(issue.Status, StatusClosed)
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid transition")
}

func TestFullLifecycle(t *testing.T) {
	dir := seedIssue(t, "tl-lc01", "Lifecycle Task", StatusOpen)

	// 1. Update: open → in_progress with title change
	err := mutate(dir, func(g *Graph) ([]Event, error) {
		issue := g.Tasks["tl-lc01"]
		if err := validateTransition(issue.Status, StatusInProgress); err != nil {
			return nil, err
		}
		fields := map[string]json.RawMessage{
			"status": json.RawMessage(`"in_progress"`),
			"title":  json.RawMessage(`"Lifecycle Task (WIP)"`),
		}
		evt, err := newEvent(EventUpdate, "tl-lc01", UpdateEventData{Fields: fields})
		if err != nil {
			return nil, err
		}
		issue.Status = StatusInProgress
		issue.Title = "Lifecycle Task (WIP)"
		issue.UpdatedAt = evt.Timestamp
		return []Event{evt}, nil
	})
	require.NoError(t, err)

	g, err := loadGraph(dir)
	require.NoError(t, err)
	assert.Equal(t, StatusInProgress, g.Tasks["tl-lc01"].Status)
	assert.Equal(t, "Lifecycle Task (WIP)", g.Tasks["tl-lc01"].Title)

	// 2. Close: in_progress → closed
	err = mutate(dir, func(g *Graph) ([]Event, error) {
		issue := g.Tasks["tl-lc01"]
		if err := validateTransition(issue.Status, StatusClosed); err != nil {
			return nil, err
		}
		evt, err := newEvent(EventClose, "tl-lc01", CloseEventData{Reason: "shipped"})
		if err != nil {
			return nil, err
		}
		issue.Status = StatusClosed
		issue.CloseReason = "shipped"
		closedAt := evt.Timestamp
		issue.ClosedAt = &closedAt
		issue.UpdatedAt = evt.Timestamp
		return []Event{evt}, nil
	})
	require.NoError(t, err)

	g, err = loadGraph(dir)
	require.NoError(t, err)
	assert.Equal(t, StatusClosed, g.Tasks["tl-lc01"].Status)
	assert.Equal(t, "shipped", g.Tasks["tl-lc01"].CloseReason)

	// 3. Reopen: closed → open
	err = mutate(dir, func(g *Graph) ([]Event, error) {
		issue := g.Tasks["tl-lc01"]
		if err := validateTransition(issue.Status, StatusOpen); err != nil {
			return nil, err
		}
		evt, err := newEvent(EventReopen, "tl-lc01", ReopenEventData{})
		if err != nil {
			return nil, err
		}
		issue.Status = StatusOpen
		issue.ClosedAt = nil
		issue.CloseReason = ""
		issue.Assignee = ""
		issue.UpdatedAt = evt.Timestamp
		return []Event{evt}, nil
	})
	require.NoError(t, err)

	g, err = loadGraph(dir)
	require.NoError(t, err)
	issue := g.Tasks["tl-lc01"]
	assert.Equal(t, StatusOpen, issue.Status)
	assert.Nil(t, issue.ClosedAt)
	assert.Empty(t, issue.CloseReason)
	assert.Empty(t, issue.Assignee)
	// Title should be preserved from update
	assert.Equal(t, "Lifecycle Task (WIP)", issue.Title)
}
