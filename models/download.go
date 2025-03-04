package models

import (
	"github.com/mjghr/tech-download-manager/client"
	"time"
)

type Download struct {
	ID           string    
	uRL          string    
	destination  string    
	queueID      string   
	status       string   
	progress     int     
	speed        int64    
	startTime    time.Time 
	retryCount   int     
}


type DownloadRequest struct {
	Url string 
	FileName   string
	Chunks     int
	ChunkSize  int
	TotalSize  int
	HttpClient *client.HTTPClient
}
