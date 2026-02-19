// ABOUTME: Tests for tl event store initialization, directory resolution, event replay, and ID generation.
// ABOUTME: Verifies append-only mutation flow and graph reconstruction from events.jsonl.

package tl

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestInitDir(t *testing.T) {
	root := t.TempDir()

	err := initDir(root)
	assert.NoError(t, err)

	info, err := os.Stat(filepath.Join(root, tlDirName))
	assert.NoError(t, err)
	assert.True(t, info.IsDir())

	_, err = os.Stat(filepath.Join(root, tlDirName, eventsFileName))
	assert.NoError(t, err)

	_, err = os.Stat(filepath.Join(root, tlDirName, lockFileName))
	assert.NoError(t, err)
}

func TestInitDirAlreadyExists(t *testing.T) {
	root := t.TempDir()

	err := initDir(root)
	assert.NoError(t, err)

	err = initDir(root)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "already initialized")
}

func TestResolveTLDir(t *testing.T) {
	root := t.TempDir()
	err := os.Mkdir(filepath.Join(root, tlDirName), 0755)
	assert.NoError(t, err)

	start := filepath.Join(root, "a", "b", "c")
	err = os.MkdirAll(start, 0755)
	assert.NoError(t, err)

	dir, err := resolveTLDir(start)
	assert.NoError(t, err)
	assert.Equal(t, filepath.Join(root, tlDirName), dir)
}

func TestLoadGraphEmpty(t *testing.T) {
	root := t.TempDir()
	err := initDir(root)
	assert.NoError(t, err)

	graph, err := loadGraph(filepath.Join(root, tlDirName))
	assert.NoError(t, err)
	assert.NotNil(t, graph)
	assert.Empty(t, graph.Tasks)
	assert.Empty(t, graph.Deps)
	assert.Empty(t, graph.RDeps)
}

func TestCreateAndReplay(t *testing.T) {
	root := t.TempDir()
	err := initDir(root)
	assert.NoError(t, err)
	dir := filepath.Join(root, tlDirName)
	ts := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	err = mutate(dir, func(_ *Graph) ([]Event, error) {
		data, err := json.Marshal(CreateEventData{
			Title:       "Implement store",
			Description: "foundational storage",
			Status:      string(StatusOpen),
			Priority:    2,
			IssueType:   string(TypeTask),
		})
		if err != nil {
			return nil, err
		}
		return []Event{{Type: EventCreate, ID: "tl-abcd", Timestamp: ts, Actor: "tester", Data: data}}, nil
	})
	assert.NoError(t, err)

	graph, err := loadGraph(dir)
	assert.NoError(t, err)

	issue, ok := graph.Tasks["tl-abcd"]
	assert.True(t, ok)
	assert.Equal(t, "Implement store", issue.Title)
	assert.Equal(t, "foundational storage", issue.Description)
	assert.Equal(t, StatusOpen, issue.Status)
	assert.Equal(t, 2, issue.Priority)
	assert.Equal(t, ts, issue.CreatedAt)
	assert.Equal(t, ts, issue.UpdatedAt)
}

func TestMultiEventReplay(t *testing.T) {
	root := t.TempDir()
	err := initDir(root)
	assert.NoError(t, err)
	dir := filepath.Join(root, tlDirName)
	t0 := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
	t1 := t0.Add(time.Minute)
	t2 := t1.Add(time.Minute)

	err = mutate(dir, func(_ *Graph) ([]Event, error) {
		createData, err := json.Marshal(CreateEventData{Title: "Task", Status: string(StatusOpen)})
		if err != nil {
			return nil, err
		}
		updateData, err := json.Marshal(UpdateEventData{Fields: map[string]json.RawMessage{"status": json.RawMessage(`"in_progress"`)}})
		if err != nil {
			return nil, err
		}
		closeData, err := json.Marshal(CloseEventData{Reason: "done"})
		if err != nil {
			return nil, err
		}

		return []Event{
			{Type: EventCreate, ID: "tl-1111", Timestamp: t0, Actor: "tester", Data: createData},
			{Type: EventUpdate, ID: "tl-1111", Timestamp: t1, Actor: "tester", Data: updateData},
			{Type: EventClose, ID: "tl-1111", Timestamp: t2, Actor: "tester", Data: closeData},
		}, nil
	})
	assert.NoError(t, err)

	graph, err := loadGraph(dir)
	assert.NoError(t, err)

	issue, ok := graph.Tasks["tl-1111"]
	assert.True(t, ok)
	assert.Equal(t, StatusClosed, issue.Status)
	assert.Equal(t, "done", issue.CloseReason)
	if assert.NotNil(t, issue.ClosedAt) {
		assert.Equal(t, t2, *issue.ClosedAt)
	}
	assert.Equal(t, t2, issue.UpdatedAt)
}

func TestDepAddReplay(t *testing.T) {
	root := t.TempDir()
	err := initDir(root)
	assert.NoError(t, err)
	dir := filepath.Join(root, tlDirName)
	t0 := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)

	err = mutate(dir, func(_ *Graph) ([]Event, error) {
		createA, err := json.Marshal(CreateEventData{Title: "A", Status: string(StatusOpen)})
		if err != nil {
			return nil, err
		}
		createB, err := json.Marshal(CreateEventData{Title: "B", Status: string(StatusOpen)})
		if err != nil {
			return nil, err
		}
		depAdd, err := json.Marshal(DepAddEventData{DependsOnID: "tl-bbbb", DepType: string(DepBlocks)})
		if err != nil {
			return nil, err
		}
		return []Event{
			{Type: EventCreate, ID: "tl-aaaa", Timestamp: t0, Actor: "tester", Data: createA},
			{Type: EventCreate, ID: "tl-bbbb", Timestamp: t0.Add(time.Minute), Actor: "tester", Data: createB},
			{Type: EventDepAdd, ID: "tl-aaaa", Timestamp: t0.Add(2 * time.Minute), Actor: "tester", Data: depAdd},
		}, nil
	})
	assert.NoError(t, err)

	graph, err := loadGraph(dir)
	assert.NoError(t, err)

	assert.Equal(t, []string{"tl-bbbb"}, graph.Deps["tl-aaaa"])
	assert.Equal(t, []string{"tl-aaaa"}, graph.RDeps["tl-bbbb"])
	if assert.Contains(t, graph.Tasks, "tl-aaaa") {
		assert.Len(t, graph.Tasks["tl-aaaa"].Dependencies, 1)
		assert.Equal(t, "tl-bbbb", graph.Tasks["tl-aaaa"].Dependencies[0].DependsOnID)
	}
}

func TestGenerateID(t *testing.T) {
	id := generateID()
	assert.True(t, len(id) == 7)
	assert.Contains(t, id, "tl-")
	assert.Equal(t, "tl-", id[:3])
}
