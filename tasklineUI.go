package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wordwrap"
)

type actions int

const (
	task actions = iota
	note
	board
	help
	quit
)

type state int

const (
	stateInitializing state = iota
	stateNormal
	stateAddTask
	stateAddNote
	stateBoard
	stateBoardFilled
	stateHelp
)

var (
	highlight = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("5"))
	bold      = lipgloss.NewStyle().Bold(true)
	greyed    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

type model struct {
	lines        []string // raw output lines from taskline (main content)
	taskSummary  []string // last two lines from taskline output
	state        state
	input        textinput.Model
	viewport     viewport.Model
	awViewport   viewport.Model // new viewport for action words
	windowHeight int
	windowWidth  int
	quitting     bool
	boardText    string
}

func loadLines() (mainLines []string, taskSummary []string) {
	// unbuffer preserves taskline's colors
	cmd := exec.Command("unbuffer", "taskline")
	out, err := cmd.Output()
	if err != nil {
		return []string{"(error running taskline)"}, nil
	}
	lines := strings.Split(strings.ReplaceAll(string(out), "\r\n", "\n"), "\n")

	// Always skip the first line and last two lines as they are empty or control codes
	if len(lines) > 3 {
		lines = lines[1 : len(lines)-2]
		mainLines = lines[:len(lines)-2]
		taskSummary = lines[len(lines)-2:]
	} else {
		mainLines = lines
		taskSummary = nil
	}
	return
}

