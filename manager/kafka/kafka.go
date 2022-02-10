package kafka

import (
	"app/base"
	"app/base/database"
	"app/base/models"
	"app/base/mqueue"
	"app/base/utils"
)

var (
	evalWriter               mqueue.Writer
	baselinesChannel         chan baselineAndAccount
	enableEvaluationRequests = utils.GetBoolEnvOrDefault("ENABLE_EVALUATION_REQUESTS", true)
)

type baselineAndAccount struct {
	BaselineID *int
	AccountID  int
}

func TryStartEvalQueue(createWriter mqueue.CreateWriter) {
	if !enableEvaluationRequests {
		return
	}
	evalTopic := utils.GetenvOrFail("EVAL_TOPIC")
	evalWriter = createWriter(evalTopic)
	baselinesChannel = make(chan baselineAndAccount)
	go runBaselineRecalcLoop()
}

func runBaselineRecalcLoop() {
	for {
		baseline := <-baselinesChannel
		inventoryIDs := getInventoryIDs(baseline.BaselineID, baseline.AccountID)
		sendInventoryIDs(inventoryIDs)
	}
}

func getInventoryIDs(baselineID *int, rhAccountID int) []mqueue.InventoryAID {
	var inventoryAIDs []mqueue.InventoryAID
	err := database.Db.Model(&models.SystemPlatform{}).
		Select("inventory_id, rh_account_id").
		Where(map[string]interface{}{"rh_account_id": rhAccountID, "baseline_id": baselineID}).
		Order("inventory_id").
		Scan(&inventoryAIDs).Error
	if err != nil {
		utils.Log("baselineID", baselineID, "err", err.Error()).
			Error("Unable to load inventory IDs for baseline")
	}
	utils.Log("nInventoryIDs", len(inventoryAIDs), "rhAccountID", rhAccountID).
		Debug("Loaded inventory IDs to evaluate")
	return inventoryAIDs
}

func sendInventoryIDs(inventoryIDs []mqueue.InventoryAID) {
	if len(inventoryIDs) == 0 {
		return
	}

	err := mqueue.SendMessages(base.Context, evalWriter, inventoryIDs...)
	if err != nil {
		utils.Log("nInventoryIDs", len(inventoryIDs), "err", err.Error()).
			Error("Inventory IDs sending failed")
	}
}

// Send all account systems of given baseline to evaluation.
// Evaluate all account systems with no baseline if baselineID is nil (used for deleted baseline).
func EvaluateBaselineSystems(baselineID *int, accountID int) {
	if !enableEvaluationRequests {
		return
	}
	baselinesChannel <- baselineAndAccount{BaselineID: baselineID, AccountID: accountID}
}
