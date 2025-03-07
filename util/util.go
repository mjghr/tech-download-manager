package util

import (
	"fmt"
	"math"
	"net/url"
	"path"
	"runtime"
)

const (
	DEFAULT_MAX_CONCURRENT_DOWNLOADS = 3
	DEFAULT_MAX_BANDWIDTH            = 500 * 1024 // 500KB/s
	DEFAULT_MAX_RETRIES              = 3
	UNLIMITED_BANDWIDTH              = -1
	UNLIMITED_RETRIES                = -1
)

func ExtractFileName(urlStr string) (string, error) {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return "", err
	}
	fileName := path.Base(parsedURL.Path)
	if fileName == "/" || fileName == "." {
		return "", fmt.Errorf("error while extracting the fileName: %s", urlStr)
	}
	return fileName, nil
}
func CalculateOptimalWorkersAndChunkSize(fileSize int) (int, int) {
	availableCores := runtime.NumCPU()
	if fileSize < 10*1024*1024 {
		return 1, fileSize
	}
	workers := int(math.Min(float64(availableCores), float64(fileSize/(10*1024*1024))))
	if fileSize > 10*1024*1024*1024 {
		workers = int(math.Min(float64(availableCores*2), float64(fileSize/(10*1024*1024))))
	}
	chunkSize := fileSize / workers
	if chunkSize < 1*1024*1024 {
		chunkSize = 1 * 1024 * 1024
	}
	return workers, chunkSize
}
