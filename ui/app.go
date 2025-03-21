package ui

import (
	"fmt"
	"net/url"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mjghr/tech-download-manager/config"
	"github.com/mjghr/tech-download-manager/controller"
	"github.com/mjghr/tech-download-manager/manager"
	"github.com/mjghr/tech-download-manager/ui/guide"
	"github.com/mjghr/tech-download-manager/ui/logs"
	"github.com/mjghr/tech-download-manager/ui/newDownloads"
	"github.com/mjghr/tech-download-manager/ui/queues"
	"github.com/mjghr/tech-download-manager/util"
)

// Add these new types near the top of the file
type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(250*time.Millisecond, func(t time.Time) tea.Msg {
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
	ready           bool // Add this field

	// Sub-models (each tab). Each one is a Bubble Tea model with a table or any other view.
	queuesModel      queues.Model
	logsModel        logs.Model
	guideModel       guide.Model
	newDownloadModel newDownloads.NewDownloadModel
}

// NewAppModel initializes the root model with default values.
func NewAppModel() AppModel {
	dm := &manager.DownloadManager{}

	return AppModel{
		tabs:            []string{"NewDownload", "Queues", "Guide", "Logs"},
		activeTab:       0,
		footerText:      "Press Tab to switch tabs | Press ESC to toggle focus | Press Q to quit",
		downloadManager: dm,
		ready:           false, // Add this

		// Create each sub-model
		queuesModel:      queues.NewModel(),
		logsModel:        logs.NewModel(),
		guideModel:       guide.NewModel(),
		newDownloadModel: newDownloads.NewModel(dm),
	}
}

// Init implements tea.Model. We can start in alt screen mode, etc.
func (m AppModel) Init() tea.Cmd {
	config.LoadEnv()
	logs.Log("Welcome to Download Manager")

	filename := "queues.json"
	loadedQueues, err := controller.LoadQueueControllers(filename)
	if err != nil {
		logs.Log(fmt.Sprintf("Error loading queues: %v", err))
	}

	// Add loaded queues to download manager
	for _, queue := range loadedQueues {
		m.downloadManager.AddQueue(queue)
	}

	// Create test URLs
	url1, err1 := url.Parse("https://upload.wikimedia.org/wikipedia/commons/3/31/Napoleon_I_of_France_by_Andrea_Appiani.jpg")
	url2, err2 := url.Parse("https://upload.wikimedia.org/wikipedia/commons/thumb/3/31/David_-_Napoleon_crossing_the_Alps_-_Malmaison1.jpg/640px-David_-_Napoleon_crossing_the_Alps_-_Malmaison1.jpg")

	if err1 != nil || err2 != nil {
		logs.Log(fmt.Sprintf("Invalid URL: %v, %v", err1, err2))
		return m.logsModel.Init()
	}

	// Create download controllers
	dc1 := m.downloadManager.NewDownloadController(url1)
	dc2 := m.downloadManager.NewDownloadController(url2)

	// Set up queue controller
	tempPath := util.GiveDefaultTempPath()
	savePath := util.GiveDefaultSavePath()
	queueID := fmt.Sprintf("queue-%d", time.Now().UnixNano())

	queueCtrl := controller.NewQueueController(
		queueID,
		tempPath,
		savePath,
		2,        // concurrent download limit
		100*1024, // speed limit (100KB/s)
	)

	// Add queue and downloads
	m.downloadManager.AddQueue(queueCtrl)
	queueCtrl.AddDownload(dc1)
	queueCtrl.AddDownload(dc2)

	// After adding all queues, update the queues model
	logs.Log(fmt.Sprintf("Updating queues model with %d queues", len(m.downloadManager.QueueList)))
	m.queuesModel.UpdateQueues(m.downloadManager.QueueList)

	return tea.Batch(
		m.logsModel.Init(),
		tick(),             // Add the ticker
		tea.EnterAltScreen, // Add this
		func() tea.Msg {
			// Force an immediate update of the queues view
			return tea.WindowSizeMsg{Width: m.width, Height: m.height}
		},
	)
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
		m.logsModel.SetSize(m.width, m.height)
		m.guideModel.SetSize(m.width, m.height)

	case tickMsg:
		// Update both models with current queues
		m.queuesModel.UpdateQueues(m.downloadManager.QueueList)
		m.newDownloadModel.UpdateQueues(m.downloadManager.QueueList)
		// Schedule next tick
		cmds = append(cmds, tick())

	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			// For demonstration, call logs.Log(fmt.Sprintf( here:
			// logs.Log(fmt.Sprintf("queue ID :", m.downloadManager.QueueList[0].QueueID))
			m.activeTab = (m.activeTab + 1) % len(m.tabs)
		case "esc":
			logs.Log(fmt.Sprintf(("esc pressed")))
			// Toggle focus on the active tab’s table (if using a table)
			m.toggleFocusOnActive()
		case "ctrl+c":
			return m, tea.Sequence(
				tea.ExitAltScreen,
				tea.Quit,
			)
		}
	}

	// Update all sub-models regardless of active tab so that background subscriptions run.
	var cmd tea.Cmd

	m.queuesModel, cmd = m.queuesModel.Update(msg)
	cmds = append(cmds, cmd)

	m.logsModel, cmd = m.logsModel.Update(msg)
	cmds = append(cmds, cmd)

	m.guideModel, cmd = m.guideModel.Update(msg)
	cmds = append(cmds, cmd)

	m.newDownloadModel, cmd = m.newDownloadModel.Update(msg)
	cmds = append(cmds, cmd)

	// Now, only the active sub-model’s view will be rendered.
	return m, tea.Batch(cmds...)
}

// toggleFocusOnActive toggles the focus on the currently active tab’s table, if it has one.
func (m *AppModel) toggleFocusOnActive() {
	switch m.activeTab {
	case 0:
		m.newDownloadModel.ToggleFocus()
	case 1:
		m.queuesModel.ToggleFocus()
	case 2:
		m.guideModel.ToggleFocus()
	case 3:
		m.logsModel.ToggleFocus()
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
		content = m.queuesModel.View()
	case 2:
		content = m.guideModel.View()
	case 3:
		content = m.logsModel.View()
	}

	// 3. Render the footer
	footer := FooterStyle.Render(m.footerText)

	return BaseStyle.Render(
		fmt.Sprintf("%s\n\n%s\n\n%s", tabBar, content, footer),
	)
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
