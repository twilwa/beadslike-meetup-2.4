// ABOUTME: Tests for event type definitions and JSON serialization.
// ABOUTME: Covers round-trip marshaling, actor resolution, and single-line JSON format.
package tl

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEventSerialization verifies that events serialize to single-line JSON
func TestEventSerialization(t *testing.T) {
	data := CreateEventData{
		Title:       "Test Task",
		Description: "A test task",
		Status:      "open",
		Priority:    1,
		IssueType:   "task",
		Labels:      []string{"test", "urgent"},
	}

	evt, err := newEvent(EventCreate, "task-123", data)
	require.NoError(t, err)

	// Marshal to JSON
	jsonBytes, err := json.Marshal(evt)
	require.NoError(t, err)

	jsonStr := string(jsonBytes)

	// Verify it's a single line (no embedded newlines)
	assert.NotContains(t, jsonStr, "\n", "Event JSON should be single line")

	// Verify it's valid JSON
	var unmarshaled Event
	err = json.Unmarshal(jsonBytes, &unmarshaled)
	require.NoError(t, err)

	// Verify basic fields
	assert.Equal(t, EventCreate, unmarshaled.Type)
	assert.Equal(t, "task-123", unmarshaled.ID)
	assert.NotEmpty(t, unmarshaled.Timestamp)
	assert.NotEmpty(t, unmarshaled.Actor)
	assert.NotEmpty(t, unmarshaled.Data)
}

// TestEventRoundTrip verifies that events can be marshaled and unmarshaled without loss
func TestEventRoundTrip(t *testing.T) {
	tests := []struct {
		name      string
		eventType string
		taskID    string
		data      interface{}
	}{
		{
			name:      "create event",
			eventType: EventCreate,
			taskID:    "task-001",
			data: CreateEventData{
				Title:       "New Task",
				Description: "Task description",
				Status:      "open",
				Priority:    2,
				IssueType:   "feature",
				Labels:      []string{"feature", "backend"},
				Metadata: map[string]json.RawMessage{
					"custom": json.RawMessage(`"value"`),
				},
			},
		},
		{
			name:      "update event",
			eventType: EventUpdate,
			taskID:    "task-002",
			data: UpdateEventData{
				Fields: map[string]json.RawMessage{
					"status":   json.RawMessage(`"in_progress"`),
					"priority": json.RawMessage(`3`),
				},
			},
		},
		{
			name:      "close event",
			eventType: EventClose,
			taskID:    "task-003",
			data: CloseEventData{
				Reason: "Completed successfully",
			},
		},
		{
			name:      "reopen event",
			eventType: EventReopen,
			taskID:    "task-004",
			data:      ReopenEventData{},
		},
		{
			name:      "dep_add event",
			eventType: EventDepAdd,
			taskID:    "task-005",
			data: DepAddEventData{
				DependsOnID: "task-001",
				DepType:     "blocks",
			},
		},
		{
			name:      "dep_remove event",
			eventType: EventDepRemove,
			taskID:    "task-006",
			data: DepRemoveEventData{
				DependsOnID: "task-001",
			},
		},
		{
			name:      "claim event",
			eventType: EventClaim,
			taskID:    "task-007",
			data: ClaimEventData{
				Agent: "claude-agent",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create event
			evt, err := newEvent(tt.eventType, tt.taskID, tt.data)
			require.NoError(t, err)

			// Marshal to JSON
			jsonBytes, err := json.Marshal(evt)
			require.NoError(t, err)

			// Unmarshal back
			var unmarshaled Event
			err = json.Unmarshal(jsonBytes, &unmarshaled)
			require.NoError(t, err)

			// Verify fields match
			assert.Equal(t, evt.Type, unmarshaled.Type)
			assert.Equal(t, evt.ID, unmarshaled.ID)
			assert.Equal(t, evt.Actor, unmarshaled.Actor)
			// Timestamps may differ slightly due to precision, so just verify both are set
			assert.False(t, unmarshaled.Timestamp.IsZero())
			// Verify data is preserved
			assert.Equal(t, evt.Data, unmarshaled.Data)
		})
	}
}

// TestActorFromEnv verifies that resolveActor respects TL_ACTOR environment variable
func TestActorFromEnv(t *testing.T) {
	// Save original env var
	originalActor := os.Getenv("TL_ACTOR")
	defer func() {
		if originalActor != "" {
			os.Setenv("TL_ACTOR", originalActor)
		} else {
			os.Unsetenv("TL_ACTOR")
		}
	}()

	// Test with TL_ACTOR set
	os.Setenv("TL_ACTOR", "test-agent")
	actor := resolveActor()
	assert.Equal(t, "test-agent", actor)

	// Test with TL_ACTOR unset (should fall back to git or "unknown")
	os.Unsetenv("TL_ACTOR")
	actor = resolveActor()
	assert.NotEmpty(t, actor)
}

// TestEventTimestampUTC verifies that event timestamps are in UTC
func TestEventTimestampUTC(t *testing.T) {
	data := CreateEventData{
		Title:  "Test",
		Status: "open",
	}

	evt, err := newEvent(EventCreate, "task-123", data)
	require.NoError(t, err)

	// Verify timestamp is UTC
	assert.Equal(t, time.UTC, evt.Timestamp.Location())
}

