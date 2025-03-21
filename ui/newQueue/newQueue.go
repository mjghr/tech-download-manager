package newQueue

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mjghr/tech-download-manager/controller"
	"github.com/mjghr/tech-download-manager/manager"
	"github.com/mjghr/tech-download-manager/ui/logs"
	"github.com/mjghr/tech-download-manager/util"
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
)

// NewQueueModel represents the model for creating a new queue
type NewQueueModel struct {
	nameInput               textinput.Model
	savePathInput           textinput.Model
	concurrentDownloadInput textinput.Model
	speedLimitInput         textinput.Model
	focused                 bool
	activeInput             int
	nameError               bool
	downloadManager         *manager.DownloadManager
	successMessage          string
	showSuccessMessage      bool
	messageTimer            int
}

// NewModel initializes a new queue model
func NewModel(dm *manager.DownloadManager) NewQueueModel {
	nameInput := textinput.New()
	nameInput.Placeholder = "Enter queue name..."
	nameInput.Focus()

	savePathInput := textinput.New()
	savePathInput.Placeholder = "Enter save path (optional)..."

	concurrentDownloadInput := textinput.New()
	concurrentDownloadInput.Placeholder = "Enter concurrent download limit (optional)..."

	speedLimitInput := textinput.New()
	speedLimitInput.Placeholder = "Enter speed limit in KB/s (optional)..."

	return NewQueueModel{
		nameInput:               nameInput,
		savePathInput:           savePathInput,
		concurrentDownloadInput: concurrentDownloadInput,
		speedLimitInput:         speedLimitInput,
		focused:                 true,
		activeInput:             0,
		nameError:               false,
		downloadManager:         dm,
		successMessage:          "",
		showSuccessMessage:      false,
		messageTimer:            0,
	}
}

// validate checks if required fields are valid
func (m *NewQueueModel) validate() bool {
	// Only validate queue name as it's the only required field
	queueName := m.nameInput.Value()
	if queueName == "" {
		m.nameError = true
		return false
	}
	m.nameError = false
	return true
}

// Update handles UI updates
func (m NewQueueModel) Update(msg tea.Msg) (NewQueueModel, tea.Cmd) {
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
		case "f6":
			// Cycle through inputs: name -> savePath -> concurrentDownload -> speedLimit
			m.activeInput = (m.activeInput + 1) % 4

			m.nameInput.Blur()
			m.savePathInput.Blur()
			m.concurrentDownloadInput.Blur()
			m.speedLimitInput.Blur()

			switch m.activeInput {
			case 0:
				m.nameInput.Focus()
			case 1:
				m.savePathInput.Focus()
			case 2:
				m.concurrentDownloadInput.Focus()
			case 3:
				m.speedLimitInput.Focus()
			}

		case "enter":
			if m.validate() {
				// Create new queue
				queueName := m.nameInput.Value()

				// Create a new queue controller
				queueCtrl := controller.NewQueueController(queueName)

				// Get optional inputs
				savePath := m.savePathInput.Value()
				if savePath == "" {
					savePath = util.GiveDefaultSavePath()
				}

				// Default values
				concurrentLimit := 1
				speedLimit := 100 * 1024 // 100KB/s

				// Parse concurrent download limit if provided
				if m.concurrentDownloadInput.Value() != "" {
					var concurrentLimitVal int
					if _, err := fmt.Sscanf(m.concurrentDownloadInput.Value(), "%d", &concurrentLimitVal); err == nil && concurrentLimitVal > 0 {
						concurrentLimit = concurrentLimitVal
					} else {
						logs.Log(fmt.Sprintf("Invalid concurrent download limit, using default: %v", err))
					}
				}

				// Parse speed limit if provided
				if m.speedLimitInput.Value() != "" {
					var speedLimitKB int
					if _, err := fmt.Sscanf(m.speedLimitInput.Value(), "%d", &speedLimitKB); err == nil && speedLimitKB > 0 {
						speedLimit = speedLimitKB * 1024 // Convert KB/s to bytes/s
					} else {
						logs.Log(fmt.Sprintf("Invalid speed limit, using default: %v", err))
					}
				}

				// Update queue controller with the parsed values
				queueCtrl.UpdateQueueController(
					savePath,
					concurrentLimit,
					speedLimit,
					time.Now(),                   // Start time is now
					time.Now().Add(24*time.Hour), // End time is 24 hours from now
				)

				// Add the queue to the download manager
				m.downloadManager.AddQueue(queueCtrl)

				// Log the number of queues in the download manager for debugging
				logs.Log(fmt.Sprintf("Created new queue: %s with ID: %s", queueName, queueCtrl.QueueID))
				logs.Log(fmt.Sprintf("Download manager now has %d queues", len(m.downloadManager.QueueList)))

				// Save all queues to queues.json
				if err := controller.SaveQueueControllers("queues.json", m.downloadManager.QueueList); err != nil {
					logs.Log(fmt.Sprintf("Error saving queues: %v", err))
				} else {
					logs.Log("Queues saved successfully to queues.json")
				}

				// Set success message
				m.successMessage = fmt.Sprintf("Queue '%s' created successfully!", queueName)
				m.showSuccessMessage = true
				m.messageTimer = 0

				// Clear input fields after successful creation
				m.nameInput.SetValue("")
				m.savePathInput.SetValue("")
				m.concurrentDownloadInput.SetValue("")
				m.speedLimitInput.SetValue("")
				m.nameError = false
			}
		}
	}

	// Handle input updates for the focused input
	if m.focused {
		switch m.activeInput {
		case 0:
			m.nameInput, cmd = m.nameInput.Update(msg)
		case 1:
			m.savePathInput, cmd = m.savePathInput.Update(msg)
		case 2:
			m.concurrentDownloadInput, cmd = m.concurrentDownloadInput.Update(msg)
		case 3:
			m.speedLimitInput, cmd = m.speedLimitInput.Update(msg)
		}
	}

	return m, cmd
}

