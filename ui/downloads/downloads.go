package downloads

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mjghr/tech-download-manager/controller"
	"github.com/mjghr/tech-download-manager/ui/logs"
)

// KeyMap defines the keybindings for the downloads list
type KeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Escape key.Binding
}

// DefaultKeyMap returns the default keybindings
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Escape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "toggle focus"),
		),
	}
}

// Model for the Downloads List tab
type Model struct {
	table         table.Model
	keymap        KeyMap
	help          help.Model
	width         int
	height        int
	focused       bool
	allDownloads  []*controller.DownloadController
	statusMessage string
	showStatus    bool
	statusExpiry  time.Time
	queues        []*controller.QueueController
}

// NewModel creates a new model for the downloads list
func NewModel() Model {
	keymap := DefaultKeyMap()
	helpModel := help.New()
	helpModel.Width = 60

	// Define columns for the downloads table
	columns := []table.Column{
		{Title: "URL", Width: 40},
		{Title: "Queue", Width: 20},
		{Title: "Status", Width: 15},
		{Title: "Progress", Width: 15},
		{Title: "Speed", Width: 15},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	// Style the table
	t.SetStyles(table.Styles{
		Header: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			Padding(0, 1).
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(lipgloss.Color("240")),
		Selected: lipgloss.NewStyle().
			Foreground(lipgloss.Color("231")).
			Background(lipgloss.Color("63")).
			Bold(true),
		Cell: lipgloss.NewStyle().
			Padding(0, 1),
	})

	return Model{
		table:         t,
		keymap:        keymap,
		help:          helpModel,
		focused:       true,
		allDownloads:  make([]*controller.DownloadController, 0),
		showStatus:    false,
		statusMessage: "",
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles key events and updates the model
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	// Check if we need to expire a status message
	now := time.Now()
	if m.showStatus && now.After(m.statusExpiry) {
		m.showStatus = false
		m.statusMessage = ""
	}

	if m.focused {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			// Pass navigation keys to the table
			case key.Matches(msg, m.keymap.Up), key.Matches(msg, m.keymap.Down):
				m.table, cmd = m.table.Update(msg)
			}
		}
	}

	// Let the table handle any other messages
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// formatStatus converts a Status value to a human-readable string
func formatStatus(status controller.Status) string {
	switch status {
	case controller.NOT_STARTED:
		return "⏳ Queued"
	case controller.PAUSED:
		return "⏸️ Paused"
	case controller.FAILED:
		return "❌ Failed"
	case controller.COMPLETED:
		return "✅ Completed"
	case controller.ONGOING:
		return "⬇️ Downloading"
	case controller.CANCELED:
		return "🚫 Canceled"
	default:
		return "Unknown"
	}
}

// View renders the component UI
func (m Model) View() string {
	if len(m.allDownloads) == 0 {
		// Empty state message when no downloads are available
		emptyStateStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			Italic(true).
			Align(lipgloss.Center).
			Width(60).
			Padding(2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240"))

		emptyState := emptyStateStyle.Render("No downloads found.\nAdd a new download using the 'New Download' tab.")
		helpView := m.helpView()

		return lipgloss.JoinVertical(
			lipgloss.Left,
			lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("205")).
				Render("Downloads List"),
			"\n",
			emptyState,
			"\n",
			helpView,
		)
	}

	// Status message
	var statusView string
	if m.showStatus {
		statusStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Padding(0, 1)
		statusView = statusStyle.Render(m.statusMessage)
	}

	// Help view with keyboard shortcuts
	helpView := m.helpView()

	// Return the complete view
	return lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			Render("Downloads List"),
		"\n",
		m.table.View(),
		"\n",
		statusView,
		"\n",
		helpView,
	)
}

// helpView returns the help text showing keyboard shortcuts
func (m Model) helpView() string {
	var helpEntries []string

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Italic(true)

	if m.focused {
		// When focused, show only navigation commands
		helpEntries = []string{
			"↑/↓: navigate",
			"esc: toggle focus",
		}
	} else {
		// When not focused, show just a hint to focus
		helpEntries = []string{
			"esc: focus downloads list",
		}
	}

	return helpStyle.Render(strings.Join(helpEntries, " | "))
}

// UpdateDownloads refreshes the downloads list
func (m *Model) UpdateDownloads(queues []*controller.QueueController) {
	logs.Log(fmt.Sprintf("UpdateDownloads called with %d queues", len(queues)))

	// Store the queues reference for use in download operations
	m.queues = queues

	// Collect all downloads from all queues
	var allDownloads []*controller.DownloadController
	for _, queue := range queues {
		for _, download := range queue.DownloadControllers {
			// Ensure the download has a proper queue ID if it's missing
			if download.QueueID == "" {
				download.QueueID = queue.QueueID
			}
			allDownloads = append(allDownloads, download)
		}
	}

	m.allDownloads = allDownloads

	// Create rows for the table
	var rows []table.Row
	for _, download := range allDownloads {
		// Calculate progress percentage
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

		// Calculate current speed (simplified version)
		speedKBps := float64(download.SpeedLimit) / 1024.0

		// Format the URL (truncate if too long)
		displayUrl := truncateString(download.Url, 40)

		// Find queue name
		queueName := "Unknown"
		for _, queue := range queues {
			// Check both the QueueID and whether download is in queue.DownloadControllers
			if queue.QueueID == download.QueueID {
				queueName = queue.QueueName
				break
			} else {
				// Double-check by iterating through the queue's downloads
				for _, queueDownload := range queue.DownloadControllers {
					if queueDownload.ID == download.ID {
						queueName = queue.QueueName
						// Also fix the QueueID for future reference
						download.QueueID = queue.QueueID
						break
					}
				}
			}
		}

		row := table.Row{
			displayUrl,
			queueName,
			formatStatus(download.Status),
			fmt.Sprintf("%.1f%%", progress),
			fmt.Sprintf("%.1f KB/s", speedKBps),
		}
		rows = append(rows, row)
	}

	m.table.SetRows(rows)
	logs.Log(fmt.Sprintf("Updated downloads list with %d downloads", len(allDownloads)))
}

// Helper function to truncate strings
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// SetSize updates the component size
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height

	// Adjust table height to fit in available space (leaving room for header, status, and help)
	tableHeight := height - 10
	if tableHeight < 5 {
		tableHeight = 5
	}

	m.table.SetHeight(tableHeight)

	// Create updated columns with new widths
	columns := []table.Column{
		{Title: "URL", Width: width / 3},
		{Title: "Queue", Width: width / 6},
		{Title: "Status", Width: width / 6},
		{Title: "Progress", Width: width / 6},
		{Title: "Speed", Width: width / 6},
	}

	// Create a new table with updated columns but preserve other properties
	newTable := table.New(
		table.WithColumns(columns),
		table.WithFocused(m.focused),
		table.WithHeight(tableHeight),
		table.WithRows(m.table.Rows()),
	)

	// Set the same table styling
	newTable.SetStyles(table.Styles{
		Header: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			Padding(0, 1).
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(lipgloss.Color("240")),
		Selected: lipgloss.NewStyle().
			Foreground(lipgloss.Color("231")).
			Background(lipgloss.Color("63")).
			Bold(true),
		Cell: lipgloss.NewStyle().
			Padding(0, 1),
	})

	// Replace the old table
	m.table = newTable

	m.help.Width = width
}

// ToggleFocus toggles the focus state of the component
func (m *Model) ToggleFocus() {
	m.focused = !m.focused
	if m.focused {
		m.table.Focus()
	} else {
		m.table.Blur()
	}
}
