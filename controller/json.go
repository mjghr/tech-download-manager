package controller

import (
	"encoding/json"
	"os"

)

func SaveQueueControllers(filename string, data []*QueueController) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

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
