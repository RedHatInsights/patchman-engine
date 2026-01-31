package utils

import (
	"context"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestRunServer(t *testing.T) {
	ConfigureLogging()

	var hook = NewTestLogHook()
	log.AddHook(hook)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(time.Millisecond * 100)
		cancel()
	}()
	err := RunServer(ctx, gin.Default(), 8888)
	assert.Nil(t, err)
	AssertEqualWait(t, 1, func() (exp, act interface{}) {
		return 2, len(hook.LogEntries)
	})
	assert.Equal(t, "gracefully shutting down server...", hook.LogEntries[0].Message)
	assert.Equal(t, "server closed successfully", hook.LogEntries[1].Message)
}
