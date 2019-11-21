package main

import (
	"app/base/core"
	"app/listener"
	"app/manager"
	"log"
	"os"
)

func main() {
	core.ConfigureApp()

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "listener":
			listener.RunListener()
			return
		case "manager":
			manager.RunManager()
			return
		}
	}
	log.Fatal("You need to provide a command")
}
