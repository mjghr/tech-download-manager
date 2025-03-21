package controller

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)


func SaveQueueControllers(filename string, data []*QueueController) error {
	// Create the directory if it doesn't exist
	dir := filepath.Dir(filename)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return fmt.Errorf("could not create directory: %v", err)
	}

	// Serialize data to JSON
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	// Write to the file
	return os.WriteFile(filename, jsonData, 0644)
}


func LoadQueueControllers(filename string) ([]*QueueController, error) {
	jsonData, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var queueControllers []*QueueController
	err = json.Unmarshal(jsonData, &queueControllers)
	if err != nil {
		return nil, err
	}

	return queueControllers, nil
}
