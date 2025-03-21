package manager

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/mjghr/tech-download-manager/client"
	"github.com/mjghr/tech-download-manager/config"
	"github.com/mjghr/tech-download-manager/controller"
	"github.com/mjghr/tech-download-manager/ui/logs"
	"github.com/mjghr/tech-download-manager/util"
)

type DownloadManager struct {
	QueueList []*controller.QueueController
}

func (d *DownloadManager) AddQueue(queue *controller.QueueController) {
	d.QueueList = append(d.QueueList, queue)
}

func (d *DownloadManager) SaveQueues() {
	controller.SaveQueueControllers(config.JSON_ADDRESS, d.QueueList)
}

func (d *DownloadManager) NewDownloadController(urlPtr *url.URL) *controller.DownloadController {
	logs.Log(fmt.Sprintf("Creating new download controller for URL: %s", urlPtr.String()))

	// Initialize HTTP client early to use for HEAD request
	httpClient := client.NewHTTPClient()

	// Get file details with HEAD request
	resp, err := httpClient.SendRequest("HEAD", urlPtr.String(), map[string]string{
		"User-Agent": "tech-idm",
	})
	if err != nil {
		logs.Log(fmt.Sprintf("Warning: Failed to get file size: %v", err))
		return &controller.DownloadController{
			Status: controller.FAILED,
			Url:    urlPtr.String(),
			ID:     fmt.Sprintf("dc-%d", time.Now().UnixNano()),
		}
	}
	defer resp.Body.Close()

	// Parse Content-Length
	contentLength := resp.Header.Get("Content-Length")
	totalSize, err := strconv.Atoi(contentLength)
	if err != nil || totalSize <= 0 {
		logs.Log(fmt.Sprintf("Warning: Invalid Content-Length '%s': %v", contentLength, err))
		return &controller.DownloadController{
			Status: controller.FAILED,
			Url:    urlPtr.String(),
			ID:     fmt.Sprintf("dc-%d", time.Now().UnixNano()),
		}
	}

	// Get speed limit from environment
	speedLimitStr := os.Getenv("SPEED_LIMIT_KB")
	speedLimit, err := strconv.Atoi(speedLimitStr)
	if err != nil {
		logs.Log(fmt.Sprintf("Invalid SPEED_LIMIT_KB value '%s', defaulting to 0: %v", speedLimitStr, err))
		speedLimit = 0
	} else {
		speedLimit = speedLimit * 1024 // Convert KB/s to bytes/s
	}

	// Extract filename from URL
	fileName, err := util.ExtractFileName(urlPtr.String())
	if err != nil {
		logs.Log(fmt.Sprintf("Warning: Failed to extract filename: %v", err))
		fileName = fmt.Sprintf("download-%d", time.Now().UnixNano())
	}

	downloadController := &controller.DownloadController{
		ID:         fmt.Sprintf("dc-%d", time.Now().UnixNano()),
		Url:        urlPtr.String(),
		Status:     controller.NOT_STARTED,
		FileName:   fileName,
		TotalSize:  totalSize,
		HttpClient: httpClient,
		SpeedLimit: speedLimit,
		Mutex:      sync.Mutex{},
		ResumeChan: make(chan bool),
		PauseChan:  make(chan bool),
	}

	// Calculate optimal chunks
	workers, chunkSize := util.CalculateOptimalWorkersAndChunkSize(totalSize)
	downloadController.Chunks = downloadController.SplitIntoChunks(workers, chunkSize)
	downloadController.CompletedBytes = make([]int, len(downloadController.Chunks))

	logs.Log(fmt.Sprintf("Created download controller %s for file %s: size=%d bytes, chunks=%d, speed_limit=%d bytes/s",
		downloadController.ID,
		downloadController.FileName,
		downloadController.TotalSize,
		len(downloadController.Chunks),
		downloadController.SpeedLimit))

	return downloadController
}
