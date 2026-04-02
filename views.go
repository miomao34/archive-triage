package main

import (
	"fmt"
	linkreader "miomao34/archive-triage/link_reader"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/log"
)

func getCulledString(str string, length int) string {
	if len(str) > length && length >= 3 {
		return str[:length-3] + "..."
	}
	if len(str) > length && length < 3 {
		return "..."[:length]
	}
	return str
}

func getStatusString(stats linkreader.DatabaseStats, width int) string {

	statString := fmt.Sprintf("unprocessed: %v, dismissed: %v, saved: %v, postponed: %v",
		stats.Unprocessed,
		stats.Dismissed,
		stats.Saved,
		stats.Snoozed,
	)
	if len(statString) > width {
		statString = fmt.Sprintf("U: %v, D: %v, S: %v, P: %v",
			stats.Unprocessed,
			stats.Dismissed,
			stats.Saved,
			stats.Snoozed,
		)
	}
	statsView := lipgloss.NewStyle().Width(width).Render(getCulledString(statString, width))

	return statsView
}

// func

func ViewWelcome(m *model) tea.View {
	messageView := nameStyle.Render(string(m.welcomeMessages[0]))
	splashScreenView := linkStyle.Render(string(m.welcomeMessages[1]))
	display := lipgloss.JoinVertical(lipgloss.Left, messageView, splashScreenView)

	view := tea.NewView(display)
	view.AltScreen = true
	return view
}

func ViewTriage(m *model) tea.View {
	nameView := nameStyle.Width(m.sizes.topCellWidth).
		Render(getCulledString(string(m.link.GetName()), m.sizes.topCellWidth))
	linkView := linkStyle.Width(m.sizes.topCellWidth).
		Render(getCulledString(string(m.link.GetHREF()), m.sizes.topCellWidth))
	nameAndLinkDisplay := lipgloss.JoinVertical(lipgloss.Left, nameView, linkView)

	dupesView := m.paginator.View()

	if len(m.duplicateIDs) == 0 {
		dupesView = duplicateStyle.Width(m.sizes.middleCellWidth).Height(m.sizes.middleCellHeight).Render("no duplicates found!")
	} else {
		start, end := m.paginator.GetSliceBounds(len(m.duplicates))
		log.Debug("slice bounds", "start", start, "end", end)

		for orderID, dupeLink := range m.duplicates[start:end] {
			dupeID := m.duplicateIDs[orderID+start]

			// -10 bc all the other characters take 10 spaces
			// /2 since we should fit both the name and the link
			// all the other math to adjust if the name is shorter
			nameOrLinkDefaultLength := (m.sizes.bottomCellWidth - 10) / 2
			dupeNameLength := min(nameOrLinkDefaultLength, len(dupeLink.GetName()))
			dupeLinkLength := nameOrLinkDefaultLength*2 - dupeNameLength

			dupeString := fmt.Sprintf("%04v | %v | %v",
				dupeID,
				getCulledString(string(dupeLink.GetName()), dupeNameLength),
				getCulledString(string(dupeLink.GetHREF()), dupeLinkLength),
			)
			dupeView := duplicateStyle.Width(m.sizes.middleCellWidth).UnsetHeight().Render(dupeString)
			if dupesView == "" {
				dupesView = dupeView
			} else {
				dupesView = lipgloss.JoinVertical(lipgloss.Left, dupesView, dupeView)
			}
		}
	}

	// duplicatesBeginID := m.sizes.middleCellHeight * m.currentDuplicatesPage
	// duplicatesEndID := min(len(m.duplicates), m.sizes.middleCellHeight*(m.currentDuplicatesPage+1))

	statsView := getStatusString(*m.stats, m.sizes.bottomCellWidth)

	view := tea.NewView(lipgloss.JoinVertical(
		lipgloss.Left,
		topCellStyle.Render(nameAndLinkDisplay),
		middleCellStyle.Height(m.sizes.middleCellHeight+1).Render(dupesView),
		bottomCellStyle.Render(statsView),
	))
	view.AltScreen = true
	return view
}

func ViewIngestPickFile(m *model) tea.View {
	nameView := nameStyle.Width(m.sizes.topCellWidth).
		Render(getCulledString(string(m.link.GetName()), m.sizes.topCellWidth))
	linkView := linkStyle.Width(m.sizes.topCellWidth).
		Render(getCulledString(string(m.link.GetHREF()), m.sizes.topCellWidth))
	display := lipgloss.JoinVertical(lipgloss.Left, nameView, linkView)

	// log.Debug(m.filepicker.Height())
	filepickerView := "Pick a file to ingest:\n" + m.filepicker.View()

	middleCellStyle = middleCellStyle.Width(m.sizes.dimensions[0])

	statsView := getStatusString(*m.stats, m.sizes.bottomCellWidth)

	view := tea.NewView(lipgloss.JoinVertical(
		lipgloss.Left,
		topCellStyle.Render(display),
		middleCellStyle.Render(filepickerView),
		bottomCellStyle.Render(statsView),
	))
	view.AltScreen = true

	return view
}

func ViewIngestPickFormat(m *model) tea.View {
	nameView := nameStyle.Width(m.sizes.topCellWidth).
		Render(getCulledString(string(m.link.GetName()), m.sizes.topCellWidth))
	linkView := linkStyle.Width(m.sizes.topCellWidth).
		Render(getCulledString(string(m.link.GetHREF()), m.sizes.topCellWidth))
	display := lipgloss.JoinVertical(lipgloss.Left, nameView, linkView)

	formatPickerView := "Pick a format:\n"
	for i, format := range m.formats {
		cursor := "  "
		if m.cursor == i {
			cursor = "> "
		}
		formatPickerView += cursor
		if m.cursor == i {
			format = formatPickerStyle.Render(format)
		}
		formatPickerView += format + "\n"
	}
	middleCellStyle = middleCellStyle.Width(m.sizes.dimensions[0])
	middleCellStyle = middleCellStyle.Height(m.sizes.middleCellHeight + 1)

	statsView := getStatusString(*m.stats, m.sizes.bottomCellWidth)

	view := tea.NewView(lipgloss.JoinVertical(
		lipgloss.Left,
		topCellStyle.Render(display),
		middleCellStyle.Render(formatPickerView),
		bottomCellStyle.Render(statsView),
	))
	view.AltScreen = true

	return view
}

func ViewTags(m *model) tea.View {
	nameView := nameStyle.Width(m.sizes.topCellWidth).
		Render(getCulledString(string(m.link.GetName()), m.sizes.topCellWidth))
	linkView := linkStyle.Width(m.sizes.topCellWidth).
		Render(getCulledString(string(m.link.GetHREF()), m.sizes.topCellWidth))
	display := lipgloss.JoinVertical(lipgloss.Left, nameView, linkView)

	m.textarea.SetStyles(tagTextAreaStyle)
	m.textarea.SetWidth(m.sizes.middleCellWidth)
	m.textarea.SetHeight(m.sizes.middleCellHeight)
	var c *tea.Cursor
	if !m.textarea.VirtualCursor() {
		c = m.textarea.Cursor()

		// Set the offset of the cursor based on the position of the textarea
		c.Y += 4
		c.X += 1
	}
	textareaView := m.textarea.View()

	statsView := getStatusString(*m.stats, m.sizes.bottomCellWidth)

	view := tea.NewView(lipgloss.JoinVertical(
		lipgloss.Left,
		topCellStyle.Render(display),
		middleCellStyle.Render(textareaView),
		bottomCellStyle.Render(statsView),
	))
	view.AltScreen = true
	view.Cursor = c

	return view
}
