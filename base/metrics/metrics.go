package metrics

import (
	"app/base/mqueue"
	"app/base/utils"
	"os"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	KafkaConnectionErrorCnt = prometheus.NewCounterVec(prometheus.CounterOpts{
		Help:      "Counter vector measuring Kafka connection issues when trying to read or write a message",
		Namespace: "patchman_engine",
		Subsystem: "core",
		Name:      "kafka_connection_errors",
	}, []string{"type"})

	EngineVersion = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Help:      "Patchman project deployment information",
		Namespace: "patchman_engine",
		Subsystem: "core",
		Name:      "info",
	}, []string{"version"})
)

func init() {
	if utils.Cfg.KafkaAddress != "" {
		prometheus.MustRegister(KafkaConnectionErrorCnt)
	}
	prometheus.MustRegister(EngineVersion)
	engineVersion, _ := os.ReadFile("VERSION")
	EngineVersion.WithLabelValues(strings.TrimSuffix(string(engineVersion), "\n")).Set(1)
}

func Configure() {
	if utils.Cfg.KafkaAddress != "" {
		mqueue.SetKafkaErrorReadCnt(KafkaConnectionErrorCnt.WithLabelValues("read"))
		mqueue.SetKafkaErrorWriteCnt(KafkaConnectionErrorCnt.WithLabelValues("write"))
	}
}
