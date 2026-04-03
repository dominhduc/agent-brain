package tui

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	ClearScreen  = "\033[2J"
	CursorHome   = "\033[H"
	HideCursor   = "\033[?25l"
	ShowCursor   = "\033[?25h"
	ResetStyle   = "\033[0m"
	BoldStyle    = "\033[1m"
	DimStyle     = "\033[2m"
	ReverseStyle = "\033[7m"
	GreenStyle   = "\033[32m"
	RedStyle     = "\033[31m"
	YellowStyle  = "\033[33m"
	CyanStyle    = "\033[36m"
)

func RenderScreen(state *ReviewState, width, height int) string {
	var b strings.Builder

	b.WriteString(ClearScreen)
	b.WriteString(CursorHome)
	b.WriteString(HideCursor)

	header := fmt.Sprintf("  %sPending Knowledge Review — %s%s%s", BoldStyle, CyanStyle, state.Profile, ResetStyle)
	b.WriteString(header)
	b.WriteString("\n\n")

	selectedCount := 0
	for _, entries := range state.Groups {
		for _, e := range entries {
			if state.Selected[e.ID] {
				selectedCount++
			}
		}
	}
	totalCount := len(state.Entries)
	summary := fmt.Sprintf("  %d entries pending", totalCount)
	if selectedCount > 0 {
		summary += fmt.Sprintf(", %s%d selected%s", GreenStyle, selectedCount, ResetStyle)
	}
	summary += ResetStyle
	b.WriteString(summary)
	b.WriteString("\n")

	if len(state.DedupGroups) > 0 {
		warning := fmt.Sprintf("  %s⚠ %d duplicate group(s) found — press %sm%s to merge%s", YellowStyle, len(state.DedupGroups), BoldStyle, "m", ResetStyle)
		b.WriteString(warning)
		b.WriteString("\n")
	}

	b.WriteString("\n")

	if len(state.Groups) == 0 {
		b.WriteString("\n  No pending entries to review.\n")
	} else {
		topicLine := renderTopicTabs(state, width)
		b.WriteString(topicLine)
		b.WriteString("\n\n")

		entries := state.CurrentEntries()
		if len(entries) == 0 {
			b.WriteString("  No entries in this topic.\n")
		} else {
			availableHeight := height - 12
			if availableHeight < 3 {
				availableHeight = 3
			}
			for i := 0; i < len(entries); i++ {
				if i >= availableHeight {
					remaining := len(entries) - i
					b.WriteString(fmt.Sprintf("  %s... and %d more%s\n", DimStyle, remaining, ResetStyle))
					break
				}
			isCurrent := i == state.CurrentEntry
			isSelected := state.Selected[entries[i].ID]
			cursor := "  "
			style := ""
			if isCurrent {
				cursor = "▶ "
				style = BoldStyle
			}
			if isSelected {
				style += GreenStyle
			}
			b.WriteString(cursor)
			b.WriteString(style)
			b.WriteString("☐ ")
			b.WriteString(entries[i].DisplayTime())
			b.WriteString("  ")
			b.WriteString(TruncateString(entries[i].Content, width-20))
			b.WriteString(ResetStyle)
			b.WriteString("\n")
			}
		}
	}

	for b.Len() < (height-2)*width {
		b.WriteByte(' ')
	}

	helpBar := renderHelpBar(state)
	b.WriteString("\033[" + fmt.Sprintf("%d", height) + ";0H")
	b.WriteString(helpBar)
	b.WriteString(ResetStyle)

	return b.String()
}

func RenderExitMessage(msg string) string {
	return fmt.Sprintf("%s%s%s%s\n", ClearScreen, CursorHome, msg, ShowCursor)
}

func renderTopicTabs(state *ReviewState, width int) string {
	var b strings.Builder
	b.WriteString("  Topics: ")
	tabWidth := 0
	for i, topic := range state.GroupOrder {
		count := len(state.Groups[topic])
		label := fmt.Sprintf("%s (%d)", topic, count)
		if i == state.CurrentGroup {
			label = fmt.Sprintf("%s%s%s", BoldStyle+ReverseStyle, label, ResetStyle)
		} else {
			label = fmt.Sprintf("%s%s%s", DimStyle, label, ResetStyle)
		}
		if tabWidth+utf8.RuneCountInString(label)+3 > width-4 {
			b.WriteString("…")
			break
		}
		if i > 0 {
			b.WriteString("  ")
			tabWidth += 2
		}
		b.WriteString(label)
		tabWidth += utf8.RuneCountInString(label)
	}
	return b.String()
}

func renderHelpBar(state *ReviewState) string {
	var b strings.Builder
	b.WriteString(DimStyle)
	b.WriteString("  ↑↓ navigate  ←→ topics  space select  a accept  r reject  m merge  q quit")
	if state.CurrentTopic() != "" {
		b.WriteString(ResetStyle)
		b.WriteString(fmt.Sprintf("  %s[%s]%s", CyanStyle, state.CurrentTopic(), ResetStyle))
	}
	return b.String()
}

func WrapText(text string, width int) []string {
	if width <= 0 {
		width = 80
	}
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}
	var lines []string
	var currentLine strings.Builder
	currentLen := 0
	for _, word := range words {
		wordLen := utf8.RuneCountInString(word)
		if currentLen > 0 && currentLen+1+wordLen > width {
			lines = append(lines, currentLine.String())
			currentLine.Reset()
			currentLen = 0
		}
		if currentLen > 0 {
			currentLine.WriteByte(' ')
			currentLen++
		}
		currentLine.WriteString(word)
		currentLen += wordLen
	}
	if currentLen > 0 {
		lines = append(lines, currentLine.String())
	}
	return lines
}

func TruncateString(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen <= 1 {
		return "…"
	}
	return string(runes[:maxLen-1]) + "…"
}
