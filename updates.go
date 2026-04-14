package main

import (
	"database/sql"
	"errors"
	linkreader "miomao34/archive-triage/link_reader"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/log"
)

// code for wayland, macos and windows is largely placeholder but
// might work; more testing is needed
func copyToClipboard(link []byte) error {
	switch runtime.GOOS {
	case "linux":
		sessionType, ok := os.LookupEnv("XDG_SESSION_TYPE")
		switch {
		case sessionType == "x11":
			command := exec.Command("xclip", "-selection", "clipboard")
			stdin, err := command.StdinPipe()
			if err != nil {
				return err
			}
			err = command.Start()
			if err != nil {
				return err
			}
			n, err := stdin.Write(link)
			if err != nil {
				return err
			}
			if n != len(link) {
				return errors.New("failed to write the whole link")
			}
			err = stdin.Close()
			if err != nil {
				return err
			}
			err = command.Wait()
			if err != nil {
				return err
			}
		case sessionType == "wayland":
			cmd := exec.Command("wl-copy", string(link))
			cmd.Start()
			cmd.Wait()
		case !ok:
			fallthrough
		default:
			log.Error("unknown session type, you're on your own here",
				"XDG_SESSION_TYPE", sessionType,
			)
		}
	case "darwin":
		exec.Command("echo", string(link), "|", "pbcopy").Start()
	case "windows":
		exec.Command("echo", string(link), "|", "clip").Start()

	default:
		log.Error("you're on your own here")
	}

	return nil
}

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
	m.paginator.SetTotalPages(len(m.duplicates))
	m.paginator.Page = 0
	if err != nil {
		log.Error("failed to get mf dupes", "err", err)
	}

	m.stats, err = m.conn.GetStats()
	if err != nil {
		log.Error("failed to get stats", "err", err)
	}

	var cmd tea.Cmd
	m.paginator, cmd = m.paginator.Update(msg)

	switch msg := msg.(type) {

	case tea.KeyMsg:

		switch {
		case key.Matches(msg, m.keys.Save):
			log.Info("switching to tags")
			m.appState = appStateTags
			m.textArea.Focus()

		case key.Matches(msg, m.keys.Ingest):
			log.Info("switching to ingest pick file")
			m.cursor = 0
			m.appState = appStateIngestPickFile
			UpdateIngestPickFile(m, nil)

		case key.Matches(msg, m.keys.Export):
			log.Info("switching to export")
			m.cursor = 0
			m.appState = appStateExportSelectFormat
			UpdateExport(m, nil)

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

		case key.Matches(msg, m.keys.Copy):
			log.Debug("copying the link")
			err := copyToClipboard(m.link.GetHREF())
			if err != nil {
				log.Error("failed to copy link", "err", err)
			}

		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		}
	}

	return m, cmd
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
		// resetting cursor since it's reused for import and export
		m.cursor = 0
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
			if m.cursor < len(m.importFormats)-1 {
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
func UpdateExportPickFormat(m *model, msg tea.Msg) (tea.Model, tea.Cmd) {

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
			if m.cursor < len(m.exportFormats)-1 {
				m.cursor++
			}
		case "enter", "space":
			// m.cursor has the same int value as the format iota
			switch ExportFormatType(m.cursor) {
			case MarkdownExportFormat:
				m.textInput.Placeholder = "export directory"
				m.appState = appStateExportMarkdown
			case BookmarkExportFormat:
				m.textInput.Placeholder = "export filename"
				m.appState = appStateExportBookmarks
			}

		case "ctrl+c", "q":
			return m, tea.Quit
		}
	}

	return m, nil
}

// unified function for appStateExportMarkdown and appStateExportBookmarks
func UpdateExport(m *model, msg tea.Msg) (tea.Model, tea.Cmd) {
	m.textInput.Focus()
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			m.textInput.Blur()
			m.appState = appStateTriage
			UpdateTriage(m, nil)
		case "enter":
			switch m.appState {
			case appStateExportMarkdown:
				saveLinksToMarkdown(m, m.textInput.Value())
			case appStateExportBookmarks:
				saveLinksToBookmarkFile(m, m.textInput.Value())
			default:
				log.Error("somehow got into UpdateExport with the wrong state",
					"appState", m.appState)
			}
			m.textInput.Blur()
			m.appState = appStateTriage
			UpdateTriage(m, nil)
		}
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
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
			if m.textArea.Focused() {
				m.textArea.Blur()
				err := tagLink(m, linkreader.LinkSaved, m.textArea.Value())
				if err != nil {
					// fuck
				}
				m.textArea.SetValue("")

				log.Info("switching to triage")
				m.appState = appStateTriage
				UpdateTriage(m, nil)

				// here - get value for tags from text area
			}
		case "ctrl+c":
			return m, tea.Quit
		default:
			if !m.textArea.Focused() {
				cmd = m.textArea.Focus()
				cmds = append(cmds, cmd)
			}
		}

		// We handle errors just like any other message
	case error:
		m.err = msg
		return m, nil
	}

	m.textArea, cmd = m.textArea.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}
