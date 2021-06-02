package base

import (
	"app/base/utils"
	"context"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

const VMaaSAPIPrefix = "/api/v3"
const RBACApiPrefix = "/api/rbac/v1"

// Go datetime parser does not like slightly incorrect RFC 3339 which we are using (missing Z )
const Rfc3339NoTz = "2006-01-02T15:04:05-07:00"

var Context context.Context
var CancelContext context.CancelFunc

func init() {
	Context, CancelContext = context.WithCancel(context.Background())
}

func HandleSignals() {
	c := make(chan os.Signal, 1)
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

type Rfc3339Timestamp time.Time

func (d Rfc3339Timestamp) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.Time().Format(Rfc3339NoTz))
}

func (d *Rfc3339Timestamp) UnmarshalJSON(data []byte) error {
	var jd string
	var err error
	if err := json.Unmarshal(data, &jd); err != nil {
		return err
	}
	t, err := time.Parse(Rfc3339NoTz, jd)
	*d = Rfc3339Timestamp(t)
	return err
}

func (d *Rfc3339Timestamp) Time() *time.Time {
	if d == nil {
		return nil
	}
	return (*time.Time)(d)
}

// TryExposeOnMetricsPort Expose app on required port if set
func TryExposeOnMetricsPort(app *gin.Engine) {
	metricsPort := utils.GetIntEnvOrDefault("METRICS_PORT", -1)
	if metricsPort == -1 {
		return // Do not expose extra metrics port if not set using METRICS_PORT var
	}
	err := utils.RunServer(Context, app, metricsPort)
	if err != nil {
		utils.Log("err", err.Error()).Error()
		panic(err)
	}
}
