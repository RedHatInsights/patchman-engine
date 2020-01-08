package listener

import (
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"context"
	"github.com/RedHatInsights/patchman-clients/vmaas"
	"github.com/antihax/optional"
	"time"
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

func getStoredAdvisoriesMap(systemID int) (*map[string]models.SystemAdvisories, error) {
	var advisories []models.SystemAdvisories
	err := database.SystemAdvisoriesQueryByID(systemID).Preload("Advisory").Find(&advisories).Error
	if err != nil {
		return nil, err
	}

	advisoriesMap := map[string]models.SystemAdvisories{}
	for _, advisory := range advisories {
		advisoriesMap[advisory.Advisory.Name] = advisory
	}
	return &advisoriesMap, nil
}

func getNewAndUnpatchedAdvisories(reported map[string]bool, stored map[string]models.SystemAdvisories) (
	[]string, []int) {
	newAdvisories := []string{}
	unpatchedAdvisories := []int{}
	for reportedAdvisory, _ := range reported {
		if storedAdvisory, found := stored[reportedAdvisory]; found {
			if storedAdvisory.WhenPatched != nil { // this advisory was already patched and now is un-patched again
				unpatchedAdvisories = append(unpatchedAdvisories, storedAdvisory.AdvisoryID)
			}
			utils.Log("advisory", storedAdvisory.Advisory.Name).Debug("still not patched")
		} else {
			newAdvisories = append(newAdvisories, reportedAdvisory)
		}
	}
	return newAdvisories, unpatchedAdvisories
}

func getPatchedAdvisories(reported map[string]bool, stored map[string]models.SystemAdvisories) []int {
	var patchedAdvisories []int
	for storedAdvisory, storedAdvisoryObj := range stored {
		if _, found := reported[storedAdvisory]; found {
			continue
		}

		// advisory contained in reported - it's patched
		if storedAdvisoryObj.WhenPatched != nil {
			// it's already marked as patched
			continue
		}

		// advisory was patched from last evaluation, let's mark it as patched
		patchedAdvisories = append(patchedAdvisories, storedAdvisoryObj.AdvisoryID)
	}
	return patchedAdvisories
}

func updateSystemAdvisoriesWhenPatched(systemID int, advisoryIDs []int, whenPatched *time.Time) error {
	err := database.Db.Model(models.SystemAdvisories{}).
		Where("system_id = ? AND advisory_id IN (?)", systemID, advisoryIDs).
		Update("when_patched", whenPatched).Error
	return err
}

func updateSystemAdvisories(systemId int, accountId int, updates vmaas.UpdatesV2Response) error {
	utils.Log().Error("System advisories not yet implemented - Depends on vmaas_sync")
	return nil
}
