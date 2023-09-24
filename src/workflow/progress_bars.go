package workflow

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	padding  = 2
	maxWidth = 80
)

type progressMsg struct {
	workflowsDone int
}

type progressDoneMsg struct{}

type updateMessageMsg struct {
	message string
}

type ProgressModel struct {
	progress    progress.Model
	mainMessage string
	message     string
	length      int
}

func (m ProgressModel) Init() tea.Cmd {
	return nil
}

var helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render

func tickCmd(msg tea.Msg) tea.Cmd {
	return tea.Tick(time.Second*1, func(t time.Time) tea.Msg {
		return msg
	})
}

func (m ProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.progress.Width = msg.Width - padding*2 - 4
		if m.progress.Width > maxWidth {
			m.progress.Width = maxWidth
		}
		return m, nil

	case progressDoneMsg:
		return m, tea.Quit

	case progressMsg:
		percentage := float64(msg.workflowsDone) / float64(m.length)
		cmd := m.progress.SetPercent(percentage)
		if percentage == 1.0 {
			cmd = tea.Batch(tickCmd(progressDoneMsg{}), cmd)
		}
		return m, cmd

	case updateMessageMsg:
		m.message = msg.message
		return m, nil

	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		return m, cmd

	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
		return m, nil
	default:
		return m, nil
	}
}

func (m ProgressModel) View() string {
	pad := strings.Repeat(" ", padding)
	return "\n" +
		pad + helpStyle(m.mainMessage) + "\n\n" +
		pad + m.progress.View() + "\n\n" +
		pad + helpStyle(m.message) + "\n\n"

}
