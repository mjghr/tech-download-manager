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
	"github.com/mjghr/tech-download-manager/util"
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
	url1, err1 := url.Parse("https://upload.wikimedia.org/wikipedia/commons/3/31/Napoleon_I_of_France_by_Andrea_Appiani.jpg")
	url2, err2 := url.Parse("https://upload.wikimedia.org/wikipedia/commons/thumb/3/31/David_-_Napoleon_crossing_the_Alps_-_Malmaison1.jpg/640px-David_-_Napoleon_crossing_the_Alps_-_Malmaison1.jpg")

	if err1 != nil || err2 != nil {
		fmt.Println("Invalid URL:", err1, err2)
		return
	}

	// Download manager to create download controllers
	dm := &manager.DownloadManager{}

	// Create download controllers
	dc1 := dm.NewDownloadController(url1)
	dc2 := dm.NewDownloadController(url2)

	// Get default paths
	tempPath := util.GiveDefaultTempPath()
	savePath := util.GiveDefaultSavePath()

	// Create queue controller
	queueID := fmt.Sprintf("queue-%d", time.Now().UnixNano())
	queueCtrl := controller.NewQueueController(
		queueID,
		tempPath,
		savePath,
		2,        // Concurrent download limit
		100*1024, // Speed limit (100KB/s)
	)

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

	// Example: Pause all downloads after 2 seconds
	time.Sleep(2 * time.Second)
	fmt.Println("\nPausing all downloads...")
	queueCtrl.PauseAll()

	// Wait 3 seconds and resume
	time.Sleep(3 * time.Second)
	fmt.Println("\nResuming all downloads...")
	queueCtrl.ResumeAll()

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
			case controller.COMPLETED:
				statusStr = "Completed"
			case controller.ONGOING:
				statusStr = "Downloading"
			}
			fmt.Printf("File: %s\nStatus: %s\n---------------\n", dc.FileName, statusStr)
		}
	}
}