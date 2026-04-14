package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/dominhduc/agent-brain/internal/knowledge"
)

func makeEntry(id, topic, content string) knowledge.PendingEntry {
	return knowledge.PendingEntry{
		ID:        id,
		Topic:     topic,
		Content:   content,
		Timestamp: time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC),
		Confidence: "high",
		Source:    "test",
	}
}

func TestParseKey(t *testing.T) {
	tests := []struct {
		ch      byte
		isArrow bool
		want    Key
	}{
		{'a', false, KeyA},
		{'A', false, KeyA},
		{'r', false, KeyR},
		{'R', false, KeyR},
		{'m', false, KeyM},
		{'q', false, KeyQ},
		{' ', false, KeySpace},
		{'\r', false, KeyEnter},
		{'\n', false, KeyEnter},
		{27, false, KeyEsc},
		{'z', false, KeyUnknown},
		{'a', true, KeyUnknown},
	}

	for _, tt := range tests {
		got := ParseKey(tt.ch, tt.isArrow)
		if got != tt.want {
			t.Errorf("ParseKey(%q, %v) = %v; want %v", tt.ch, tt.isArrow, got, tt.want)
		}
	}
}

func TestParseArrowKey(t *testing.T) {
	tests := []struct {
		seq  []byte
		want Key
	}{
		{[]byte{27, '[', 'A'}, KeyUp},
		{[]byte{27, '[', 'B'}, KeyDown},
		{[]byte{27, '[', 'C'}, KeyRight},
		{[]byte{27, '[', 'D'}, KeyLeft},
		{[]byte{27, '[', 'Z'}, KeyUnknown},
		{[]byte{27, 'A'}, KeyUnknown},
		{[]byte{27}, KeyUnknown},
	}

	for _, tt := range tests {
		got := ParseArrowKey(tt.seq)
		if got != tt.want {
			t.Errorf("ParseArrowKey(%v) = %v; want %v", tt.seq, got, tt.want)
		}
	}
}

func TestKeyString(t *testing.T) {
	tests := []struct {
		key Key
		want string
	}{
		{KeyA, "a"},
		{KeyR, "r"},
		{KeyM, "m"},
		{KeyQ, "q"},
		{KeySpace, "space"},
		{KeyUp, "↑"},
		{KeyDown, "↓"},
		{KeyLeft, "←"},
		{KeyRight, "→"},
		{KeyEnter, "enter"},
		{KeyEsc, "esc"},
		{KeyUnknown, "unknown"},
	}

	for _, tt := range tests {
		got := tt.key.String()
		if got != tt.want {
			t.Errorf("Key(%d).String() = %q; want %q", tt.key, got, tt.want)
		}
	}
}

func TestWrapText(t *testing.T) {
	tests := []struct {
		text  string
		width int
		want  int
	}{
		{"hello world", 80, 1},
		{"hello world", 5, 2},
		{"a b c d", 3, 2},
		{"", 80, 0},
		{"abc", 0, 1},
	}

	for _, tt := range tests {
		lines := WrapText(tt.text, tt.width)
		if len(lines) != tt.want {
			t.Errorf("WrapText(%q, %d) got %d lines; want %d", tt.text, tt.width, len(lines), tt.want)
		}
	}
}

func TestWrapTextContent(t *testing.T) {
	lines := WrapText("hello world foo", 11)
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if lines[0] != "hello world" {
		t.Errorf("line 0 = %q; want %q", lines[0], "hello world")
	}
	if lines[1] != "foo" {
		t.Errorf("line 1 = %q; want %q", lines[1], "foo")
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		s      string
		maxLen int
		want   string
	}{
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello", 4, "hel…"},
		{"hello", 1, "…"},
		{"hello", 0, ""},
		{"hello", -1, ""},
		{"日本語", 3, "日本語"},
	}

	for _, tt := range tests {
		got := TruncateString(tt.s, tt.maxLen)
		if got != tt.want {
			t.Errorf("TruncateString(%q, %d) = %q; want %q", tt.s, tt.maxLen, got, tt.want)
		}
	}
}

