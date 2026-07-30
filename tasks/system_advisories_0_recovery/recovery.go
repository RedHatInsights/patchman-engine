package system_advisories_0_recovery

import (
	"app/base"
	"app/base/core"
	"app/base/mqueue"
	"app/base/utils"
	"app/tasks"
	"time"
)

const (
	systemAdvisoriesPartitions = 32
	systemAdvisoriesRemainder  = 0
	// recoveryBatchSize matches the cutover plan: 500 systems per Kafka message.
	recoveryBatchSize = 500
)

var evalWriter mqueue.Writer

func Configure() {
	core.ConfigureApp()
	evalTopic := utils.FailIfEmpty(utils.CoreCfg.EvalTopic, "EVAL_TOPIC")
	evalWriter = mqueue.NewKafkaWriterFromEnv(evalTopic)
}

func Run() {
	tasks.HandleContextCancel(tasks.WaitAndExit)
	Configure()
	defer utils.LogPanics(true)

	if !tasks.EnableSystemAdvisories0Recovery {
		utils.LogInfo("system_advisories_0_recovery disabled (set system_advisories_0_recovery=true in JOBS_CONFIG), skipping") //nolint:lll
		return
	}

	utils.LogInfo("Starting system_advisories_0 recovery recalc publish")
	if err := publishBucket0Recalc(); err != nil {
		utils.LogError("err", err, "system_advisories_0 recovery failed")
		return
	}
	utils.LogInfo("system_advisories_0 recovery recalc publish finished")
}

func publishBucket0Recalc() error {
	inventoryAIDs, err := getNonStaleBucket0InventoryIDs()
	if err != nil {
		return err
	}
	utils.LogInfo("count", len(inventoryAIDs), "non-stale bucket-0 systems selected for recovery recalc")

	start := time.Now()
	err = mqueue.EvalDataSlice(inventoryAIDs).WriteEventsSkipNotifications(base.Context, evalWriter, recoveryBatchSize)
	if err != nil {
		utils.LogError("err", err, "sending recovery recalc messages failed")
		return err
	}
	utils.LogInfo("count", len(inventoryAIDs), "seconds", time.Since(start).Seconds(),
		"systems sent to recovery recalc with skip_notifications")
	return nil
}

func getNonStaleBucket0InventoryIDs() ([]mqueue.EvalData, error) {
	var inventoryAIDs []mqueue.EvalData
	err := tasks.CancelableDB().Table("system_inventory si").
		Select("si.inventory_id, si.rh_account_id, ra.org_id").
		Joins("JOIN rh_account ra ON ra.id = si.rh_account_id").
		Where("si.stale = false").
		Where("satisfies_hash_partition('system_advisories'::regclass, ?, ?, si.rh_account_id)",
			systemAdvisoriesPartitions, systemAdvisoriesRemainder).
		Order("si.rh_account_id, si.id").
		Scan(&inventoryAIDs).Error
	if err != nil {
		return nil, err
	}
	return inventoryAIDs, nil
}
