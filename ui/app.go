package ui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mjghr/tech-download-manager/config"
	"github.com/mjghr/tech-download-manager/controller"
	"github.com/mjghr/tech-download-manager/manager"
	"github.com/mjghr/tech-download-manager/ui/downloads"
	"github.com/mjghr/tech-download-manager/ui/guide"
	"github.com/mjghr/tech-download-manager/ui/logs"
	"github.com/mjghr/tech-download-manager/ui/newDownloads"
	"github.com/mjghr/tech-download-manager/ui/newQueue"
	"github.com/mjghr/tech-download-manager/ui/queues"
)

// Add these new types near the top of the file
type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// AppModel is our root Bubble Tea model.
type AppModel struct {
	tabs            []string
	activeTab       int
	width           int
	height          int
	footerText      string
	downloadManager *manager.DownloadManager
	ready           bool

	// Sub-models (each tab)
	queuesModel        queues.Model
	guideModel         guide.Model
	newDownloadModel   newDownloads.NewDownloadModel
	newQueueModel      newQueue.NewQueueModel
	downloadsListModel downloads.Model
}

// NewAppModel initializes the root model with default values.
func NewAppModel() AppModel {
	dm := &manager.DownloadManager{}

	return AppModel{
		tabs:            []string{"NewDownload", "NewQueue", "Queues", "Downloads", "Guide"},
		activeTab:       0,
		footerText:      "Press Tab to switch tabs | Press ESC to toggle focus | Press Q to quit",
		downloadManager: dm,
		ready:           false,

		// Create each sub-model
		queuesModel:        queues.NewModel(),
		guideModel:         guide.NewModel(),
		newDownloadModel:   newDownloads.NewModel(dm),
		newQueueModel:      newQueue.NewModel(dm),
		downloadsListModel: downloads.NewModel(),
	}
}

// Init implements tea.Model. We can start in alt screen mode, etc.
func (m AppModel) Init() tea.Cmd {
	config.LoadEnv()
	logs.Log("Welcome to Download Manager")


	// Load existing queues from the JSON file
	filename := "queues.json"
	loadedQueues, err := controller.LoadQueueControllers(filename)

	if err != nil {
		logs.Log(fmt.Sprintf("Error loading queues: %v", err))
	} else if len(loadedQueues) > 0 {
		logs.Log(fmt.Sprintf("Loaded %d queues from %s", len(loadedQueues), filename))

		// Add loaded queues to download manager
		for _, queue := range loadedQueues {
			m.downloadManager.AddQueue(queue)
		}
	} else {
		logs.Log("No existing queues found, creating a default one")

		// Create example queue with test downloads when no queues are found
		url1, err1 := url.Parse("https://upload.wikimedia.org/wikipedia/commons/3/31/Napoleon_I_of_France_by_Andrea_Appiani.jpg")
		url2, err2 := url.Parse("https://upload.wikimedia.org/wikipedia/commons/thumb/3/31/David_-_Napoleon_crossing_the_Alps_-_Malmaison1.jpg/640px-David_-_Napoleon_crossing_the_Alps_-_Malmaison1.jpg")

		if err1 == nil && err2 == nil {
			// Create download controllers for test URLs
			dc1 := m.downloadManager.NewDownloadController(url1)
			dc2 := m.downloadManager.NewDownloadController(url2)

			// Set up an example queue controller
			savePath := util.GiveDefaultSavePath()
			queueCtrl := controller.NewQueueController("Example Queue")
			queueCtrl.UpdateQueueController(
				savePath,
				2,        // concurrent download limit
				100*1024, // speed limit (100KB/s)
				time.Now(),
				time.Now().Add(time.Hour*24),
			)

			// Add queue to download manager and add downloads to the queue
			m.downloadManager.AddQueue(queueCtrl)
			queueCtrl.AddDownload(dc1)
			queueCtrl.AddDownload(dc2)

			// Save the example queue to disk
			if err := controller.SaveQueueControllers(filename, m.downloadManager.QueueList); err != nil {
				logs.Log(fmt.Sprintf("Error saving initial queues: %v", err))
			}
		} else {
			logs.Log(fmt.Sprintf("Error creating example URLs: %v, %v", err1, err2))
		}
	}

	// Update all models with current queue state
	logs.Log(fmt.Sprintf("Initializing models with %d queues", len(m.downloadManager.QueueList)))
	m.updateModels()

	return tea.Batch(
		tick(),             // Start the ticker
		tea.EnterAltScreen, // Enter alternative screen mode
		tea.ClearScreen,    // Clear the screen immediately
		func() tea.Msg {
			// Force an immediate window size update

			return tea.WindowSizeMsg{Width: m.width, Height: m.height}
		},
	)
}

