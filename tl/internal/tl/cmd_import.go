// ABOUTME: Import command — reads beads JSONL and imports tasks with content-hash deduplication.
// ABOUTME: Implements `tl import --from <path>` preserving beads IDs and unknown fields.

package tl

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var importFromPath string

type importCounts struct {
	Imported int `json:"imported"`
	Updated  int `json:"updated"`
	Skipped  int `json:"skipped"`
}

func init() {
	importCmd.Flags().StringVar(&importFromPath, "from", ".beads/issues.jsonl", "Path to beads JSONL file")
	importCmd.RunE = runImport
}

func runImport(cmd *cobra.Command, args []string) error {
	dir, err := tlDir(GlobalOptions{Dir: tlDirFlag})
	if err != nil {
		return err
	}

	counts := importCounts{}
	err = mutate(dir, func(graph *Graph) ([]Event, error) {
		file, err := os.Open(importFromPath)
		if err != nil {
			return nil, err
		}
		defer file.Close()

		issues, err := ParseBeadsJSONL(file)
		if err != nil {
			return nil, err
		}

		events := make([]Event, 0, len(issues))
		for _, incoming := range issues {
			existing, found := graph.Tasks[incoming.ID]
			if !found {
				evt, err := buildCreateEvent(incoming)
				if err != nil {
					return nil, err
				}
				events = append(events, evt)
				counts.Imported++

				graph.Tasks[incoming.ID] = cloneIssue(incoming)
				continue
			}

			existingHash := ComputeContentHash(existing)
			incomingHash := ComputeContentHash(incoming)
			if existingHash == incomingHash {
				counts.Skipped++
				continue
			}

			if !incoming.UpdatedAt.After(existing.UpdatedAt) {
				counts.Skipped++
				continue
			}

			fields, err := buildUpdateFields(existing, incoming)
			if err != nil {
				return nil, err
			}
			if len(fields) == 0 {
				counts.Skipped++
				continue
			}

			evt, err := newEvent(EventUpdate, incoming.ID, UpdateEventData{Fields: fields})
			if err != nil {
				return nil, err
			}
			evt.Timestamp = incoming.UpdatedAt
			events = append(events, evt)
			counts.Updated++

			applyUpdateFields(existing, fields)
			existing.UpdatedAt = incoming.UpdatedAt
		}

		for _, issue := range issues {
			for _, dep := range issue.Dependencies {
				dependsOnID := dep.DependsOnID
				if dependsOnID == "" {
					continue
				}
				if issue.ID == dependsOnID {
					continue
				}
				if hasDependency(graph, issue.ID, dependsOnID) {
					continue
				}

				depType := string(dep.Type)
				if depType == "" {
					depType = string(DepBlocks)
				}

				evt, err := newEvent(EventDepAdd, issue.ID, DepAddEventData{
					DependsOnID: dependsOnID,
					DepType:     depType,
				})
				if err != nil {
					return nil, err
				}
				if !dep.CreatedAt.IsZero() {
					evt.Timestamp = dep.CreatedAt
				}
				events = append(events, evt)

				graph.Deps[issue.ID] = append(graph.Deps[issue.ID], dependsOnID)
				graph.RDeps[dependsOnID] = append(graph.RDeps[dependsOnID], issue.ID)
				if src, ok := graph.Tasks[issue.ID]; ok {
					src.Dependencies = append(src.Dependencies, &Dependency{
						IssueID:     issue.ID,
						DependsOnID: dependsOnID,
						Type:        DependencyType(depType),
						CreatedAt:   evt.Timestamp,
						CreatedBy:   dep.CreatedBy,
					})
				}
			}
		}

		return events, nil
	})
	if err != nil {
		return err
	}

	if jsonOutput {
		payload, err := json.Marshal(counts)
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(payload))
		return nil
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Imported %d, Updated %d, Skipped %d from %s\n", counts.Imported, counts.Updated, counts.Skipped, importFromPath)
	return nil
}

func buildCreateEvent(issue *Issue) (Event, error) {
	data := CreateEventData{
		Title:       issue.Title,
		Description: issue.Description,
		Status:      string(issue.Status),
		Priority:    issue.Priority,
		IssueType:   string(issue.IssueType),
		Labels:      issue.Labels,
		Metadata:    issue.Metadata,
	}

	evt, err := newEvent(EventCreate, issue.ID, data)
	if err != nil {
		return Event{}, err
	}
	evt.ID = issue.ID
	evt.Timestamp = issue.CreatedAt
	return evt, nil
}

