package utils

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
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
