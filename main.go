package main

import (
	"bufio"
	"fmt"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/mjghr/tech-download-manager/config"
	"github.com/mjghr/tech-download-manager/controller"
	"github.com/mjghr/tech-download-manager/manager"
)

func main() {
	config.LoadEnv()
	fmt.Println(config.WELCOME_MESSAGE)

	filename := "queues.json"

	loadedQueues, err := controller.LoadQueueControllers(filename)
	if err != nil {
		fmt.Println("Error loading:", err)
		return
	}

	fmt.Println(loadedQueues)

	// Parse example URLs
	url1, err1 := url.Parse("https://s1.netpaak.ir/GAME/Marvels.S.M.2-R/Marvels.Spider-Man.2-RUNE_VGdl.ir.part08.rar")
	url2, err2 := url.Parse("https://s1.netpaak.ir/GAME/Marvels.S.M.2-R/Marvels.Spider-Man.2-RUNE_VGdl.ir.part09.rar")

	if err1 != nil || err2 != nil {
		fmt.Println("Invalid URL:", err1, err2)
		return
	}

	// Download manager to create download controllers
	dm := &manager.DownloadManager{}

	// Create download controllers
	dc1 := dm.NewDownloadController(url1)
	dc2 := dm.NewDownloadController(url2)
	queueCtrl := controller.NewQueueController("abbas")

	// Add downloads to queue
	queueCtrl.AddDownload(dc1)
	queueCtrl.AddDownload(dc2)

	// Set a time window for downloads (optional)
	now := time.Now()
	oneHourLater := now.Add(1 * time.Hour)
	queueCtrl.SetTimeWindow(now, oneHourLater)

	// Start monitoring goroutine
	go monitorDownloads(queueCtrl)

	// Start the queue processing
	go func() {
		if err := queueCtrl.Start(); err != nil {
			log.Printf("Error processing queue: %v", err)
		}
	}()

	// // Example: Pause all downloads after 2 seconds
	// time.Sleep(2 * time.Second)
	// fmt.Println("\nPausing all downloads...")
	// queueCtrl.PauseAll()

	// // Wait 3 seconds and resume
	// time.Sleep(3 * time.Second)
	// fmt.Println("\nResuming all downloads...")
	// queueCtrl.ResumeAll()

	time.Sleep(10 * time.Second)
	fmt.Println("\nCanceling all downloads...")
	queueCtrl.CancelAll()

	// Wait for user input
	fmt.Println("\nPress Enter to exit.")
	reader := bufio.NewReader(os.Stdin)
	reader.ReadString('\n')
}

func monitorDownloads(qc *controller.QueueController) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		fmt.Print("\033[H\033[2J") // Clear screen
		fmt.Println("Download Status:")
		fmt.Println("---------------")

		for _, dc := range qc.DownloadControllers {
			status := dc.GetStatus()
			var statusStr string
			switch status {
			case controller.NOT_STARTED:
				statusStr = "Not Started"
			case controller.PAUSED:
				statusStr = "Paused"
			case controller.FAILED:
				statusStr = "Failed"
			case controller.CANCELED:
				statusStr = "Cancelled"
			case controller.COMPLETED:
				statusStr = "Completed"
			case controller.ONGOING:
				statusStr = "Downloading"
			}
			fmt.Printf("File: %s\nStatus: %s\n---------------\n", dc.FileName, statusStr)
		}
	}
}
