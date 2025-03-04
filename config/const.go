package config

import (
	"github.com/joho/godotenv"
	"log"
	"os"
)

var (
	WELCOME_MESSAGE string
)

func LoadEnv() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading env file.")
	}
	WELCOME_MESSAGE = getEnv("WELCOME_MESSAGE", "8080")
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
