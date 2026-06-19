package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wordwrap"

	"visrosa/tasklineUI/internal/taskdata"
)

type actions int

const (
	task actions = iota
	board
	help
	quit
)

type state int

const (
	stateInitializing state = iota
	stateNormal
	stateAddTask
	stateBoard
	stateBoardFilled
	stateHelp
)

var (
	highlight     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("5"))
	bold          = lipgloss.NewStyle().Bold(true)
	greyed        = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	logo          = lipgloss.NewStyle().Foreground(lipgloss.Color("54")).Italic(true).Bold(true)
	logoUI        = lipgloss.NewStyle().Background(lipgloss.Color("54")).Foreground(lipgloss.Color("5"))
	sectionHeader = lipgloss.NewStyle().Bold(true).Underline(true)
	dim           = lipgloss.NewStyle().Faint(true)
	italic        = lipgloss.NewStyle().Italic(true)
)

type model struct {
	tasks        []taskdata.Task
	loadErr      string
	taskSummary  []string // derived from the loaded pending tasks, not a separate query
	state        state
	input        textinput.Model
	viewport     viewport.Model
	awViewport   viewport.Model // action-words strip
	windowHeight int
	windowWidth  int
	quitting     bool
	boardText    string
	taskText     string
}

// reload re-reads pending tasks from taskwarrior's store and recomputes the summary line.
func (m *model) reload() {
	tasks, err := taskdata.LoadTasks()
	if err != nil {
		m.loadErr = err.Error()
		m.tasks = nil
		m.taskSummary = nil
		return
	}
	m.loadErr = ""
	m.tasks = tasks
	m.taskSummary = summaryLines(tasks)
}

// contentLines is what gets wrapped and handed to the main viewport.
func (m model) contentLines() []string {
	if m.loadErr != "" {
		return []string{"(error loading tasks: " + m.loadErr + ")"}
	}
	return renderTasks(m.tasks)
}

func (m model) setViewportContent() viewport.Model {
	vp := m.viewport
	wrapped := wordwrap.String(strings.Join(m.contentLines(), "\n"), vp.Width)
	vp.SetContent(wrapped)
	return vp
}

