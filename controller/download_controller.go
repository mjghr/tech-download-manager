package controller

import (
	"fmt"
	"log"
	"os"
	"io"

	"github.com/mjghr/tech-download-manager/client"
	"github.com/mjghr/tech-download-manager/config"
)


type DownloadRequest struct {
	Url        string
	FileName   string
	Chunks     int
	ChunkSize  int
	TotalSize  int
	HttpClient *client.HTTPClient
}

func (d *DownloadRequest) SplitIntoChunks() [][2]int {
	arr := make([][2]int, d.Chunks)
	for i := range d.Chunks {
		if i == 0 {
			arr[i][0] = 0
			arr[i][1] = d.ChunkSize
		} else if i == d.Chunks-1 {
			arr[i][0] = arr[i-1][1] + 1
			arr[i][1] = d.TotalSize - 1
		} else {
			arr[i][0] = arr[i-1][1] + 1
			arr[i][1] = arr[i][0] + d.ChunkSize
		}
	}
	return arr
}

func (d *DownloadRequest) Download(idx int, byteChunk [2]int) error {
	log.Printf("Downloading chunk %v", idx)
	method := "GET"
	headers := map[string]string{
		"User-Agent": "CFD Downloader",
		"Range":      fmt.Sprintf("bytes=%v-%v", byteChunk[0], byteChunk[1]),
	}
	resp, err := d.HttpClient.SendRequest(method, d.Url, headers)
	if err != nil {
		return fmt.Errorf("chunk fail %v", resp.StatusCode)
	}
	if resp.StatusCode > 299 {
		return fmt.Errorf("can't process, response is %v", resp.StatusCode)
	}
	fileName := fmt.Sprintf("%v-%v.tmp", config.TMP_FILE_PREFIX, idx)
	file, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("can't create a file %v", fileName)
	}
	defer file.Close()
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write to file: %v", err)
	}
	log.Printf("Wrote chunk %v to file", idx)
	return nil
}


func (d *DownloadRequest) MergeDownloads() error {
	out, err := os.Create(d.FileName)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer out.Close()
	for idx := 0; idx < d.Chunks; idx++ {
		fileName := fmt.Sprintf("%v-%v.tmp", config.TMP_FILE_PREFIX, idx)
		in, err := os.Open(fileName)
		if err != nil {
			return fmt.Errorf("failed to open chunk file %s: %v", fileName, err)
		}
		defer in.Close()
		_, err = io.Copy(out, in)
		if err != nil {
			return fmt.Errorf("failed to merge chunk file %s: %v", fileName, err)
		}
	}
	fmt.Println("file chunks merged successfully...")
	return nil
}

func (d *DownloadRequest) CleanupTmpFiles() error {
	log.Println("Starting to clean tmp downloaded files...")
	for idx := range d.Chunks {
		fileName := fmt.Sprintf("%v-%v.tmp", config.TMP_FILE_PREFIX, idx)
		err := os.Remove(fileName)
		if err != nil {
			return fmt.Errorf("failed to remove chunk file %s: %v", fileName, err)
		}
	}
	return nil
}
