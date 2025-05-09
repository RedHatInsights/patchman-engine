package api

import (
	"app/base/utils"
	"bytes"
	"context"
	"io"
	"net/http"

	"github.com/bytedance/sonic"
	"github.com/pkg/errors"
)

type Client struct {
	HTTPClient     *http.Client
	Debug          bool
	DefaultHeaders map[string]string
}

func (o *Client) Request(ctx *context.Context, method, url string,
	requestPtr interface{}, responseOutPtr interface{}) (*http.Response, error) {
	body := &bytes.Buffer{}
	if requestPtr != nil {
		err := sonic.ConfigDefault.NewEncoder(body).Encode(requestPtr)
		if err != nil {
			return nil, errors.Wrap(err, "JSON encoding failed")
		}
	}

	httpReq, err := http.NewRequestWithContext(*ctx, method, url, body)
	if err != nil {
		return nil, errors.Wrap(err, "Request failed")
	}
	httpReq.Header.Add("Content-Type", "application/json")
	addHeaders(httpReq, o.DefaultHeaders)

	httpResp, err := utils.CallAPI(o.HTTPClient, httpReq, o.Debug)
	if err != nil {
		return httpResp, errors.Wrap(err, "Request failed")
	}

	err = sonic.ConfigDefault.NewDecoder(httpResp.Body).Decode(responseOutPtr)
	if err != nil {
		if errors.Is(err, io.EOF) {
			// empty response body
			return httpResp, nil
		}
		return httpResp, errors.Wrap(err, "Response body reading failed")
	}

	return httpResp, nil
}

func addHeaders(request *http.Request, headersMap map[string]string) {
	if headersMap == nil {
		return
	}
	for k, v := range headersMap {
		request.Header.Add(k, v)
	}
}
