package utils

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunServer(t *testing.T) {
	prev := CoreCfg.LogLevel
	CoreCfg.LogLevel = "debug"
	defer func() { CoreCfg.LogLevel = prev }()

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

func TestRunServerWaitsForActiveRequests(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := listener.Addr().(*net.TCPAddr).Port
	require.NoError(t, listener.Close())
	address := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	started := make(chan struct{})
	release := make(chan struct{})
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}()
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- RunServer(ctx, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			close(started)
			<-release
			w.WriteHeader(http.StatusNoContent)
		}), port)
	}()
	require.Eventually(t, func() bool {
		conn, dialErr := net.DialTimeout("tcp", address, 100*time.Millisecond)
		if dialErr != nil {
			return false
		}
		return conn.Close() == nil
	}, time.Second, 10*time.Millisecond)
	requestDone := make(chan error, 1)
	go func() {
		client := &http.Client{Timeout: 3 * time.Second}
		resp, requestErr := client.Get("http://" + address) // #nosec G704 -- local test server.
		if requestErr == nil {
			requestErr = resp.Body.Close()
		}
		requestDone <- requestErr
	}()
	<-started
	cancel()
	select {
	case err := <-serverDone:
		t.Fatalf("server returned before its handler finished: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	close(release)
	require.NoError(t, <-requestDone)
	require.NoError(t, <-serverDone)
}
