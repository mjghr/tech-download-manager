package queues

import (
	"fmt"

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

// Update the NewModel function to initialize an empty table
func NewModel() Model {
	// Define columns
	columns := []table.Column{
		{Title: "Queue ID", Width: 20},
		{Title: "Speed Limit", Width: 15},
		{Title: "Concurrent Limit", Width: 18},
		{Title: "Start Time", Width: 20},
		{Title: "End Time", Width: 20},
	}

	// Initialize an empty table with no rows
	t := table.New(
		table.WithColumns(columns),
		table.WithRows([]table.Row{}), // Start with no rows
		table.WithFocused(true),       // Start focused by default
		table.WithHeight(7),
	)

	return Model{
		table:   t,
		focused: true,
		queues:  []*controller.QueueController{},
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

// Update the View function to display the table
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

// Add this method to update the table with real queue data
func (m *Model) UpdateQueues(queues []*controller.QueueController) {
	m.queues = queues

	// Convert queue data into table rows
	rows := []table.Row{}
	for _, queue := range queues {
		startTime := "--:--:--"
		endTime := "--:--:--"

		// Format start and end times if they are set
		if !queue.StartTime.IsZero() {
			startTime = queue.StartTime.Format("15:04:05")
		}
		if !queue.EndTime.IsZero() {
			endTime = queue.EndTime.Format("15:04:05")
		}

		// Add a row for each queue
		rows = append(rows, table.Row{
			queue.QueueID,
			fmt.Sprintf("%d KB/s", queue.SpeedLimit/1024), // Convert speed limit to KB/s
			fmt.Sprintf("%d", queue.ConcurrenDownloadtLimit),
			startTime,
			endTime,
		})
	}

	// Update the table with the new rows
	m.table.SetRows(rows)
}