func initialModel() model {
	input := textinput.New()
	input.CharLimit = 256
	input.Width = 120
	input.Prompt = ""
	mainLines, taskSummary := loadLines()
	vp := viewport.New(80, 20) // Default size before detecting the actual dimensions
	wrapped := wordwrap.String(strings.Join(mainLines, "\n"), vp.Width-2)
	vp.SetContent(wrapped)
	// This sets a visible scrollbar style (you can customize colors)
	// vp.Style = vp.Style.
	// 	Border(lipgloss.NormalBorder()).
	// 	BorderForeground(lipgloss.Color("8"))

	awVp := viewport.New(vp.Width, 3) // Action words viewport: height 3
	return model{
		lines:       mainLines,
		taskSummary: taskSummary,
		state:       stateInitializing,
		input:       input,
		viewport:    vp,
		awViewport:  awVp,
	}
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
		// case "t":
		// 	m.state = stateAddTask
		// 	m.input.SetValue("")
		// 	m.input.Focus()
		// 	return m, textinput.Blink
		// case "n":
		// 	m.state = stateAddNote
		// 	m.input.SetValue("")
		// 	m.input.Focus()
		// 	return m, textinput.Blink
		// case "b":
		// 	m.state = stateBoard
		// 	m.input.SetValue("")
		// 	m.input.Focus()
		// 	return m, textinput.Blink
		case "q":
			if m.state == stateNormal || m.state == stateHelp {
				m.quitting = true
				return m, tea.Quit
			}
		case "h", "?":
			if m.state == stateHelp {
				m.state = stateNormal
				return m, nil
			}
			m.state = stateHelp
			return m, nil
		}
		switch m.state {
		case stateNormal:
			switch key {
			case "t":
				m.state = stateAddTask
				m.input.SetValue("")
				m.input.Focus()
				return m, textinput.Blink
			case "n":
				m.state = stateAddNote
				m.input.SetValue("")
				m.input.Focus()
				return m, textinput.Blink
			case "b":
				m.state = stateBoard
				m.input.SetValue("")
				m.input.Focus()
				return m, textinput.Blink
			case "q":
				m.quitting = true
				return m, tea.Quit
			case "h", "?":
				m.state = stateHelp
				return m, nil
			}
		case stateHelp:
			switch key {
			case "esc":
				m.state = stateNormal
				return m, nil
			}

		case stateAddTask, stateAddNote, stateBoard:
			switch key {
			case "esc":
				m.input.Blur()
				m.state = stateNormal
				return m, nil
			case "tab":
				if m.state == stateAddTask || m.state == stateAddNote {
					m.state = stateBoard
				}
			case "enter":
				val := strings.TrimSpace(m.input.Value())
				if val != "" {
					var cmd *exec.Cmd
					if m.state == stateAddTask {
						if m.boardText != "" {
							cmd = exec.Command("taskline", "t", val, "-b", m.boardText)
						} else {
							cmd = exec.Command("taskline", "t", val)
						}
					} else if m.state == stateAddNote {
						if m.boardText != "" {
							cmd = exec.Command("taskline", "n", val, "-b", m.boardText)
						} else {
							cmd = exec.Command("taskline", "n", val)
						}
					} else if m.state == stateBoard {
						m.boardText = val
						m.state = stateBoardFilled
						m.input.Blur()
						return m, nil
					} else {
						return m, nil // Should not happen, but just in case
					}
					cmd.Run() // ignore error for now
					m.input.Blur()
					mainLines, taskSummary := loadLines()
					m.lines = mainLines
					m.taskSummary = taskSummary
					m.viewport.SetContent(strings.Join(mainLines, "\n"))
					if m.state == stateAddTask || m.state == stateAddNote {
						if m.boardText != "" {
							m.state = stateBoardFilled
						} else {
							m.state = stateNormal
						}
					}
					return m, nil
				}
			}
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		case stateBoardFilled:
			switch key {
			case "t":
				m.state = stateAddTask
				m.input.SetValue("")
				m.input.Focus()
				return m, textinput.Blink
			case "n":
				m.state = stateAddNote
				m.input.SetValue("")
				m.input.Focus()
				return m, textinput.Blink
			case "b":
				m.state = stateBoard
				m.input.SetValue(m.boardText)
				m.input.Focus()
				return m, textinput.Blink
			}
			// Optionally handle other keys for navigation, etc.
			return m, nil
		}
	case tea.WindowSizeMsg:
		m.windowHeight = msg.Height
		m.windowWidth = msg.Width
		if m.state == stateInitializing {
			m.viewport = viewport.New(m.windowWidth-2, m.windowHeight-9)
			wrapped := wordwrap.String(strings.Join(m.lines, "\n"), m.viewport.Width)
			m.viewport.SetContent(wrapped)
			m.state = stateNormal
		} else {
			m.viewport.Height = m.windowHeight - 9
			m.viewport.Width = (m.windowWidth - 2)
			wrapped := wordwrap.String(strings.Join(m.lines, "\n"), m.viewport.Width)
			m.viewport.SetContent(wrapped)
		}
		return m, nil
	}
	return m, nil
}

func (m model) actionWord(aW actions) string {
	switch m.state {
	case stateAddTask:
		if aW == task {
			return highlight.Render("task")
		}
		if aW == board {
			return highlight.Render("⭾ ") + "board" + greyed.Render(m.boardText)
		}
		return greyed.Render(map[actions]string{note: "note", help: "help", quit: "quit"}[aW])

	case stateAddNote:
		if aW == note {
			return highlight.Render("note")
		}
		if aW == board {
			return highlight.Render("⭾") + " board"
		}
		return greyed.Render(map[actions]string{task: "task", help: "help", quit: "quit"}[aW])

	case stateBoardFilled:
		if aW == board {
			return highlight.Render("b") + "oard " + greyed.Render(m.boardText)
		}
		fallthrough

	default:
		var key, rest string
		switch aW {
		case task:
			key, rest = "t", "ask"
		case note:
			key, rest = "n", "ote"
		case board:
			key, rest = "b", "oard"
		case help:
			key, rest = "H", "elp"
		case quit:
			key, rest = "Q", "uit"
		}
		return highlight.Bold(true).Render(key) + lipgloss.NewStyle().Bold(true).Render(rest)
	}
}

