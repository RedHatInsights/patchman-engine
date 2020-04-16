package base

import (
	"app/base/utils"
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

const VMaaSAPIPrefix = "/api"
const RBACApiPrefix = "/api/rbac/v1"

// Go datetime parser does not like slightly incorrect RFC 3339 which we are using (missing Z )
const Rfc3339NoTz = "2006-01-02T15:04:05-07:00"

var Context context.Context
var CancelContext context.CancelFunc

func init() {
	Context, CancelContext = context.WithCancel(context.Background())
}

func HandleSignals() {
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-c
		CancelContext()
		utils.Log().Info("SIGTERM/SIGINT handled")
	}()
}

func remove(r rune) rune {
	if r == 0 {
		return -1
	}
	return r
}

// Removes characters, which are not accepted by postgresql driver
// in parameter values
func RemoveInvalidChars(s string) string {
	return strings.Map(remove, s)
}
