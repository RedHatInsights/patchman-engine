package utils

import (
	"context"
	"os"
	"regexp"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestInitLogging(t *testing.T) {
	assert.Nil(t, os.Setenv("LOG_STYLE", "json"))
	ConfigureLogging()

	var hook = NewTestLogHook()
	log.AddHook(hook)

	LogInfo("num", 1, "str", "text", "info log")

	assert.Equal(t, 1, len(hook.LogEntries))
	entry := hook.LogEntries[0]
	assert.Equal(t, 2, len(entry.Data))
	assert.Equal(t, 1, entry.Data["num"])
	assert.Equal(t, "text", entry.Data["str"])
	assert.Equal(t, "info log", entry.Message)
}

func TestEvenArgs(t *testing.T) {
	assert.Nil(t, os.Setenv("LOG_STYLE", "json"))
	ConfigureLogging()

	var hook = NewTestLogHook()
	log.AddHook(hook)

	LogInfo("num", 1, "str", "text")

	assert.Equal(t, 1, len(hook.LogEntries))
	entry := hook.LogEntries[0]
	assert.Equal(t, 2, len(entry.Data))
	assert.Equal(t, 1, entry.Data["num"])
	assert.Equal(t, "text", entry.Data["str"])
}

func TestLogInfoWithSpanContext(t *testing.T) {
	assert.Nil(t, os.Setenv("LOG_STYLE", "json"))
	ConfigureLogging()

	var hook = NewTestLogHook()
	log.AddHook(hook)

	tp := sdktrace.NewTracerProvider()
	defer func() { _ = tp.Shutdown(context.Background()) }()
	ctx, span := tp.Tracer("test").Start(context.Background(), "log-test")
	defer span.End()

	LogInfo(ctx, "k", "v", "msg")

	require.Equal(t, 1, len(hook.LogEntries))
	entry := hook.LogEntries[0]
	assert.Equal(t, "msg", entry.Message)
	assert.Equal(t, "v", entry.Data["k"])

	traceID, ok := entry.Data["trace_id"].(string)
	require.True(t, ok)
	spanID, ok := entry.Data["span_id"].(string)
	require.True(t, ok)
	assert.Regexp(t, regexp.MustCompile(`^[0-9a-f]{32}$`), traceID)
	assert.Regexp(t, regexp.MustCompile(`^[0-9a-f]{16}$`), spanID)
}