// getHeader returns the header string, using Kitty double-size if available
func getHeader() string {
	if os.Getenv("KITTY_WINDOW_ID") != "" {
		// return "\x1b]66;s=3;taskline\xE2\x83\xAC\x07\n "
		// return "\x1b[3m\x1b]66;s=3;t͟a͟s͟k͟l͟i͟n͟e͟\x07\x1b[0m" + "\x1b[2m\x1b]66;s=3;ꭐ\x07\x1b[0m" + "\n "
		//return "\x1b[3m\x1b]66;s=3;taskline\x07\x1b[0m" + "\x1b[2;3m\x1b]66;s=3;ꭐ\x07\x1b[0m" + "\n "
		return "\x1b[3m\x1b[38;5;54m\x1b]66;s=3;taskline\x07\x1b[0m" + "\x1b[3m\x1b[38;5;5m\x1b[48;5;54m\x1b]66;s=3;ꭐ\x07\x1b[0m" + "\n "
	} // color 127 for highlight
	return "\ntaskline\n "
}

func (m model) viewTitle() string {
	return lipgloss.NewStyle().Render(getHeader())
}

func (m model) viewActionWords() string {
	headerTask := m.actionWord(task)
	headerNote := m.actionWord(note)
	headerBoard := m.actionWord(board)

	if m.state == stateAddTask {
		headerTask = lipgloss.NewStyle().Width(32).Render(headerTask + " " + m.input.View())
	} else if m.state == stateAddNote {
		headerNote = lipgloss.NewStyle().Width(32).Render(headerNote + " " + m.input.View())
	} else if m.state == stateBoard {
		headerBoard = lipgloss.NewStyle().Width(32).Render(headerBoard + " " + m.input.View())
	}

	// Calculate max width across all three lines
	maxWidth := lipgloss.Width(headerTask)
	if w := lipgloss.Width(headerNote); w > maxWidth {
		maxWidth = w
	}
	if w := lipgloss.Width(headerBoard); w > maxWidth {
		maxWidth = w
	}

	// Pad all lines to max width for vertical stability
	headerTask = lipgloss.NewStyle().Width(maxWidth).Render(headerTask)
	headerNote = lipgloss.NewStyle().Width(maxWidth).Render(headerNote)
	headerBoard = lipgloss.NewStyle().Width(maxWidth).Render(headerBoard)

	content := lipgloss.JoinVertical(lipgloss.Left, headerTask, headerNote, headerBoard)
	m.awViewport.Width = maxWidth
	m.awViewport.Height = 3
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

// func (m model) preFooter() string {
// 	info := lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).Padding(0, 1).Render(fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))
// 	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(info)))
// 	return lipgloss.JoinHorizontal(lipgloss.Center, line, info)
// }

func (m model) viewSummary() string {
	if len(m.taskSummary) == 0 {
		return ""
	}
	return lipgloss.NewStyle().MarginTop(1).Width(m.viewport.Width).Render(strings.Join(m.taskSummary, "\n"))
}

func (m model) viewFooter() string {
	footerText := lipgloss.JoinHorizontal(1, m.actionWord(help), " | ", m.actionWord(quit))
	// footaW := lipgloss.NewStyle().Align(lipgloss.Right).Width(m.windowWidth).Render(footerText)

	// Define your help text here
	var helpText strings.Builder
	helpText.WriteString(highlight.Render("↑") + "/" + highlight.Render("↓"))
	helpText.WriteString("|")
	helpText.WriteString(highlight.Render("⇞") + "/" + highlight.Render("⇟"))
	helpText.WriteString(" scroll")
	helpText.WriteString("	")
	helpText.WriteString(lipgloss.NewStyle().Bold(true).Render("Highlighted glyphs are shortcuts"))
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
	// b.WriteString(m.preFooter())
	summary := m.viewSummary()
	if summary != "" {
		// b.WriteString("\n")
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
