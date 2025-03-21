package controller

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"github.com/mjghr/tech-download-manager/client"
	"github.com/mjghr/tech-download-manager/config"
)

type Status int

const (
	NOT_STARTED Status = iota
	PAUSED
	FAILED
	COMPLETED
	ONGOING
)

type DownloadController struct {
	ID             string `json:"id"`
	QueueID        string `json:"queueId"`
	Url            string `json:"url"`
	Status         Status `json:"status"`
	FileName       string `json:"fileName"`
	Chunks         [][2]int `json:"chunks"`
	CompletedBytes []int `json:"completedBytes"`
	TotalSize      int `json:"totalSize"`
	HttpClient     *client.HTTPClient `json:"httpClient"`
	SpeedLimit     int `json:"speedLimit"`

	PauseChan      chan bool `json:"-"`
	Mutex          sync.Mutex `json:"-"`
	ResumeChan     chan bool `json:"-"`
	TokenBucket chan struct{} `json:"-"`

}

func (d *DownloadController) SplitIntoChunks(workers, chunkSize int) [][2]int {
	log.Printf("Starting to split download %s into %d chunks (total size: %d bytes)", d.ID, d.Chunks, d.TotalSize)
	arr := make([][2]int, workers)

	if d.TotalSize <= 0 {
		log.Printf("Error: Total size is %d, cannot split into chunks", d.TotalSize)
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
		log.Printf("Created chunk %d for %s: bytes %d-%d", i, d.ID, start, end)
	}

	log.Printf("Successfully split %s into %d chunks", d.ID, d.Chunks)
	return arr
}

func (d *DownloadController) Download(idx int, byteChunk [2]int, tmpPath string) error {
	log.Printf("Starting download of chunk %d for %s (bytes %d-%d, speed limit: %d bytes/s)", idx, d.FileName, byteChunk[0], byteChunk[1], d.SpeedLimit)
	method := "GET"
	headers := map[string]string{
		"User-Agent": "tech-idm",
		"Range":      fmt.Sprintf("bytes=%d-%d", byteChunk[0], byteChunk[1]),
	}

	log.Printf("Sending HTTP request for chunk %d of %s", idx, d.FileName)
	resp, err := d.HttpClient.SendRequest(method, d.Url, headers)
	if err != nil {
		log.Printf("Failed to send request for chunk %d of %s: %v", idx, d.FileName, err)
		return fmt.Errorf("failed to send request for chunk %d: %w", idx, err)
	}
	if resp.StatusCode > 299 {
		log.Printf("Received invalid response for chunk %d of %s: status code %d", idx, d.FileName, resp.StatusCode)
		return fmt.Errorf("invalid response for chunk %d: status code %d", idx, resp.StatusCode)
	}
	defer resp.Body.Close()

	// Define chunk file
	fileName := fmt.Sprintf("%s/%s-%s-%d.tmp", tmpPath, config.TMP_FILE_PREFIX, d.FileName, idx)
	log.Printf("Creating temporary file for chunk %d: %s", idx, fileName)
	file, err := os.Create(fileName)
	if err != nil {
		log.Printf("Failed to create file %s for chunk %d: %v", fileName, idx, err)
		return fmt.Errorf("failed to create file %s for chunk %d: %w", fileName, idx, err)
	}
	defer file.Close()

	startTime := time.Now()
	totalRead := 0
	buffer := make([]byte, 32*1024) // 32 KB buffer for efficient reading

	for {
		d.checkPause() // Blocks if paused, ensuring pause time isn't counted

		n, readErr := resp.Body.Read(buffer)
		if n > 0 {
			log.Printf("Read %d bytes for chunk %d of %s", n, idx, d.FileName)
			_, writeErr := file.Write(buffer[:n])
			if writeErr != nil {
				log.Printf("Failed to write %d bytes to file %s for chunk %d: %v", n, fileName, idx, writeErr)
				return fmt.Errorf("failed writing %d bytes to %s for chunk %d: %w", n, fileName, idx, writeErr)
			}
			totalRead += n
			d.CompletedBytes[idx] = totalRead
			log.Printf("Chunk %d of %s: total bytes downloaded so far: %d", idx, d.FileName, d.CompletedBytes[idx])

			if d.SpeedLimit > 0 {
				expectedTime := float64(totalRead) / float64(d.SpeedLimit) // seconds
				elapsed := time.Since(startTime).Seconds()
				if elapsed < expectedTime {
					sleepDuration := time.Duration((expectedTime - elapsed) * float64(time.Second))
					log.Printf("Chunk %d of %s: sleeping for %.2f seconds to respect speed limit of %d bytes/s", idx, d.FileName, sleepDuration.Seconds(), d.SpeedLimit)
					time.Sleep(sleepDuration)
				}
			}
		}

		if readErr == io.EOF {
			log.Printf("Finished reading chunk %d of %s: reached EOF", idx, d.FileName)
			break
		}
		if readErr != nil {
			log.Printf("Error reading chunk %d of %s: %v", idx, d.FileName, readErr)
			return fmt.Errorf("error reading chunk %d of %s: %w", idx, d.FileName, readErr)
		}
	}

	elapsed := time.Since(startTime).Seconds()
	log.Printf("Completed download of chunk %d for %s in %.2f seconds (total bytes: %d)", idx, d.FileName, elapsed, totalRead)
	return nil
}

