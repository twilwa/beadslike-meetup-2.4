// ABOUTME: Append-only JSONL event store with replay-on-read for tl task management.
// ABOUTME: All mutations are serialized under flock; reads replay events to build in-memory graph.

package tl

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const (
	tlDirName      = ".tl"
	eventsFileName = "events.jsonl"
	lockFileName   = "lock"
)

type GlobalOptions struct {
	JSON bool
	Dir  string
}

func resolveTLDir(start string) (string, error) {
	current := start
	for {
		candidate := filepath.Join(current, tlDirName)
		info, err := os.Stat(candidate)
		if err == nil {
			if info.IsDir() {
				return candidate, nil
			}
			return "", fmt.Errorf("%s exists but is not a directory", candidate)
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		if current == filepath.Dir(current) {
			break
		}
		current = filepath.Dir(current)
	}

	if filepath.Base(start) == tlDirName {
		info, err := os.Stat(start)
		if err == nil {
			if info.IsDir() {
				return start, nil
			}
			return "", fmt.Errorf("%s exists but is not a directory", start)
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
	}

	return "", ErrNoTLDir
}

func tlDir(opts GlobalOptions) (string, error) {
	start := opts.Dir
	if start == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		start = wd
	}
	return resolveTLDir(start)
}

func initDir(path string) error {
	dirPath := filepath.Join(path, tlDirName)
	if _, err := os.Stat(dirPath); err == nil {
		return fmt.Errorf("already initialized at %s", path)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return err
	}

	if err := createEmptyFile(filepath.Join(dirPath, eventsFileName)); err != nil {
		return err
	}
	if err := createEmptyFile(filepath.Join(dirPath, lockFileName)); err != nil {
		return err
	}
	return nil
}

func createEmptyFile(path string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	return f.Close()
}

func readEvents(path string) ([]Event, error) {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	const maxEventLineBytes = 10 * 1024 * 1024

	endsWithNewline := false
	if info, err := file.Stat(); err == nil && info.Size() > 0 {
		last := make([]byte, 1)
		if _, err := file.ReadAt(last, info.Size()-1); err == nil {
			endsWithNewline = last[0] == '\n'
		}
	}

	var events []Event
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), maxEventLineBytes)
	var pending []byte
	pendingNo := 0
	currentNo := 0

	processLine := func(lineNo int, line []byte) error {
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) == 0 {
			return nil
		}
		var event Event
		if err := json.Unmarshal(trimmed, &event); err != nil {
			return fmt.Errorf("%s:%d: invalid JSON in events log: %w", path, lineNo, err)
		}
		events = append(events, event)
		return nil
	}

	for scanner.Scan() {
		currentNo++
		line := append([]byte(nil), scanner.Bytes()...)
		if pending != nil {
			if err := processLine(pendingNo, pending); err != nil {
				return nil, err
			}
		}
		pending = line
		pendingNo = currentNo
	}
	if err := scanner.Err(); err != nil {
		if errors.Is(err, bufio.ErrTooLong) {
			return nil, fmt.Errorf("%s: event line too long (> %d bytes); file may be corrupted", path, maxEventLineBytes)
		}
		return nil, err
	}

	if pending != nil {
		if err := processLine(pendingNo, pending); err != nil {
			if !endsWithNewline {
				return events, nil
			}
			return nil, err
		}
	}
	return events, nil
}

