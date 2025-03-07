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

type DownloadController struct {
	ID                  string
	queueID             string
	url                 string
	tempFileDestination string
	status              Status
	progress            float64
	speed               int
}

type TimeWindow struct {
	StartTime time.Time
	EndTime   time.Time
	Enabled   bool
}

type QueueConfig struct {
	SaveDestination        string
	MaxConcurrentDownloads int
	MaxBandwidth           int // in bytes per second, -1 for unlimited
	ActiveTimeWindow       TimeWindow
	MaxRetries             int // -1 for unlimited
}

type Queue struct {
	ID        string
	Config    QueueConfig
	Downloads []*DownloadController
	Status    Status
	CreatedAt time.Time
	UpdatedAt time.Time
}

type DownloadRequest struct {
	Url        string
	FileName   string
	Chunks     int
	ChunkSize  int
	TotalSize  int
	HttpClient *client.HTTPClient
}
