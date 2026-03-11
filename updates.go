package main

import (
	linkreader "miomao34/archive-triage/link_reader"
	"os/exec"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/log"
)

func UpdateWelcome(m *model, msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch {

		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.keys.Welcome):
			// this space intentionally left blank
		default:
			m.appState = appStateTriage

			var err error
			m.duplicateIDs, m.duplicates, err = m.conn.GetSimilarLinks(m.link)
			if err != nil {
				log.Debug("failed to get mf dupes", "err", err)
			}
		}
	}

	return m, nil
}

func UpdateTriage(m *model, msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:

		switch {
		case key.Matches(msg, m.keys.Save):
			var err error
			m.duplicateIDs, m.duplicates, err = m.conn.GetSimilarLinks(m.link)
			if err != nil {
				log.Debug("failed to get mf dupes", "err", err)
			}

			m.appState = appStateTags
			m.textarea.Focus()

		case key.Matches(msg, m.keys.Help):
			m.appState = appStateHelp
		case key.Matches(msg, m.keys.Welcome):
			m.appState = appStateWelcome
		case key.Matches(msg, m.keys.Postpone):
			id, err := m.conn.InsertLink(m.link)
			if err != nil {
				// todo
			}
			err = m.conn.MarkLinkById(id, linkreader.LinkSnoozed)
			if err != nil {
				// todo
			}
			err = m.NextLink()
			if err != nil {
				// todo
			}
			m.duplicateIDs, m.duplicates, err = m.conn.GetSimilarLinks(m.link)
			if err != nil {
				log.Debug("failed to get mf dupes", "err", err)
			}

		case key.Matches(msg, m.keys.Discard):
			err := m.NextLink()
			if err != nil {
				// todo
			}
			m.duplicateIDs, m.duplicates, err = m.conn.GetSimilarLinks(m.link)
			if err != nil {
				log.Debug("failed to get mf dupes", "err", err)
			}
		case key.Matches(msg, m.keys.Open):
			exec.Command("xdg-open", string(m.link.GetHREF())).Start()

		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		}
	}

	return m, nil
}

func insertWithTags(m *model, resolution linkreader.LinkResolution, tags string) error {
	id, err := m.conn.InsertLink(m.link)
	if err != nil {
		return err
	}
	err = m.conn.MarkLinkById(id, resolution)
	if err != nil {
		return err
	}
	for _, tag := range strings.Split(tags, "\n") {
		if tag == "" {
			continue
		}
		tag = strings.ReplaceAll(tag, " ", "_")
		err = m.conn.TagLink(id, tag)
		if err != nil {
			return err
		}
	}
	return nil
}

func UpdateTags(m *model, msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {

	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			if m.textarea.Focused() {
				m.textarea.Blur()
				err := insertWithTags(m, linkreader.LinkSaved, m.textarea.Value())
				if err != nil {
					// fuck
				}
				m.textarea.SetValue("")

				err = m.NextLink()
				if err != nil {
					// fuck
				}

				m.appState = appStateTriage

				// here - get value for tags from text area
			}
		case "ctrl+c":
			return m, tea.Quit
		default:
			if !m.textarea.Focused() {
				cmd = m.textarea.Focus()
				cmds = append(cmds, cmd)
			}
		}

		// We handle errors just like any other message
	case error:
		m.err = msg
		return m, nil
	}

	m.textarea, cmd = m.textarea.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}
