package main

import (
	"fmt"
	"log"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mjghr/tech-download-manager/ui"
	"github.com/mjghr/tech-download-manager/ui/logs"
)


func main() {
	p := tea.NewProgram(ui.NewAppModel(), tea.WithAltScreen())
	logs.Log(fmt.Sprintf(("Starting download manager...")))
	if _, err := p.Run(); err != nil {
		log.Fatal("Error running program:", err)
	}

}
