package vmaas_sync //nolint:revive,stylecheck

import (
	"app/base"
	"app/base/types"
	"app/base/utils"
	"app/base/vmaas"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

func isSyncNeeded(lastSyncTS *string) bool {
	if vmaasClient == nil {
		panic("VMaaS client is nil")
	}
	if lastSyncTS == nil {
		utils.Log("lastSyncTS", lastSyncTS).Info("Last sync disabled - sync needed")
		return true
	}

	var emptyTS types.Rfc3339Timestamp
	ts, err := time.Parse(types.Rfc3339NoTz, *lastSyncTS)
	if err != nil {
		utils.Log("err", err).Error("Couldn't parse `lastSyncTS` timestamp")
		return true
	}
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
	case ts.Before(*exported.Time()):
		return true
	default:
		utils.Log().Info("No need to sync vmaas")
		return false
	}
}

func vmaasDBChangeRequest() (*vmaas.DBChangeResponse, error) {
	vmaasCallFunc := func() (interface{}, *http.Response, error) {
		resp, err := vmaasClient.Request(&base.Context, http.MethodGet, vmaasDBChangeURL, nil, nil)
		return nil, resp, err
	}

	vmaasDataPtr, err := utils.HTTPCallRetry(base.Context, vmaasCallFunc, vmaasCallExpRetry, vmaasCallMaxRetries)
	if err != nil {
		vmaasCallCnt.WithLabelValues("error-dbchange").Inc()
		return nil, errors.Wrap(err, "Checking DBChange")
	}
	vmaasCallCnt.WithLabelValues("success").Inc()
	return vmaasDataPtr.(*vmaas.DBChangeResponse), nil
}
