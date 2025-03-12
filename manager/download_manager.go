package manager

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/mjghr/tech-download-manager/client"
	"github.com/mjghr/tech-download-manager/controller"
	"github.com/mjghr/tech-download-manager/models"
	"github.com/mjghr/tech-download-manager/util"
)

type DownloadManager struct{}

func (d *DownloadManager) DownloadQueue(queue *models.Queue) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, queue.ConcurrentDownloadLimit) // Limit concurrent downloads

	log.Printf("Starting download queue: %s with %d concurrent downloads limit", queue.ID, queue.ConcurrentDownloadLimit)

	for i := range queue.DownloadControllers {
		wg.Add(1)
		sem <- struct{}{}

		go func(dc *controller.DownloadController) {
			defer wg.Done()
			defer func() { <-sem }()

			startTime := time.Now()
			d.StartDownload(dc)
			endTime := time.Now()
			log.Printf("Download completed for: %s | Duration: %v", dc.FileName, endTime.Sub(startTime))
		}(&queue.DownloadControllers[i])
	}
	wg.Wait()

	log.Println("All downloads in queue completed")
}

func (d *DownloadManager) StartDownload(downloadController *controller.DownloadController) {
	httpRequestSender := client.NewHTTPClient()
	reqMethod := "HEAD"
	url := downloadController.Url
	headers := map[string]string{
		"user_agent": "tech-idm",
	}
	response, err := httpRequestSender.SendRequest(reqMethod, url, headers)
	if err != nil {
		log.Fatal(err)
	}

	// acceptRangers := response.Header.Get("Accept-Ranges")
	// log.Println(acceptRangers)

	contentLength := response.Header.Get("Content-Length")
	// log.Println("Content-Length:", contentLength)
	contentLengthInBytes, err := strconv.Atoi(contentLength)
	if err != nil {
		log.Fatal("empty, can't download the file", err)
	}
	// log.Println("Content-Length:", contentLengthInBytes)
	workers, chunkSize := util.CalculateOptimalWorkersAndChunkSize(contentLengthInBytes)
	// log.Println("Optimal Workers:", workers)
	// log.Println("Optimal Chunk Size:", chunkSize)
	fileName, err := util.ExtractFileName(url)
	if err != nil {
		log.Fatal("Error while extracting file name...")
	}

	// log.Println("Filename extracted: ", fileName)
	downloadController.ChunkSize = chunkSize
	downloadController.TotalSize = contentLengthInBytes
	downloadController.HttpClient = httpRequestSender
	downloadController.SpeedLimit = 1024 * 1000
	downloadController.Chunks = workers
	downloadController.FileName = fileName

	byteRangeArray := make([][2]int, workers)
	byteRangeArray = downloadController.SplitIntoChunks()
	// fmt.Println(byteRangeArray)

	var tmpPath, downPath string
	tmpPath = util.GiveDefaultTempPath()
	downPath = util.GiveDefaultSavePath()

	// Create directories if they don't exist
	if err := os.MkdirAll(tmpPath, 0755); err != nil {
		log.Fatal("Failed to create tmp directory:", err)
	}
	if err := os.MkdirAll(downPath, 0755); err != nil {
		log.Fatal("Failed to create download directory:", err)
	}

	fmt.Println("started downloading", fileName)
	var wg sync.WaitGroup
	for idx, byteChunk := range byteRangeArray {
		wg.Add(1)
		go func(idx int, byteChunk [2]int) {
			defer wg.Done()
			err := downloadController.Download(idx, byteChunk, tmpPath)
			if err != nil {
				log.Fatal(fmt.Sprintf("Failed to download chunk %v", idx), err)
			}
		}(idx, byteChunk)
	}
	wg.Wait()
	fmt.Println("done downloading now merging and removing tmp files", fileName)

	err = downloadController.MergeDownloads(tmpPath, downPath)
	if err != nil {
		log.Fatal("Failed merging tmp downloaded files...", err)
	}

	err = downloadController.CleanupTmpFiles(tmpPath)
	if err != nil {
		log.Fatal("Failed cleaning up tmp downloaded files...", err)
	}

	log.Printf("file generated: %v\n\n", downloadController.FileName)

}

func (d *DownloadManager) NewDownloadController(urlPtr *url.URL) *controller.DownloadController {
	downloadController := &controller.DownloadController{
		Url:        urlPtr.String(),
		Status:     controller.NOT_STARTED,
		PauseMutex: sync.Mutex{},
		ResumeChan: make(chan bool),
		PauseChan:  make(chan bool),
	}
	return downloadController
}
