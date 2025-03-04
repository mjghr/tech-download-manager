package main

import (
	"fmt"
	"log"
	"github.com/joho/godotenv"
	"os"
)

func main(){
	err := godotenv.Load()
	if err != nil {
		log.Fatal("failed loading env file")
	}

	welcomeMessage := os.Getenv("WELCOME_MESSAGE")
	fmt.Println(welcomeMessage)

}