package vmaas_sync //nolint:revive,stylecheck

import (
	"app/base"
	"app/base/database"
	"app/base/types"
	"app/base/types/vmaas"
	"app/base/utils"
	"net/http"

	"github.com/pkg/errors"
)

func isSyncNeeded() bool {
	if vmaasClient == nil {
		panic("VMaaS client is nil")
	}

	ts, err := database.GetTimestampKVValue(LastSync)
	if err != nil || ts == nil {
		utils.Log("ts", ts, "err", err).Info("Last sync disabled - sync needed")
		return true
	}

	var emptyTS types.Rfc3339TimestampNoT
	dbchange, err := vmaasDBChangeRequest()
	if err != nil {
		utils.Log("err", err).Error("Could'n query vmaas dbchange")
		return true
	}
	// check only `exported` timestamp
	// we can possibly check each timestamp to reduce syncing of CVEs, errata, and repos
	exported := dbchange.GetExported()
	utils.Log("last sync", ts, "dbchange.exported", exported).Info()
	switch {
	case exported == emptyTS:
		return true
	case ts.Time().Before(*exported.Time()):
		return true
	default:
		utils.Log().Info("No need to sync vmaas")
		return false
	}
}

func vmaasDBChangeRequest() (*vmaas.DBChangeResponse, error) {
	vmaasCallFunc := func() (interface{}, *http.Response, error) {
		response := vmaas.DBChangeResponse{}
		resp, err := vmaasClient.Request(&base.Context, http.MethodGet, vmaasDBChangeURL, nil, &response)
		return &response, resp, err
	}

	vmaasDataPtr, err := utils.HTTPCallRetry(base.Context, vmaasCallFunc, vmaasCallExpRetry, vmaasCallMaxRetries)
	if err != nil {
		vmaasCallCnt.WithLabelValues("error-dbchange").Inc()
		return nil, errors.Wrap(err, "Checking DBChange")
	}
	vmaasCallCnt.WithLabelValues("success").Inc()
	return vmaasDataPtr.(*vmaas.DBChangeResponse), nil
}
