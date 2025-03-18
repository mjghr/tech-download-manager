package models

import (
	"time"

	"github.com/mjghr/tech-download-manager/client"
)

type Status int

const (
	NOT_STARTED Status = iota
	PAUSED
	FAILED
	COMPLETED
	ONGOING
)

type Queue struct {
	ID                      string
	SaveDestination         string
	SpeedLimit              int
	ConcurrentDownloadLimit int
	StartTime               time.Time
	EndTime                 time.Time
}

type DownloadRequest struct {
	Url        string
	FileName   string
	Chunks     int
	ChunkSize  int
	TotalSize  int
	HttpClient *client.HTTPClient
}
