package main

import (
	"app/base/core"
	"app/evaluator"
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
		case "manager":
			manager.RunManager()
			return
		case "listener":
			listener.RunListener()
			return
		case "evaluator":
			evaluator.RunEvaluator()
			return
		case "vmaas_sync":
			vmaas_sync.RunVmaasSync()
			return
		}
	}
	log.Fatal("You need to provide a command")
}
