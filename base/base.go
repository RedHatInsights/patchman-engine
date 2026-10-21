package base

import (
	"app/base/telemetry"
	"app/base/utils"
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

const VMaaSAPIPrefix = "/api/v3"
const RBACApiPrefix = "/api/rbac/v1"
const defaultK8sGracePeriod = 30 * time.Second

var Context context.Context
var CancelContext context.CancelFunc

func init() {
	Context, CancelContext = context.WithCancel(context.Background())
}

func HandleSignals() {
	log.RegisterExitHandler(ShutdownTelemetry)
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-c
		utils.LogInfo("starting grace period for " + sig.String())
		time.Sleep(defaultK8sGracePeriod / 4)
		CancelContext()
		utils.LogInfo("SIGTERM/SIGINT handled")
	}()
}

// ShutdownTelemetry flushes finished spans after application work has stopped.
// The application context is already canceled during signal-driven shutdown.
func ShutdownTelemetry() {
	ctx, cancel := context.WithTimeout(context.Background(), defaultK8sGracePeriod/3)
	defer cancel()
	if err := telemetry.Shutdown(ctx); err != nil {
		utils.LogError("err", err, "OpenTelemetry shutdown failed")
	}
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

// TryExposeOnMetricsPort Expose app on required port if set
func TryExposeOnMetricsPort(app *gin.Engine) {
	metricsPort := utils.CoreCfg.MetricsPort
	if metricsPort == -1 {
		return // Do not expose extra metrics port if not set
	}
	err := utils.RunServer(Context, app, metricsPort)
	if err != nil {
		utils.LogError("err", err)
		panic(err)
	}
}
