package middlewares

import (
	"app/base/api"
	"app/base/utils"
	"context"
	"net/http"
	"os"
	"strconv"
	"strings"
)

var (
	debugRequest = os.Getenv("LOG_LEVEL") == "trace"
	httpClient   = &http.Client{}
)

const xRHIdentity = "x-rh-identity"

// make client on demand with specified identity
func makeClient(identity string) *api.Client {
	client := api.Client{
		HTTPClient:     httpClient,
		Debug:          debugRequest,
		DefaultHeaders: map[string]string{xRHIdentity: identity},
	}
	return &client
}

func makeRequest(client *api.Client, ctx *context.Context, url, svc string, out interface{}) error {
	res, err := client.Request(ctx, http.MethodGet, url, nil, &out)
	if res != nil && res.Body != nil {
		defer res.Body.Close()
	}

	if err != nil {
		utils.Log("err", err.Error()).Errorf("Call to %s svc failed", svc)
		status := http.StatusInternalServerError
		if res != nil {
			status = res.StatusCode
		}
		serviceErrorCnt.WithLabelValues(strings.ToLower(svc), strconv.Itoa(status)).Inc()
	}
	return err
}
