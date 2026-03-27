package main

// These imports will be used later on the tutorial. If you save the file
// now, Go might complain they are unused, but that's fine.
// You may also need to run `go mod tidy` to download bubbletea and its
// dependencies.
import (
	"fmt"
	linkreader "miomao34/archive-triage/link_reader"
	"os"

	"charm.land/bubbles/v2/filepicker"
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/log"
)

const (
	appStateWelcome = iota
	appStateTriage
	appStateIngestPickFile
	appStateIngestPickFormat
	appStateHelp
	appStateTags
)

type keyMap struct {
	Back           key.Binding
	Save           key.Binding
	Edit           key.Binding
	Postpone       key.Binding
	ResetPostponed key.Binding
	Open           key.Binding
	Ingest         key.Binding
	Discard        key.Binding
	Undo           key.Binding
	Context        key.Binding
	Welcome        key.Binding
	Help           key.Binding
	Quit           key.Binding
}

type sizes struct {
	dimensions []int

	topCellWidth    int
	middleCellWidth int
	bottomCellWidth int

	topCellHeight    int
	middleCellHeight int
	bottomCellHeight int

	numberOfDupesPerPage int
}

type model struct {
	id   int
	link linkreader.Linker

	appState int
	sizes    sizes

	conn *linkreader.DatabaseConnector

	selected        map[int]struct{}
	welcomeMessages map[int]string

	duplicateIDs []int
	duplicates   []linkreader.Linker

	stats *linkreader.DatabaseStats

	textarea     textarea.Model
	filepicker   filepicker.Model
	selectedFile string

	formats []string
	cursor  int

	err error

	keys keyMap
	help help.Model
}

func initialModel(conn *linkreader.DatabaseConnector) model {
	ta := textarea.New()
	ta.Placeholder = "one tag per line"
	ta.SetVirtualCursor(false)
	ta.SetStyles(tagTextAreaStyle)

	// ta.Focus()

	fp := filepicker.New()
	fp.AllowedTypes = []string{".txt", ".html"}
	fp.AutoHeight = false
	fp.CurrentDirectory, _ = os.Getwd()

	var keys = keyMap{
		Back: key.NewBinding(
			key.WithKeys("b"),
			key.WithHelp("b", "go back"),
		),
		Save: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "save link"),
		),
		Edit: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "edit link"),
		),
		Postpone: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "postpone/snooze link"),
		),
		ResetPostponed: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "mark postponed links unprocessed again"),
		),
		Open: key.NewBinding(
			key.WithKeys("o"),
			key.WithHelp("o", "open link"),
		),
		Ingest: key.NewBinding(
			key.WithKeys("i"),
			key.WithHelp("i", "ingest a link file"),
		),
		Discard: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "discard link"),
		),
		Undo: key.NewBinding(
			key.WithKeys("u"),
			key.WithHelp("u", "mark last processed link unprocessed again, delete its tags"),
		),
		Context: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "increase link context window"),
		),
		Welcome: key.NewBinding(
			key.WithKeys("w"),
			key.WithHelp("w", "go to the splash screen"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "toggle help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "esc", "ctrl+c"),
			key.WithHelp("q/esc/ctrl-c", "quit"),
		),
	}

	sizes := sizes{
		dimensions: make([]int, 2),
	}

	return model{
		id:   0,
		link: linkreader.Link{},

		conn: conn,

		appState: appStateWelcome,
		sizes:    sizes,

		// A map which indicates which choices are selected. We're using
		// the  map like a mathematical set. The keys refer to the indexes
		// of the `choices` slice, above.
		selected: make(map[int]struct{}),
		welcomeMessages: map[int]string{
			0: "hello and welcome to the black mesa research facility",
			1: `    ___              __    _            
   /   |  __________/ /_  / /   _____   
  / /| | / ___/ ___/ __ \/ / | / / _ \  
 / ___ |/ /  / /__/ / / / /| |/ /  __/  
/_/  |_/_/   \___/_/ /_/ / |___/\___/   
           ______     / /
          /_  __/____/ /___  ____  ___ 
           / / / ___/ / __ \/ __ \/ _ \
          / / / /  / / /_/ / /_/ /  __/
         /_/ /_/  /_/\__,_/\__, /\___/ 
                          /____/       
`,
		},

		stats: &(linkreader.DatabaseStats{}),

		textarea:   ta,
		filepicker: fp,

		cursor: 0,
		formats: []string{
			"ExtensionExportFormat",
			"BookmarkExportFormat",
			"FirefoxShareTabsExportFormat",
		},

		keys: keys,
		help: help.New(),
	}
}

