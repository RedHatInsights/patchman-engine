package main

import (
	"app/base/core"
	"app/listener"
	"app/manager"
	"app/vmaas_sync"
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
		case "vmaas_sync":
			vmaas_sync.RunVmaasSync()
			return
		}
	}
	log.Fatal("You need to provide a command")
}
