package controller

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/mjghr/tech-download-manager/client"
	"github.com/mjghr/tech-download-manager/config"
	"github.com/mjghr/tech-download-manager/ui/logs"
)

type Status int

const (
	NOT_STARTED Status = iota
	PAUSED
	FAILED
	COMPLETED
	ONGOING
	CANCELED
)

type DownloadController struct {
	ID             string             `json:"id"`
	QueueID        string             `json:"queueId"`
	Url            string             `json:"url"`
	Status         Status             `json:"status"`
	FileName       string             `json:"fileName"`
	Chunks         [][2]int           `json:"chunks"`
	CompletedBytes []int              `json:"completedBytes"`
	TotalSize      int                `json:"totalSize"`
	HttpClient     *client.HTTPClient `json:"httpClient"`
	SpeedLimit     int                `json:"speedLimit"`

	PauseChan   chan bool            `json:"-"`
	Mutex       sync.Mutex           `json:"-"`
	ResumeChan  chan bool            `json:"-"`
	TokenBucket chan struct{}        `json:"-"`
	CancelFuncs []context.CancelFunc `json:"-"`
	ctx         context.Context      `json:"-"`
}

func (d *DownloadController) SplitIntoChunks(workers, chunkSize int) [][2]int {
	logs.Log(fmt.Sprintf(("Starting to split download %s into %d chunks (total size: %d bytes)"), d.ID, d.Chunks, d.TotalSize))
	arr := make([][2]int, workers)

	if d.TotalSize <= 0 {
		logs.Log(fmt.Sprintf(("Error: Total size is %d, cannot split into chunks"), d.TotalSize))
		return arr
	}

	remainder := d.TotalSize % workers

	var start, end int
	for i := 0; i < workers; i++ {
		start = i * chunkSize
		end = start + chunkSize - 1

		// Add remainder to last chunk
		if i == workers-1 {
			end += remainder
		}

		arr[i] = [2]int{start, end}
		logs.Log(fmt.Sprintf(("Created chunk %d for %s: bytes %d-%d"), i, d.ID, start, end))
	}

	logs.Log(fmt.Sprintf(("Successfully split %s into %d chunks"), d.ID, d.Chunks))
	return arr
}

func (d *DownloadController) Cancel(tmp string) {
	d.Mutex.Lock()
	defer d.Mutex.Unlock()

	if d.Status == ONGOING || d.Status == PAUSED {
		d.Status = CANCELED
		logs.Log(fmt.Sprintf("Download %s has been canceled", d.ID))

		// Cancel all ongoing goroutines
		for _, cancelFunc := range d.CancelFuncs {
			cancelFunc()
		}

		// Clean up temporary files
		err := d.CleanupTmpFiles(tmp)
		if err != nil {
			logs.Log(fmt.Sprintf("Warning: failed to clean up temp files for %s: %v", d.ID, err))
		}

		// Notify any waiting goroutines
		close(d.PauseChan)
		close(d.ResumeChan)
	} else {
		logs.Log(fmt.Sprintf("Download %s is not ongoing or paused, no action taken", d.ID))
	}
}

