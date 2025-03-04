package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

var (
	WELCOME_MESSAGE string
	WORKERS_NUM     int
	TMP_FILE_PREFIX string
)

func LoadEnv() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading env file.")
	}
	WELCOME_MESSAGE = getEnvString("WELCOME_MESSAGE", "")
	WORKERS_NUM = getEnvInt("WORKERS_NUM", 0)
	TMP_FILE_PREFIX = getEnvString("TMP_FILE_PREFIX","")
}

func getEnvString(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		intValue, err := strconv.Atoi(value)
		if err != nil {
			log.Printf("Warning: %s is not a valid integer, using default value %d\n", key, defaultValue)
			return defaultValue
		}
		return intValue
	}
	return defaultValue
}