func TestNewReviewState(t *testing.T) {
	entries := []knowledge.PendingEntry{
		makeEntry("1", "gotcha", "bug: nil pointer dereference"),
		makeEntry("2", "pattern", "use context with timeout"),
		makeEntry("3", "gotcha", "race condition in map access"),
	}

	state := NewReviewState(entries, "test-profile")

	if state.Profile != "test-profile" {
		t.Errorf("Profile = %q; want %q", state.Profile, "test-profile")
	}
	if len(state.Groups) != 2 {
		t.Errorf("len(Groups) = %d; want 2", len(state.Groups))
	}
	if len(state.GroupOrder) != 2 {
		t.Errorf("len(GroupOrder) = %d; want 2", len(state.GroupOrder))
	}
	if state.GroupOrder[0] != "gotcha" {
		t.Errorf("GroupOrder[0] = %q; want %q", state.GroupOrder[0], "gotcha")
	}
	if state.CurrentGroup != 0 {
		t.Errorf("CurrentGroup = %d; want 0", state.CurrentGroup)
	}
	if state.CurrentEntry != 0 {
		t.Errorf("CurrentEntry = %d; want 0", state.CurrentEntry)
	}
	if len(state.Selected) != 0 {
		t.Errorf("len(Selected) = %d; want 0", len(state.Selected))
	}
}

func TestReviewStateNavigation(t *testing.T) {
	entries := []knowledge.PendingEntry{
		makeEntry("1", "alpha", "entry one"),
		makeEntry("2", "alpha", "entry two"),
		makeEntry("3", "alpha", "entry three"),
		makeEntry("4", "beta", "entry four"),
	}

	state := NewReviewState(entries, "test")

	state.MoveDown()
	if state.CurrentEntry != 1 {
		t.Errorf("MoveDown: CurrentEntry = %d; want 1", state.CurrentEntry)
	}

	state.MoveDown()
	state.MoveDown()
	if state.CurrentEntry != 2 {
		t.Errorf("MoveDown at end: CurrentEntry = %d; want 2", state.CurrentEntry)
	}

	state.MoveUp()
	if state.CurrentEntry != 1 {
		t.Errorf("MoveUp: CurrentEntry = %d; want 1", state.CurrentEntry)
	}

	state.MoveUp()
	state.MoveUp()
	if state.CurrentEntry != 0 {
		t.Errorf("MoveUp at start: CurrentEntry = %d; want 0", state.CurrentEntry)
	}

	state.NextGroup()
	if state.CurrentGroup != 1 {
		t.Errorf("NextGroup: CurrentGroup = %d; want 1", state.CurrentGroup)
	}
	if state.CurrentEntry != 0 {
		t.Errorf("NextGroup resets CurrentEntry: got %d; want 0", state.CurrentEntry)
	}

	state.NextGroup()
	if state.CurrentGroup != 1 {
		t.Errorf("NextGroup at end: CurrentGroup = %d; want 1", state.CurrentGroup)
	}

	state.PrevGroup()
	if state.CurrentGroup != 0 {
		t.Errorf("PrevGroup: CurrentGroup = %d; want 0", state.CurrentGroup)
	}

	state.PrevGroup()
	if state.CurrentGroup != 0 {
		t.Errorf("PrevGroup at start: CurrentGroup = %d; want 0", state.CurrentGroup)
	}
}

func TestReviewStateToggle(t *testing.T) {
	entries := []knowledge.PendingEntry{
		makeEntry("1", "alpha", "entry one"),
		makeEntry("2", "alpha", "entry two"),
	}

	state := NewReviewState(entries, "test")

	state.ToggleSelected()
	if !state.Selected["1"] {
		t.Error("ToggleSelected: entry 1 should be selected")
	}

	state.ToggleSelected()
	if state.Selected["1"] {
		t.Error("ToggleSelected: entry 1 should be deselected")
	}
}

