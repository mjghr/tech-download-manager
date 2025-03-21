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
	tables        []table.Model
	focused       bool
	width         int
	height        int
	queues        []*controller.QueueController
	activeTable   int
	statusMessage string
	showStatus    bool
	statusExpiry  time.Time
}

// NewModel creates a new model for the queues tab
func NewModel() Model {
	return Model{
		tables:        make([]table.Model, 0),
		focused:       true,
		activeTable:   0,
		showStatus:    false,
		statusMessage: "",
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles the key events
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	// Check if we need to expire a status message
	now := time.Now()
	if m.showStatus && now.After(m.statusExpiry) {
		m.showStatus = false
		m.statusMessage = ""
	}

	if m.focused && len(m.tables) > 0 {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "f1", "f2", "f3", "f4":
				// Handle function keys for entire queue operations
				if len(m.queues) > 0 {
					queueName := m.queues[m.activeTable].QueueName

					switch msg.String() {
					case "f1":
						// Start all downloads in the queue
						// Start in the main goroutine to ensure UI updates
						err := m.queues[m.activeTable].Start()
						if err != nil {
							logs.Log(fmt.Sprintf("Error starting queue: %v", err))
							m.statusMessage = fmt.Sprintf("Error: %v", err)
						} else {
							logs.Log(fmt.Sprintf("Started all downloads in queue: %s", queueName))
							m.statusMessage = "Started all downloads in queue"
						}
						m.showStatus = true
						m.statusExpiry = now.Add(3 * time.Second)
					case "f2":
						// Pause all downloads in the queue
						m.queues[m.activeTable].PauseAll()
						logs.Log(fmt.Sprintf("Paused all downloads in queue: %s", queueName))
						m.statusMessage = "Paused all downloads in queue"
						m.showStatus = true
						m.statusExpiry = now.Add(3 * time.Second)
					case "f3":
						// Resume all downloads in the queue
						m.queues[m.activeTable].ResumeAll()
						logs.Log(fmt.Sprintf("Resumed all downloads in queue: %s", queueName))
						m.statusMessage = "Resumed all downloads in queue"
						m.showStatus = true
						m.statusExpiry = now.Add(3 * time.Second)
					case "f4":
						// Cancel all downloads in the queue
						m.queues[m.activeTable].CancelAll()
						logs.Log(fmt.Sprintf("Cancelled all downloads in queue: %s", queueName))
						m.statusMessage = "Cancelled all downloads in queue"
						m.showStatus = true
						m.statusExpiry = now.Add(3 * time.Second)
					}

					// Force a screen refresh to show the updated state
					return m, tea.Batch(cmd, tea.ClearScreen)
				}
				// Still pass the message to the table to maintain its state
				m.tables[m.activeTable], cmd = m.tables[m.activeTable].Update(msg)

			case "up", "down":
				// Pass these keys to the active table
				m.tables[m.activeTable], cmd = m.tables[m.activeTable].Update(msg)

			case "j":
				// Cycle to the next queue, wrapping around if needed
				if len(m.tables) > 0 {
					m.tables[m.activeTable].Blur()
					m.activeTable = (m.activeTable + 1) % len(m.tables) // Wrap around
					m.tables[m.activeTable].Focus()
					// Force full screen refresh when switching queues
					return m, tea.Batch(cmd, tea.ClearScreen)
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

// Add styles
var headerStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color("205"))

var helpStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("241")).
	Italic(true)

var emptyStateStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("243")).
	Italic(true).
	Align(lipgloss.Center).
	Width(60).
	Padding(2).
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("240"))

// UpdateQueues updates the queue data
func (m *Model) UpdateQueues(queues []*controller.QueueController) {
	logs.Log(fmt.Sprintf("UpdateQueues called with %d queues", len(queues)))
	m.queues = queues
	m.tables = make([]table.Model, len(queues))

	// If no queues, return early
	if len(queues) == 0 {
		logs.Log("No queues available to display")
		return
	}

	// Define columns for download information
	columns := []table.Column{
		{Title: "ID", Width: 10},
		{Title: "Filename", Width: 30},
		{Title: "Status", Width: 12},
		{Title: "Progress", Width: 15},
		{Title: "Size", Width: 15},
		{Title: "Speed", Width: 15},
	}

	for i, queue := range queues {
		// Create table for this queue
		t := table.New(
			table.WithColumns(columns),
			table.WithFocused(i == m.activeTable && m.focused),
			table.WithHeight(10), // Increased height for better visibility
		)

		// Adjust table styling with more attractive colors
		t.SetStyles(table.Styles{
			Header: lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("205")). // Match our label style color
				Padding(0, 1).
				Border(lipgloss.NormalBorder(), false, false, true, false). // Bottom border only
				BorderForeground(lipgloss.Color("240")),
			Selected: lipgloss.NewStyle().
				Foreground(lipgloss.Color("231")). // Bright white for text
				Background(lipgloss.Color("63")).  // Deep blue background
				Bold(true),
			Cell: lipgloss.NewStyle().
				Padding(0, 1),
		})

		// Create rows for each download in the queue
		var rows []table.Row
		for _, download := range queue.DownloadControllers {
			// Calculate progress percentage (handle zero total size case)
			totalCompleted := 0
			for _, bytes := range download.CompletedBytes {
				totalCompleted += bytes
			}

			var progress float64
			if download.TotalSize > 0 {
				progress = float64(totalCompleted) / float64(download.TotalSize) * 100
			} else {
				progress = 0
			}

			// Format size in MB (handle zero case)
			var sizeMB float64
			if download.TotalSize > 0 {
				sizeMB = float64(download.TotalSize) / 1024 / 1024
			}

			// Format status with more engaging visual display
			statusText := formatStatus(download.Status)

			row := table.Row{
				download.ID,
				download.FileName,
				statusText,
				fmt.Sprintf("%.1f%%", progress),
				fmt.Sprintf("%.2f MB", sizeMB),
				fmt.Sprintf("%d KB/s", download.SpeedLimit/1024),
			}
			rows = append(rows, row)
		}

		t.SetRows(rows)
		m.tables[i] = t
		logs.Log(fmt.Sprintf("Table updated with %d rows for queue: %s (name: %s)",
			len(rows), queue.QueueID, queue.QueueName))
	}
}

