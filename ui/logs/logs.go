package logs

import (
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
)

// GlobalLogChannel is the channel used for sending log messages.
// var GlobalLogChannel = make(chan string, 1000)

// Log can be called from anywhere in your program to send a log message.
// For example: logs.Log(fmt.Sprintf(("Download started: file1.zip")
func Log(message string) {
	// GlobalLogChannel <- message
}

// LogMsg is a Bubble Tea message that wraps a log string.
type LogMsg string

// Model represents the logs tab's state.
type Model struct {
	table   table.Model
	focused bool
	width   int
	height  int
}

// NewModel creates a new logs model with an initial table.
func NewModel() Model {
	columns := []table.Column{
		{Title: "Time", Width: 8},     // Keep the "Time" column width as is
		{Title: "Message", Width: 50}, // Increase the "Message" column width
	}

	// Start with an initial log entry.
	rows := []table.Row{
		{"--:--:--", "Application started"},
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

// logListener is a command that waits for a log message from GlobalLogChannel.
// It blocks until a log is received, then returns it wrapped as a LogMsg.
func logListener() tea.Cmd {
	return func() tea.Msg {
		// msg := <-GlobalLogChannel
		// return LogMsg(msg)
		return ""
	}
}

// Init starts the logListener command.
func (m Model) Init() tea.Cmd {
	return logListener()
}

// Update listens for log messages and updates the table accordingly.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case LogMsg:
		// Only update if the message isn't empty.
		if msg != "" {
			timestamp := time.Now().Format("15:04:05")
			newRow := table.Row{timestamp, string(msg)}
			// Append the new row to the current rows.
			rows := append(m.table.Rows(), newRow)
			m.table.SetRows(rows)
		}
		// Restart the logListener to wait for the next message.
		return m, logListener()

	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, msg.Height)
		return m, nil

	case error:
		// In case of an error message, log it.
		timestamp := time.Now().Format("15:04:05")
		errorMsg := table.Row{timestamp, "Error: " + msg.Error()}
		rows := append(m.table.Rows(), errorMsg)
		m.table.SetRows(rows)
		return m, logListener()
	}

	// When the table is focused, let it process other messages.
	if m.focused {
		var tableCmd tea.Cmd
		m.table, tableCmd = m.table.Update(msg)
		cmd = tea.Batch(cmd, tableCmd)
	}

	return m, cmd
}

// View renders the logs table.
func (m Model) View() string {
	return m.table.View()
}

// SetSize allows the parent model to adjust the table's size on window resize.
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height

	// Dynamically adjust the column widths
	messageColumnWidth := width - 12 // Subtract the "Time" column width and padding
	if messageColumnWidth < 20 {     // Ensure a minimum width for the "Message" column
		messageColumnWidth = 20
	}

	m.table.SetColumns([]table.Column{
		{Title: "Time", Width: 8},
		{Title: "Message", Width: messageColumnWidth},
	})

	m.table.SetWidth(width - 4)
	m.table.SetHeight(height - 10)
}

// ToggleFocus toggles the table's focus.
func (m *Model) ToggleFocus() {
	m.focused = !m.focused
	if m.focused {
		m.table.Focus()
	} else {
		m.table.Blur()
	}
}
