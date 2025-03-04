package models

import (
	"github.com/mjghr/tech-download-manager/client"
	"time"
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
	ID                  string
	queueID             string
	url                 string
	tempFileDestination string
	status              Status
	progress            float64
	speed               int
}

type Queue struct {
	ID                  string
	saveDestination     string
	speedLimit          int
	concurrentDownloadLimit       int
	startTime           time.Time
	endTime             time.Time
	downloadControllers []DownloadController
}

type DownloadRequest struct {
	Url        string
	FileName   string
	Chunks     int
	ChunkSize  int
	TotalSize  int
	HttpClient *client.HTTPClient
}
