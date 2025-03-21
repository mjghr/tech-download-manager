package controller

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/mjghr/tech-download-manager/ui/logs"
	"github.com/mjghr/tech-download-manager/util"
)

// QueueController manages a download queue with features like pause, resume, and concurrent download limits
type QueueController struct {
	QueueID                 string                `json:"queueId"`
	SpeedLimit              int                   `json:"speedLimit"`
	ConcurrentDownloadLimit int                   `json:"concurrentDownloadLimit"`
	StartTime               time.Time             `json:"startTime"`
	EndTime                 time.Time             `json:"endTime"`
	DownloadControllers     []*DownloadController `json:"downloadControllers"`
	TempPath                string                `json:"tempPath"`
	SavePath                string                `json:"savePath"`
	QueueName               string                `json:"name"`

	mutex sync.Mutex     `json:"-"`
	wg    sync.WaitGroup `json:"-"`
}

func (qc *QueueController) UpdateQueueController(savePath string, concurrentDownloadLimit, speedLimit int, startTime, endTime time.Time) {
	if savePath != "" {
		qc.SavePath = savePath
	}
	if concurrentDownloadLimit != 0 {
		qc.ConcurrentDownloadLimit = concurrentDownloadLimit
	}
	if speedLimit != 0 {
		qc.SpeedLimit = speedLimit
	}
	if !startTime.IsZero() {
		qc.StartTime = startTime
	}
	if !endTime.IsZero() {
		qc.EndTime = endTime
	}
}

func NewQueueController(name string) *QueueController {
	return &QueueController{
		QueueID:                 fmt.Sprintf("queue-%d", time.Now().UnixNano()),
		QueueName:               name,
		ConcurrentDownloadLimit: 1,
		SpeedLimit:              100 * 1024,
		TempPath:                util.GiveDefaultTempPath(),
		SavePath:                util.GiveDefaultSavePath(),
		DownloadControllers:     make([]*DownloadController, 0),
	}
}

// Start begins processing the download queue
func (qc *QueueController) Start() error {
	logs.Log(fmt.Sprintf("Starting queue %s processing", qc.QueueID))

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
	logs.Log(fmt.Sprintf("Queue %s processing completed", qc.QueueID))
	return nil
}

func (qc *QueueController) processDownload(dc *DownloadController) {
	defer qc.wg.Done()

	// // Skip if already completed or failed
	if dc.GetStatus() == COMPLETED {
		logs.Log(fmt.Sprintf("Download %s skipped: already %v", dc.ID, dc.GetStatus()))
		return
	}

	// Wait for a slot to become available
	qc.waitForDownloadSlot(dc)

	// Check if start time is in the future
	now := time.Now()
	if !qc.StartTime.IsZero() && now.Before(qc.StartTime) {
		waitDuration := qc.StartTime.Sub(now)
		logs.Log(fmt.Sprintf("Waiting %v for scheduled start time for download %s", waitDuration, dc.ID))
		time.Sleep(waitDuration)
	}

	// Check if we're already past end time
	if !qc.EndTime.IsZero() && now.After(qc.EndTime) {
		logs.Log(fmt.Sprintf("Download %s skipped as current time is past the end time", dc.ID))
		return
	}

	// Set speed limit from queue if not set individually
	if dc.SpeedLimit == 0 {
		dc.SpeedLimit = qc.SpeedLimit
	}

	// Mark this download as in progress
	dc.SetStatus(ONGOING)
	logs.Log(fmt.Sprintf("Starting download %s in queue %s", dc.ID, qc.QueueID))

	// Split file into chunks
	chunks := dc.Chunks

	ctx, cancel := context.WithCancel(context.Background())
	dc.CancelFuncs = append(dc.CancelFuncs, cancel)
	dc.ctx = ctx

	// Download each chunk
	var downloadErr error
	var chunkWg sync.WaitGroup

	for i, chunk := range chunks {
		chunkWg.Add(1)
		go func(idx int, byteChunk [2]int) {
			defer chunkWg.Done()
			if dc.GetStatus() != ONGOING { // Check status before proceeding
				logs.Log(fmt.Sprintf("Chunk %d for %s skipped: download not ONGOING", idx, dc.ID))
				return
			}
			err := dc.Download(idx, byteChunk, qc.TempPath, ctx)
			if err != nil {
				logs.Log(fmt.Sprintf("Error downloading chunk %d for %s: %v", idx, dc.FileName, err))
				if errors.Is(downloadErr, context.Canceled) {
					dc.SetStatus(CANCELED)
				} else {
					dc.SetStatus(CANCELED)
				}
				downloadErr = err
			}
		}(i, chunk)
	}

	// Wait for all chunks to complete
	chunkWg.Wait()

	if downloadErr != nil {
		if errors.Is(downloadErr, context.Canceled) {
			logs.Log(fmt.Sprintf("Download %s canceled: %v", dc.ID, downloadErr))
			dc.SetStatus(CANCELED)
		} else {
			logs.Log(fmt.Sprintf("Download %s failed: %v", dc.ID, downloadErr))
			dc.SetStatus(FAILED)
		}

		// Clean up temporary files on failure or cancellation
		if err := dc.CleanupTmpFiles(qc.TempPath); err != nil {
			logs.Log(fmt.Sprintf("Warning: failed to clean up temp files for %s: %v", dc.ID, err))
		}
		return
	}

	// Check if we're past the end time
	if !qc.EndTime.IsZero() && time.Now().After(qc.EndTime) {
		logs.Log(fmt.Sprintf("Download %s completed chunks but current time is past the end time, not merging", dc.ID))
		dc.Status = FAILED
		return
	}

	// Merge chunks and cleanup
	err := dc.MergeDownloads(qc.TempPath, qc.SavePath)
	if err != nil {
		logs.Log(fmt.Sprintf("Failed to merge chunks for %s: %v", dc.ID, err))
		dc.Status = FAILED
		return
	}

	err = dc.CleanupTmpFiles(qc.TempPath)
	if err != nil {
		logs.Log(fmt.Sprintf("Warning: failed to clean up temp files for %s: %v", dc.ID, err))
		// Still consider the download complete even if cleanup fails
	}

	dc.Status = COMPLETED
	logs.Log(fmt.Sprintf("Download %s completed successfully", dc.ID))
}

