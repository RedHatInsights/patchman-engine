package vmaas_sync

import "github.com/prometheus/client_golang/prometheus"

var (
	messagesReceivedCnt = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "patchman_engine",
		Subsystem: "vmaas_sync",
		Name:      "websocket_msgs",
	}, []string{"type"})
)
