package listener

import (
	"app/base/candlepin"
	"app/base/models"
	"app/base/utils"
	"context"
	"net/http"

	"github.com/pkg/errors"
	"gorm.io/gorm"
)

func getTemplate(db *gorm.DB, accountID int, environments []string) (*int64, error) {
	var templateID *int64
	if len(environments) == 0 {
		// no environments
		return templateID, nil
	}

	// get template ids for given environments
	var environmentTemplates []int64
	err := db.Model(models.Template{}).
		Where("rh_account_id = ? AND environment_id IN (?)", accountID, environments).
		Select("id").
		Scan(&environmentTemplates).Error
	if err != nil {
		return nil, err
	}

	if len(environmentTemplates) == 0 {
		return templateID, nil
	}

	templateID = &environmentTemplates[0]
	if len(environmentTemplates) > 1 {
		utils.LogWarn(
			"account", accountID, "environments", environments, "templates", environmentTemplates,
			"Multiple templates found in account rhsm environments",
		)
	}
	return templateID, nil
}

func callCandlepinEnvironment(ctx context.Context, consumer string) (
	*candlepin.ConsumersDetailResponse, error) {
	candlepinEnvConsumersURL := utils.CoreCfg.CandlepinAddress + "/consumers/" + consumer
	candlepinFunc := func() (interface{}, *http.Response, error) {
		candlepinResp := candlepin.ConsumersDetailResponse{}
		resp, err := candlepinClient.Request(&ctx, http.MethodGet, candlepinEnvConsumersURL, nil, &candlepinResp)
		statusCode := utils.TryGetStatusCode(resp)
		utils.LogDebug("candlepin_url", candlepinEnvConsumersURL, "status_code", statusCode, "err", err)
		if err != nil {
			err = errors.Wrap(candlepin.ErrCandlepin, err.Error())
		} else if statusCode != http.StatusOK && statusCode != http.StatusNoContent {
			err = errors.Errorf("candlepin API status %d", statusCode)
		}
		return &candlepinResp, resp, err
	}

	candlepinRespPtr, err := utils.HTTPCallRetry(candlepinFunc,
		candlepin.CandlepinExpRetries, candlepin.CandlepinRetries, http.StatusServiceUnavailable)
	if err != nil {
		return nil, errors.Wrap(err, "candlepin /consumers call failed")
	}
	return candlepinRespPtr.(*candlepin.ConsumersDetailResponse), nil
}