func (d *DownloadController) Download(idx int, byteChunk [2]int, tmpPath string, ctx context.Context) error {
	logs.Log(fmt.Sprintf("Starting download of chunk %d for %s (bytes %d-%d, speed limit: %d bytes/s)", idx, d.FileName, byteChunk[0], byteChunk[1], d.SpeedLimit))

	// Define chunk file
	fileName := fmt.Sprintf("%s/%s-%s-%d.tmp", tmpPath, config.TMP_FILE_PREFIX, d.FileName, idx)
	logs.Log(fmt.Sprintf("Creating temporary file for chunk %d: %s", idx, fileName))

	// Check if the file already exists
	var file *os.File
	var err error
	var startOffset int

	if _, err := os.Stat(fileName); err == nil {
		// File exists, open it in append mode
		file, err = os.OpenFile(fileName, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			logs.Log(fmt.Sprintf("Failed to open file %s for chunk %d: %v", fileName, idx, err))
			return fmt.Errorf("failed to open file %s for chunk %d: %w", fileName, idx, err)
		}

		// Get the current size of the file
		fileInfo, err := file.Stat()
		if err != nil {
			logs.Log(fmt.Sprintf("Failed to get file info for %s: %v", fileName, err))
			return fmt.Errorf("failed to get file info for %s: %w", fileName, err)
		}
		startOffset = int(fileInfo.Size())
		logs.Log(fmt.Sprintf("Resuming download of chunk %d from byte %d", idx, startOffset))
	} else {
		// File does not exist, create it
		file, err = os.Create(fileName)
		if err != nil {
			logs.Log(fmt.Sprintf("Failed to create file %s for chunk %d: %v", fileName, idx, err))
			return fmt.Errorf("failed to create file %s for chunk %d: %w", fileName, idx, err)
		}
		startOffset = 0
		logs.Log(fmt.Sprintf("Starting new download of chunk %d", idx))
	}
	defer file.Close()

	headers := map[string]string{
		"User-Agent": "tech-idm",
		"Range":      fmt.Sprintf("bytes=%d-%d", byteChunk[0]+startOffset, byteChunk[1]),
	}

	resp, err := d.HttpClient.SendRequestWithContext(ctx, "GET", d.Url, headers)
	if err != nil {
		logs.Log(fmt.Sprintf("Failed to send request for chunk %d of %s: %v", idx, d.FileName, err))
		return fmt.Errorf("failed to send request for chunk %d: %w", idx, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		logs.Log(fmt.Sprintf("Received invalid response for chunk %d of %s: status code %d", idx, d.FileName, resp.StatusCode))
		return fmt.Errorf("invalid response for chunk %d: status code %d", idx, resp.StatusCode)
	}

	startTime := time.Now()
	totalRead := startOffset
	buffer := make([]byte, 32*1024)

	for {
		select {
		case <-ctx.Done():
			logs.Log(fmt.Sprintf("Download of chunk %d for %s canceled", idx, d.FileName))
			return ctx.Err()
		default:
			d.checkPause()

			n, readErr := resp.Body.Read(buffer)
			if n > 0 {
				logs.Log(fmt.Sprintf("Read %d bytes for chunk %d of %s", n, idx, d.FileName))
				_, writeErr := file.Write(buffer[:n])
				if writeErr != nil {
					logs.Log(fmt.Sprintf("Failed to write %d bytes to file %s for chunk %d: %v", n, fileName, idx, writeErr))
					return fmt.Errorf("failed writing %d bytes to %s for chunk %d: %w", n, fileName, idx, writeErr)
				}
				totalRead += n
				d.CompletedBytes[idx] = totalRead
				logs.Log(fmt.Sprintf("Chunk %d of %s: total bytes downloaded so far: %d", idx, d.FileName, d.CompletedBytes[idx]))

				if d.SpeedLimit > 0 {
					expectedTime := float64(totalRead) / float64(d.SpeedLimit) // seconds
					elapsed := time.Since(startTime).Seconds()
					if elapsed < expectedTime {
						sleepDuration := time.Duration((expectedTime - elapsed) * float64(time.Second))
						logs.Log(fmt.Sprintf("Chunk %d of %s: sleeping for %.2f seconds to respect speed limit of %d bytes/s", idx, d.FileName, sleepDuration.Seconds(), d.SpeedLimit))
						time.Sleep(sleepDuration)
					}
				}
			}

			if readErr == io.EOF {
				logs.Log(fmt.Sprintf("Finished reading chunk %d of %s: reached EOF", idx, d.FileName))
				return nil
			}
			if readErr != nil {
				logs.Log(fmt.Sprintf("Error reading chunk %d of %s: %v", idx, d.FileName, readErr))
				return fmt.Errorf("error reading chunk %d of %s: %w", idx, d.FileName, readErr)
			}
		}
	}
}

func (d *DownloadController) MergeDownloads(dirPath, mergeDir string) error {
	outFile := fmt.Sprintf("%s/%s", mergeDir, d.FileName)
	logs.Log(fmt.Sprintf(("Starting to merge chunks into final file: %s"), outFile))
	out, err := os.Create(outFile)
	if err != nil {
		logs.Log(fmt.Sprintf(("Failed to create output file %s: %v"), outFile, err))
		return fmt.Errorf("failed to create output file %s: %w", outFile, err)
	}
	defer out.Close()

	for idx := range d.Chunks {
		fileName := fmt.Sprintf("%s/%s-%s-%d.tmp", dirPath, config.TMP_FILE_PREFIX, d.FileName, idx)
		logs.Log(fmt.Sprintf(("Opening chunk %d file for merging: %s"), idx, fileName))
		in, err := os.Open(fileName)
		if err != nil {
			logs.Log(fmt.Sprintf(("Failed to open chunk file %s: %v"), fileName, err))
			return fmt.Errorf("failed to open chunk file %s: %w", fileName, err)
		}

		logs.Log(fmt.Sprintf(("Merging chunk %d from %s into %s"), idx, fileName, outFile))
		_, err = io.Copy(out, in)
		in.Close() // Close immediately after copying
		if err != nil {
			logs.Log(fmt.Sprintf(("Failed to merge chunk file %s into %s: %v"), fileName, outFile, err))
			return fmt.Errorf("failed to merge chunk file %s: %w", fileName, err)
		}
		logs.Log(fmt.Sprintf(("Successfully merged chunk %d from %s"), idx, fileName))
	}

	logs.Log(fmt.Sprintf(("Successfully merged all chunks into %s"), outFile))
	return nil
}

func (d *DownloadController) CleanupTmpFiles(tmpPath string) error {
	logs.Log(fmt.Sprintf(("Starting cleanup of temporary files for %s"), d.FileName))
	for idx := range d.Chunks {
		fileName := fmt.Sprintf("%s/%s-%s-%d.tmp", tmpPath, config.TMP_FILE_PREFIX, d.FileName, idx)
		logs.Log(fmt.Sprintf(("Attempting to remove temporary file: %s"), fileName))
		err := os.Remove(fileName)
		if err != nil {
			logs.Log(fmt.Sprintf(("Failed to remove temporary file %s: %v"), fileName, err))
			return fmt.Errorf("failed to remove temporary file %s: %w", fileName, err)
		}
		logs.Log(fmt.Sprintf(("Successfully removed temporary file %s"), fileName))
	}
	logs.Log(fmt.Sprintf(("Completed cleanup of all temporary files for %s"), d.FileName))
	return nil
}

func (d *DownloadController) checkPause() {
	d.Mutex.Lock()
	if d.Status == PAUSED {
		logs.Log(fmt.Sprintf(("Download %s is paused, waiting for resume signal"), d.ID))
		d.Mutex.Unlock()
		<-d.ResumeChan
		logs.Log(fmt.Sprintf(("Received resume signal for download %s"), d.ID))
	} else {
		d.Mutex.Unlock()
	}
}

func (d *DownloadController) Pause() {
	d.Mutex.Lock()
	if d.Status == ONGOING {
		d.Status = PAUSED
		logs.Log(fmt.Sprintf(("Download %s has been paused"), d.ID))
	} else {
		logs.Log(fmt.Sprintf(("Download %s is already paused or not ongoing, no action taken"), d.ID))
	}
	d.Mutex.Unlock()
}

func (d *DownloadController) Resume() {
	d.Mutex.Lock()
	if d.Status == PAUSED {
		d.Status = ONGOING
		logs.Log(fmt.Sprintf(("Download %s has been resumed"), d.ID))
		d.ResumeChan <- true // Notify goroutines to resume
	} else {
		logs.Log(fmt.Sprintf(("Download %s is not paused, no action taken"), d.ID))
	}
	d.Mutex.Unlock()
}

func (dc *DownloadController) GetStatus() Status {
	dc.Mutex.Lock()
	defer dc.Mutex.Unlock()
	return dc.Status
}

func (dc *DownloadController) SetStatus(newStatus Status) {
	dc.Mutex.Lock()
	defer dc.Mutex.Unlock()
	dc.Status = newStatus
}

func (d *DownloadController) Retry(idx int, byteChunk [2]int, tmpPath string, maxRetries int) error {
	var err error
	for retry := 0; retry < maxRetries; retry++ {
		logs.Log(fmt.Sprintf("Attempting retry %d for chunk %d of %s", retry+1, idx, d.FileName))
		err = d.Download(idx, byteChunk, tmpPath, d.ctx)
		if err == nil {
			logs.Log(fmt.Sprintf("Successfully retried chunk %d of %s", idx, d.FileName))
			return nil
		}
		logs.Log(fmt.Sprintf("Retry %d for chunk %d failed: %v", retry+1, idx, err))
		time.Sleep(time.Second * 2) // Wait before retrying
	}
	return fmt.Errorf("failed to download chunk %d after %d retries: %w", idx, maxRetries, err)
}
