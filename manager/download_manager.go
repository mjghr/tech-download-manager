package manager

import (
	"log"
	"net/url"
	"strconv"
	"github.com/mjghr/tech-download-manager/client"
	"github.com/mjghr/tech-download-manager/models"
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
	downReq := &models.DownloadRequest{
		Url:        url,
		FileName:   fileName,
		Chunks:     workers,
		ChunkSize:  chunkSize,
		TotalSize:  contentLengthInBytes,
		HttpClient: httpRequestSender,
	}


}
