package tui

import (
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/dominhduc/agent-brain/internal/review"
)

type Key int

const (
	KeyUnknown Key = iota
	KeyA
	KeyR
	KeyM
	KeyQ
	KeySpace
	KeyUp
	KeyDown
	KeyLeft
	KeyRight
	KeyEnter
	KeyEsc
)

func ParseKey(ch byte, isArrow bool) Key {
	if isArrow {
		return KeyUnknown
	}
	switch ch {
	case 'a', 'A':
		return KeyA
	case 'r', 'R':
		return KeyR
	case 'm', 'M':
		return KeyM
	case 'q', 'Q':
		return KeyQ
	case ' ':
		return KeySpace
	case '\r', '\n':
		return KeyEnter
	case 27:
		return KeyEsc
	default:
		return KeyUnknown
	}
}

func ParseArrowKey(seq []byte) Key {
	if len(seq) < 3 {
		return KeyUnknown
	}
	switch seq[2] {
	case 'A':
		return KeyUp
	case 'B':
		return KeyDown
	case 'C':
		return KeyRight
	case 'D':
		return KeyLeft
	default:
		return KeyUnknown
	}
}

func (k Key) String() string {
	switch k {
	case KeyUnknown:
		return "unknown"
	case KeyA:
		return "a"
	case KeyR:
		return "r"
	case KeyM:
		return "m"
	case KeyQ:
		return "q"
	case KeySpace:
		return "space"
	case KeyUp:
		return "↑"
	case KeyDown:
		return "↓"
	case KeyLeft:
		return "←"
	case KeyRight:
		return "→"
	case KeyEnter:
		return "enter"
	case KeyEsc:
		return "esc"
	default:
		return "unknown"
	}
}

type ReviewState struct {
	Entries     []review.PendingEntry
	Groups      map[string][]review.PendingEntry
	GroupOrder  []string
	CurrentGroup  int
	CurrentEntry  int
	Selected    map[string]bool
	DedupGroups []review.DedupGroup
	Profile     string
	Exit        bool
	ExitAccepted []review.PendingEntry
	ExitRejected []string
	ExitErr     error
}

func NewReviewState(entries []review.PendingEntry, profile string) *ReviewState {
	groups := review.GroupByTopic(entries)

	var topics []string
	for topic := range groups {
		topics = append(topics, topic)
	}
	sort.Strings(topics)

	dedups := review.FindDuplicateGroups(entries)

	return &ReviewState{
		Entries:     entries,
		Groups:      groups,
		GroupOrder:  topics,
		CurrentGroup:  0,
		CurrentEntry:  0,
		Selected:    make(map[string]bool),
		DedupGroups: dedups,
		Profile:     profile,
	}
}

func (s *ReviewState) CurrentTopic() string {
	if s.CurrentGroup < 0 || s.CurrentGroup >= len(s.GroupOrder) {
		return ""
	}
	return s.GroupOrder[s.CurrentGroup]
}

func (s *ReviewState) CurrentEntries() []review.PendingEntry {
	topic := s.CurrentTopic()
	if topic == "" {
		return nil
	}
	return s.Groups[topic]
}

func (s *ReviewState) MoveUp() {
	if s.CurrentEntry > 0 {
		s.CurrentEntry--
	}
}

func (s *ReviewState) MoveDown() {
	entries := s.CurrentEntries()
	if s.CurrentEntry < len(entries)-1 {
		s.CurrentEntry++
	}
}

func (s *ReviewState) NextGroup() {
	if s.CurrentGroup < len(s.GroupOrder)-1 {
		s.CurrentGroup++
		s.CurrentEntry = 0
	}
}

func (s *ReviewState) PrevGroup() {
	if s.CurrentGroup > 0 {
		s.CurrentGroup--
		s.CurrentEntry = 0
	}
}

func (s *ReviewState) ToggleSelected() {
	entries := s.CurrentEntries()
	if s.CurrentEntry < 0 || s.CurrentEntry >= len(entries) {
		return
	}
	id := entries[s.CurrentEntry].ID
	if s.Selected[id] {
		delete(s.Selected, id)
	} else {
		s.Selected[id] = true
	}
}

func (s *ReviewState) AcceptSelected() []review.PendingEntry {
	var accepted []review.PendingEntry
	for _, e := range s.Entries {
		if s.Selected[e.ID] {
			accepted = append(accepted, e)
		}
	}
	return accepted
}

