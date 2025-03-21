package guide

import (
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	table   table.Model
	focused bool
	width   int
	height  int
}

func NewModel() Model {
	columns := []table.Column{
		{Title: "Key", Width: 10},
		{Title: "Action", Width: 40},
	}

	rows := []table.Row{
		{"Tab", "Switch to next tab"},
		{"ESC", "Toggle focus on the current tab’s table"},
		{"Q", "Quit the application"},
		{"Ctrl+C", "Quit the application"},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(false),
		table.WithHeight(7),
	)

	return Model{
		table:   t,
		focused: false,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	if m.focused {
		m.table, cmd = m.table.Update(msg)
	}

	return m, cmd
}

func (m Model) View() string {
	return m.table.View()
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.table.SetWidth(width - 4)
	m.table.SetHeight(height - 10)
}

func (m *Model) ToggleFocus() {
	m.focused = !m.focused
	if m.focused {
		m.table.Focus()
	} else {
		m.table.Blur()
	}
}