func TestReviewStateAcceptReject(t *testing.T) {
	entries := []knowledge.PendingEntry{
		makeEntry("1", "alpha", "entry one"),
		makeEntry("2", "alpha", "entry two"),
		makeEntry("3", "beta", "entry three"),
	}

	state := NewReviewState(entries, "test")

	state.Selected["1"] = true
	state.Selected["3"] = true

	accepted := state.AcceptSelected()
	if len(accepted) != 2 {
		t.Errorf("AcceptSelected: got %d entries; want 2", len(accepted))
	}

	rejected := state.RejectSelected()
	if len(rejected) != 1 {
		t.Errorf("RejectSelected: got %d IDs; want 1", len(rejected))
	}
	if rejected[0] != "2" {
		t.Errorf("RejectSelected: got %q; want %q", rejected[0], "2")
	}
}

func TestReviewStateSelectAllDeselectAll(t *testing.T) {
	entries := []knowledge.PendingEntry{
		makeEntry("1", "alpha", "entry one"),
		makeEntry("2", "alpha", "entry two"),
		makeEntry("3", "beta", "entry three"),
	}

	state := NewReviewState(entries, "test")

	state.SelectAll()
	if !state.Selected["1"] || !state.Selected["2"] {
		t.Error("SelectAll: entries in current topic should be selected")
	}
	if state.Selected["3"] {
		t.Error("SelectAll: entries in other topics should NOT be selected")
	}

	state.DeselectAll()
	if state.Selected["1"] || state.Selected["2"] {
		t.Error("DeselectAll: entries in current topic should be deselected")
	}
}

func TestReviewStateCurrentTopic(t *testing.T) {
	entries := []knowledge.PendingEntry{
		makeEntry("1", "alpha", "entry one"),
		makeEntry("2", "beta", "entry two"),
	}

	state := NewReviewState(entries, "test")

	if state.CurrentTopic() != "alpha" {
		t.Errorf("CurrentTopic = %q; want %q", state.CurrentTopic(), "alpha")
	}

	curEntries := state.CurrentEntries()
	if len(curEntries) != 1 {
		t.Errorf("CurrentEntries count = %d; want 1", len(curEntries))
	}
}

func TestReviewStateEmpty(t *testing.T) {
	state := NewReviewState(nil, "test")

	if state.CurrentTopic() != "" {
		t.Errorf("CurrentTopic on empty = %q; want empty", state.CurrentTopic())
	}
	if len(state.CurrentEntries()) != 0 {
		t.Error("CurrentEntries on empty should be empty")
	}
}

func TestReviewStateDedupGroups(t *testing.T) {
	entries := []knowledge.PendingEntry{
		makeEntry("1", "gotcha", "always check for nil"),
		makeEntry("2", "gotcha", "Always check for nil"),
	}

	state := NewReviewState(entries, "test")

	if len(state.DedupGroups) != 1 {
		t.Errorf("DedupGroups count = %d; want 1", len(state.DedupGroups))
	}
}

func TestRenderScreen(t *testing.T) {
	entries := []knowledge.PendingEntry{
		makeEntry("1", "alpha", "entry one"),
	}

	state := NewReviewState(entries, "test")

	output := RenderScreen(state, 80, 24)

	if !strings.Contains(output, "test") {
		t.Error("RenderScreen: missing profile name")
	}
	if !strings.Contains(output, "1 entries pending") {
		t.Error("RenderScreen: missing entry count")
	}
	if !strings.Contains(output, "alpha") {
		t.Error("RenderScreen: missing topic name")
	}
}

func TestRenderExitMessage(t *testing.T) {
	output := RenderExitMessage("goodbye")
	if !strings.Contains(output, ClearScreen) {
		t.Error("RenderExitMessage: missing ClearScreen")
	}
	if !strings.Contains(output, ShowCursor) {
		t.Error("RenderExitMessage: missing ShowCursor")
	}
	if !strings.Contains(output, "goodbye") {
		t.Error("RenderExitMessage: missing message")
	}
}