func (d *DownloadController) MergeDownloads(dirPath, mergeDir string) error {
	outFile := fmt.Sprintf("%s/%s", mergeDir, d.FileName)
	log.Printf("Starting to merge chunks into final file: %s", outFile)
	out, err := os.Create(outFile)
	if err != nil {
		log.Printf("Failed to create output file %s: %v", outFile, err)
		return fmt.Errorf("failed to create output file %s: %w", outFile, err)
	}
	defer out.Close()

	for idx := range d.Chunks {
		fileName := fmt.Sprintf("%s/%s-%s-%d.tmp", dirPath, config.TMP_FILE_PREFIX, d.FileName, idx)
		log.Printf("Opening chunk %d file for merging: %s", idx, fileName)
		in, err := os.Open(fileName)
		if err != nil {
			log.Printf("Failed to open chunk file %s: %v", fileName, err)
			return fmt.Errorf("failed to open chunk file %s: %w", fileName, err)
		}

		log.Printf("Merging chunk %d from %s into %s", idx, fileName, outFile)
		_, err = io.Copy(out, in)
		in.Close() // Close immediately after copying
		if err != nil {
			log.Printf("Failed to merge chunk file %s into %s: %v", fileName, outFile, err)
			return fmt.Errorf("failed to merge chunk file %s: %w", fileName, err)
		}
		log.Printf("Successfully merged chunk %d from %s", idx, fileName)
	}

	log.Printf("Successfully merged all chunks into %s", outFile)
	return nil
}

func (d *DownloadController) CleanupTmpFiles(tmpPath string) error {
	log.Printf("Starting cleanup of temporary files for %s", d.FileName)
	for idx := range d.Chunks {
		fileName := fmt.Sprintf("%s/%s-%s-%d.tmp", tmpPath, config.TMP_FILE_PREFIX, d.FileName, idx)
		log.Printf("Attempting to remove temporary file: %s", fileName)
		err := os.Remove(fileName)
		if err != nil {
			log.Printf("Failed to remove temporary file %s: %v", fileName, err)
			return fmt.Errorf("failed to remove temporary file %s: %w", fileName, err)
		}
		log.Printf("Successfully removed temporary file %s", fileName)
	}
	log.Printf("Completed cleanup of all temporary files for %s", d.FileName)
	return nil
}

func (d *DownloadController) checkPause() {
	d.Mutex.Lock()
	if d.Status == PAUSED {
		log.Printf("Download %s is paused, waiting for resume signal", d.ID)
		d.Mutex.Unlock()
		<-d.ResumeChan
		log.Printf("Received resume signal for download %s", d.ID)
	} else {
		d.Mutex.Unlock()
	}
}

func (d *DownloadController) Pause() {
	d.Mutex.Lock()
	if d.Status == ONGOING {
		d.Status = PAUSED
		log.Printf("Download %s has been paused", d.ID)
	} else {
		log.Printf("Download %s is already paused or not ongoing, no action taken", d.ID)
	}
	d.Mutex.Unlock()
}

func (d *DownloadController) Resume() {
	d.Mutex.Lock()
	if d.Status == PAUSED {
		d.Status = ONGOING
		log.Printf("Download %s has been resumed", d.ID)
		d.ResumeChan <- true // Notify goroutines to resume
	} else {
		log.Printf("Download %s is not paused, no action taken", d.ID)
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