func initialModel() model {
	input := textinput.New()
	input.CharLimit = 256
	input.Width = 120
	input.Prompt = ""

	m := model{
		state: stateInitializing,
		input: input,
	}
	m.reload()

	vp := viewport.New(80, 20) // Default size before detecting the actual dimensions
	wrapped := wordwrap.String(strings.Join(m.contentLines(), "\n"), vp.Width-2)
	vp.SetContent(wrapped)
	m.viewport = vp

	m.awViewport = viewport.New(vp.Width, 2) // Action-words strip: task + board rows

	return m
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := strings.ToLower(msg.String())
		switch key {
		case "up":
			m.viewport.ScrollUp(1)
			return m, nil
		case "down":
			m.viewport.ScrollDown(1)
			return m, nil
		case "pgup":
			m.viewport.HalfPageUp()
			return m, nil
		case "pgdown":
			m.viewport.HalfPageDown()
			return m, nil

		case "q":
			if m.state == stateNormal || m.state == stateHelp || m.state == stateBoardFilled {
				m.quitting = true
				return m, tea.Quit
			}
		case "h", "?":
			if m.state == stateNormal || m.state == stateBoardFilled {
				m.state = stateHelp
				return m, nil
			}

			if m.state == stateHelp {
				m.state = stateNormal
				return m, nil
			}

		case "esc":
			if m.state == stateNormal {
				m.quitting = true
				return m, tea.Quit
			}
			if m.state == stateAddTask || m.state == stateBoard {
				m.input.Blur()
				m.state = stateNormal
				return m, nil
			}
			m.state = stateNormal
			return m, nil

		}
		switch m.state {
		case stateNormal:
			switch key {
			case "t":
				m.state = stateAddTask
				m.input.SetValue(m.taskText)
				m.input.Focus()
				return m, textinput.Blink
			case "b":
				m.state = stateBoard
				m.input.SetValue("")
				m.input.Focus()
				return m, textinput.Blink
			}
		case stateBoardFilled:
			switch key {
			case "t":
				m.state = stateAddTask
				m.input.SetValue("")
				m.input.Focus()
				return m, textinput.Blink
			case "b":
				m.state = stateBoard
				m.input.SetValue(m.boardText)
				m.input.Focus()
				return m, textinput.Blink
			case "esc":
				m.input.SetValue("")
				m.state = stateNormal
				return m, nil
			}
			// Optionally handle other keys for navigation, etc.
			return m, nil

		case stateAddTask, stateBoard:
			switch key {

			case "tab":
				if m.state == stateAddTask {
					m.taskText = strings.TrimSpace(m.input.Value())
					m.input.SetValue(m.taskText)
					m.state = stateBoard
				}
				return m, nil
			case "enter":
				val := strings.TrimSpace(m.input.Value())
				if val != "" {
					switch m.state {
					case stateBoard:
						m.boardText = val
						m.state = stateBoardFilled
						m.input.Blur()
						return m, nil
					case stateAddTask:
						taskdata.AddTask(val, m.boardText) // ignore error for now
						m.input.Blur()
						m.reload()
						m.viewport = m.setViewportContent()
						if m.boardText != "" {
							m.state = stateBoardFilled
						} else {
							m.state = stateNormal
						}
						return m, nil
					}
				}
			}
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd

		}
	case tea.WindowSizeMsg:
		m.windowHeight = msg.Height - 7 //3 for header + 2
		m.windowWidth = msg.Width - 2
		if m.state == stateInitializing {
			m.viewport = viewport.New(m.windowWidth, m.windowHeight)
			m.viewport = m.setViewportContent()
			m.state = stateNormal
		} else {
			m.viewport.Height = m.windowHeight
			m.viewport.Width = (m.windowWidth)
			m.viewport = m.setViewportContent()
		}
		return m, nil
	}
	return m, nil
}

func (m model) actionWord(aW actions) string {

	switch m.state {
	case stateAddTask:
		if aW == task {
			return highlight.PaddingLeft(1).Render("task")
		}
		if aW == board {
			return highlight.Render("⭾") + "board" + greyed.Render(m.boardText)
		}
		return greyed.PaddingLeft(1).Render(map[actions]string{help: "help", quit: "quit"}[aW])

	case stateBoardFilled:
		if aW == board {
			return highlight.PaddingLeft(1).Render("b") + "oard " + greyed.Render(m.boardText)
		}
		fallthrough

	default:
		var key, rest string
		switch aW {
		case task:
			key, rest = "t", "ask"
		case board:
			key, rest = "b", "oard"
		case help:
			key, rest = "H", "elp"
		case quit:
			key, rest = "Q", "uit"
		}
		return highlight.PaddingLeft(1).Render(key) + bold.Render(rest)
	}
}

// getHeader returns the header string, using Kitty double-size if available
func getHeader() string {
	if os.Getenv("KITTY_WINDOW_ID") != "" {
		return "\x1b[3m\x1b[38;5;54m\x1b]66;s=3;taskline\x07\x1b[0m" + "\x1b[3m\x1b[38;5;5m\x1b[48;5;54m\x1b]66;s=3;ꭐ\x07\x1b[0m" + "\n "
	} // color 127 for highlight

	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Render(logo.Render("taskline") + logoUI.Render("ꭐ"))
}

func (m model) viewTitle() string {
	return lipgloss.NewStyle().Render(getHeader())
}

