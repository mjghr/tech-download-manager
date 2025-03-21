package main

import (
	"fmt"
	"log"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mjghr/tech-download-manager/config"
	"github.com/mjghr/tech-download-manager/controller"
	"github.com/mjghr/tech-download-manager/ui"
	"github.com/mjghr/tech-download-manager/ui/logs"
)


func main() {
	// Load environment variables
	config.LoadEnv()

	// Log loaded queues for debugging
	filename := "queues.json"
	loadedQueues, err := controller.LoadQueueControllers(filename)
	if err != nil {
		logs.Log(fmt.Sprintf("Error loading queues: %v", err))
	} else {
		logs.Log(fmt.Sprintf("Loaded %d queues from %s", len(loadedQueues), filename))
	}

	// Create and run the app
	p := tea.NewProgram(ui.NewAppModel(), tea.WithAltScreen())
	logs.Log("Starting download manager...")

	if _, err := p.Run(); err != nil {
		log.Fatal("Error running program:", err)
	}

	// Note: Queues are saved in the app.go file when pressing q/ctrl+c
	logs.Log("Download manager closed.")
}
