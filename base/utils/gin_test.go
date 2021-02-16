package utils

import (
	"context"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
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
	err := RunServer(ctx, gin.Default(), ":8888")
	assert.Nil(t, err)
	AssertWait(t, 1, func() bool {
		return len(hook.LogEntries) > 0
	})
	assert.Equal(t, "server closed successfully", hook.LogEntries[0].Message)
}
