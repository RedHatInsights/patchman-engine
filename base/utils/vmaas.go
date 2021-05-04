package utils

import (
	"context"
	"github.com/lestrrat-go/backoff"
	"github.com/pkg/errors"
	"net/http"
	"time"
)

func HTTPCallRetry(ctx context.Context, vmaasCallFun func() (vmaasDataPtr interface{}, resp *http.Response, err error),
	exponentialRetry bool, maxRetries int, codesToRetry ...int) (vmaasDataPtr interface{}, err error) {
	backoffState, cancel := startBackoff(ctx, exponentialRetry, maxRetries)
	defer cancel()
	for backoff.Continue(backoffState) {
		responseData, resp, callErr := vmaasCallFun()
		if callErr == nil {
			return responseData, nil
		}

		if len(codesToRetry) == 0 { // no "retry" codes specified, continue always
			continue
		}

		if statusCodeFound(resp, codesToRetry) {
			continue
		}

		responseDetails := TryGetResponseDetails(resp)
		return nil, errors.Wrap(err, "HTTP call failed"+responseDetails)
	}
	return nil, errors.New("HTTP retry call failed")
}

func startBackoff(ctx context.Context, exponential bool, maxRetries int) (backoff.Backoff, backoff.CancelFunc) {
	opts := []backoff.Option{backoff.WithInterval(time.Second)}
	if maxRetries > 0 {
		opts = append(opts, backoff.WithMaxRetries(maxRetries))
	}
	if exponential {
		var policy = backoff.NewExponential(opts...)
		backoffState, cancel := policy.Start(ctx)
		return backoffState, cancel
	}
	var policy = backoff.NewConstant(time.Second, opts...)
	backoffState, cancel := policy.Start(ctx)
	return backoffState, cancel
}

func statusCodeFound(response *http.Response, statusCodes []int) bool {
	if response == nil {
		return false
	}

	for _, code := range statusCodes {
		if code == response.StatusCode {
			return true
		}
	}
	return false
}