// View renders the UI
func (m NewQueueModel) View() string {
	// Create a fixed-size container for consistent rendering
	containerStyle := lipgloss.NewStyle().
		Width(m.nameInput.Width + 20).
		Padding(1).
		BorderStyle(lipgloss.HiddenBorder())

	var view strings.Builder

	// Queue name input with validation style
	view.WriteString(labelStyle.Render("Queue Name (required):") + "\n")
	nameView := m.nameInput.View()
	if m.nameError {
		nameView = errorStyle.Render(nameView)
	} else if m.nameInput.Focused() {
		nameView = focusedStyle.Render(nameView)
	} else {
		nameView = blurredStyle.Render(nameView)
	}
	view.WriteString(nameView + "\n\n")

	// Optional save path input
	view.WriteString(labelStyle.Render("Save Path (optional):") + "\n")
	savePathView := m.savePathInput.View()
	if m.savePathInput.Focused() {
		savePathView = focusedStyle.Render(savePathView)
	} else {
		savePathView = blurredStyle.Render(savePathView)
	}
	view.WriteString(savePathView + "\n\n")

	// Optional concurrent download limit input
	view.WriteString(labelStyle.Render("Concurrent Download Limit (optional):") + "\n")
	concurrentView := m.concurrentDownloadInput.View()
	if m.concurrentDownloadInput.Focused() {
		concurrentView = focusedStyle.Render(concurrentView)
	} else {
		concurrentView = blurredStyle.Render(concurrentView)
	}
	view.WriteString(concurrentView + "\n\n")

	// Optional speed limit input
	view.WriteString(labelStyle.Render("Speed Limit KB/s (optional):") + "\n")
	speedView := m.speedLimitInput.View()
	if m.speedLimitInput.Focused() {
		speedView = focusedStyle.Render(speedView)
	} else {
		speedView = blurredStyle.Render(speedView)
	}
	view.WriteString(speedView + "\n\n")

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
		Width(m.nameInput.Width + 16)

	hint := "Press Enter to create queue | Press F6 to switch between fields"
	view.WriteString(hintStyle.Render(hint))

	// Wrap in the container for consistent sizing
	return containerStyle.Render(view.String())
}

// SetSize updates the width of the input fields
func (m *NewQueueModel) SetSize(width, height int) {
	m.nameInput.Width = width - 4
	m.savePathInput.Width = width - 4
	m.concurrentDownloadInput.Width = width - 4
	m.speedLimitInput.Width = width - 4
}

// ToggleFocus toggles focus state
func (m *NewQueueModel) ToggleFocus() {
	m.focused = !m.focused
	if m.focused {
		switch m.activeInput {
		case 0:
			m.nameInput.Focus()
		case 1:
			m.savePathInput.Focus()
		case 2:
			m.concurrentDownloadInput.Focus()
		case 3:
			m.speedLimitInput.Focus()
		}
	} else {
		m.nameInput.Blur()
		m.savePathInput.Blur()
		m.concurrentDownloadInput.Blur()
		m.speedLimitInput.Blur()
	}
}
