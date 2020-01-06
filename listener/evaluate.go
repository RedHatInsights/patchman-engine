package listener

import (
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"context"
	"github.com/RedHatInsights/patchman-clients/vmaas"
	"github.com/antihax/optional"
)


func evaluate(systemId int, accountId int, ctx context.Context, updatesReq vmaas.UpdatesRequest) {
	vmaasCallArgs := vmaas.AppUpdatesHandlerV2PostPostOpts{
		UpdatesRequest: optional.NewInterface(updatesReq),
	}

	vmaasData, resp, err := vmaasClient.UpdatesApi.AppUpdatesHandlerV2PostPost(ctx, &vmaasCallArgs)
	if err != nil {
		utils.Log("err", err.Error()).Error("Saving account into the database")
		return
	}
	err = updateSystemAdvisories(systemId, accountId, vmaasData)
	if err != nil {
		utils.Log("err", err.Error()).Error("Updating system advisories")
		return
	}
	utils.Log("res", resp).Debug("VMAAS query complete")
}

func getReportedAdvisories(vmaasData vmaas.UpdatesV2Response) map[string]bool {
	advisories := map[string]bool{}
	for _, updates := range vmaasData.UpdateList {
		for _, update := range updates.AvailableUpdates {
			advisories[update.Erratum] = true
		}
	}
	return advisories
}

func getStoredAdvisoriesMap(inventoryId string) (*map[string]models.AdvisoryMetadata, error) {
	var advisories []models.AdvisoryMetadata
	query := database.SystemAdvisoriesQuery(inventoryId)
	err := query.Find(&advisories).Error
	if err != nil {
		return nil, err
	}

	advisoriesMap := map[string]models.AdvisoryMetadata{}
	for _, advisory := range advisories {
		advisoriesMap[advisory.Name] = advisory
	}
	return &advisoriesMap, nil
}

func updateSystemAdvisories(systemId int, accountId int, updates vmaas.UpdatesV2Response) error {
	utils.Log().Error("System advisories not yet implemented - Depends on vmaas_sync")
	return nil
}
