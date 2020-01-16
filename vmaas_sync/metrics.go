package vmaas_sync //nolint:golint,stylecheck

import "github.com/prometheus/client_golang/prometheus"

var (
	messagesReceivedCnt = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "patchman_engine",
		Subsystem: "vmaas_sync",
		Name:      "websocket_msgs",
	}, []string{"type"})

	vmaasCallCnt = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "patchman_engine",
		Subsystem: "vmaas_sync",
		Name:      "vmaas_call",
	}, []string{"type"})

	storeAdvisoriesCnt = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "patchman_engine",
		Subsystem: "vmaas_sync",
		Name:      "store_advisories",
	}, []string{"type"})
)
