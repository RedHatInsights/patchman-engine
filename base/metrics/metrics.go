package metrics

import (
	"app/base/mqueue"
	"app/base/utils"
	"os"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	Namespace          = "patchman_engine"
	SubsystemCore      = "core"
	SubsystemEvaluator = "evaluator"
	SubsystemListener  = "listener"
	SubsystemManager   = "manager"
	SubsystemTasks     = "tasks"
	SubsystemVmaasSync = "vmaas_sync"
	LabelType          = "type"
	LabelVersion       = "version"
	LabelEvent         = "event"
)

var (
	KafkaConnectionErrorCnt = prometheus.NewCounterVec(prometheus.CounterOpts{
		Help:      "Counter vector measuring Kafka connection issues when trying to read or write a message",
		Namespace: Namespace,
		Subsystem: SubsystemCore,
		Name:      "kafka_connection_errors",
	}, []string{LabelType})

	EngineVersion = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Help:      "Patchman project deployment information",
		Namespace: Namespace,
		Subsystem: SubsystemCore,
		Name:      "info",
	}, []string{LabelVersion})
)

func init() {
	if utils.CoreCfg.KafkaAddress != "" {
		prometheus.MustRegister(KafkaConnectionErrorCnt)
	}
	prometheus.MustRegister(EngineVersion)
	engineVersion, _ := os.ReadFile("VERSION")
	EngineVersion.WithLabelValues(strings.TrimSuffix(string(engineVersion), "\n")).Set(1)
}

func Configure() {
	if utils.CoreCfg.KafkaAddress != "" {
		mqueue.SetKafkaErrorReadCnt(KafkaConnectionErrorCnt.WithLabelValues("read"))
		mqueue.SetKafkaErrorWriteCnt(KafkaConnectionErrorCnt.WithLabelValues("write"))
	}
}
