package knowledge

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestHubWithBehavior(t *testing.T) *Hub {
	t.Helper()
	hub := setupTestHub(t)
	return hub
}

func TestLoadBehaviorEmpty(t *testing.T) {
	hub := setupTestHubWithBehavior(t)

	signals, err := hub.LoadBehavior()
	if err != nil {
		t.Fatal(err)
	}
	if signals.CommandCounts == nil {
		t.Fatal("expected non-nil CommandCounts")
	}
	if signals.TopicAccess == nil {
		t.Fatal("expected non-nil TopicAccess")
	}
	if signals.SearchQueries == nil {
		t.Fatal("expected non-nil SearchQueries")
	}
}

func TestSaveAndLoadBehavior(t *testing.T) {
	hub := setupTestHubWithBehavior(t)

	signals, _ := hub.LoadBehavior()
	signals.CommandCounts["test"] = 42
	signals.EvalOutcomes.Good = 5
	signals.EvalOutcomes.Bad = 2

	err := hub.SaveBehavior(signals)
	if err != nil {
		t.Fatal(err)
	}

	loaded, err := hub.LoadBehavior()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.CommandCounts["test"] != 42 {
		t.Errorf("expected CommandCounts[test]=42, got %d", loaded.CommandCounts["test"])
	}
	if loaded.EvalOutcomes.Good != 5 {
		t.Errorf("expected Good=5, got %d", loaded.EvalOutcomes.Good)
	}
	if loaded.EvalOutcomes.Bad != 2 {
		t.Errorf("expected Bad=2, got %d", loaded.EvalOutcomes.Bad)
	}
}

func TestTrackCommand(t *testing.T) {
	hub := setupTestHubWithBehavior(t)

	err := hub.TrackCommand("get")
	if err != nil {
		t.Fatal(err)
	}

	signals, _ := hub.LoadBehavior()
	if signals.CommandCounts["get"] != 1 {
		t.Errorf("expected CommandCounts[get]=1, got %d", signals.CommandCounts["get"])
	}

	err = hub.TrackCommand("get")
	if err != nil {
		t.Fatal(err)
	}

	signals, _ = hub.LoadBehavior()
	if signals.CommandCounts["get"] != 2 {
		t.Errorf("expected CommandCounts[get]=2, got %d", signals.CommandCounts["get"])
	}
}

func TestTrackTopicAccess(t *testing.T) {
	hub := setupTestHubWithBehavior(t)

	err := hub.TrackTopicAccess("gotchas")
	if err != nil {
		t.Fatal(err)
	}

	signals, _ := hub.LoadBehavior()
	info, ok := signals.TopicAccess["gotchas"]
	if !ok {
		t.Fatal("expected gotchas in TopicAccess")
	}
	if info.Retrievals != 1 {
		t.Errorf("expected Retrievals=1, got %d", info.Retrievals)
	}

	err = hub.TrackTopicAccess("gotchas")
	if err != nil {
		t.Fatal(err)
	}

	signals, _ = hub.LoadBehavior()
	info, _ = signals.TopicAccess["gotchas"]
	if info.Retrievals != 2 {
		t.Errorf("expected Retrievals=2, got %d", info.Retrievals)
	}
}

func TestTrackSearch(t *testing.T) {
	hub := setupTestHubWithBehavior(t)

	err := hub.TrackSearch("how to prune")
	if err != nil {
		t.Fatal(err)
	}

	signals, _ := hub.LoadBehavior()
	if len(signals.SearchQueries) != 1 {
		t.Fatalf("expected 1 search query, got %d", len(signals.SearchQueries))
	}
	if signals.SearchQueries[0].Query != "how to prune" {
		t.Errorf("expected query 'how to prune', got %s", signals.SearchQueries[0].Query)
	}
	if signals.SearchQueries[0].Count != 1 {
		t.Errorf("expected count=1, got %d", signals.SearchQueries[0].Count)
	}

	err = hub.TrackSearch("how to prune")
	if err != nil {
		t.Fatal(err)
	}

	signals, _ = hub.LoadBehavior()
	if signals.SearchQueries[0].Count != 2 {
		t.Errorf("expected count=2, got %d", signals.SearchQueries[0].Count)
	}

	err = hub.TrackSearch("new query")
	if err != nil {
		t.Fatal(err)
	}

	signals, _ = hub.LoadBehavior()
	if len(signals.SearchQueries) != 2 {
		t.Fatalf("expected 2 search queries, got %d", len(signals.SearchQueries))
	}
}

func TestTrackEvalOutcome(t *testing.T) {
	hub := setupTestHubWithBehavior(t)

	err := hub.TrackEvalOutcome(true)
	if err != nil {
		t.Fatal(err)
	}

	signals, _ := hub.LoadBehavior()
	if signals.EvalOutcomes.Good != 1 {
		t.Errorf("expected Good=1, got %d", signals.EvalOutcomes.Good)
	}
	if signals.EvalOutcomes.TotalSessions != 1 {
		t.Errorf("expected TotalSessions=1, got %d", signals.EvalOutcomes.TotalSessions)
	}

	err = hub.TrackEvalOutcome(false)
	if err != nil {
		t.Fatal(err)
	}

	signals, _ = hub.LoadBehavior()
	if signals.EvalOutcomes.Bad != 1 {
		t.Errorf("expected Bad=1, got %d", signals.EvalOutcomes.Bad)
	}
	if signals.EvalOutcomes.TotalSessions != 2 {
		t.Errorf("expected TotalSessions=2, got %d", signals.EvalOutcomes.TotalSessions)
	}
}

func TestBehaviorPath(t *testing.T) {
	hub := setupTestHubWithBehavior(t)
	path := hub.behaviorPath()
	expected := filepath.Join(hub.dir, "behavior", "signals.json")
	if path != expected {
		t.Errorf("expected %s, got %s", expected, path)
	}
}

func TestBehaviorFileCorrupt(t *testing.T) {
	hub := setupTestHubWithBehavior(t)

	dir := filepath.Dir(hub.behaviorPath())
	os.MkdirAll(dir, 0755)
	os.WriteFile(hub.behaviorPath(), []byte("not valid json"), 0600)

	signals, err := hub.LoadBehavior()
	if err != nil {
		t.Fatal(err)
	}
	if signals.CommandCounts == nil {
		t.Fatal("expected empty behavior on corrupt file")
	}
}
