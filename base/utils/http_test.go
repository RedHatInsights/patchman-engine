package utils

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

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
	data, err := HTTPCallRetry(context.Background(), createFirstlyFailingCallFun(), false, 0)
	dataStrPtr := data.(*string)
	assert.Nil(t, err)
	assert.Equal(t, "some data", *dataStrPtr)
}

func TestHTTPCallRetryFail(t *testing.T) {
	// nolint:bodyclose
	data, err := HTTPCallRetry(context.Background(), createFirstlyFailingCallFun(), false, 1)
	assert.NotNil(t, err)
	assert.Equal(t, "HTTP retry call failed, attempts: 2", err.Error())
	assert.Nil(t, data)
}
