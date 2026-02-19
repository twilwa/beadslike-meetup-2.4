// ABOUTME: Tests for content hash utility ensuring determinism and correctness
// ABOUTME: Validates null-byte separator pattern and collision resistance

package tl

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestHashDeterministic verifies that the same Issue produces the same hash
func TestHashDeterministic(t *testing.T) {
	issue := &Issue{
		ID:                 "test-1",
		Title:              "Test Issue",
		Description:        "This is a test",
		Design:             "Design notes",
		AcceptanceCriteria: "AC1, AC2",
		Notes:              "Some notes",
		SpecID:             "spec-123",
		Status:             StatusOpen,
		Priority:           1,
		IssueType:          TypeFeature,
		Assignee:           "alice",
		Owner:              "bob",
		CreatedBy:          "charlie",
		Pinned:             false,
		Metadata:           nil,
	}

	hash1 := ComputeContentHash(issue)
	hash2 := ComputeContentHash(issue)

	assert.Equal(t, hash1, hash2, "same issue should produce same hash")
}

// TestHashDifferentContent verifies that different Issues produce different hashes
func TestHashDifferentContent(t *testing.T) {
	issue1 := &Issue{
		ID:                 "test-1",
		Title:              "Issue One",
		Description:        "Description 1",
		Design:             "",
		AcceptanceCriteria: "",
		Notes:              "",
		SpecID:             "",
		Status:             StatusOpen,
		Priority:           1,
		IssueType:          TypeFeature,
		Assignee:           "",
		Owner:              "",
		CreatedBy:          "",
		Pinned:             false,
		Metadata:           nil,
	}

	issue2 := &Issue{
		ID:                 "test-2",
		Title:              "Issue Two",
		Description:        "Description 2",
		Design:             "",
		AcceptanceCriteria: "",
		Notes:              "",
		SpecID:             "",
		Status:             StatusOpen,
		Priority:           1,
		IssueType:          TypeFeature,
		Assignee:           "",
		Owner:              "",
		CreatedBy:          "",
		Pinned:             false,
		Metadata:           nil,
	}

	hash1 := ComputeContentHash(issue1)
	hash2 := ComputeContentHash(issue2)

	assert.NotEqual(t, hash1, hash2, "different issues should produce different hashes")
}

// TestHashNullSeparator verifies that null-byte separators prevent concatenation collisions
// Issue{Title:"AB"} should NOT equal Issue{Title:"A", Description:"B"}
func TestHashNullSeparator(t *testing.T) {
	issue1 := &Issue{
		ID:                 "test-1",
		Title:              "AB",
		Description:        "",
		Design:             "",
		AcceptanceCriteria: "",
		Notes:              "",
		SpecID:             "",
		Status:             StatusOpen,
		Priority:           0,
		IssueType:          TypeTask,
		Assignee:           "",
		Owner:              "",
		CreatedBy:          "",
		Pinned:             false,
		Metadata:           nil,
	}

	issue2 := &Issue{
		ID:                 "test-2",
		Title:              "A",
		Description:        "B",
		Design:             "",
		AcceptanceCriteria: "",
		Notes:              "",
		SpecID:             "",
		Status:             StatusOpen,
		Priority:           0,
		IssueType:          TypeTask,
		Assignee:           "",
		Owner:              "",
		CreatedBy:          "",
		Pinned:             false,
		Metadata:           nil,
	}

	hash1 := ComputeContentHash(issue1)
	hash2 := ComputeContentHash(issue2)

	assert.NotEqual(t, hash1, hash2, "null-byte separators should prevent concatenation collisions")
}

// TestHashPriorityZero verifies that different priority values produce different hashes
func TestHashPriorityZero(t *testing.T) {
	issue1 := &Issue{
		ID:                 "test-1",
		Title:              "Test",
		Description:        "",
		Design:             "",
		AcceptanceCriteria: "",
		Notes:              "",
		SpecID:             "",
		Status:             StatusOpen,
		Priority:           0,
		IssueType:          TypeTask,
		Assignee:           "",
		Owner:              "",
		CreatedBy:          "",
		Pinned:             false,
		Metadata:           nil,
	}

	issue2 := &Issue{
		ID:                 "test-2",
		Title:              "Test",
		Description:        "",
		Design:             "",
		AcceptanceCriteria: "",
		Notes:              "",
		SpecID:             "",
		Status:             StatusOpen,
		Priority:           1,
		IssueType:          TypeTask,
		Assignee:           "",
		Owner:              "",
		CreatedBy:          "",
		Pinned:             false,
		Metadata:           nil,
	}

	hash1 := ComputeContentHash(issue1)
	hash2 := ComputeContentHash(issue2)

	assert.NotEqual(t, hash1, hash2, "different priority values should produce different hashes")
}

// TestHashPinnedFlag verifies that the pinned flag affects the hash
func TestHashPinnedFlag(t *testing.T) {
	baseIssue := &Issue{
		ID:                 "test-1",
		Title:              "Test",
		Description:        "",
		Design:             "",
		AcceptanceCriteria: "",
		Notes:              "",
		SpecID:             "",
		Status:             StatusOpen,
		Priority:           1,
		IssueType:          TypeTask,
		Assignee:           "",
		Owner:              "",
		CreatedBy:          "",
		Pinned:             false,
		Metadata:           nil,
	}

	pinnedIssue := &Issue{
		ID:                 "test-2",
		Title:              "Test",
		Description:        "",
		Design:             "",
		AcceptanceCriteria: "",
		Notes:              "",
		SpecID:             "",
		Status:             StatusOpen,
		Priority:           1,
		IssueType:          TypeTask,
		Assignee:           "",
		Owner:              "",
		CreatedBy:          "",
		Pinned:             true,
		Metadata:           nil,
	}

	hash1 := ComputeContentHash(baseIssue)
	hash2 := ComputeContentHash(pinnedIssue)

	assert.NotEqual(t, hash1, hash2, "pinned flag should affect the hash")
}

// TestHashMetadata verifies that metadata affects the hash
func TestHashMetadata(t *testing.T) {
	issue1 := &Issue{
		ID:                 "test-1",
		Title:              "Test",
		Description:        "",
		Design:             "",
		AcceptanceCriteria: "",
		Notes:              "",
		SpecID:             "",
		Status:             StatusOpen,
		Priority:           1,
		IssueType:          TypeTask,
		Assignee:           "",
		Owner:              "",
		CreatedBy:          "",
		Pinned:             false,
		Metadata:           nil,
	}

	metadataValue, _ := json.Marshal(map[string]string{"key": "value"})
	issue2 := &Issue{
		ID:                 "test-2",
		Title:              "Test",
		Description:        "",
		Design:             "",
		AcceptanceCriteria: "",
		Notes:              "",
		SpecID:             "",
		Status:             StatusOpen,
		Priority:           1,
		IssueType:          TypeTask,
		Assignee:           "",
		Owner:              "",
		CreatedBy:          "",
		Pinned:             false,
		Metadata: map[string]json.RawMessage{
			"key": metadataValue,
		},
	}

	hash1 := ComputeContentHash(issue1)
	hash2 := ComputeContentHash(issue2)

	assert.NotEqual(t, hash1, hash2, "different metadata should produce different hashes")
}
