package newDownloads

import (
	"fmt"
	"net/url"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mjghr/tech-download-manager/controller"
	"github.com/mjghr/tech-download-manager/manager"
	"github.com/mjghr/tech-download-manager/ui/logs"
)

// Add validation styles at the top of the file
var (
	normalStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240"))

	errorStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("196")) // Red color
)

// Add download manager to the model struct
type NewDownloadModel struct {
	urlInput        textinput.Model
	queues          []*controller.QueueController
	selectedQueue   int
	focused         bool
	activeInput     int
	urlError        bool
	downloadManager *manager.DownloadManager
}

// Update NewModel to accept download manager
func NewModel(dm *manager.DownloadManager) NewDownloadModel {
	urlInput := textinput.New()
	urlInput.Placeholder = "Enter download URL..."
	urlInput.Focus()

	return NewDownloadModel{
		urlInput:        urlInput,
		focused:         true,
		activeInput:     0,
		urlError:        false,
		downloadManager: dm,
	}
}

func (m *NewDownloadModel) UpdateQueues(queues []*controller.QueueController) {
	m.queues = queues
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

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "l":
			m.activeInput = (m.activeInput + 1) % 2 // URL input + queue selection

			m.urlInput.Blur()
			if m.activeInput == 0 {
				m.urlInput.Focus()
			}

		case "enter":
			if m.activeInput == 1 { // Queue selection
				if m.validate() {
					if urlStr := m.urlInput.Value(); urlStr != "" {
						if parsedURL, err := url.Parse(urlStr); err == nil {
							if len(m.queues) > 0 {
								queue := m.queues[m.selectedQueue]
								logs.Log(fmt.Sprintf("Creating new download controller for URL: %s with Queue ID: %s", parsedURL.String(), queue.QueueID))

								// Create new download controller using manager
								dc := m.downloadManager.NewDownloadController(parsedURL)
								if dc != nil {
									queue.AddDownload(dc)
									logs.Log(fmt.Sprintf("Added download %s to queue %s", dc.ID, queue.QueueID))

									// Clear input and reset validation
									m.urlInput.SetValue("")
									m.urlError = false
								} else {
									logs.Log("Failed to create download controller")
								}
							} else {
								logs.Log("Warning: No queues available to add download")
							}
						}
					}
				}
			}

		case "up", "down":
			if m.activeInput == 1 { // Queue selection
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
	var view string

	// Show URL input with validation style
	view += "URL:\n"
	urlView := m.urlInput.View()
	if m.urlError {
		urlView = errorStyle.Render(urlView)
	} else {
		urlView = normalStyle.Render(urlView)
	}
	view += urlView + "\n\n"

	// Show queue selection
	view += "Select Queue (use up/down):\n"
	for i, queue := range m.queues {
		prefix := "  "
		if i == m.selectedQueue && m.activeInput == 1 {
			prefix = "➜ "
		}
		view += prefix + queue.QueueName + "\n"
	}

	view += "\nPress Enter to add download"

	return view
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
