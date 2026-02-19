// ABOUTME: Content hash utility for deduplication during beads import
// ABOUTME: Provides deterministic SHA256 hashing of Issue content fields

package tl

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"hash"
)

// ComputeContentHash creates a deterministic hash of the issue's content.
// Uses all substantive fields (excluding ID, timestamps, and compaction metadata)
// to ensure that identical content produces identical hashes across all clones.
func ComputeContentHash(issue *Issue) string {
	h := sha256.New()
	w := hashFieldWriter{h}

	// Core fields in stable order (matching beads for import dedup compatibility)
	w.str(issue.Title)
	w.str(issue.Description)
	w.str(issue.Design)
	w.str(issue.AcceptanceCriteria)
	w.str(issue.Notes)
	w.str(issue.SpecID)
	w.str(string(issue.Status))
	w.int(issue.Priority)
	w.str(string(issue.IssueType))
	w.str(issue.Assignee)
	w.str(issue.Owner)
	w.str(issue.CreatedBy)

	// Optional fields (write null separator even if field doesn't exist in tl.Issue)
	w.flag(issue.Pinned, "pinned")

	// Metadata as JSON string
	if issue.Metadata != nil {
		metadataJSON, _ := json.Marshal(issue.Metadata)
		w.h.Write(metadataJSON)
	}
	w.h.Write([]byte{0})

	return fmt.Sprintf("%x", h.Sum(nil))
}

// hashFieldWriter provides helper methods for writing fields to a hash.
// Each method writes the value followed by a null separator for consistency.
type hashFieldWriter struct {
	h hash.Hash
}

func (w hashFieldWriter) str(s string) {
	w.h.Write([]byte(s))
	w.h.Write([]byte{0})
}

func (w hashFieldWriter) int(n int) {
	w.h.Write([]byte(fmt.Sprintf("%d", n)))
	w.h.Write([]byte{0})
}

func (w hashFieldWriter) flag(b bool, label string) {
	if b {
		w.h.Write([]byte(label))
	}
	w.h.Write([]byte{0})
}
