package jrex

import (
	"errors"
	"fmt"
	"strings"

	"github.com/cgalvisleon/et/stdrout"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	cliInputStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
	cliSeparatorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

const cliHelpText = `available commands:
  /build [major|minor|release]   bump the version (default: release) and publish to the store
  /help    show this help message
  /quit, /q exit the CLI`

type cliLogMsg struct {
	kind    string
	message string
}

type cliCmdResultMsg struct {
	kind    string
	message string
	err     error
}

type cliModel struct {
	jrex     *Jrex
	viewport viewport.Model
	input    textinput.Model
	lines    []string
	ready    bool
}

/**
* newCliModel: Builds the CLI model that pairs a scrolling log viewport with a
* command input line for the given Jrex instance.
* @param jrex *Jrex
* @return cliModel
**/
func newCliModel(jrex *Jrex) cliModel {
	input := textinput.New()
	input.Prompt = "> "
	input.Placeholder = "/build"
	input.PromptStyle = cliInputStyle
	input.Focus()

	return cliModel{
		jrex:  jrex,
		input: input,
	}
}

/**
* Init: Starts the input cursor blinking.
* @return tea.Cmd
**/
func (m cliModel) Init() tea.Cmd {
	return textinput.Blink
}

/**
* Update: Handles window resizes, key presses and async log/command messages.
* @param msg tea.Msg
* @return tea.Model, tea.Cmd
**/
func (m cliModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.resize(msg.Width, msg.Height)
		return m, nil
	case tea.MouseMsg:
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			line := strings.TrimSpace(m.input.Value())
			m.input.SetValue("")
			if line == "" {
				return m, nil
			}
			m.appendLine(fmt.Sprintf("> %s", line))
			return m, m.dispatch(line)
		}
	case cliLogMsg:
		m.appendLog(msg.kind, msg.message)
		return m, nil
	case cliCmdResultMsg:
		if msg.err != nil {
			m.appendLog(msg.kind+" error", msg.err.Error())
		} else {
			m.appendLog(msg.kind, msg.message)
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

/**
* View: Renders the log viewport, a separator, and the command input line.
* @return string
**/
func (m cliModel) View() string {
	if !m.ready {
		return "Starting jrex...\n"
	}

	separator := cliSeparatorStyle.Render(strings.Repeat("─", m.viewport.Width))
	return fmt.Sprintf("%s\n%s\n%s", m.viewport.View(), separator, m.input.View())
}

/**
* resize: Adjusts the viewport and input widths to the terminal size, leaving
* room for the separator and input line.
* @params width int, height int
**/
func (m *cliModel) resize(width, height int) {
	const reservedLines = 2 // separator + input line

	viewportHeight := max(0, height-reservedLines)
	if !m.ready {
		m.viewport = viewport.New(width, viewportHeight)
		m.viewport.SetContent(strings.Join(m.lines, "\n"))
		m.ready = true
	} else {
		m.viewport.Width = width
		m.viewport.Height = viewportHeight
	}
	m.input.Width = max(0, width-len(m.input.Prompt)-1)
}

/**
* appendLine: Appends a line to the log viewport and scrolls to the bottom.
* @param line string
**/
func (m *cliModel) appendLine(line string) {
	m.lines = append(m.lines, line)
	if m.ready {
		m.viewport.SetContent(strings.Join(m.lines, "\n"))
		m.viewport.GotoBottom()
	}
}

/**
* appendLog: Appends a kind/message pair to the log viewport, splitting the
* message on line breaks so each line renders as its own row in the viewport.
* @params kind, message string
**/
func (m *cliModel) appendLog(kind, message string) {
	for line := range strings.SplitSeq(message, "\n") {
		m.appendLine(fmt.Sprintf("[%s] %s", kind, line))
	}
}

/**
* dispatch: Parses a command line and returns the tea.Cmd that runs it.
* @param line string
* @return tea.Cmd
**/
func (m *cliModel) dispatch(line string) tea.Cmd {
	fields := strings.Fields(line)
	name := fields[0]

	switch name {
	case "/build":
		part := "same"
		if len(fields) > 1 {
			part = fields[1]
		}
		return func() tea.Msg {
			return m.runBuild(part)
		}
	case "/help":
		m.appendLine(cliHelpText)
		return nil
	case "/quit", "/q":
		return tea.Quit
	default:
		m.appendLine(fmt.Sprintf("unknown command: %s (try /help)", name))
		return nil
	}
}

/**
* runBuild: Runs Build with the default release bump as a tea.Cmd.
* @return tea.Msg
**/
func (m *cliModel) runBuild(part string) tea.Msg {
	if map[string]bool{
		"same":    true,
		"major":   true,
		"minor":   true,
		"release": true,
	}[part] == false {
		return cliCmdResultMsg{kind: "Build", err: errors.New("invalid part")}
	}

	err := m.jrex.Build(Part(part))
	if err != nil {
		return cliCmdResultMsg{kind: "Build", err: err}
	}
	return cliCmdResultMsg{kind: "Build", message: fmt.Sprintf("done — v%s", m.jrex.Version)}
}

/**
* RunCli: Launches the split-pane CLI (log viewport + command input) bound to
* this Jrex instance, starts hot-reload watching, and blocks until the user exits.
* @return error
**/
func (s *Jrex) RunCli() error {
	program := tea.NewProgram(newCliModel(s), tea.WithAltScreen(), tea.WithMouseCellMotion())
	s.program = program
	stdrout.SetStdout(s)
	defer func() { s.program = nil }()

	if s.onStart != nil {
		go func() {
			if err := s.onStart(s); err != nil {
				s.Notify("Error", err.Error())
			}
		}()
	}

	go func() {
		if err := s.hotReload(); err != nil {
			s.Notify("Error", err.Error())
		}
	}()

	_, err := program.Run()
	return err
}
