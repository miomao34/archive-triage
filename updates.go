package main

import (
	"database/sql"
	"errors"
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
			log.Info("switching to triage")
			m.appState = appStateTriage
			UpdateTriage(m, nil)

			var err error
			m.duplicateIDs, m.duplicates, err = m.conn.GetSimilarLinks(m.link)
			if err != nil {
				log.Debug("failed to get mf dupes", "err", err)
			}
		}
	}
	// necessary for the file picker to work; spent solid 6hrs debugging it
	m.filepicker, _ = m.filepicker.Update(msg)

	return m, nil
}

func UpdateTriage(m *model, msg tea.Msg) (tea.Model, tea.Cmd) {
	err := m.NextLink()
	if err != nil {
		newLink := linkreader.Link{}
		newLink.SetName([]byte("no more unprocessed links. congratulations!"))
		newLink.SetHREF([]byte("press i to ingest a new file, or r to reset postponed links"))
		m.link = newLink
	}
	m.duplicateIDs, m.duplicates, err = m.conn.GetSimilarLinks(m.link)
	if err != nil {
		log.Error("failed to get mf dupes", "err", err)
	}

	m.stats, err = m.conn.GetStats()
	if err != nil {
		log.Error("failed to get stats", "err", err)
	}

	switch msg := msg.(type) {

	case tea.KeyMsg:

		switch {
		case key.Matches(msg, m.keys.Save):
			log.Info("switching to tags")
			m.appState = appStateTags
			m.textarea.Focus()

		case key.Matches(msg, m.keys.Ingest):
			log.Info("switching to ingest pick file")
			m.appState = appStateIngestPickFile
			UpdateIngestPickFile(m, nil)

		case key.Matches(msg, m.keys.Help):
			log.Info("switching to help (one day)")
			m.appState = appStateHelp

		case key.Matches(msg, m.keys.Welcome):
			log.Info("switching to welcome")
			m.appState = appStateWelcome

		case key.Matches(msg, m.keys.Postpone):
			err := m.conn.MarkLinkById(m.id, linkreader.LinkSnoozed)
			if err != nil {
				// todo
			}
			return UpdateTriage(m, nil)

		case key.Matches(msg, m.keys.ResetPostponed):
			log.Debug("resetting snoozed links...")
			err := m.conn.ResetSnoozedLinks()
			if err != nil {
				log.Error("failed to reset snoozed links", "err", err)
			} else {
				log.Debug("snoozed links reset!")
			}
			return UpdateTriage(m, nil)

		case key.Matches(msg, m.keys.Discard):
			err := m.conn.MarkLinkById(m.id, linkreader.LinkDismissed)
			if err != nil {
				log.Error("failed to mark link discarded", "err", err)
			} else {
				log.Info("discarded link")
			}
			return UpdateTriage(m, nil)
		case key.Matches(msg, m.keys.Undo):
			id, err := m.conn.UnmarkLastLink()
			if err != nil {
				log.Error("failed to unmark link", "err", err)
				if errors.Is(err, sql.ErrNoRows) {
					log.Debug("no more marked links!")
				}
			} else {
				err = m.conn.UntagLink(id)
				if err != nil {
					log.Error("failed to untag unmarked link", "err", err)
				} else {
					log.Info("untagged an unmarked link")
				}
			}
			return UpdateTriage(m, nil)

		case key.Matches(msg, m.keys.Open):
			exec.Command("xdg-open", string(m.link.GetHREF())).Start()

		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		}
	}

	return m, nil
}

func UpdateIngestPickFile(m *model, msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			log.Info("switching to triage")
			m.appState = appStateTriage
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.filepicker, cmd = m.filepicker.Update(msg)

	// Did the user select a file?
	if didSelect, path := m.filepicker.DidSelectFile(msg); didSelect {
		// Get the path of the selected file.
		m.selectedFile = path
		log.Info("switching to ingest pick format")
		m.appState = appStateIngestPickFormat
	}

	return m, cmd
}

func UpdateIngestPickFormat(m *model, msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			log.Info("switching to triage")
			m.appState = appStateTriage
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.formats)-1 {
				m.cursor++
			}
		case "enter", "space":
			// m.cursor has the same int value as the format iota
			generator := linkreader.LinkGenerator{}
			log.Debug("opening generator", "selectedFile", m.selectedFile, "LinkFileFormatType", m.cursor)
			err := generator.ReadBookmarksFile(m.selectedFile, linkreader.LinkFileFormatType(m.cursor))
			if err != nil {
				log.Debug("whelp")
			}
			generator.Run()
			for link := range generator.ReturnChannel {
				id, err := m.conn.InsertLink(link)
				if err != nil {
					log.Error("failed to insert link", "name", string(link.GetName()), "href", string(link.GetHREF()))
				} else {
					log.Info("inserted link", "name", string(link.GetName()), "href", string(link.GetHREF()))
				}
				m.conn.MarkLinkSource(id, m.selectedFile)
			}

			log.Info("switching to triage")
			UpdateTriage(m, nil)
			m.appState = appStateTriage

		case "ctrl+c", "q":
			return m, tea.Quit
		}
	}

	return m, nil
}

func tagLink(m *model, resolution linkreader.LinkResolution, tags string) error {
	err := m.conn.MarkLinkById(m.id, resolution)
	if err != nil {
		return err
	}
	for _, tag := range strings.Split(tags, "\n") {
		if tag == "" {
			continue
		}
		tag = strings.ReplaceAll(tag, " ", "_")
		err = m.conn.TagLink(m.id, tag)
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
				err := tagLink(m, linkreader.LinkSaved, m.textarea.Value())
				if err != nil {
					// fuck
				}
				m.textarea.SetValue("")

				log.Info("switching to triage")
				m.appState = appStateTriage
				UpdateTriage(m, nil)

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
