package telemetry

import (
	"app/base/utils"
	"os"
	"strconv"
	"strings"
)

const RHServiceName = "patchman-engine"

func samplingRate() float64 {
	raw := os.Getenv("OTEL_SAMPLING_RATE")
	if raw == "" {
		return 1.0
	}
	n, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 1.0
	}
	if n < 0 {
		return 0
	}
	if n > 1 {
		return 1
	}
	return n
}

func otelEnabled() bool {
	return strings.ToLower(os.Getenv("OTEL_ENABLED")) == "true"
}

func Enabled() bool {
	return enabled
}

func MQEnabled() bool {
	return enabled && utils.GetBoolEnvOrDefault("OTEL_MQ_ENABLED", true)
}

func HTTPInboundEnabled() bool {
	return enabled && utils.GetBoolEnvOrDefault("OTEL_HTTP_INBOUND_ENABLED", true)
}

func HTTPOutboundEnabled() bool {
	return enabled && utils.GetBoolEnvOrDefault("OTEL_HTTP_OUTBOUND_ENABLED", true)
}

func SQLEnabled() bool {
	return enabled && utils.GetBoolEnvOrDefault("OTEL_SQL_ENABLED", true)
}
