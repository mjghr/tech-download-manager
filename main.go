package main

import (
	"bufio"
	"fmt"
	"log"
	"net/url"
	"os"
	"github.com/mjghr/tech-download-manager/util"
)

func main(){
	util.LoadEnv();
	welcomeMessage := os.Getenv("WELCOME_MESSAGE")
	fmt.Println(welcomeMessage)

	url, err := GetUrlFromUser()
	if err != nil {
		log.Fatal("invalid URL:", err)
	}

	fmt.Println(url)
}


func GetUrlFromUser()(*url.URL, error){
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