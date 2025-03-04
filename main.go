package main

import (
	"bufio"
	"fmt"
	"github.com/mjghr/tech-download-manager/config"
	"log"
	"net/url"
	"os"
)

func main() {
	config.LoadEnv()
	fmt.Println(config.WELCOME_MESSAGE)

	url, err := GetUrlFromUser()
	if err != nil {
		log.Fatal("invalid URL:", err)
	}

	fmt.Println(url)
}

func GetUrlFromUser() (*url.URL, error) {
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