// Update all models with current queue information
func (m *AppModel) updateModels() {
	// Log current queue count for debugging
	queueCount := len(m.downloadManager.QueueList)
	logs.Log(fmt.Sprintf("Updating models with %d queues", queueCount))

	// Update all models that need the queue list
	m.queuesModel.UpdateQueues(m.downloadManager.QueueList)
	m.newDownloadModel.UpdateQueues(m.downloadManager.QueueList)
	m.downloadsListModel.UpdateDownloads(m.downloadManager.QueueList)
}

// Update implements tea.Model and handles incoming messages.
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if !m.ready {
			// First time initialization
			m.ready = true
		}
		// Let each sub-model know about the size change
		m.queuesModel.SetSize(m.width, m.height)
		m.guideModel.SetSize(m.width, m.height)
		m.newQueueModel.SetSize(m.width, m.height)
		m.newDownloadModel.SetSize(m.width, m.height)
		m.downloadsListModel.SetSize(m.width, m.height)

	case tickMsg:
		// Update all models with current queue state every tick
		m.updateModels()

		// Schedule next tick
		cmds = append(cmds, tick())

	case tea.KeyMsg:
		// Handle global key events here
		switch msg.String() {
		case "tab":
			// Cycle through tabs
			previousTab := m.activeTab
			m.activeTab = (m.activeTab + 1) % len(m.tabs)
			logs.Log(fmt.Sprintf("Switched to tab: %s", m.tabs[m.activeTab]))

			// Make sure models are updated when switching tabs
			m.updateModels()

			// Force a full refresh to clear the screen on tab change
			if previousTab != m.activeTab {
				cmds = append(cmds, tea.ClearScreen)
			}

			return m, tea.Batch(cmds...)
		case "ctrl+c":
			// Save queues before quitting
			if err := controller.SaveQueueControllers("queues.json", m.downloadManager.QueueList); err != nil {
				logs.Log(fmt.Sprintf("Error saving queues: %v", err))
			}
			return m, tea.Sequence(
				tea.ExitAltScreen,
				tea.Quit,
			)
		case "esc":
			// Toggle focus on the active tab
			m.toggleFocusOnActive()
			// Don't return early here, we want the active tab to also know about the esc press
		}

		// Only forward key events to the active tab
		var cmd tea.Cmd
		switch m.activeTab {
		case 0:
			m.newDownloadModel, cmd = m.newDownloadModel.Update(msg)
			cmds = append(cmds, cmd)
		case 1:
			m.newQueueModel, cmd = m.newQueueModel.Update(msg)
			cmds = append(cmds, cmd)
		case 2:
			// For the queues tab, update model data immediately after key press
			// to reflect changes in UI state
			m.queuesModel, cmd = m.queuesModel.Update(msg)
			cmds = append(cmds, cmd)

			// For specific operation keys, ensure we immediately update the queue data
			if msg.String() == "f1" || msg.String() == "f2" || msg.String() == "f3" || msg.String() == "f4" {
				// Ensure the queue data is updated right away
				m.updateModels()
				// Force refresh the screen
				cmds = append(cmds, tea.ClearScreen)
			}
		case 3:
			// For the downloads list tab (view only)
			m.downloadsListModel, cmd = m.downloadsListModel.Update(msg)
			cmds = append(cmds, cmd)

			// Removed special handling for operation keys since downloads tab is now view-only
		case 4:
			m.guideModel, cmd = m.guideModel.Update(msg)
			cmds = append(cmds, cmd)
		}

		return m, tea.Batch(cmds...)
	}

	// For non-key events, update all models to handle things like window resize and ticks
	var cmd tea.Cmd

	// Update all sub-models, but only add their commands for non-key events
	m.queuesModel, cmd = m.queuesModel.Update(msg)
	cmds = append(cmds, cmd)

	m.guideModel, cmd = m.guideModel.Update(msg)
	cmds = append(cmds, cmd)

	m.newDownloadModel, cmd = m.newDownloadModel.Update(msg)
	cmds = append(cmds, cmd)

	m.newQueueModel, cmd = m.newQueueModel.Update(msg)
	cmds = append(cmds, cmd)

	m.downloadsListModel, cmd = m.downloadsListModel.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// toggleFocusOnActive toggles the focus on the currently active tab's table, if it has one.
