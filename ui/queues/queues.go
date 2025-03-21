package queues

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mjghr/tech-download-manager/controller"
	"github.com/mjghr/tech-download-manager/ui/logs"
)

// Model for the "Queues" tab.
type Model struct {
	tables      []table.Model
	focused     bool
	width       int
	height      int
	queues      []*controller.QueueController
	activeTable int
}

// Update the NewModel function
func NewModel() Model {
	return Model{
		tables:      make([]table.Model, 0),
		focused:     true,
		activeTable: 0,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

// Update the Update method to handle table navigation
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	if m.focused && len(m.tables) > 0 {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "up", "down":
				// Pass these keys to the active table
				m.tables[m.activeTable], cmd = m.tables[m.activeTable].Update(msg)
			case "j":
				// Move to next table
				if m.activeTable < len(m.tables)-1 {
					m.tables[m.activeTable].Blur()
					m.activeTable++
					m.tables[m.activeTable].Focus()
				}
			case "k":
				// Move to previous table
				if m.activeTable > 0 {
					m.tables[m.activeTable].Blur()
					m.activeTable--
					m.tables[m.activeTable].Focus()
				}
			default:
				// Pass other keys to the active table
				m.tables[m.activeTable], cmd = m.tables[m.activeTable].Update(msg)
			}
		}
	}

	return m, cmd
}

// Add a helper function to create a queue info header
func createQueueInfoHeader(queue *controller.QueueController) string {
	return fmt.Sprintf(
		"Queue ID: %s | Speed Limit: %d KB/s | Concurrent Limit: %d | Start: %s | End: %s",
		queue.QueueID,
		queue.SpeedLimit/1024,
		queue.ConcurrentDownloadLimit,
		formatTime(queue.StartTime),
		formatTime(queue.EndTime),
	)
}

// Helper function for time formatting
func formatTime(t time.Time) string {
	if t.IsZero() {
		return "--:--:--"
	}
	return t.Format("15:04:05")
}

// Update the UpdateQueues method
func (m *Model) UpdateQueues(queues []*controller.QueueController) {
	logs.Log(fmt.Sprintf("UpdateQueues called with %d queues", len(queues)))
	m.queues = queues
	m.tables = make([]table.Model, len(queues))

	// Define columns for download information
	columns := []table.Column{
		{Title: "ID", Width: 10},
		{Title: "Filename", Width: 30},
		{Title: "Status", Width: 10},
		{Title: "Progress", Width: 15},
		{Title: "Size", Width: 15},
		{Title: "Speed", Width: 15},
	}

	for i, queue := range queues {
		// Create table for this queue
		t := table.New(
			table.WithColumns(columns),
			table.WithFocused(i == m.activeTable),
			table.WithHeight(5),
		)

		// Create rows for each download in the queue
		var rows []table.Row
		for _, download := range queue.DownloadControllers {
			// Calculate progress percentage
			totalCompleted := 0
			for _, bytes := range download.CompletedBytes {
				totalCompleted += bytes
			}
			progress := float64(totalCompleted) / float64(download.TotalSize) * 100

			// Format size in MB
			sizeMB := float64(download.TotalSize) / 1024 / 1024

			row := table.Row{
				download.ID,
				download.FileName,
				formatStatus(download.Status),
				fmt.Sprintf("%.1f%%", progress),
				fmt.Sprintf("%.2f MB", sizeMB),
				fmt.Sprintf("%d KB/s", download.SpeedLimit/1024),
			}
			rows = append(rows, row)
		}

		t.SetRows(rows)
		m.tables[i] = t
		logs.Log(fmt.Sprintf("Table updated with %d rows for queue: %s", len(rows), queue.QueueID))
	}
}

// Helper function to format status
func formatStatus(status controller.Status) string {
	switch status {
	case controller.NOT_STARTED:
		return "Pending"
	case controller.PAUSED:
		return "Paused"
	case controller.FAILED:
		return "Failed"
	case controller.COMPLETED:
		return "Done"
	case controller.ONGOING:
		return "Active"
	default:
		return "Unknown"
	}
}

// Update the View method to apply styles
func (m Model) View() string {
	if len(m.queues) == 0 {
		logs.Log("Queue view called with empty queues")
		return "No queues available"
	}

	var sb strings.Builder
	for i, queue := range m.queues {
		// Add queue header
		sb.WriteString(headerStyle.Render(createQueueInfoHeader(queue)))
		sb.WriteString("\n\n")

		// Add queue's downloads table with appropriate style
		var tableContent string
		if i == m.activeTable && m.focused {
			tableContent = lipgloss.NewStyle().
				BorderStyle(lipgloss.DoubleBorder()).
				BorderForeground(lipgloss.Color("63")).Render(m.tables[i].View())
		} else {
			tableContent = lipgloss.NewStyle().
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("240")).Render(m.tables[i].View())
		}
		sb.WriteString(tableContent)
		sb.WriteString("\n\n")
	}

	logs.Log(fmt.Sprintf("Rendering table view with %d queues", len(m.queues)))
	return sb.String()
}

// SetSize allows the parent model to pass the new window dimensions on resize.
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height

	// Only resize tables if we have any
	if len(m.tables) > 0 {
		tableHeight := (height - 10) / len(m.tables)
		for i := range m.tables {
			m.tables[i].SetWidth(width - 4)
			m.tables[i].SetHeight(tableHeight)
		}
	}
}

// ToggleFocus toggles whether this table is focused.
func (m *Model) ToggleFocus() {
	m.focused = !m.focused

	// Only modify table focus if we have tables
	if len(m.tables) > 0 {
		if m.focused {
			m.tables[m.activeTable].Focus()
		} else {
			m.tables[m.activeTable].Blur()
		}
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
			fmt.Sprintf("%d", queue.ConcurrentDownloadLimit),
			startTime,
			endTime,
		})
	}

	// Update the table with the new rows
	m.table.SetRows(rows)
}
