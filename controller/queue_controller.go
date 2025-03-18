package controller

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

// QueueController manages a download queue with features like pause, resume, and concurrent download limits
type QueueController struct {
	QueueID             string
	SaveDestination     string
	SpeedLimit          int
	ConcurrentLimit     int
	StartTime           time.Time
	EndTime             time.Time
	DownloadControllers []*DownloadController
	ActiveDownloads     int
	TempPath            string
	SavePath            string
	mutex               sync.Mutex
	downloadInProgress  map[string]bool
	wg                  sync.WaitGroup
}

// NewQueueController creates a new queue controller
func NewQueueController(queueID, tempPath, savePath string, concurrentLimit, speedLimit int) *QueueController {
	return &QueueController{
		QueueID:             queueID,
		ConcurrentLimit:     concurrentLimit,
		SpeedLimit:          speedLimit,
		ActiveDownloads:     0,
		TempPath:            tempPath,
		SavePath:            savePath,
		SaveDestination:     savePath,
		DownloadControllers: make([]*DownloadController, 0),
		downloadInProgress:  make(map[string]bool),
	}
}

// Start begins processing the download queue
func (qc *QueueController) Start() error {
	log.Printf("Starting queue %s processing", qc.QueueID)

	// Check if temp directory exists, create if not
	if err := os.MkdirAll(qc.TempPath, 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Check if save directory exists, create if not
	if err := os.MkdirAll(qc.SavePath, 0755); err != nil {
		return fmt.Errorf("failed to create save directory: %w", err)
	}

	for _, dc := range qc.DownloadControllers {
		qc.wg.Add(1)
		go qc.processDownload(dc)
	}

	qc.wg.Wait()
	log.Printf("Queue %s processing completed", qc.QueueID)
	return nil
}

// processDownload handles an individual download in the queue
func (qc *QueueController) processDownload(dc *DownloadController) {
	defer qc.wg.Done()

	// Wait for a slot to become available
	qc.waitForDownloadSlot(dc)

	// Check if start time is in the future
	now := time.Now()
	if !qc.StartTime.IsZero() && now.Before(qc.StartTime) {
		waitDuration := qc.StartTime.Sub(now)
		log.Printf("Waiting %v for scheduled start time for download %s", waitDuration, dc.ID)
		time.Sleep(waitDuration)
	}

	// Check if we're already past end time
	if !qc.EndTime.IsZero() && now.After(qc.EndTime) {
		log.Printf("Download %s skipped as current time is past the end time", dc.ID)
		return
	}

	// Mark this download as in progress
	qc.mutex.Lock()
	qc.downloadInProgress[dc.ID] = true
	qc.ActiveDownloads++
	qc.mutex.Unlock()

	// Ensure we decrement active downloads when done
	defer func() {
		qc.mutex.Lock()
		qc.ActiveDownloads--
		delete(qc.downloadInProgress, dc.ID)
		qc.mutex.Unlock()
	}()

	// Initialize required channels for the download controller
	dc.PauseChan = make(chan bool)
	dc.ResumeChan = make(chan bool)

	// Set speed limit from queue if not set individually
	if dc.SpeedLimit == 0 {
		dc.SpeedLimit = qc.SpeedLimit
	}

	log.Printf("Starting download %s in queue %s", dc.ID, qc.QueueID)
	dc.Status = ONGOING

	// Split file into chunks
	chunks := dc.SplitIntoChunks()

	// Download each chunk
	var downloadErr error
	var chunkWg sync.WaitGroup

	for i, chunk := range chunks {
		chunkWg.Add(1)
		go func(idx int, byteChunk [2]int) {
			defer chunkWg.Done()
			err := dc.Download(idx, byteChunk, qc.TempPath)
			if err != nil {
				log.Printf("Error downloading chunk %d for %s: %v", idx, dc.FileName, err)
				downloadErr = err
				dc.Status = FAILED
			}
		}(i, chunk)
	}

	// Wait for all chunks to complete
	chunkWg.Wait()

	// Check if download failed
	if downloadErr != nil {
		log.Printf("Download %s failed: %v", dc.ID, downloadErr)
		return
	}

	// Check if we're past the end time
	if !qc.EndTime.IsZero() && time.Now().After(qc.EndTime) {
		log.Printf("Download %s completed chunks but current time is past the end time, not merging", dc.ID)
		dc.Status = FAILED
		return
	}

	// Merge chunks and cleanup
	err := dc.MergeDownloads(qc.TempPath, qc.SavePath)
	if err != nil {
		log.Printf("Failed to merge chunks for %s: %v", dc.ID, err)
		dc.Status = FAILED
		return
	}

	err = dc.CleanupTmpFiles(qc.TempPath)
	if err != nil {
		log.Printf("Warning: failed to clean up temp files for %s: %v", dc.ID, err)
		// Still consider the download complete even if cleanup fails
	}

	dc.Status = COMPLETED
	log.Printf("Download %s completed successfully", dc.ID)
}

// waitForDownloadSlot waits until a download slot is available
func (qc *QueueController) waitForDownloadSlot(dc *DownloadController) {
	for {
		qc.mutex.Lock()
		if qc.ActiveDownloads < qc.ConcurrentLimit {
			qc.mutex.Unlock()
			return
		}
		qc.mutex.Unlock()

		// Wait a bit before checking again
		time.Sleep(500 * time.Millisecond)

		// Check if we're already past end time
		if !qc.EndTime.IsZero() && time.Now().After(qc.EndTime) {
			log.Printf("Download %s skipped while waiting for slot as current time is past the end time", dc.ID)
			return
		}
	}
}

// PauseAll pauses all active downloads in the queue
func (qc *QueueController) PauseAll() {
	log.Printf("Pausing all downloads in queue %s", qc.QueueID)
	for _, dc := range qc.DownloadControllers {
		dc.Pause()
	}
}

// ResumeAll resumes all paused downloads in the queue
func (qc *QueueController) ResumeAll() {
	log.Printf("Resuming all downloads in queue %s", qc.QueueID)
	for _, dc := range qc.DownloadControllers {
		dc.Resume()
	}
}

// PauseDownload pauses a specific download in the queue
func (qc *QueueController) PauseDownload(downloadID string) error {
	for _, dc := range qc.DownloadControllers {
		if dc.ID == downloadID {
			dc.Pause()
			return nil
		}
	}
	return fmt.Errorf("download %s not found in queue", downloadID)
}

// ResumeDownload resumes a specific download in the queue
func (qc *QueueController) ResumeDownload(downloadID string) error {
	for _, dc := range qc.DownloadControllers {
		if dc.ID == downloadID {
			dc.Resume()
			return nil
		}
	}
	return fmt.Errorf("download %s not found in queue", downloadID)
}

// AddDownload adds a new download to the queue
func (qc *QueueController) AddDownload(dc *DownloadController) {
	qc.mutex.Lock()
	defer qc.mutex.Unlock()

	qc.DownloadControllers = append(qc.DownloadControllers, dc)
	log.Printf("Added download %s to queue %s", dc.ID, qc.QueueID)
}

// RemoveDownload removes a download from the queue
func (qc *QueueController) RemoveDownload(downloadID string) error {
	qc.mutex.Lock()
	defer qc.mutex.Unlock()

	for i, dc := range qc.DownloadControllers {
		if dc.ID == downloadID {
			// Remove this element
			qc.DownloadControllers = append(
				qc.DownloadControllers[:i],
				qc.DownloadControllers[i+1:]...,
			)
			log.Printf("Removed download %s from queue %s", downloadID, qc.QueueID)
			return nil
		}
	}

	return fmt.Errorf("download %s not found in queue", downloadID)
}

// SetConcurrentLimit updates the concurrent download limit
func (qc *QueueController) SetConcurrentLimit(limit int) {
	qc.mutex.Lock()
	defer qc.mutex.Unlock()

	qc.ConcurrentLimit = limit
	log.Printf("Updated concurrent download limit to %d for queue %s", limit, qc.QueueID)
}

// SetTimeWindow sets the time window for downloads
func (qc *QueueController) SetTimeWindow(startTime, endTime time.Time) {
	qc.mutex.Lock()
	defer qc.mutex.Unlock()

	qc.StartTime = startTime
	qc.EndTime = endTime
	log.Printf("Updated time window for queue %s: start=%v, end=%v",
		qc.QueueID, startTime.Format(time.RFC3339), endTime.Format(time.RFC3339))
}

// SetPaths updates the temporary and save paths
func (qc *QueueController) SetPaths(tempPath, savePath string) error {
	// Check if the directories exist or can be created
	if err := os.MkdirAll(tempPath, 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}

	if err := os.MkdirAll(savePath, 0755); err != nil {
		return fmt.Errorf("failed to create save directory: %w", err)
	}

	qc.mutex.Lock()
	defer qc.mutex.Unlock()

	qc.TempPath = tempPath
	qc.SavePath = savePath
	qc.SaveDestination = savePath

	log.Printf("Updated paths for queue %s: temp=%s, save=%s", qc.QueueID, tempPath, savePath)
	return nil
}
