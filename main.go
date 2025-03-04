package main

import (
	"bufio"
	"fmt"
	"log"
	"net/url"
	"os"

	"github.com/mjghr/tech-download-manager/config"
	"github.com/mjghr/tech-download-manager/manager"
)

func main() {
	config.LoadEnv()
	fmt.Println(config.WELCOME_MESSAGE)
	url, err := getUrlFromUser()
	if err != nil {
		log.Fatal("invalid URL:", err)
	}
	manager.Download(url)
}

func getUrlFromUser() (*url.URL, error) {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("Enter the file URL to download: ")
	scanner.Scan()
	userInput := scanner.Text()
	parsedURL, err := url.Parse(userInput)
	if err != nil {
		return nil, err
	}

	return parsedURL, nil
}
