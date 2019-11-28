package vmaas_client

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
)

// Call VMaaS for updates
func CallVMaaSUpdates(url, payload string) (*VMaaSUpdatesResponse, error) {
	resp, err := http.Post(url, "application/json", bytes.NewBufferString(payload))
	if err != nil {
		return nil, err
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	parsedResp := VMaaSUpdatesResponse{}
	err = json.Unmarshal(bodyBytes, &parsedResp)
	if err != nil {
		return nil, err
	}
	parsedResp.StatusCode = resp.StatusCode
	return &parsedResp, nil
}