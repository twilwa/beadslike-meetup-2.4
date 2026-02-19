// ABOUTME: Core domain types for task management (Issue, Dependency, Graph, Status, DependencyType, IssueType, Priority)
// ABOUTME: Includes state machine validation and type methods for workflow transitions

package tl

import (
	"encoding/json"
	"errors"
	"time"
)

// Status represents the workflow state of an issue
type Status string

const (
	StatusOpen       Status = "open"
	StatusInProgress Status = "in_progress"
	StatusBlocked    Status = "blocked"
	StatusDeferred   Status = "deferred"
	StatusClosed     Status = "closed"
	StatusPinned     Status = "pinned"
	StatusHooked     Status = "hooked"
)

// IsValid checks if the status is a known built-in status
func (s Status) IsValid() bool {
	switch s {
	case StatusOpen, StatusInProgress, StatusBlocked, StatusDeferred, StatusClosed, StatusPinned, StatusHooked:
		return true
	}
	return false
}

// DependencyType represents the kind of relationship between issues
type DependencyType string

const (
	DepBlocks            DependencyType = "blocks"
	DepParentChild       DependencyType = "parent-child"
	DepConditionalBlocks DependencyType = "conditional-blocks"
	DepWaitsFor          DependencyType = "waits-for"
	DepRelated           DependencyType = "related"
	DepDiscoveredFrom    DependencyType = "discovered-from"
)

// AffectsReadyWork returns true if this dependency type blocks work
func (d DependencyType) AffectsReadyWork() bool {
	return d == DepBlocks || d == DepParentChild || d == DepConditionalBlocks || d == DepWaitsFor
}

// IssueType represents the category of work
type IssueType string

const (
	TypeBug      IssueType = "bug"
	TypeFeature  IssueType = "feature"
	TypeTask     IssueType = "task"
	TypeEpic     IssueType = "epic"
	TypeChore    IssueType = "chore"
	TypeDecision IssueType = "decision"
)

// Issue represents a work item in the task management system
type Issue struct {
	ID                 string                     `json:"id"`
	Title              string                     `json:"title"`
	Description        string                     `json:"description,omitempty"`
	Design             string                     `json:"design,omitempty"`
	AcceptanceCriteria string                     `json:"acceptance_criteria,omitempty"`
	Notes              string                     `json:"notes,omitempty"`
	SpecID             string                     `json:"spec_id,omitempty"`
	Status             Status                     `json:"status,omitempty"`
	Priority           int                        `json:"priority"`
	IssueType          IssueType                  `json:"issue_type,omitempty"`
	Assignee           string                     `json:"assignee,omitempty"`
	Owner              string                     `json:"owner,omitempty"`
	CreatedBy          string                     `json:"created_by,omitempty"`
	CreatedAt          time.Time                  `json:"created_at"`
	UpdatedAt          time.Time                  `json:"updated_at"`
	ClosedAt           *time.Time                 `json:"closed_at,omitempty"`
	CloseReason        string                     `json:"close_reason,omitempty"`
	DeferUntil         *time.Time                 `json:"defer_until,omitempty"`
	Labels             []string                   `json:"labels,omitempty"`
	Dependencies       []*Dependency              `json:"dependencies,omitempty"`
	Pinned             bool                       `json:"pinned,omitempty"`
	Ephemeral          bool                       `json:"ephemeral,omitempty"`
	Metadata           map[string]json.RawMessage `json:"metadata,omitempty"`
}

// Dependency represents a relationship between two issues
type Dependency struct {
	IssueID     string          `json:"issue_id"`
	DependsOnID string          `json:"depends_on_id"`
	Type        DependencyType  `json:"type"`
	CreatedAt   time.Time       `json:"created_at"`
	CreatedBy   string          `json:"created_by,omitempty"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
}

// Graph represents the in-memory state of the task graph after event replay
type Graph struct {
	Tasks map[string]*Issue   // id → issue
	Deps  map[string][]string // issueID → []dependsOnID
	RDeps map[string][]string // dependsOnID → []issueID (reverse index)
}

// validateTransition checks if a status transition is valid
func validateTransition(from, to Status) error {
	// No-op is always valid
	if from == to {
		return nil
	}

	// Define valid transitions
	validTransitions := map[Status]map[Status]struct{}{
		StatusOpen: {
			StatusInProgress: {},
			StatusBlocked:    {},
			StatusDeferred:   {},
			StatusClosed:     {},
		},
		StatusInProgress: {
			StatusOpen:    {},
			StatusBlocked: {},
			StatusClosed:  {},
		},
		StatusBlocked: {
			StatusOpen:       {},
			StatusInProgress: {},
			StatusClosed:     {},
		},
		StatusDeferred: {
			StatusOpen: {},
		},
		StatusClosed: {
			StatusOpen: {},
		},
		StatusPinned: {},
		StatusHooked: {},
	}

	// Check if from status is known
	allowed, ok := validTransitions[from]
	if !ok {
		return errors.New("unknown status: " + string(from))
	}

	// Check if to status is allowed
	if _, valid := allowed[to]; !valid {
		return errors.New("invalid transition: " + string(from) + " → " + string(to))
	}

	return nil
}

// Sentinel error constants
var (
	ErrNoTLDir  = errors.New("no .tl directory found (run tl init)")
	ErrLockBusy = errors.New("lock busy, retry")
	ErrNotFound = errors.New("not found")
	ErrCycle    = errors.New("dependency would create a cycle")
)
