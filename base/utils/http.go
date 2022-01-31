package utils

import (
	"context"
	"fmt"
	"github.com/lestrrat-go/backoff"
	"github.com/pkg/errors"
	"net/http"
	"net/http/httputil"
	"time"
)

func HTTPCallRetry(ctx context.Context, httpCallFun func() (outputDataPtr interface{}, resp *http.Response, err error),
	exponentialRetry bool, maxRetries int, codesToRetry ...int) (outputDataPtr interface{}, err error) {
	backoffState, cancel := startBackoff(ctx, exponentialRetry, maxRetries)
	defer cancel()
	attempt := 0
	for backoff.Continue(backoffState) {
		attempt++
		outDataPtr, resp, callErr := httpCallFun()
		if statusCodeFound(resp, codesToRetry) {
			Log("attempt", attempt, "status_code", TryGetStatusCode(resp)).
				Warn("HTTP call ended with wrong status code")
			continue
		}

		if callErr == nil {
			return outDataPtr, nil
		}

		if len(codesToRetry) == 0 { // no "retry" codes specified, continue always
			Log("attempt", attempt, "err", callErr.Error()).Warn("HTTP call failed, trying again")
			continue
		}

		responseDetails := tryGetResponseDetails(resp)
		return nil, errors.Wrap(callErr, "HTTP call failed"+responseDetails)
	}
	return nil, errors.Errorf("HTTP retry call failed, attempts: %d", attempt)
}

func CallAPI(client *http.Client, request *http.Request, debugEnabled bool) (*http.Response, error) {
	if debugEnabled {
		dump, err := httputil.DumpRequestOut(request, true)
		if err != nil {
			return nil, err
		}
		Log("dump", fmt.Sprintf("\n%s\n", string(dump))).Debug("http call")
	}

	resp, err := client.Do(request)
	if err != nil {
		return resp, err
	}

	if debugEnabled {
		dump, err := httputil.DumpResponse(resp, true)
		if err != nil {
			return resp, err
		}
		Log("dump", fmt.Sprintf("\n%s\n", string(dump))).Debug("http response")
	}
	return resp, err
}

func tryGetResponseDetails(response *http.Response) string {
	details := ""
	if response != nil {
		details = fmt.Sprintf(", status code: %d", response.StatusCode)
	}
	return details
}

func TryGetStatusCode(response *http.Response) int {
	if response == nil {
		return 0
	}
	return response.StatusCode
}

func startBackoff(ctx context.Context, exponential bool, maxRetries int) (backoff.Backoff, backoff.CancelFunc) {
	opts := []backoff.Option{backoff.WithInterval(time.Second), backoff.WithMaxRetries(maxRetries)}
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
