package newDownloads

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mjghr/tech-download-manager/controller"
	"github.com/mjghr/tech-download-manager/manager"
	"github.com/mjghr/tech-download-manager/ui/logs"
)

// Setup styles for different input states
var (
	focusedStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(0, 1).
			Width(50)

	blurredStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1).
			Width(50)

	errorStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("196")).
			Padding(0, 1).
			Width(50)

	normalStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1).
			Width(50)

	labelStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205"))

	selectorStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(0, 1)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")). // Green color
			Bold(true)

	warningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("208")). // Orange color
			Italic(true)
)

// Add download manager to the model struct
type NewDownloadModel struct {
	urlInput           textinput.Model
	queues             []*controller.QueueController
	selectedQueue      int
	focused            bool
	activeInput        int
	urlError           bool
	downloadManager    *manager.DownloadManager
	successMessage     string
	showSuccessMessage bool
	messageTimer       int
}

// Update NewModel to accept download manager
func NewModel(dm *manager.DownloadManager) NewDownloadModel {
	urlInput := textinput.New()
	urlInput.Placeholder = "Enter download URL..."
	urlInput.Focus()

	return NewDownloadModel{
		urlInput:           urlInput,
		focused:            true,
		activeInput:        0,
		urlError:           false,
		downloadManager:    dm,
		successMessage:     "",
		showSuccessMessage: false,
		messageTimer:       0,
	}
}

func (m *NewDownloadModel) UpdateQueues(queues []*controller.QueueController) {
	m.queues = queues

	// Ensure selected queue is valid
	if len(m.queues) > 0 && m.selectedQueue >= len(m.queues) {
		m.selectedQueue = 0
	}
}

func (m *NewDownloadModel) validate() bool {
	// Only validate URL
	urlStr := m.urlInput.Value()
	if urlStr == "" {
		m.urlError = true
		return false
	}
	if _, err := url.Parse(urlStr); err != nil {
		m.urlError = true
		return false
	}
	m.urlError = false
	return true
}

func (m NewDownloadModel) Update(msg tea.Msg) (NewDownloadModel, tea.Cmd) {
	var cmd tea.Cmd

	// Handle the timer for success message
	if m.showSuccessMessage {
		m.messageTimer++
		if m.messageTimer > 10 { // roughly 2.5 seconds with a 250ms tick
			m.showSuccessMessage = false
			m.messageTimer = 0
		}
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "f5":
			m.activeInput = (m.activeInput + 1) % 2 // URL input + queue selection

			m.urlInput.Blur()
			if m.activeInput == 0 {
				m.urlInput.Focus()
			}

		case "enter":
			if m.activeInput == 1 { // Queue selection
				if len(m.queues) == 0 {
					logs.Log("Cannot add download: no queues available")
					m.successMessage = "Please create a queue first in the NewQueue tab."
					m.showSuccessMessage = true
					m.messageTimer = 0
					return m, cmd
				}

				if m.validate() {
					if urlStr := m.urlInput.Value(); urlStr != "" {
						if parsedURL, err := url.Parse(urlStr); err == nil {
							queue := m.queues[m.selectedQueue]
							logs.Log(fmt.Sprintf("Creating new download controller for URL: %s with Queue ID: %s", parsedURL.String(), queue.QueueID))

							// Create new download controller using manager
							dc := m.downloadManager.NewDownloadController(parsedURL)
							if dc != nil {
								queue.AddDownload(dc)
								logs.Log(fmt.Sprintf("Added download %s to queue %s", dc.ID, queue.QueueID))

								// Save all queues to queues.json after adding download
								if err := controller.SaveQueueControllers("queues.json", m.downloadManager.QueueList); err != nil {
									logs.Log(fmt.Sprintf("Error saving queues: %v", err))

								}

								// Set success message
								m.successMessage = fmt.Sprintf("Added '%s' to queue '%s'", dc.FileName, queue.QueueName)
								m.showSuccessMessage = true
								m.messageTimer = 0

								// Clear input and reset validation
								m.urlInput.SetValue("")
								m.urlError = false
							} else {
								logs.Log("Failed to create download controller")
								m.successMessage = "Failed to create download - check URL and try again."
								m.showSuccessMessage = true
								m.messageTimer = 0
							}
						}
					}
				}
			}

		case "up", "down":
			if m.activeInput == 1 && len(m.queues) > 0 { // Queue selection
				if msg.String() == "up" {
					m.selectedQueue = (m.selectedQueue - 1 + len(m.queues)) % len(m.queues)
				} else {
					m.selectedQueue = (m.selectedQueue + 1) % len(m.queues)
				}
			}
		}
	}

	// Handle input updates
	if m.focused && m.activeInput == 0 {
		m.urlInput, cmd = m.urlInput.Update(msg)
	}

	return m, cmd
}

func (m NewDownloadModel) View() string {
	// Create a fixed-size container for consistent rendering
	containerStyle := lipgloss.NewStyle().
		Width(m.urlInput.Width + 20).
		Padding(1).
		BorderStyle(lipgloss.HiddenBorder())

	var view strings.Builder

	// URL input with validation style
	view.WriteString(labelStyle.Render("URL (required):") + "\n")
	urlView := m.urlInput.View()
	if m.urlError {
		urlView = errorStyle.Render(urlView)
	} else if m.urlInput.Focused() {
		urlView = focusedStyle.Render(urlView)
	} else {
		urlView = blurredStyle.Render(urlView)
	}
	view.WriteString(urlView + "\n\n")

	// Queue selection
	view.WriteString(labelStyle.Render("Select Queue:") + "\n")

	if len(m.queues) == 0 {
		warningStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Italic(true).
			Padding(0, 1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("214"))

		warningText := "No queues available! Create a queue first."
		view.WriteString(warningStyle.Render(warningText) + "\n\n")
	} else {
		// Create a styled dropdown for queue selection
		queueBox := selectorStyle.Copy().Width(m.urlInput.Width)
		var queueContent strings.Builder

		for i, queue := range m.queues {
			if i == m.selectedQueue {
				queueContent.WriteString(lipgloss.NewStyle().
					Foreground(lipgloss.Color("205")).
					Bold(true).
					Background(lipgloss.Color("236")).
					Padding(0, 1).
					Render("▶ " + queue.QueueName))
			} else {
				queueContent.WriteString(lipgloss.NewStyle().
					Foreground(lipgloss.Color("252")).
					Padding(0, 1).
					Render("• " + queue.QueueName))
			}
			queueContent.WriteString("\n")
		}

		view.WriteString(queueBox.Render(queueContent.String()) + "\n\n")
	}

	// Show success message if needed
	if m.showSuccessMessage {
		successStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")).
			Bold(true).
			Padding(1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("36"))
		view.WriteString(successStyle.Render(m.successMessage) + "\n\n")
	}

	// Add hint text at the bottom with nice styling
	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Italic(true).
		Align(lipgloss.Center).
		Width(m.urlInput.Width + 16)

	hint := "Press Enter to add download | Press F5 to switch input and queue"
	view.WriteString(hintStyle.Render(hint))

	// Wrap in the container for consistent sizing
	return containerStyle.Render(view.String())
}

func (m *NewDownloadModel) SetSize(width, height int) {
	m.urlInput.Width = width - 4
}

func (m *NewDownloadModel) ToggleFocus() {
	m.focused = !m.focused
	if m.focused && m.activeInput == 0 {
		m.urlInput.Focus()
	} else {
		m.urlInput.Blur()
	}
}
