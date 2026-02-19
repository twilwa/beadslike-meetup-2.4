// ABOUTME: Parses beads JSONL format into Issue structs with unknown field preservation
// ABOUTME: Handles truncated final lines gracefully and maps beads fields to tl Issue model

package tl

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// ParseBeadsJSONL reads a beads JSONL file and returns a slice of Issues.
// Each line is expected to be a JSON object representing a beads Issue.
// Unknown fields are preserved in the Metadata map.
// If the file does not end with a newline, the final incomplete line is skipped without error.
func ParseBeadsJSONL(r io.Reader) ([]*Issue, error) {
	var issues []*Issue
	scanner := bufio.NewScanner(r)

	// Check if input is a file to detect truncation
	var endsWithNewline bool
	if f, ok := r.(*os.File); ok {
		info, err := f.Stat()
		if err == nil && info.Size() > 0 {
			last := make([]byte, 1)
			if _, err := f.ReadAt(last, info.Size()-1); err == nil {
				endsWithNewline = last[0] == '\n'
			}
		}
	} else {
		// For non-file readers, assume proper termination
		endsWithNewline = true
	}

	lineNo := 0
	var pending []byte
	var pendingNo int

	for scanner.Scan() {
		lineNo++
		line := scanner.Bytes()
		trimmed := make([]byte, len(line))
		copy(trimmed, line)

		// Skip empty lines
		if len(trimmed) == 0 {
			continue
		}

		// Process the previous pending line (if any)
		if pending != nil {
			issue, err := parseBeadsLine(pending)
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", pendingNo, err)
			}
			issues = append(issues, issue)
		}

		// Store current line as pending
		pending = trimmed
		pendingNo = lineNo
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanner error: %w", err)
	}

	// Process the final pending line with truncation tolerance
	if pending != nil {
		issue, err := parseBeadsLine(pending)
		if err != nil {
			// Only ignore if file doesn't end with newline (truncated)
			if !endsWithNewline {
				return issues, nil
			}
			return nil, fmt.Errorf("line %d: %w", pendingNo, err)
		}
		issues = append(issues, issue)
	}

	return issues, nil
}

// parseBeadsLine unmarshals a single JSONL line into an Issue.
// It first unmarshals into a map to capture all fields, then maps known fields
// to the Issue struct and preserves unknown fields in Metadata.
func parseBeadsLine(line []byte) (*Issue, error) {
	// First pass: unmarshal into raw map to capture all fields
	var rawMap map[string]json.RawMessage
	if err := json.Unmarshal(line, &rawMap); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	issue := &Issue{
		Metadata: make(map[string]json.RawMessage),
	}

	// Known beads fields that map to Issue struct
	knownFields := map[string]bool{
		"id":                  true,
		"title":               true,
		"description":         true,
		"design":              true,
		"acceptance_criteria": true,
		"notes":               true,
		"spec_id":             true,
		"status":              true,
		"priority":            true,
		"issue_type":          true,
		"assignee":            true,
		"owner":               true,
		"created_by":          true,
		"created_at":          true,
		"updated_at":          true,
		"closed_at":           true,
		"close_reason":        true,
		"defer_until":         true,
		"labels":              true,
		"dependencies":        true,
		"pinned":              true,
		"ephemeral":           true,
	}

	// Process each field
	for key, value := range rawMap {
		if !knownFields[key] {
			// Unknown field: preserve in Metadata
			issue.Metadata[key] = value
			continue
		}

		// Known field: unmarshal into struct
		switch key {
		case "id":
			if err := json.Unmarshal(value, &issue.ID); err != nil {
				return nil, fmt.Errorf("invalid id: %w", err)
			}
		case "title":
			if err := json.Unmarshal(value, &issue.Title); err != nil {
				return nil, fmt.Errorf("invalid title: %w", err)
			}
		case "description":
			if err := json.Unmarshal(value, &issue.Description); err != nil {
				return nil, fmt.Errorf("invalid description: %w", err)
			}
		case "design":
			if err := json.Unmarshal(value, &issue.Design); err != nil {
				return nil, fmt.Errorf("invalid design: %w", err)
			}
		case "acceptance_criteria":
			if err := json.Unmarshal(value, &issue.AcceptanceCriteria); err != nil {
				return nil, fmt.Errorf("invalid acceptance_criteria: %w", err)
			}
		case "notes":
			if err := json.Unmarshal(value, &issue.Notes); err != nil {
				return nil, fmt.Errorf("invalid notes: %w", err)
			}
		case "spec_id":
			if err := json.Unmarshal(value, &issue.SpecID); err != nil {
				return nil, fmt.Errorf("invalid spec_id: %w", err)
			}
		case "status":
			if err := json.Unmarshal(value, &issue.Status); err != nil {
				return nil, fmt.Errorf("invalid status: %w", err)
			}
		case "priority":
			if err := json.Unmarshal(value, &issue.Priority); err != nil {
				return nil, fmt.Errorf("invalid priority: %w", err)
			}
		case "issue_type":
			if err := json.Unmarshal(value, &issue.IssueType); err != nil {
				return nil, fmt.Errorf("invalid issue_type: %w", err)
			}
		case "assignee":
			if err := json.Unmarshal(value, &issue.Assignee); err != nil {
				return nil, fmt.Errorf("invalid assignee: %w", err)
			}
		case "owner":
			if err := json.Unmarshal(value, &issue.Owner); err != nil {
				return nil, fmt.Errorf("invalid owner: %w", err)
			}
		case "created_by":
			if err := json.Unmarshal(value, &issue.CreatedBy); err != nil {
				return nil, fmt.Errorf("invalid created_by: %w", err)
			}
		case "created_at":
			if err := json.Unmarshal(value, &issue.CreatedAt); err != nil {
				return nil, fmt.Errorf("invalid created_at: %w", err)
			}
		case "updated_at":
			if err := json.Unmarshal(value, &issue.UpdatedAt); err != nil {
				return nil, fmt.Errorf("invalid updated_at: %w", err)
			}
		case "closed_at":
			if err := json.Unmarshal(value, &issue.ClosedAt); err != nil {
				return nil, fmt.Errorf("invalid closed_at: %w", err)
			}
		case "close_reason":
			if err := json.Unmarshal(value, &issue.CloseReason); err != nil {
				return nil, fmt.Errorf("invalid close_reason: %w", err)
			}
		case "defer_until":
			if err := json.Unmarshal(value, &issue.DeferUntil); err != nil {
				return nil, fmt.Errorf("invalid defer_until: %w", err)
			}
		case "labels":
			if err := json.Unmarshal(value, &issue.Labels); err != nil {
				return nil, fmt.Errorf("invalid labels: %w", err)
			}
		case "dependencies":
			var deps []*Dependency
			if err := json.Unmarshal(value, &deps); err != nil {
				return nil, fmt.Errorf("invalid dependencies: %w", err)
			}
			issue.Dependencies = deps
		case "pinned":
			if err := json.Unmarshal(value, &issue.Pinned); err != nil {
				return nil, fmt.Errorf("invalid pinned: %w", err)
			}
		case "ephemeral":
			if err := json.Unmarshal(value, &issue.Ephemeral); err != nil {
				return nil, fmt.Errorf("invalid ephemeral: %w", err)
			}
		}
	}

	// Clean up empty metadata
	if len(issue.Metadata) == 0 {
		issue.Metadata = nil
	}

	return issue, nil
}