func (m *AppModel) toggleFocusOnActive() {
	switch m.activeTab {
	case 0:
		m.newDownloadModel.ToggleFocus()
	case 1:
		m.newQueueModel.ToggleFocus()
	case 2:
		m.queuesModel.ToggleFocus()
	case 3:
		m.downloadsListModel.ToggleFocus()
	case 4:
		m.guideModel.ToggleFocus()
	}
}

// View implements tea.Model and returns a string to display.
func (m AppModel) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	// 1. Render tab bar
	tabBar := m.renderTabBar()

	// 2. Render the active sub-model
	var content string
	switch m.activeTab {
	case 0:
		content = m.newDownloadModel.View()
	case 1:
		content = m.newQueueModel.View()
	case 2:
		content = m.queuesModel.View()
	case 3:
		content = m.downloadsListModel.View()
	case 4:
		content = m.guideModel.View()
	}

	// Calculate available content area dimensions
	contentWidth := m.width - 4
	contentHeight := m.height - 10

	// Create a fixed-size container for all tab content
	contentContainerStyle := lipgloss.NewStyle().
		Width(contentWidth).
		Height(contentHeight).
		Padding(0).
		AlignVertical(lipgloss.Top).
		BorderStyle(lipgloss.HiddenBorder())

	// Ensure content has consistent size by wrapping in the container
	content = contentContainerStyle.Render(content)

	// 3. Render the footer with tab-specific text
	footerText := m.getFooterText()
	footer := FooterStyle.Render(footerText)

	// Return the complete view
	return BaseStyle.Render(
		fmt.Sprintf("%s\n\n%s\n\n%s", tabBar, content, footer),
	)
}

// Get tab-specific footer text
func (m AppModel) getFooterText() string {
	switch m.activeTab {
	case 0: // NewDownload tab
		return "Tab: switch tabs | ESC: toggle focus | F5: switch input | Enter: add download | Q: quit"
	case 1: // NewQueue tab
		return "Tab: switch tabs | ESC: toggle focus | Enter: submit | Q: quit"
	case 2: // Queues tab
		return "Tab: switch tabs | ESC: toggle focus | F1: start all | F2: pause all | F3: resume all | F4: cancel all | J: next queue | Q: quit"
	case 3: // Downloads tab
		return "Tab: switch tabs | ESC: toggle focus | Q: quit"
	case 4: // Guide tab
		return "Tab: switch tabs | ↑/↓: scroll | Q: quit"
	default:
		return m.footerText
	}
}

// renderTabBar returns a string with the tab names styled according to which one is active.
func (m AppModel) renderTabBar() string {
	var result string
	for i, tab := range m.tabs {
		if i == m.activeTab {
			result += ActiveTabStyle.Render(tab) + " "
		} else {
			result += InactiveTabStyle.Render(tab) + " "
		}
	}
	return result
}