// waitForDownloadSlot waits until a download slot is available
func (qc *QueueController) waitForDownloadSlot(dc *DownloadController) {
	for {
		qc.mutex.Lock()
		activeCount := 0
		for _, download := range qc.DownloadControllers {
			if download.Status == ONGOING {
				activeCount++
			}
		}
		if activeCount < qc.ConcurrentDownloadLimit {
			qc.mutex.Unlock()
			return
		}
		qc.mutex.Unlock()

		// Wait a bit before checking again
		time.Sleep(500 * time.Millisecond)

		// Check if we're already past end time
		if !qc.EndTime.IsZero() && time.Now().After(qc.EndTime) {
			logs.Log(fmt.Sprintf("Download %s skipped while waiting for slot as current time is past the end time", dc.ID))
			return
		}
	}
}

// PauseAll pauses all active downloads in the queue
func (qc *QueueController) PauseAll() {
	logs.Log(fmt.Sprintf("Pausing all downloads in queue %s", qc.QueueID))
	for _, dc := range qc.DownloadControllers {
		dc.Pause()
	}
}

// ResumeAll resumes all paused downloads in the queue
func (qc *QueueController) ResumeAll() {
	logs.Log(fmt.Sprintf("Resuming all downloads in queue %s", qc.QueueID))
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
	logs.Log(fmt.Sprintf("Added download %s to queue %s", dc.ID, qc.QueueID))
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
			logs.Log(fmt.Sprintf("Removed download %s from queue %s", downloadID, qc.QueueID))
			return nil
		}
	}

	return fmt.Errorf("download %s not found in queue", downloadID)
}

// SetConcurrentLimit updates the concurrent download limit
func (qc *QueueController) SetConcurrentLimit(limit int) {
	qc.mutex.Lock()
	defer qc.mutex.Unlock()

	qc.ConcurrentDownloadLimit = limit
	logs.Log(fmt.Sprintf("Updated concurrent download limit to %d for queue %s", limit, qc.QueueID))
}

// SetTimeWindow sets the time window for downloads
func (qc *QueueController) SetTimeWindow(startTime, endTime time.Time) {
	qc.mutex.Lock()
	defer qc.mutex.Unlock()

	qc.StartTime = startTime
	qc.EndTime = endTime
	logs.Log(fmt.Sprintf("Updated time window for queue %s: start=%v, end=%v",
		qc.QueueID, startTime.Format(time.RFC3339), endTime.Format(time.RFC3339)))
}

func (qc *QueueController) SetPaths(tempPath, savePath string) error {
	qc.mutex.Lock()
	defer qc.mutex.Unlock()

	for _, dc := range qc.DownloadControllers {
		if dc.GetStatus() == ONGOING {
			return fmt.Errorf("cannot change paths while downloads are ongoing")
		}
	}

	if err := os.MkdirAll(tempPath, 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	if err := os.MkdirAll(savePath, 0755); err != nil {
		return fmt.Errorf("failed to create save directory: %w", err)
	}

	qc.TempPath = tempPath
	qc.SavePath = savePath
	logs.Log(fmt.Sprintf("Updated paths for queue %s: temp=%s, save=%s", qc.QueueID, tempPath, savePath))
	return nil
}

func (qc *QueueController) CancelDownload(downloadID string) error {
	qc.mutex.Lock()
	defer qc.mutex.Unlock()

	for _, dc := range qc.DownloadControllers {
		if dc.ID == downloadID {
			dc.Cancel(qc.TempPath)
			return nil
		}
	}

	return fmt.Errorf("download %s not found in queue", downloadID)
}

func (qc *QueueController) CancelAll() error {
	qc.mutex.Lock()
	defer qc.mutex.Unlock()

	for _, dc := range qc.DownloadControllers {
		dc.Cancel(qc.TempPath)
	}

	return fmt.Errorf("failed to cancel al downloads")
}
