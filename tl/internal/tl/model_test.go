// ABOUTME: Tests for core domain types (Status, DependencyType, Priority, state transitions)
// ABOUTME: Validates type methods, JSON marshaling, and workflow state machine

package tl

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestStatusIsValid validates the IsValid method for Status type
func TestStatusIsValid(t *testing.T) {
	tests := []struct {
		status Status
		valid  bool
	}{
		{StatusOpen, true},
		{StatusInProgress, true},
		{StatusBlocked, true},
		{StatusDeferred, true},
		{StatusClosed, true},
		{StatusPinned, true},
		{StatusHooked, true},
		{Status("unknown"), false},
		{Status("custom_workflow"), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			assert.Equal(t, tt.valid, tt.status.IsValid())
		})
	}
}

// TestStatusRoundTrip validates that unknown statuses round-trip through JSON
func TestStatusRoundTrip(t *testing.T) {
	issue := Issue{
		ID:        "test-1",
		Title:     "Test Issue",
		Status:    Status("custom_workflow"),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Marshal to JSON
	data, err := json.Marshal(issue)
	assert.NoError(t, err)

	// Unmarshal back
	var unmarshaled Issue
	err = json.Unmarshal(data, &unmarshaled)
	assert.NoError(t, err)

	// Status should be preserved
	assert.Equal(t, Status("custom_workflow"), unmarshaled.Status)
}

// TestPriorityZeroNotOmitted validates that Priority: 0 is included in JSON
func TestPriorityZeroNotOmitted(t *testing.T) {
	issue := Issue{
		ID:        "test-1",
		Title:     "Test Issue",
		Priority:  0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	data, err := json.Marshal(issue)
	assert.NoError(t, err)

	// Check that "priority":0 is in the JSON
	assert.Contains(t, string(data), `"priority":0`)
}

// TestDependencyTypeAffectsReadyWork validates the AffectsReadyWork method
func TestDependencyTypeAffectsReadyWork(t *testing.T) {
	tests := []struct {
		depType DependencyType
		affects bool
	}{
		{DepBlocks, true},
		{DepParentChild, true},
		{DepConditionalBlocks, true},
		{DepWaitsFor, true},
		{DepRelated, false},
		{DepDiscoveredFrom, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.depType), func(t *testing.T) {
			assert.Equal(t, tt.affects, tt.depType.AffectsReadyWork())
		})
	}
}

// TestDependencyTypeUnknownAccepted validates that unknown dependency types are accepted
func TestDependencyTypeUnknownAccepted(t *testing.T) {
	customDep := DependencyType("custom-link")

	// Should not panic or error
	dep := Dependency{
		IssueID:     "issue-1",
		DependsOnID: "issue-2",
		Type:        customDep,
		CreatedAt:   time.Now(),
	}

	// Should marshal and unmarshal successfully
	data, err := json.Marshal(dep)
	assert.NoError(t, err)

	var unmarshaled Dependency
	err = json.Unmarshal(data, &unmarshaled)
	assert.NoError(t, err)

	assert.Equal(t, customDep, unmarshaled.Type)
}

// TestValidateTransitionValid validates valid state transitions
func TestValidateTransitionValid(t *testing.T) {
	tests := []struct {
		from Status
		to   Status
	}{
		{StatusOpen, StatusInProgress},
		{StatusOpen, StatusBlocked},
		{StatusOpen, StatusDeferred},
		{StatusOpen, StatusClosed},
		{StatusInProgress, StatusOpen},
		{StatusInProgress, StatusBlocked},
		{StatusInProgress, StatusClosed},
		{StatusBlocked, StatusOpen},
		{StatusBlocked, StatusInProgress},
		{StatusBlocked, StatusClosed},
		{StatusDeferred, StatusOpen},
		{StatusClosed, StatusOpen},
		{StatusOpen, StatusOpen},     // no-op
		{StatusClosed, StatusClosed}, // no-op
	}

	for _, tt := range tests {
		t.Run(string(tt.from)+"→"+string(tt.to), func(t *testing.T) {
			err := validateTransition(tt.from, tt.to)
			assert.NoError(t, err)
		})
	}
}

// TestValidateTransitionInvalid validates invalid state transitions
func TestValidateTransitionInvalid(t *testing.T) {
	tests := []struct {
		from Status
		to   Status
	}{
		{StatusClosed, StatusInProgress},
		{StatusClosed, StatusBlocked},
		{StatusClosed, StatusDeferred},
		{StatusDeferred, StatusInProgress},
		{StatusDeferred, StatusBlocked},
		{StatusDeferred, StatusClosed},
		{StatusPinned, StatusOpen},
		{StatusPinned, StatusInProgress},
		{StatusHooked, StatusOpen},
	}

	for _, tt := range tests {
		t.Run(string(tt.from)+"→"+string(tt.to), func(t *testing.T) {
			err := validateTransition(tt.from, tt.to)
			assert.Error(t, err)
		})
	}
}

// TestValidateTransitionUnknownStatus validates that unknown statuses return error
func TestValidateTransitionUnknownStatus(t *testing.T) {
	err := validateTransition(Status("unknown"), StatusOpen)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown status")
}
