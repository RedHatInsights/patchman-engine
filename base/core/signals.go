package core

import (
	"app/base/utils"
	"os"
	"os/signal"
	"syscall"
)

func HandleSignals() {
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGTERM)
	go func() {
		<-c
		utils.Log().Info("SIGTERM handled")
		os.Exit(1)
	}()
}
