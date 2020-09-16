package listener

import (
	"app/base"
	"app/base/core"
	"app/base/utils"
	"app/manager/middlewares"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	EventUpload             = "upload"
	EventDelete             = "delete"
	ReceivedSuccess         = "success-eval"
	ReceivedSuccessNoEval   = "success-unchanged"
	ReceivedDeleted         = "success-deleted"
	ReceivedErrorIdentity   = "error-identity"
	ReceivedErrorProcessing = "error-processing"
	ReceivedErrorOtherType  = "error-other-type"
	ReceivedWarnNoRows      = "warn-no-rows"
	ReceivedWarnNoPackages  = "warn-no-packages"
)

var (
	messagesReceivedCnt = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "patchman_engine",
		Subsystem: "listener",
		Name:      "kafka_message_received",
	}, []string{"event", "type"})

	messageHandlingDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "patchman_engine",
		Subsystem: "listener",
		Name:      "kafka_message_handling_duration_seconds",
	}, []string{"event"})

	reposAddedCnt = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "patchman_engine",
		Subsystem: "listener",
		Name:      "repos_added",
	})

	receivedFromReporter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "patchman_engine",
		Subsystem: "listener",
		Name:      "received_from_reporter",
	}, []string{"reporter"})
)

func RunMetrics() {
	prometheus.MustRegister(messagesReceivedCnt, messageHandlingDuration, reposAddedCnt, receivedFromReporter)

	// create web app
	app := gin.New()
	core.InitProbes(app)
	middlewares.Prometheus().Use(app)

	err := utils.RunServer(base.Context, app, ":8081")
	if err != nil {
		utils.Log("err", err.Error()).Error()
		panic(err)
	}
}
