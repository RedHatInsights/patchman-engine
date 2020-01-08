package evaluator

import (
	"app/base"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"context"
	"github.com/RedHatInsights/patchman-clients/vmaas"
	"github.com/antihax/optional"
	"time"
)

const unknown = "unknown"

var (
	vmaasClient     *vmaas.APIClient
)

func Configure() {
	traceApi := utils.GetenvOrFail("LOG_LEVEL") == "trace"

	vmaasConfig := vmaas.NewConfiguration()
	vmaasConfig.BasePath = utils.GetenvOrFail("VMAAS_ADDRESS") + base.VMAAS_API_PREFIX
	vmaasConfig.Debug = traceApi
	vmaasClient = vmaas.NewAPIClient(vmaasConfig)
}

func Evaluate(systemId int, accountId int, ctx context.Context, updatesReq vmaas.UpdatesRequest) {
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

// Return advisory IDs, created advisories count, error
func ensureAdvisoriesInDb(advisories []string) (*[]int, int, error) {
	var existingAdvisories []models.AdvisoryMetadata
	err := database.Db.Where("name IN (?)", advisories).Find(&existingAdvisories).Error
	if err != nil {
		return nil, 0, err
	}

	var existingAdvisoryIDs []int
	for _, existingAdvisory := range existingAdvisories {
		existingAdvisoryIDs = append(existingAdvisoryIDs, existingAdvisory.ID)
	}

	if len(existingAdvisories) == len(advisories) {
		// all advisories are in database
		return &existingAdvisoryIDs, 0, nil
	}

	createdAdvisoryIDs, err := createNewAdvisories(advisories, existingAdvisories)
	if err != nil {
		return nil, 0, err
	}
	existingAdvisoryIDs = append(existingAdvisoryIDs, *createdAdvisoryIDs...)

	return &existingAdvisoryIDs, len(*createdAdvisoryIDs), nil
}

// Return created advisories IDs, created advisories, error
func createNewAdvisories(advisories []string, existingAdvisories []models.AdvisoryMetadata) (
	*[]int, error) {
	existingAdvisoriesMap := map[string]bool{}
	for _, advisoryObj := range existingAdvisories {
		existingAdvisoriesMap[advisoryObj.Name] = true
	}

	var createdAdvisoryIDs []int
	for _, advisory := range advisories {
		if _, found := existingAdvisoriesMap[advisory]; found {
			continue // advisory is already stored in database
		}

		item := models.AdvisoryMetadata{Name: advisory,
			Description: unknown, Synopsis: unknown, Summary: unknown, Solution: unknown}
		err := database.Db.Create(&item).Error
		if err != nil {
			return nil, err
		}
		createdAdvisoryIDs = append(createdAdvisoryIDs, item.ID)
		utils.Log("advisory", advisory).Info("unknown advisory created")
	}

	return &createdAdvisoryIDs, nil
}

func addNewSystemAdvisories(systemID int, advisoryIDs []int) error {
	advisoriesObjs := models.SystemAdvisoriesSlice{}
	for _, advisoryID := range advisoryIDs {
		advisoriesObjs = append(advisoriesObjs,
			models.SystemAdvisories{SystemID: systemID, AdvisoryID: advisoryID})
	}

	interfaceSlice := advisoriesObjs.ToInterfaceSlice()
	err := database.BulkInsert(database.Db, interfaceSlice)
	if err != nil {
		return err
	}
	return nil
}

func updateSystemAdvisories(systemId int, accountId int, updates vmaas.UpdatesV2Response) error {
	utils.Log().Error("System advisories not yet implemented - Depends on vmaas_sync")
	return nil
}
