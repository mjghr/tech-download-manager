package manager

import (
	"log"
	"net/url"
	"strconv"
	"fmt"
	"sync"

	"github.com/mjghr/tech-download-manager/client"
	"github.com/mjghr/tech-download-manager/controller"
	"github.com/mjghr/tech-download-manager/util"
)

func Download(urlPtr *url.URL) {
	httpRequestSender := client.NewHTTPClient()
	url := urlPtr.String()
	reqMethod := "HEAD"
	headers := map[string]string{
		"user_agent": "tech-idm",
	}
	response, err := httpRequestSender.SendRequest(reqMethod, url, headers)
	if err != nil {
		log.Fatal(err)
	}

	acceptRangers := response.Header.Get("Accept-Ranges")
	log.Println(acceptRangers)

	contentLength := response.Header.Get("Content-Length")
	log.Println("Content-Length:", contentLength)
	contentLengthInBytes, err := strconv.Atoi(contentLength)
	if err != nil {
		log.Fatal("empty, can't download the file", err)
	}
	log.Println("Content-Length:", contentLengthInBytes)
	workers, chunkSize := util.CalculateOptimalWorkersAndChunkSize(contentLengthInBytes)
	log.Println("Optimal Workers:", workers)
	log.Println("Optimal Chunk Size:", chunkSize)
	fileName, err := util.ExtractFileName(url)
	if err != nil {
		log.Fatal("Error while extracting file name...")
	}

	
	log.Println("Filename extracted: ", fileName)
	downReq := &controller.DownloadRequest{
		Url:        url,
		FileName:   fileName,
		Chunks:     workers,
		ChunkSize:  chunkSize,
		TotalSize:  contentLengthInBytes,
		HttpClient: httpRequestSender,
	}

	byteRangeArray := make([][2]int, workers)
	byteRangeArray = downReq.SplitIntoChunks()
	fmt.Println(byteRangeArray)

	var wg sync.WaitGroup
	for idx, byteChunk := range byteRangeArray {
		wg.Add(1) 
		go func(idx int, byteChunk [2]int) {
			defer wg.Done()
			err := downReq.Download(idx, byteChunk)
			if err != nil {
				log.Fatal(fmt.Sprintf("Failed to download chunk %v", idx), err)
			}
		}(idx, byteChunk)
	}
	wg.Wait()

	err = downReq.MergeDownloads()
	if err != nil {
		log.Fatal("Failed merging tmp downloaded files...", err)
	}

	err = downReq.CleanupTmpFiles()
	if err != nil {
		log.Fatal("Failed cleaning up tmp downloaded files...", err)
	}

	log.Println(fmt.Sprintf("File generated: %v\n\n", downReq.FileName))

}
