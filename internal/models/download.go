package models

import "time"

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