func (s *ReviewState) RejectSelected() []string {
	var rejected []string
	for _, e := range s.Entries {
		if !s.Selected[e.ID] {
			rejected = append(rejected, e.ID)
		}
	}
	return rejected
}

func (s *ReviewState) SelectAll() {
	for _, e := range s.CurrentEntries() {
		s.Selected[e.ID] = true
	}
}

func (s *ReviewState) DeselectAll() {
	for _, e := range s.CurrentEntries() {
		delete(s.Selected, e.ID)
	}
}

func RunReview(entries []review.PendingEntry, profile string, writer io.Writer) ([]review.PendingEntry, []string, error) {
	state := NewReviewState(entries, profile)

	oldState, err := EnableRawMode()
	if err != nil {
		return nil, nil, fmt.Errorf("enabling raw mode: %w", err)
	}
	defer oldState.Restore()

	buf := make([]byte, 3)

	for {
		width, height, _ := GetTerminalSize()
		if width <= 0 {
			width = 80
		}
		if height <= 0 {
			height = 24
		}

		output := RenderScreen(state, width, height)
		fmt.Fprint(writer, output)

		n, readErr := os.Stdin.Read(buf[:1])
		if readErr != nil {
			if readErr == io.EOF {
				return nil, nil, nil
			}
			return nil, nil, fmt.Errorf("reading input: %w", readErr)
		}
		if n == 0 {
			continue
		}

		ch := buf[0]

		if ch == 27 {
			_, err1 := os.Stdin.Read(buf[1:2])
			if err1 != nil {
				if err1 == io.EOF {
					handleKey(state, KeyEsc, writer)
					continue
				}
				continue
			}
			if buf[1] == 0 {
				handleKey(state, KeyEsc, writer)
				continue
			}
			if buf[1] == '[' {
				_, err2 := os.Stdin.Read(buf[2:3])
				if err2 != nil {
					continue
				}
				if buf[2] == 0 {
					continue
				}
				key := ParseArrowKey(buf[:3])
				handleNavigation(state, key)
				continue
			}
			if buf[1] == 27 {
				fmt.Fprint(writer, RenderExitMessage("\n  Review cancelled.\n"))
				return nil, nil, nil
			}
			continue
		}

		key := ParseKey(ch, false)
		handleKey(state, key, writer)

		if state.Exit {
			return state.ExitAccepted, state.ExitRejected, state.ExitErr
		}
	}
}

func handleKey(state *ReviewState, key Key, writer io.Writer) {
	switch key {
	case KeyQ:
		fmt.Fprint(writer, RenderExitMessage("\n  Review cancelled.\n"))
		state.Exit = true
		state.ExitAccepted = nil
		state.ExitRejected = nil
		state.ExitErr = nil
	case KeyEsc:
		fmt.Fprint(writer, RenderExitMessage("\n  Review cancelled.\n"))
		state.Exit = true
		state.ExitAccepted = nil
		state.ExitRejected = nil
		state.ExitErr = nil
	case KeyUp:
		state.MoveUp()
	case KeyDown:
		state.MoveDown()
	case KeyLeft:
		state.PrevGroup()
	case KeyRight:
		state.NextGroup()
	case KeySpace:
		state.ToggleSelected()
	case KeyA:
		fmt.Fprint(writer, RenderExitMessage(fmt.Sprintf("\n  Accepted %d entries.\n", len(state.AcceptSelected()))))
		state.Exit = true
		state.ExitAccepted = state.AcceptSelected()
		state.ExitRejected = state.RejectSelected()
		state.ExitErr = nil
	case KeyM:
		if len(state.DedupGroups) > 0 {
			for _, group := range state.DedupGroups {
				for _, e := range group.Entries[1:] {
					delete(state.Selected, e.ID)
				}
			}
			fmt.Fprintf(writer, "\r  Merged %d duplicate group(s). Press 'a' to accept, 'q' to quit.\n", len(state.DedupGroups))
		}
	case KeyR:
		entries := state.CurrentEntries()
		if state.CurrentEntry < len(entries) {
			delete(state.Selected, entries[state.CurrentEntry].ID)
		}
	}
}

func handleNavigation(state *ReviewState, key Key) {
	switch key {
	case KeyUp:
		state.MoveUp()
	case KeyDown:
		state.MoveDown()
	case KeyLeft:
		state.PrevGroup()
	case KeyRight:
		state.NextGroup()
	}
}
