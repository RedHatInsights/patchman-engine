package utils

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

type closeTrackingBody struct {
	rc        io.ReadCloser
	readBytes atomic.Int64
	closed    atomic.Bool
}

func newCloseTrackingBody(content string) *closeTrackingBody {
	return &closeTrackingBody{rc: io.NopCloser(strings.NewReader(content))}
}

func (b *closeTrackingBody) Read(p []byte) (int, error) {
	n, err := b.rc.Read(p)
	if n > 0 {
		b.readBytes.Add(int64(n))
	}
	return n, err
}

func (b *closeTrackingBody) Close() error {
	b.closed.Store(true)
	return b.rc.Close()
}

func trackingResponse(status int, body string) (*http.Response, *closeTrackingBody) {
	tb := newCloseTrackingBody(body)
	return &http.Response{
		StatusCode: status,
		Body:       tb,
	}, tb
}

func assertBodyDrainedAndClosed(t *testing.T, body *closeTrackingBody, content string) {
	t.Helper()
	assert.True(t, body.closed.Load())
	expected := int64(len(content))
	if expected > maxHTTPResponseDrain {
		expected = maxHTTPResponseDrain
	}
	assert.Equal(t, expected, body.readBytes.Load())
}

func createFirstlyFailingCallFun() func() (interface{}, *http.Response, error) {
	i := 0
	httpCallFun := func() (interface{}, *http.Response, error) {
		if i < 2 {
			i++
			return nil, nil, errors.New("testing error")
		}
		strPtr := "some data"
		return &strPtr, nil, nil
	}
	return httpCallFun
}

func TestHTTPCallRetrySucc(t *testing.T) {
	// nolint:bodyclose
	data, err := HTTPCallRetry(createFirstlyFailingCallFun(), false, 0)
	dataStrPtr := data.(*string)
	assert.Nil(t, err)
	assert.Equal(t, "some data", *dataStrPtr)
}

func TestHTTPCallRetryFail(t *testing.T) {
	// nolint:bodyclose
	data, err := HTTPCallRetry(createFirstlyFailingCallFun(), false, 1)
	assert.NotNil(t, err)
	assert.Equal(t, "HTTP retry call failed, attempts: 2", err.Error())
	assert.Nil(t, data)
}

func TestHTTPCallRetryClosesBodyOnSuccess(t *testing.T) {
	const content = "ok"
	resp, body := trackingResponse(http.StatusOK, content)
	data, err := HTTPCallRetry(func() (interface{}, *http.Response, error) {
		return "data", resp, nil
	}, false, 1)

	assert.NoError(t, err)
	assert.Equal(t, "data", data)
	assertBodyDrainedAndClosed(t, body, content)
}

func TestHTTPCallRetryClosesBodyOnRetryableStatus(t *testing.T) {
	const firstContent = "retry"
	const secondContent = "ok"
	first, firstBody := trackingResponse(http.StatusServiceUnavailable, firstContent)
	second, secondBody := trackingResponse(http.StatusOK, secondContent)
	attempt := 0
	data, err := HTTPCallRetry(func() (interface{}, *http.Response, error) {
		attempt++
		if attempt == 1 {
			return nil, first, nil
		}
		return "data", second, nil
	}, false, 3, http.StatusServiceUnavailable)

	assert.NoError(t, err)
	assert.Equal(t, "data", data)
	assertBodyDrainedAndClosed(t, firstBody, firstContent)
	assertBodyDrainedAndClosed(t, secondBody, secondContent)
}

func TestHTTPCallRetryClosesBodyOnNonRetryableError(t *testing.T) {
	const content = "nope"
	resp, body := trackingResponse(http.StatusBadRequest, content)
	data, err := HTTPCallRetry(func() (interface{}, *http.Response, error) {
		return nil, resp, errors.New("bad request")
	}, false, 3, http.StatusServiceUnavailable)

	assert.Error(t, err)
	assert.Nil(t, data)
	assertBodyDrainedAndClosed(t, body, content)
}

func TestCloseHTTPResponseDrainsBody(t *testing.T) {
	const content = "some body data"
	body := newCloseTrackingBody(content)
	closeHTTPResponse(&http.Response{StatusCode: http.StatusOK, Body: body})
	assertBodyDrainedAndClosed(t, body, content)
}

func TestCloseHTTPResponseLimitsDrain(t *testing.T) {
	content := strings.Repeat("x", maxHTTPResponseDrain+1024)
	body := newCloseTrackingBody(content)
	closeHTTPResponse(&http.Response{StatusCode: http.StatusOK, Body: body})
	assert.True(t, body.closed.Load())
	assert.Equal(t, int64(maxHTTPResponseDrain), body.readBytes.Load())
}
