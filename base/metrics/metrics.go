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
)

func init() {
	if os.Getenv("KAFKA_ADDRESS") != "" {
		prometheus.MustRegister(KafkaConnectionErrorCnt)
	}
}

func Configure() {
	if os.Getenv("KAFKA_ADDRESS") != "" {
		mqueue.SetKafkaErrorReadCnt(KafkaConnectionErrorCnt.WithLabelValues("read"))
		mqueue.SetKafkaErrorWriteCnt(KafkaConnectionErrorCnt.WithLabelValues("write"))
	}
}