func (m model) viewActionWords() string {
	headerTask := m.actionWord(task)
	headerBoard := m.actionWord(board)

	// Calculate max width across both lines
	availableWidth := m.windowWidth - 12 // Width of title+shortcuts

	if os.Getenv("KITTY_WINDOW_ID") != "" {
		availableWidth = m.windowWidth - 28
	}

	headerStyle := lipgloss.NewStyle().Width(availableWidth)
	if m.state == stateAddTask {
		headerTask = headerStyle.Render(headerTask + " " + m.input.View())
	} else if m.state == stateBoard {
		headerBoard = headerStyle.Render(headerBoard + " " + m.input.View())
	}

	//Pad all lines to max width for vertical stability
	headerTask = headerStyle.Render(headerTask) + greyed.Render(m.taskText)
	headerBoard = headerStyle.Render(headerBoard) + greyed.Render(m.boardText)

	content := lipgloss.JoinVertical(lipgloss.Left, headerTask, headerBoard)
	m.awViewport.Width = availableWidth
	m.awViewport.Height = 2
	m.awViewport.SetContent(content)
	return m.awViewport.View()
}

func (m model) viewHeader() string {
	return lipgloss.JoinHorizontal(lipgloss.Top, m.viewTitle(), " ", m.viewActionWords())
}

func (m model) viewScrollbar() string {

	// The middle part: repeat " │ \n" for (height-2) lines, to account for header/footer
	barCount := m.viewport.Height - 3
	if barCount < 0 {
		barCount = 0
	}

	// Header and footer for the scrollbar
	header := "🭽\n▏" + highlight.Render("↑") + "\n"
	bars := strings.Repeat("▏\n", barCount)
	// Add up arrow at top, down arrow at bottom (optional, can stylize)
	footer := "▏" + highlight.Render("↓") + "\n🭼"
	// Combine everything
	scrollbar := header + bars + footer
	if m.viewport.TotalLineCount() != m.viewport.VisibleLineCount() {

		return lipgloss.NewStyle().Width(3).Height(m.viewport.Height).Render(scrollbar)
	} else {
		return ""
	}
}

func (m model) viewViewport() string {
	viewp := lipgloss.NewStyle().Width(m.viewport.Width).Height(m.viewport.Height).MarginTop(0).Render(m.viewport.View())
	return lipgloss.JoinHorizontal(lipgloss.Top, m.viewScrollbar(), viewp)
}

func (m model) viewSummary() string {
	if len(m.taskSummary) == 0 {
		return ""
	}
	return lipgloss.NewStyle().MarginTop(1).Width(m.viewport.Width).Render(strings.Join(m.taskSummary, "\n"))
}

func (m model) viewFooter() string {
	footerText := lipgloss.JoinHorizontal(1, m.actionWord(help), " | ", m.actionWord(quit))

	// Define your help text here
	var helpText strings.Builder
	helpText.WriteString(highlight.Render("↑") + "/" + highlight.Render("↓"))
	helpText.WriteString("|")
	helpText.WriteString(highlight.Render("⇞") + "/" + highlight.Render("⇟"))
	helpText.WriteString(" scroll")
	helpText.WriteString("	")
	helpText.WriteString(lipgloss.NewStyle().Bold(true).Render("Shortcuts are highlighted"))
	helpText.WriteString("	")
	helpText.WriteString(highlight.Render("h") + "|" + highlight.Render("⎋") + ": hide help" + "	")
	helpText.WriteString(highlight.Render("q") + "|" + highlight.Render("⎈c") + ": quit")

	if m.state == stateHelp {

		left := lipgloss.NewStyle().Width(m.windowWidth - lipgloss.Width(footerText)).Align(lipgloss.Center).Render(helpText.String())
		return lipgloss.JoinHorizontal(lipgloss.Top, left, footerText)
	}

	// Default: action words only, right aligned
	return lipgloss.NewStyle().Width(m.windowWidth).Align(lipgloss.Right).Render(footerText)
}

func (m model) View() string {
	if m.state == stateInitializing {
		return "Initializing...\n"
	}

	var b strings.Builder
	b.WriteString(m.viewHeader())
	b.WriteString("\n")
	b.WriteString(m.viewViewport())
	summary := m.viewSummary()
	if summary != "" {
		b.WriteString(summary)
	}

	b.WriteString("\n")
	b.WriteString(m.viewFooter())
	return b.String()
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	_, err := p.Run()
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
