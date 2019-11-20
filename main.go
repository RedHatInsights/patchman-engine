package main

import (
	"app/base/core"
	"app/listener"
	"app/webserver"
	"log"
	"os"
)

func main() {
	core.ConfigureApp()

	// optionally run listener
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "listener":
			listener.RunListener()
			return
		case "webserver":
			webserver.RunWebserver()
			return
		}
	}
	log.Fatal("You need to provide a command")
}