func replayEvents(events []Event) (*Graph, error) {
	graph := &Graph{
		Tasks: make(map[string]*Issue),
		Deps:  make(map[string][]string),
		RDeps: make(map[string][]string),
	}

	for _, event := range events {
		switch event.Type {
		case EventCreate:
			var data CreateEventData
			if err := json.Unmarshal(event.Data, &data); err != nil {
				return nil, err
			}
			status := Status(data.Status)
			if status == "" {
				status = StatusOpen
			}
			graph.Tasks[event.ID] = &Issue{
				ID:          event.ID,
				Title:       data.Title,
				Description: data.Description,
				Status:      status,
				Priority:    data.Priority,
				IssueType:   IssueType(data.IssueType),
				Labels:      data.Labels,
				CreatedAt:   event.Timestamp,
				UpdatedAt:   event.Timestamp,
				Metadata:    data.Metadata,
			}

		case EventUpdate:
			var data UpdateEventData
			if err := json.Unmarshal(event.Data, &data); err != nil {
				return nil, err
			}
			issue, ok := graph.Tasks[event.ID]
			if !ok {
				continue
			}

			for field, value := range data.Fields {
				switch field {
				case "status":
					var status Status
					if err := json.Unmarshal(value, &status); err != nil {
						return nil, err
					}
					issue.Status = status
				case "title":
					var title string
					if err := json.Unmarshal(value, &title); err != nil {
						return nil, err
					}
					issue.Title = title
				case "description":
					var description string
					if err := json.Unmarshal(value, &description); err != nil {
						return nil, err
					}
					issue.Description = description
				case "priority":
					var priority int
					if err := json.Unmarshal(value, &priority); err != nil {
						return nil, err
					}
					issue.Priority = priority
				case "assignee":
					var assignee string
					if err := json.Unmarshal(value, &assignee); err != nil {
						return nil, err
					}
					issue.Assignee = assignee
				case "close_reason":
					var closeReason string
					if err := json.Unmarshal(value, &closeReason); err != nil {
						return nil, err
					}
					issue.CloseReason = closeReason
				default:
					if issue.Metadata == nil {
						issue.Metadata = make(map[string]json.RawMessage)
					}
					issue.Metadata[field] = value
				}
			}
			issue.UpdatedAt = event.Timestamp

		case EventClose:
			var data CloseEventData
			if err := json.Unmarshal(event.Data, &data); err != nil {
				return nil, err
			}
			issue, ok := graph.Tasks[event.ID]
			if !ok {
				continue
			}
			issue.Status = StatusClosed
			issue.CloseReason = data.Reason
			closedAt := event.Timestamp
			issue.ClosedAt = &closedAt
			issue.UpdatedAt = event.Timestamp
			clearBlockingEdges(graph, event.ID)

		case EventReopen:
			issue, ok := graph.Tasks[event.ID]
			if !ok {
				continue
			}
			issue.Status = StatusOpen
			issue.ClosedAt = nil
			issue.CloseReason = ""
			issue.Assignee = ""
			issue.UpdatedAt = event.Timestamp

		case EventClaim:
			var data ClaimEventData
			if err := json.Unmarshal(event.Data, &data); err != nil {
				return nil, err
			}
			issue, ok := graph.Tasks[event.ID]
			if !ok {
				continue
			}
			issue.Status = StatusInProgress
			issue.Assignee = data.Agent
			issue.UpdatedAt = event.Timestamp

		case EventDepAdd:
			var data DepAddEventData
			if err := json.Unmarshal(event.Data, &data); err != nil {
				return nil, err
			}
			graph.Deps[event.ID] = append(graph.Deps[event.ID], data.DependsOnID)
			graph.RDeps[data.DependsOnID] = append(graph.RDeps[data.DependsOnID], event.ID)
			issue, ok := graph.Tasks[event.ID]
			if !ok {
				continue
			}
			issue.Dependencies = append(issue.Dependencies, &Dependency{
				IssueID:     event.ID,
				DependsOnID: data.DependsOnID,
				Type:        DependencyType(data.DepType),
				CreatedAt:   event.Timestamp,
				CreatedBy:   event.Actor,
			})

		case EventDepRemove:
			var data DepRemoveEventData
			if err := json.Unmarshal(event.Data, &data); err != nil {
				return nil, err
			}
			graph.Deps[event.ID] = removeString(graph.Deps[event.ID], data.DependsOnID)
			if len(graph.Deps[event.ID]) == 0 {
				delete(graph.Deps, event.ID)
			}

			graph.RDeps[data.DependsOnID] = removeString(graph.RDeps[data.DependsOnID], event.ID)
			if len(graph.RDeps[data.DependsOnID]) == 0 {
				delete(graph.RDeps, data.DependsOnID)
			}

			issue, ok := graph.Tasks[event.ID]
			if !ok {
				continue
			}
			issue.Dependencies = removeDependency(issue.Dependencies, data.DependsOnID)

		default:
			continue
		}
	}

	return graph, nil
}

func clearBlockingEdges(graph *Graph, closedID string) {
	blocked := append([]string(nil), graph.RDeps[closedID]...)
	for _, blockedID := range blocked {
		graph.Deps[blockedID] = removeString(graph.Deps[blockedID], closedID)
		if len(graph.Deps[blockedID]) == 0 {
			delete(graph.Deps, blockedID)
		}
		issue, ok := graph.Tasks[blockedID]
		if !ok {
			continue
		}
		issue.Dependencies = removeDependency(issue.Dependencies, closedID)
	}
	delete(graph.RDeps, closedID)
}

func removeString(values []string, target string) []string {
	if len(values) == 0 {
		return values
	}
	out := values[:0]
	for _, value := range values {
		if value != target {
			out = append(out, value)
		}
	}
	return out
}

func removeDependency(deps []*Dependency, dependsOnID string) []*Dependency {
	if len(deps) == 0 {
		return deps
	}
	out := deps[:0]
	for _, dep := range deps {
		if dep.DependsOnID != dependsOnID {
			out = append(out, dep)
		}
	}
	return out
}

func loadGraph(dir string) (*Graph, error) {
	eventsPath := filepath.Join(dir, eventsFileName)
	events, err := readEvents(eventsPath)
	if err != nil {
		return nil, err
	}
	return replayEvents(events)
}

func appendEventsToFile(path string, events []Event) error {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, event := range events {
		data, err := json.Marshal(event)
		if err != nil {
			return err
		}
		line := append(data, '\n')
		if err := writeAll(file, line); err != nil {
			return err
		}
	}
	return nil
}

func writeAll(w io.Writer, data []byte) error {
	for len(data) > 0 {
		n, err := w.Write(data)
		if err != nil {
			return err
		}
		data = data[n:]
	}
	return nil
}

func generateID() string {
	bytes := make([]byte, 4)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return "tl-" + hex.EncodeToString(bytes)[:4]
}

func mutate(dir string, fn func(*Graph) ([]Event, error)) error {
	lockPath := filepath.Join(dir, lockFileName)
	eventsPath := filepath.Join(dir, eventsFileName)
	return withLock(lockPath, func() error {
		graph, err := loadGraph(dir)
		if err != nil {
			return err
		}
		events, err := fn(graph)
		if err != nil {
			return err
		}
		return appendEventsToFile(eventsPath, events)
	})
}
