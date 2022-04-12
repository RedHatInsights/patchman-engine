package metrics

import (
	"app/base/mqueue"
	"github.com/prometheus/client_golang/prometheus"
	"os"
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

	// ENGINEVERSION - DO NOT EDIT this variable MANUALLY - it is modified by generate_docs.sh
	ENGINEVERSION = "v1.18.67"
)

func init() {
	if os.Getenv("KAFKA_ADDRESS") != "" {
		prometheus.MustRegister(KafkaConnectionErrorCnt)
	}
	prometheus.MustRegister(EngineVersion)
	EngineVersion.WithLabelValues(ENGINEVERSION).Set(1)
}

func Configure() {
	if os.Getenv("KAFKA_ADDRESS") != "" {
		mqueue.SetKafkaErrorReadCnt(KafkaConnectionErrorCnt.WithLabelValues("read"))
		mqueue.SetKafkaErrorWriteCnt(KafkaConnectionErrorCnt.WithLabelValues("write"))
	}
}
