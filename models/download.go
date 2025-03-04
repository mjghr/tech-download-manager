package models

import (
	"net/http"
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
	HttpClient *http.Client
}

type HTTPClient struct {
	client *http.Client
}
