package models

import (
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

type DownloadRequest struct {
	Url        string
	FileName   string
	Chunks     int
	ChunkSize  int
	TotalSize  int
	HttpClient *client.HTTPClient
}