func (m model) Init() tea.Cmd {
	return m.filepicker.Init()
}

func (m *model) NextLink() error {
	id, newLink, err := m.conn.GetUnresolvedLink()
	if err != nil {
		log.Info("no more unresolved links")
		return err
	}

	m.id = id
	m.link = newLink
	return nil
}

func (m *model) SizeCalculations(width int, height int) {
	m.sizes.dimensions[0], m.sizes.dimensions[1] = width, height

	// 2 spaces for borders
	m.sizes.topCellWidth = m.sizes.dimensions[0] - 2
	m.sizes.middleCellWidth = m.sizes.dimensions[0] - 2
	m.sizes.bottomCellWidth = m.sizes.dimensions[0] - 2

	// 2, one for link and one for name
	m.sizes.topCellHeight = 2
	// 4 for borders, 2 for top cell, 1 for bottom cell
	m.sizes.middleCellHeight = m.sizes.dimensions[1] - 4 - 2 - 1
	m.sizes.bottomCellHeight = 1

	// 3 for all borders, 2 for current name and link
	// 1 dupe is 4 characters high
	m.sizes.numberOfDupesPerPage = (m.sizes.dimensions[1] - 3 - 2) / 4

	// idk why this 1 is necessary
	// fixme move me someplace else
	m.filepicker.SetHeight(m.sizes.middleCellHeight - 2)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.link.GetHREF() == nil {
		err := m.NextLink()
		if err != nil {
			// todo
		}
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SizeCalculations(msg.Width, msg.Height)
	}

	switch m.appState {
	case appStateWelcome:
		return UpdateWelcome(&m, msg)
	case appStateTriage:
		return UpdateTriage(&m, msg)
	case appStateIngestPickFile:
		return UpdateIngestPickFile(&m, msg)
	case appStateIngestPickFormat:
		return UpdateIngestPickFormat(&m, msg)
	case appStateTags:
		return UpdateTags(&m, msg)
	}

	return m, nil
}

func (m model) View() tea.View {
	switch m.appState {
	case appStateWelcome:
		return ViewWelcome(&m)
	case appStateTriage:
		return ViewTriage(&m)
	case appStateIngestPickFile:
		return ViewIngestPickFile(&m)
	case appStateIngestPickFormat:
		return ViewIngestPickFormat(&m)
	case appStateTags:
		return ViewTags(&m)
	}

	return tea.NewView("")

	// link_display := lipgloss.JoinVertical(lipgloss.Top, nameStyle.Render(string(m.currentName)), linkStyle.Render(string(m.currentLink)))
	// return lipgloss.JoinVertical(lipgloss.Top, link_display, m.buttons.View())
}

func main() {
	f, err := os.OpenFile("log.txt", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		log.Fatal("failed to open logging to file! da hell", "err", err)
	}
	log.SetOutput(f)

	// log.SetFormatter(log.LogfmtFormatter)
	log.SetLevel(log.DebugLevel)
	log.Debug(">>>>starting!<<<<")

	if len(os.Args) < 2 {
		fmt.Println(`usage:
		./archive-triage <db-filename>`)
		os.Exit(1)
	}

	dbFilename := os.Args[1]
	conn, err := linkreader.OpenConnection(dbFilename)
	if err != nil {
		log.Fatal("failed to open db connection:", "err", err)
	}
	defer func() {
		err = conn.Close()
		if err != nil {
			log.Error("failed to close db connection:", "err", err)
		}
	}()

	p := tea.NewProgram(initialModel(conn))
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
