package main

import (
	"fmt"

	"github.com/braginantonev/mhserver/internal/application"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file", err)
	}

	app := application.NewApplication()
	app.Run()
}
