package kafka

import (
	"app/base"
	"app/base/mqueue"
	"app/base/utils"
)

var (
	evalWriter       mqueue.Writer
	inventoryIDsChan chan inventoryIDsBatch
)

type inventoryIDsBatch struct {
	InventoryIDs []mqueue.EvalData
}

func TryStartEvalQueue(createWriter mqueue.CreateWriter) {
	evalTopic := utils.FailIfEmpty(utils.CoreCfg.EvalTopic, "EVAL_TOPIC")
	evalWriter = createWriter(evalTopic)
	inventoryIDsChan = make(chan inventoryIDsBatch)
	go runRecalcLoop()
}

func runRecalcLoop() {
	for {
		batch := <-inventoryIDsChan
		sendInventoryIDs(batch.InventoryIDs)
	}
}

func InventoryIDs2InventoryAIDs(accountID int, inventoryIDs []string) []mqueue.EvalData {
	inventoryAIDs := make([]mqueue.EvalData, 0, len(inventoryIDs))
	for _, v := range inventoryIDs {
		inventoryAIDs = append(inventoryAIDs, mqueue.EvalData{InventoryID: v, RhAccountID: accountID})
	}
	return inventoryAIDs
}

func sendInventoryIDs(inventoryIDs mqueue.EvalDataSlice) {
	if len(inventoryIDs) == 0 {
		return
	}

	err := mqueue.SendMessages(base.Context, evalWriter, &inventoryIDs)
	if err != nil {
		utils.LogError("nInventoryIDs", len(inventoryIDs), "err", err.Error(),
			"Inventory IDs sending failed")
	}
}

// Send given systems to re-evaluation.
func RecalcSystems(inventoryAIDs []mqueue.EvalData) {
	batch := inventoryIDsBatch{InventoryIDs: inventoryAIDs}
	utils.LogDebug("systems", inventoryAIDs, "systems sent to recalc")
	inventoryIDsChan <- batch
}
