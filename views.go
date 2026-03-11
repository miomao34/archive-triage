package main

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func ViewCulled(str string, length int) string {
	if len(str) > length {
		return str[:length-3] + "..."
	}
	return str
}

func ViewWelcome(m *model) tea.View {
	messageView := nameStyle.Render(string(m.welcomeMessages[0]))
	splashScreenView := linkStyle.Render(string(m.welcomeMessages[1]))
	display := lipgloss.JoinVertical(lipgloss.Left, messageView, splashScreenView)

	view := tea.NewView(display)
	view.AltScreen = true
	return view
}

func ViewTriage(m *model) tea.View {
	nameView := nameStyle.Width(m.sizes.linkAndNameLengthLimit).
		Render(ViewCulled(string(m.link.GetName()), m.sizes.linkAndNameLengthLimit))
	linkView := linkStyle.Width(m.sizes.linkAndNameLengthLimit).
		Render(ViewCulled(string(m.link.GetHREF()), m.sizes.linkAndNameLengthLimit))
	nameAndLinkDisplay := lipgloss.JoinVertical(lipgloss.Left, nameView, linkView)

	var dupesView string

	if len(m.duplicateIDs) == 0 {
		dupesView = duplicateStyle.Width(m.sizes.dimensions[0] - 2).Render("no duplicates found!")
	}

	for dupeID := range m.duplicateIDs {
		dupeLink := m.duplicates[dupeID]
		dupeString := fmt.Sprintf("%04v | %v | %v",
			dupeID,
			ViewCulled(string(dupeLink.GetName()), (m.sizes.dupeLengthLimit-10)/2),
			ViewCulled(string(dupeLink.GetHREF()), (m.sizes.dupeLengthLimit-10)/2),
		)
		dupeView := duplicateStyle.Width(m.sizes.dupeLengthLimit).Render(dupeString)
		if dupesView == "" {
			dupesView = dupeView
		} else {
			dupesView = lipgloss.JoinVertical(lipgloss.Left, dupesView, dupeView)
		}
	}

	view := tea.NewView(lipgloss.JoinVertical(lipgloss.Left, topCellStyle.Render(nameAndLinkDisplay), middleCellStyle.Render(dupesView)))
	view.AltScreen = true
	return view
}

func ViewTags(m *model) tea.View {
	nameView := nameStyle.Width(m.sizes.linkAndNameLengthLimit).
		Render(ViewCulled(string(m.link.GetName()), m.sizes.linkAndNameLengthLimit))
	linkView := linkStyle.Width(m.sizes.linkAndNameLengthLimit).
		Render(ViewCulled(string(m.link.GetHREF()), m.sizes.linkAndNameLengthLimit))
	display := lipgloss.JoinVertical(lipgloss.Left, nameView, linkView)

	m.textarea.SetStyles(tagTextAreaStyle)
	m.textarea.SetWidth(m.sizes.dupeLengthLimit)
	m.textarea.SetHeight(m.sizes.dimensions[1] - 5)
	var c *tea.Cursor
	if !m.textarea.VirtualCursor() {
		c = m.textarea.Cursor()

		// Set the offset of the cursor based on the position of the textarea
		c.Y += 4
		c.X += 1
	}
	textareaView := m.textarea.View()

	view := tea.NewView(lipgloss.JoinVertical(lipgloss.Left, topCellStyle.Render(display), bottomCellStyle.Render(textareaView)))
	view.AltScreen = true
	view.Cursor = c

	return view
}

// FullHelp returns keybindings for the expanded help view. It's part of the
// key.Map interface.
func (k keyMap) FullHelp() [][]key.Binding {
	// columns
	return [][]key.Binding{
		{k.Back, k.Save, k.Postpone, k.Discard}, // first column
		{k.Context, k.Help, k.Quit},             // second column
	}
}
