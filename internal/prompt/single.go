package prompt

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	questionStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	answerStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

type Result struct {
	Answer  string
	Skipped bool
}

type model struct {
	question string
	input    string
	done     bool
	skipped  bool
	timeout  *time.Timer
	timedOut bool
}

type timeoutMsg struct{}

func (m model) Init() tea.Cmd {
	if m.timeout != nil {
		return func() tea.Msg {
			<-m.timeout.C
			return timeoutMsg{}
		}
	}
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case timeoutMsg:
		m.skipped = true
		m.done = true
		return m, tea.Quit

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			m.done = true
			return m, tea.Quit
		case tea.KeyEsc, tea.KeyCtrlC:
			m.skipped = true
			m.done = true
			return m, tea.Quit
		case tea.KeyBackspace, tea.KeyDelete:
			if len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
			}
		case tea.KeySpace:
			m.input += " "
		default:
			if msg.Type == tea.KeyRunes {
				m.input += string(msg.Runes)
			}
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.done {
		return ""
	}
	cursor := dimStyle.Render("▌")
	return "\n" + questionStyle.Render(m.question) + "\n" +
		dimStyle.Render("  › ") + answerStyle.Render(m.input) + cursor + "\n" +
		dimStyle.Render("  (Enter to submit, Esc to skip)") + "\n"
}

// Ask shows a single prompt and returns the result. timeoutSecs ≤ 0 means no timeout.
func Ask(question string, timeoutSecs int) Result {
	m := model{question: question}
	if timeoutSecs > 0 {
		m.timeout = time.NewTimer(time.Duration(timeoutSecs) * time.Second)
		defer m.timeout.Stop()
	}

	opts := []tea.ProgramOption{tea.WithInputTTY()}
	p := tea.NewProgram(m, opts...)
	final, err := p.Run()
	if err != nil {
		return Result{Skipped: true}
	}
	fm := final.(model)
	if fm.skipped || fm.input == "" {
		return Result{Skipped: true}
	}
	return Result{Answer: fm.input}
}
