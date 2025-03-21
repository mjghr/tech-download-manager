package main

import (
	"fmt"
	"log"
	"net/url"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mjghr/tech-download-manager/config"
	"github.com/mjghr/tech-download-manager/controller"
	"github.com/mjghr/tech-download-manager/manager"
	"github.com/mjghr/tech-download-manager/ui"
	"github.com/mjghr/tech-download-manager/ui/logs"
	"github.com/mjghr/tech-download-manager/util"
)

type model struct {
	DownloadManager *manager.DownloadManager // Fixed typo in field name (optional)
}

// Init uses a pointer receiver to modify the original model.
func (m *model) Init() tea.Cmd {
	config.LoadEnv()
	logs.Log(fmt.Sprintf((config.WELCOME_MESSAGE)))

	url1, err1 := url.Parse("https://upload.wikimedia.org/wikipedia/commons/3/31/Napoleon_I_of_France_by_Andrea_Appiani.jpg")
	url2, err2 := url.Parse("https://upload.wikimedia.org/wikipedia/commons/thumb/3/31/David_-_Napoleon_crossing_the_Alps_-_Malmaison1.jpg/640px-David_-_Napoleon_crossing_the_Alps_-_Malmaison1.jpg")

	if err1 != nil || err2 != nil {
		logs.Log(fmt.Sprintf((fmt.Sprint("Invalid URL:", err1, err2))))
		return nil
	}

	dm := &manager.DownloadManager{}
	dc1 := dm.NewDownloadController(url1)
	dc2 := dm.NewDownloadController(url2)

	tempPath := util.GiveDefaultTempPath()
	savePath := util.GiveDefaultSavePath()

	queueID := fmt.Sprintf("queue-%d", time.Now().UnixNano())
	queueCtrl := controller.NewQueueController(
		queueID,
		tempPath,
		savePath,
		2,
		100*1024,
	)
	dm.AddQueue(queueCtrl)
	queueCtrl.AddDownload(dc1)
	queueCtrl.AddDownload(dc2)

	m.DownloadManager = dm // Assign to the original model's field
	return nil
}

// Update uses a pointer receiver.
func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	}
	return m, nil
}

// View uses a pointer receiver.
func (m *model) View() string {
	if m.DownloadManager == nil { // Safety check
		return "Initializing..."
	}
	str := ""
	for _, queueController := range m.DownloadManager.QueueList {
		for _, downloadController := range queueController.DownloadControllers {
			str += fmt.Sprintf("downloadController with id : %v has Status : %v \n", downloadController.ID, downloadController.Status)
		}
	}
	return str
}

func main() {
	p := tea.NewProgram(ui.NewAppModel(), tea.WithAltScreen())
	logs.Log(fmt.Sprintf(("Starting download manager...")))
	if _, err := p.Run(); err != nil {
		log.Fatal("Error running program:", err)
	}

}
