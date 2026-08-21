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
	io.Reader
	closed atomic.Bool
}

func (b *closeTrackingBody) Close() error {
	b.closed.Store(true)
	return nil
}

func trackingResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       &closeTrackingBody{Reader: strings.NewReader(body)},
	}
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
	resp := trackingResponse(http.StatusOK, "ok")
	data, err := HTTPCallRetry(func() (interface{}, *http.Response, error) {
		return "data", resp, nil
	}, false, 1)

	assert.NoError(t, err)
	assert.Equal(t, "data", data)
	assert.True(t, resp.Body.(*closeTrackingBody).closed.Load())
}

func TestHTTPCallRetryClosesBodyOnRetryableStatus(t *testing.T) {
	first := trackingResponse(http.StatusServiceUnavailable, "retry")
	second := trackingResponse(http.StatusOK, "ok")
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
	assert.True(t, first.Body.(*closeTrackingBody).closed.Load())
	assert.True(t, second.Body.(*closeTrackingBody).closed.Load())
}

func TestHTTPCallRetryClosesBodyOnNonRetryableError(t *testing.T) {
	resp := trackingResponse(http.StatusBadRequest, "nope")
	data, err := HTTPCallRetry(func() (interface{}, *http.Response, error) {
		return nil, resp, errors.New("bad request")
	}, false, 3, http.StatusServiceUnavailable)

	assert.Error(t, err)
	assert.Nil(t, data)
	assert.True(t, resp.Body.(*closeTrackingBody).closed.Load())
}
