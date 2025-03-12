package util

import (
	"fmt"
	"log"
	"math"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
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

func GiveDefaultTempPath() string {
	var tmpPath string
	switch runtime.GOOS {
	case "windows":
		// Windows paths
		userHome, err := os.UserHomeDir()
		if err != nil {
			log.Fatal("Failed to get user home directory:", err)
		}
		tmpPath = filepath.Join(userHome, "Desktop", "tmp")
	case "darwin":
		// macOS paths
		userHome, err := os.UserHomeDir()
		if err != nil {
			log.Fatal("Failed to get user home directory:", err)
		}
		tmpPath = filepath.Join(userHome, "Downloads", "tmp")
	case "linux":
		// Linux paths
		userHome, err := os.UserHomeDir()
		if err != nil {
			log.Fatal("Failed to get user home directory:", err)
		}
		tmpPath = filepath.Join(userHome, "Downloads", "tmp")
	default:
		log.Fatal("Unsupported operating system")
	}

	return tmpPath
}

func GiveDefaultSavePath() string {
	var savePath string
	switch runtime.GOOS {
	case "windows":
		// Windows paths
		userHome, err := os.UserHomeDir()
		if err != nil {
			log.Fatal("Failed to get user home directory:", err)
		}
		savePath = filepath.Join(userHome, "Desktop", "download")
	case "darwin":
		// macOS paths
		userHome, err := os.UserHomeDir()
		if err != nil {
			log.Fatal("Failed to get user home directory:", err)
		}
		savePath = filepath.Join(userHome, "Downloads", "download")
	case "linux":
		// Linux paths
		userHome, err := os.UserHomeDir()
		if err != nil {
			log.Fatal("Failed to get user home directory:", err)
		}
		savePath = filepath.Join(userHome, "Downloads", "download")
	default:
		log.Fatal("Unsupported operating system")
	}

	return savePath
}