// TestEventDataMarshaling verifies that complex data structures marshal correctly
func TestEventDataMarshaling(t *testing.T) {
	metadata := map[string]json.RawMessage{
		"custom_field": json.RawMessage(`"custom_value"`),
		"nested":       json.RawMessage(`{"key":"value"}`),
	}

	data := CreateEventData{
		Title:       "Complex Task",
		Description: "With metadata",
		Status:      "open",
		Priority:    1,
		IssueType:   "bug",
		Labels:      []string{"critical", "p0"},
		Metadata:    metadata,
	}

	evt, err := newEvent(EventCreate, "task-complex", data)
	require.NoError(t, err)

	// Marshal and unmarshal
	jsonBytes, err := json.Marshal(evt)
	require.NoError(t, err)

	var unmarshaled Event
	err = json.Unmarshal(jsonBytes, &unmarshaled)
	require.NoError(t, err)

	// Unmarshal the data back to CreateEventData
	var unmarshaledData CreateEventData
	err = json.Unmarshal(unmarshaled.Data, &unmarshaledData)
	require.NoError(t, err)

	// Verify metadata is preserved
	assert.Equal(t, metadata, unmarshaledData.Metadata)
	assert.Equal(t, data.Title, unmarshaledData.Title)
	assert.Equal(t, data.Labels, unmarshaledData.Labels)
}

// TestEventJSONFormat verifies the JSON format matches expected structure
func TestEventJSONFormat(t *testing.T) {
	data := CreateEventData{
		Title:  "Format Test",
		Status: "open",
	}

	evt, err := newEvent(EventCreate, "task-fmt", data)
	require.NoError(t, err)

	jsonBytes, err := json.Marshal(evt)
	require.NoError(t, err)

	// Parse as generic JSON to verify structure
	var parsed map[string]interface{}
	err = json.Unmarshal(jsonBytes, &parsed)
	require.NoError(t, err)

	// Verify required fields exist
	assert.Contains(t, parsed, "type")
	assert.Contains(t, parsed, "id")
	assert.Contains(t, parsed, "ts")
	assert.Contains(t, parsed, "actor")
	assert.Contains(t, parsed, "data")

	// Verify field values
	assert.Equal(t, EventCreate, parsed["type"])
	assert.Equal(t, "task-fmt", parsed["id"])
	assert.NotNil(t, parsed["ts"])
	assert.NotNil(t, parsed["actor"])
	assert.NotNil(t, parsed["data"])
}

// TestEventAppendFormat verifies that events can be appended as single lines
func TestEventAppendFormat(t *testing.T) {
	data1 := CreateEventData{
		Title:  "Task 1",
		Status: "open",
	}
	data2 := UpdateEventData{
		Fields: map[string]json.RawMessage{
			"status": json.RawMessage(`"in_progress"`),
		},
	}

	evt1, err := newEvent(EventCreate, "task-1", data1)
	require.NoError(t, err)

	evt2, err := newEvent(EventUpdate, "task-1", data2)
	require.NoError(t, err)

	// Marshal both events
	json1, err := json.Marshal(evt1)
	require.NoError(t, err)

	json2, err := json.Marshal(evt2)
	require.NoError(t, err)

	// Simulate appending to a file
	combined := string(json1) + "\n" + string(json2) + "\n"

	// Verify we can split and parse them back
	lines := strings.Split(strings.TrimSpace(combined), "\n")
	assert.Equal(t, 2, len(lines))

	var evt1Parsed, evt2Parsed Event
	err = json.Unmarshal([]byte(lines[0]), &evt1Parsed)
	require.NoError(t, err)

	err = json.Unmarshal([]byte(lines[1]), &evt2Parsed)
	require.NoError(t, err)

	assert.Equal(t, EventCreate, evt1Parsed.Type)
	assert.Equal(t, EventUpdate, evt2Parsed.Type)
}

// TestUpdateEventDataFields verifies that UpdateEventData can hold arbitrary field updates
func TestUpdateEventDataFields(t *testing.T) {
	fields := map[string]json.RawMessage{
		"title":       json.RawMessage(`"Updated Title"`),
		"status":      json.RawMessage(`"in_progress"`),
		"priority":    json.RawMessage(`5`),
		"labels":      json.RawMessage(`["updated","label"]`),
		"custom_data": json.RawMessage(`{"nested":"value"}`),
	}

	data := UpdateEventData{Fields: fields}

	evt, err := newEvent(EventUpdate, "task-update", data)
	require.NoError(t, err)

	// Marshal and unmarshal
	jsonBytes, err := json.Marshal(evt)
	require.NoError(t, err)

	var unmarshaled Event
	err = json.Unmarshal(jsonBytes, &unmarshaled)
	require.NoError(t, err)

	// Unmarshal the data
	var unmarshaledData UpdateEventData
	err = json.Unmarshal(unmarshaled.Data, &unmarshaledData)
	require.NoError(t, err)

	// Verify all fields are preserved
	assert.Equal(t, fields, unmarshaledData.Fields)
}
