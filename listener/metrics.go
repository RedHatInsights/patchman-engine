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
	EventUpload                  = "upload"
	EventDelete                  = "delete"
	ReceivedSuccess              = "success-eval"
	ReceivedSuccessNoEval        = "success-unchanged"
	ReceivedDeleted              = "success-deleted"
	ReceivedErrorIdentity        = "error-identity"
	ReceivedErrorProcessing      = "error-processing"
	ReceivedErrorOtherType       = "error-other-type"
	ReceivedBypassed             = "bypassed"
	ReceivedWarnNoRows           = "warn-no-rows"
	ReceivedWarnNoPackages       = "warn-no-packages"
	ReceivedWarnExcludedReporter = "warn-excluded-reporter"
	ReceivedWarnExcludedHostType = "warn-excluded-host-type"
	ReceivedWarnBadPackages      = "warn-malformed-packages"
)

var (
	eventMsgsReceivedCnt = prometheus.NewCounterVec(prometheus.CounterOpts{
		Help:      "How many messages received by listener, which event type, which process result",
		Namespace: "patchman_engine",
		Subsystem: "listener",
		Name:      "kafka_message_received",
	}, []string{"event", "type"})

	templateMsgReceivedCnt = prometheus.NewCounterVec(prometheus.CounterOpts{
		Help:      "Number of template messages received by type",
		Namespace: "patchman_engine",
		Subsystem: "listener",
		Name:      "template_message_received",
	}, []string{"event", "type"})

	messageHandlingDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Help:      "How long it took to process the message",
		Namespace: "patchman_engine",
		Subsystem: "listener",
		Name:      "kafka_message_handling_duration_seconds",
	}, []string{"event"})

	templateMsgHandlingDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Help:      "How long it took to process the message",
		Namespace: "patchman_engine",
		Subsystem: "listener",
		Name:      "template_message_handling_duration_seconds",
	}, []string{"event"})
	reposAddedCnt = prometheus.NewCounter(prometheus.CounterOpts{
		Help:      "How many repositories were added",
		Namespace: "patchman_engine",
		Subsystem: "listener",
		Name:      "repos_added",
	})

	receivedFromReporter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Help:      "How many messages were received from which reporter",
		Namespace: "patchman_engine",
		Subsystem: "listener",
		Name:      "received_from_reporter",
	}, []string{"reporter"})

	messagePartDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Help:      "How long particular listener part took",
		Namespace: "patchman_engine",
		Subsystem: "listener",
		Name:      "kafka_message_part_duration_seconds",
	}, []string{"part"})
)

func RunMetrics() {
	prometheus.MustRegister(eventMsgsReceivedCnt, messageHandlingDuration, reposAddedCnt, receivedFromReporter,
		messagePartDuration)

	// create web app
	app := gin.New()
	core.InitProbes(app)
	middlewares.Prometheus().Use(app)

	go base.TryExposeOnMetricsPort(app)

	publicPort := utils.CoreCfg.PublicPort
	err := utils.RunServer(base.Context, app, publicPort)
	if err != nil {
		utils.LogError("err", err.Error())
		panic(err)
	}
}
