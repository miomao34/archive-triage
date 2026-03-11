package main

// These imports will be used later on the tutorial. If you save the file
// now, Go might complain they are unused, but that's fine.
// You may also need to run `go mod tidy` to download bubbletea and its
// dependencies.
import (
	"errors"
	"fmt"
	linkreader "miomao34/archive-triage/link_reader"
	"os"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/log"
)

const (
	appStateWelcome = iota
	appStateTriage
	appStateHelp
	appStateTags
)

type keyMap struct {
	Back     key.Binding
	Save     key.Binding
	Edit     key.Binding
	Postpone key.Binding
	Open     key.Binding
	Discard  key.Binding
	Context  key.Binding
	Welcome  key.Binding
	Help     key.Binding
	Quit     key.Binding
}

type sizes struct {
	dimensions []int

	linkAndNameLengthLimit int
	dupeLengthLimit        int

	numberOfDupesPerPage int
}

type model struct {
	link linkreader.Linker

	appState int
	sizes    sizes

	generator linkreader.LinkGenerator
	conn      *linkreader.DatabaseConnector

	selected        map[int]struct{}
	welcomeMessages map[int]string

	duplicateIDs         []int
	duplicates           []linkreader.Linker
	dupeIDNumberOfDigits int
	scroll               int

	textarea textarea.Model
	err      error

	keys keyMap
	help help.Model
}

func initialModel(generator linkreader.LinkGenerator, conn *linkreader.DatabaseConnector) model {
	err := generator.Run()
	if err != nil {
		// hehe
		log.Fatal("failed to run generator:", "err", err)
	}
	log.Debug("started link generator")

	ta := textarea.New()
	ta.Placeholder = "one tag per line"
	ta.SetVirtualCursor(false)
	ta.SetStyles(tagTextAreaStyle)

	// ta.Focus()

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
		Open: key.NewBinding(
			key.WithKeys("o"),
			key.WithHelp("o", "open link"),
		),
		Discard: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "discard link"),
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
		link: linkreader.Link{},

		generator: generator,
		conn:      conn,

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

		textarea: ta,

		keys: keys,
		help: help.New(),
	}
}

func (m model) Init() tea.Cmd {
	// Just return `nil`, which means "no I/O right now, please."
	return nil
}

func (m *model) NextLink() error {
	newLink, ok := <-m.generator.ReturnChannel
	if !ok {
		log.Debug("channel is closed!")
		return errors.New("channel is closed!")
	}

	m.link = newLink
	return nil
}

func (m *model) SizeCalculations(width int, height int) {
	m.sizes.dimensions[0], m.sizes.dimensions[1] = width, height

	// 2 spaces for borders
	m.sizes.linkAndNameLengthLimit = m.sizes.dimensions[0] - 2
	m.sizes.dupeLengthLimit = m.sizes.dimensions[0] - 2

	// 3 for all borders, 2 for current name and link
	// 1 dupe is 4 characters high
	m.sizes.numberOfDupesPerPage = (m.sizes.dimensions[1] - 3 - 2) / 4
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
	case appStateTags:
		return ViewTags(&m)
	}

	return tea.NewView("")

	// link_display := lipgloss.JoinVertical(lipgloss.Top, nameStyle.Render(string(m.currentName)), linkStyle.Render(string(m.currentLink)))
	// return lipgloss.JoinVertical(lipgloss.Top, link_display, m.buttons.View())
}

func main() {
	log.SetLevel(log.DebugLevel)
	log.Debug("starting!")

	if len(os.Args) < 4 {
		log.Fatal(`usage:
						./archive-triage <format> <input-filename> <output-db-filename>)
						format - one of: bookmark, extension, firefox`)
	}

	dbFilename := os.Args[3]
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

	log.Debug("opened db connection", "dbFilename", dbFilename)

	formatString := os.Args[1]
	var format linkreader.LinkFileFormatType
	filename := os.Args[2]
	switch formatString {
	case "bookmark":
		format = linkreader.BookmarkExportFormat
	case "extension":
		format = linkreader.ExtensionExportFormat
	case "firefox":
		format = linkreader.FirefoxShareTabsExportFormat
	default:
		log.Fatal("format should be one of: bookmark, extension, firefox")
	}
	generator := linkreader.LinkGenerator{}
	err = generator.ReadBookmarksFile(filename, format)
	if err != nil {
		// haha
		log.Fatal("failed to create generator:", "err", err)
	}
	log.Debug("created link generator")

	p := tea.NewProgram(initialModel(generator, conn))
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
