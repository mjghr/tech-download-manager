package main

import (
	"bufio"
	"fmt"
	// "log"
	"net/url"
	"os"

	"github.com/mjghr/tech-download-manager/config"
	"github.com/mjghr/tech-download-manager/controller"
	"github.com/mjghr/tech-download-manager/manager"
	"github.com/mjghr/tech-download-manager/models"
)

func main() {
	config.LoadEnv()
	fmt.Println(config.WELCOME_MESSAGE)
	// url, err := getUrlFromUser()
	// if err != nil {
	// 	log.Fatal("invalid URL:", err)
	// }
	url1, err1 := url.Parse("https://upload.wikimedia.org/wikipedia/commons/3/31/Napoleon_I_of_France_by_Andrea_Appiani.jpg")
	url2, err2 := url.Parse("https://upload.wikimedia.org/wikipedia/commons/thumb/3/31/David_-_Napoleon_crossing_the_Alps_-_Malmaison1.jpg/640px-David_-_Napoleon_crossing_the_Alps_-_Malmaison1.jpg")
	url3, err3 := url.Parse("https://upload.wikimedia.org/wikipedia/commons/thumb/b/b5/Jacques-Louis_David_-_The_Emperor_Napoleon_in_His_Study_at_the_Tuileries_-_Google_Art_Project_2.jpg/1200px-Jacques-Louis_David_-_The_Emperor_Napoleon_in_His_Study_at_the_Tuileries_-_Google_Art_Project_2.jpg")
	if err1 != nil && err2 != nil && err3 != nil {

	}
	manager := &manager.DownloadManager{}
	// dmc1 := manager.NewDownloadController(url)
	dmc2 := manager.NewDownloadController(url1)
	dmc3 := manager.NewDownloadController(url2)
	dmc4 := manager.NewDownloadController(url3)
	dmcs := []controller.DownloadController{
		// *dmc1,
		*dmc2,
		*dmc3,
		*dmc4,
	}
	queue := &models.Queue{
		ID:                      "1",
		SaveDestination:         "/downloads",
		SpeedLimit:              1024 * 1000, // Example: 100KB/s
		ConcurrentDownloadLimit: 3,          // Max 3 downloads at a time
		DownloadControllers: dmcs,
	}

	manager.DownloadQueue(queue)

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