func buildUpdateFields(existing *Issue, incoming *Issue) (map[string]json.RawMessage, error) {
	fields := make(map[string]json.RawMessage)

	if existing.Title != incoming.Title {
		raw, err := json.Marshal(incoming.Title)
		if err != nil {
			return nil, err
		}
		fields["title"] = raw
	}
	if existing.Description != incoming.Description {
		raw, err := json.Marshal(incoming.Description)
		if err != nil {
			return nil, err
		}
		fields["description"] = raw
	}
	if existing.Status != incoming.Status {
		raw, err := json.Marshal(incoming.Status)
		if err != nil {
			return nil, err
		}
		fields["status"] = raw
	}
	if existing.Priority != incoming.Priority {
		raw, err := json.Marshal(incoming.Priority)
		if err != nil {
			return nil, err
		}
		fields["priority"] = raw
	}
	if existing.Assignee != incoming.Assignee {
		raw, err := json.Marshal(incoming.Assignee)
		if err != nil {
			return nil, err
		}
		fields["assignee"] = raw
	}
	if existing.CloseReason != incoming.CloseReason {
		raw, err := json.Marshal(incoming.CloseReason)
		if err != nil {
			return nil, err
		}
		fields["close_reason"] = raw
	}
	if existing.IssueType != incoming.IssueType {
		raw, err := json.Marshal(incoming.IssueType)
		if err != nil {
			return nil, err
		}
		fields["issue_type"] = raw
	}
	if !stringSlicesEqual(existing.Labels, incoming.Labels) {
		raw, err := json.Marshal(incoming.Labels)
		if err != nil {
			return nil, err
		}
		fields["labels"] = raw
	}

	for key, value := range incoming.Metadata {
		if existing.Metadata == nil || !bytes.Equal(existing.Metadata[key], value) {
			fields[key] = cloneRawMessage(value)
		}
	}

	return fields, nil
}

func applyUpdateFields(issue *Issue, fields map[string]json.RawMessage) {
	for field, value := range fields {
		switch field {
		case "status":
			var v Status
			_ = json.Unmarshal(value, &v)
			issue.Status = v
		case "title":
			var v string
			_ = json.Unmarshal(value, &v)
			issue.Title = v
		case "description":
			var v string
			_ = json.Unmarshal(value, &v)
			issue.Description = v
		case "priority":
			var v int
			_ = json.Unmarshal(value, &v)
			issue.Priority = v
		case "assignee":
			var v string
			_ = json.Unmarshal(value, &v)
			issue.Assignee = v
		case "close_reason":
			var v string
			_ = json.Unmarshal(value, &v)
			issue.CloseReason = v
		case "issue_type":
			var v IssueType
			_ = json.Unmarshal(value, &v)
			issue.IssueType = v
		case "labels":
			var v []string
			_ = json.Unmarshal(value, &v)
			issue.Labels = v
		default:
			if issue.Metadata == nil {
				issue.Metadata = make(map[string]json.RawMessage)
			}
			issue.Metadata[field] = cloneRawMessage(value)
		}
	}
}

func hasDependency(graph *Graph, issueID string, dependsOnID string) bool {
	for _, existing := range graph.Deps[issueID] {
		if existing == dependsOnID {
			return true
		}
	}
	return false
}

func cloneIssue(issue *Issue) *Issue {
	if issue == nil {
		return nil
	}

	cloned := *issue
	cloned.Labels = append([]string(nil), issue.Labels...)
	if issue.Metadata != nil {
		cloned.Metadata = make(map[string]json.RawMessage, len(issue.Metadata))
		for k, v := range issue.Metadata {
			cloned.Metadata[k] = cloneRawMessage(v)
		}
	}
	if issue.Dependencies != nil {
		cloned.Dependencies = make([]*Dependency, 0, len(issue.Dependencies))
		for _, dep := range issue.Dependencies {
			if dep == nil {
				continue
			}
			d := *dep
			d.Metadata = cloneRawMessage(dep.Metadata)
			cloned.Dependencies = append(cloned.Dependencies, &d)
		}
	}

	return &cloned
}

func cloneRawMessage(raw json.RawMessage) json.RawMessage {
	if raw == nil {
		return nil
	}
	return append(json.RawMessage(nil), raw...)
}

func stringSlicesEqual(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
