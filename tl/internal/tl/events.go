// ABOUTME: Event type definitions and JSON serialization for append-only event log.
// ABOUTME: Provides Event struct, typed data structs, and round-trip marshaling for events.jsonl.
package tl

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Event constants for event types
const (
	EventCreate    = "create"
	EventUpdate    = "update"
	EventClose     = "close"
	EventReopen    = "reopen"
	EventDepAdd    = "dep_add"
	EventDepRemove = "dep_remove"
	EventClaim     = "claim"
)

// Event is the base event written to events.jsonl
type Event struct {
	Type      string          `json:"type"`
	ID        string          `json:"id"`
	Timestamp time.Time       `json:"ts"`
	Actor     string          `json:"actor"`
	Data      json.RawMessage `json:"data"`
}

// CreateEventData is the typed data for create events
type CreateEventData struct {
	Title       string                     `json:"title"`
	Description string                     `json:"description,omitempty"`
	Status      string                     `json:"status"`
	Priority    int                        `json:"priority"`
	IssueType   string                     `json:"issue_type,omitempty"`
	Labels      []string                   `json:"labels,omitempty"`
	Metadata    map[string]json.RawMessage `json:"metadata,omitempty"`
}

// UpdateEventData is the typed data for update events
type UpdateEventData struct {
	Fields map[string]json.RawMessage `json:"fields"`
}

// CloseEventData is the typed data for close events
type CloseEventData struct {
	Reason string `json:"reason,omitempty"`
}

// ReopenEventData is the typed data for reopen events
type ReopenEventData struct {
}

// DepAddEventData is the typed data for dependency add events
type DepAddEventData struct {
	DependsOnID string `json:"depends_on_id"`
	DepType     string `json:"dep_type"`
}

// DepRemoveEventData is the typed data for dependency remove events
type DepRemoveEventData struct {
	DependsOnID string `json:"depends_on_id"`
}

// ClaimEventData is the typed data for claim events
type ClaimEventData struct {
	Agent string `json:"agent"`
}

// resolveActor returns the actor name from environment or git config
// Priority: TL_ACTOR env var → git config user.name → "unknown"
func resolveActor() string {
	// Check TL_ACTOR environment variable
	if actor := os.Getenv("TL_ACTOR"); actor != "" {
		return actor
	}

	// Try git config user.name
	cmd := exec.Command("git", "config", "user.name")
	output, err := cmd.Output()
	if err == nil {
		name := strings.TrimSpace(string(output))
		if name != "" {
			return name
		}
	}

	return "unknown"
}

// newEvent creates a new Event with the given type, task ID, and data
// It sets Timestamp to now (UTC) and Actor from resolveActor()
// The data is marshaled to json.RawMessage for Event.Data
func newEvent(eventType, taskID string, data interface{}) (Event, error) {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return Event{}, fmt.Errorf("failed to marshal event data: %w", err)
	}

	return Event{
		Type:      eventType,
		ID:        taskID,
		Timestamp: time.Now().UTC(),
		Actor:     resolveActor(),
		Data:      json.RawMessage(dataBytes),
	}, nil
}

// MarshalJSON ensures Event is serialized as compact JSON (single line)
func (e Event) MarshalJSON() ([]byte, error) {
	type Alias Event
	return json.Marshal((*Alias)(&e))
}
