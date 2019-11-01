package main

import (
	"os"

	"gin-container/app/core"
	"gin-container/app/listener"
	"gin-container/app/webserver"
)

func main() {
	core.ConfigureApp()

	// optionally run listener
	if len(os.Args) > 1 && os.Args[1] == "--listener" {
		go listener.RunListener()
	}

	webserver.RunWebserver()
}
