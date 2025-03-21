package queues

import (
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mjghr/tech-download-manager/controller"
)

// Model for the "Queues" tab.
type Model struct {
	table   table.Model
	focused bool
	width   int
	height  int
	queues  []*controller.QueueController
}

// NewModel creates a new Queues model with a sample table of downloads.
func NewModel() Model {
	// Define columns
	columns := []table.Column{
		{Title: "File", Width: 20},
		{Title: "Status", Width: 10},
		{Title: "Progress", Width: 10},
	}

	// Define some sample rows
	rows := []table.Row{
		{"file1.zip", "downloading", "56%"},
		{"video.mp4", "paused", "22%"},
		{"music.mp3", "complete", "100%"},
		{"image.png", "downloading", "10%"},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true), // start focused by default
		table.WithHeight(7),
	)

	// You can customize table styles here or rely on your global style
	return Model{
		table:   t,
		focused: true,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages for the queues model.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	if m.focused {
		// Pass messages to the table if focused
		m.table, cmd = m.table.Update(msg)
	}

	return m, cmd
}

// View returns a string representation of this tab’s UI.
func (m Model) View() string {
	return m.table.View()
}

// SetSize allows the parent model to pass the new window dimensions on resize.
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height

	// Adjust table height/width if desired:
	m.table.SetWidth(width/2 - 4)  // or any logic you want
	m.table.SetHeight(height - 10) // or any logic you want
}

// ToggleFocus toggles whether this table is focused.
func (m *Model) ToggleFocus() {
	m.focused = !m.focused
	m.table.Blur()
	if m.focused {
		m.table.Focus()
	} else {
		m.table.Blur()
	}
}

// Add this new method
func (m *Model) UpdateQueues(queues []*controller.QueueController) {
	m.queues = queues
	// Update the table or other UI elements with the queue information
	// This will depend on how you want to display the queues
}
