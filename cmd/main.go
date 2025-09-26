package main

import (
	"github.com/braginantonev/mhserver/internal/application"
)

func main() {
	app := application.NewApplication()
	app.Run()
}