// Helper function to format status with color-coded indicators
func formatStatus(status controller.Status) string {
	switch status {
	case controller.NOT_STARTED:
		return "○ Pending"
	case controller.PAUSED:
		return "⏸️ Paused"
	case controller.FAILED:
		return "✕ Failed"
	case controller.COMPLETED:
		return "✓ Done"
	case controller.ONGOING:
		return "➤ Active"
	case controller.CANCELED:
		return "⊘ Cancelled"
	default:
		return "? Unknown"
	}
}

// View renders the UI
func (m Model) View() string {
	// Create a container with fixed dimensions to prevent display issues
	containerStyle := lipgloss.NewStyle().
		Width(m.width-10).
		MaxHeight(m.height-10).
		Padding(1, 2).
		BorderStyle(lipgloss.HiddenBorder())

	// Start with an empty content buffer
	var sb strings.Builder

	if len(m.queues) == 0 {
		emptyMessage := "No queues available\n\nUse the NewQueue tab to create a queue first."
		return containerStyle.Render(emptyStateStyle.Render(emptyMessage))
	}

	// Display queue selector with a distinct style
	selectorBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(0, 1).
		MarginBottom(1).
		Width(m.width - 20)

	var queueSelector strings.Builder
	queueSelector.WriteString(lipgloss.NewStyle().Bold(true).Render("Select Queue: "))
	for i, queue := range m.queues {
		if i == m.activeTable {
			queueSelector.WriteString(lipgloss.NewStyle().
				Foreground(lipgloss.Color("205")).
				Bold(true).
				Background(lipgloss.Color("236")).
				Padding(0, 1).
				Render(queue.QueueName))
		} else {
			queueSelector.WriteString(lipgloss.NewStyle().
				Foreground(lipgloss.Color("252")).
				Padding(0, 1).
				Render(queue.QueueName))
		}

		if i < len(m.queues)-1 {
			queueSelector.WriteString(" | ")
		}
	}

	// Add the styled queue selector to the main view
	sb.WriteString(selectorBoxStyle.Render(queueSelector.String()))
	sb.WriteString("\n")

	// Display status message if active
	if m.showStatus {
		statusStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")). // Green color
			Bold(true).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("36")).
			Width(m.width-20).
			Align(lipgloss.Center).
			Padding(0, 1).
			MarginBottom(1)

		sb.WriteString(statusStyle.Render(m.statusMessage))
		sb.WriteString("\n")
	}

	// Display the selected queue details
	if m.activeTable < len(m.queues) {
		queue := m.queues[m.activeTable]

		// Queue details in its own styled box
		detailsBoxStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("241")).
			Padding(0, 1).
			MarginBottom(1).
			Width(m.width - 20)

		queueDetails := fmt.Sprintf(
			"Speed: %d KB/s • Concurrent: %d • Path: %s",
			queue.SpeedLimit/1024,
			queue.ConcurrentDownloadLimit,
			queue.SavePath,
		)

		sb.WriteString(detailsBoxStyle.Render(queueDetails))
		sb.WriteString("\n")

		// Show a message for queues with no downloads
		if len(queue.DownloadControllers) == 0 {
			noDownloadsMsg := lipgloss.NewStyle().
				Foreground(lipgloss.Color("243")).
				Italic(true).
				Align(lipgloss.Center).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("240")).
				Width(m.width - 24).
				Padding(1).
				Render("No downloads in this queue. Use the NewDownload tab to add downloads.")
			sb.WriteString(noDownloadsMsg)
		} else {
			// Add the table for the selected queue
			tableView := m.tables[m.activeTable].View()
			tableStyle := lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("63"))

			sb.WriteString(tableStyle.Render(tableView))
		}
	}

	// Add help text at the bottom with nice styling
	helpBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		Italic(true).
		MarginTop(2).
		Width(m.width - 20)

	sb.WriteString("\n\n" + helpBoxStyle.Render(helpText))

	// Render content inside the container to ensure consistent dimensions
	return containerStyle.Render(sb.String())
}

// SetSize allows the parent model to pass the new window dimensions on resize.
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height

	// Only resize tables if we have any
	if len(m.tables) > 0 {
		tableWidth := width - 20
		tableHeight := 10

		for i := range m.tables {
			m.tables[i].SetWidth(tableWidth)
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

const helpText = `
Controls:
  ↑/↓: Navigate rows
  j: Cycle through queues
  F1: Start all downloads in queue
  F2: Pause all downloads in queue
  F3: Resume all downloads in queue
  F4: Cancel all downloads in queue
`
